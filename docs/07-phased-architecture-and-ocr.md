# OA 智审平台 — 分阶段架构说明与 OCR 设计

> 文档版本：v1.0 | 更新日期：2026-03-09  
> 本文档说明三个容易产生疑问的设计决策：(1) 定时任务表为何在规则配置阶段提前建立，(2) Chat 与 ChatViaPython 两条调用链路的分工，(3) OCR 解析的触发机制与字段类型关联。

---

## 一、定时任务表提前建立的原因

### 1.1 现象

在"规则配置与 OA 集成"这个 spec 中，数据库迁移 000008 创建了 `cron_tasks` 和 `cron_task_type_configs` 两张表，迁移 000009 创建了 `audit_logs`、`cron_logs`、`archive_logs` 三张表。但本阶段并没有实现定时任务的业务逻辑（没有 Cron 调度 service、没有 handler、没有路由）。

### 1.2 原因

用户个人配置表 `user_personal_configs` 的 JSONB 字段在结构上引用了这些概念：

```json
{
  "audit_details": [...],    // 引用 process_audit_configs、audit_rules
  "cron_details": [...],     // 引用 cron_tasks、cron_task_type_configs
  "archive_details": [...]   // 引用 archive_logs 的流程类型
}
```

如果不提前建表：
- 后续迁移 000010（user_personal_configs）的 JSONB 数据在语义上引用了不存在的表概念，虽然 JSONB 不强制外键，但数据模型文档会出现断层
- 后续实现定时任务功能时，需要在已有迁移序列中间插入新迁移，容易造成迁移顺序混乱
- `cron_logs` 表有外键 `REFERENCES cron_tasks(id)`，必须先建 `cron_tasks`

### 1.3 当前状态

| 内容 | 已完成 | 待后续 spec |
|------|--------|------------|
| 表结构（DDL） | ✅ | — |
| Go Model 定义 | ✅ `cron_task.go`、`audit_log.go` | — |
| Repository 层 | — | ✅ |
| Service 层（Cron 调度、批量审核） | — | ✅ |
| Handler / Router | — | ✅ |
| 前端 cron.vue 页面对接 | — | ✅ |


---

## 二、Chat 与 ChatViaPython 两条调用链路

### 2.1 为什么有两个方法

`AIModelCallerService` 提供了两个调用入口，对应不同的知识库模式（`kb_mode`）：

| 方法 | 调用链路 | 适用 kb_mode | 阶段 |
|------|---------|-------------|------|
| `Chat()` | Go → LLM API（直连） | `rules_only` | 第一阶段 |
| `ChatViaPython()` | Go → Python AI 服务 → LLM API | `rag_only`、`hybrid` | 第二阶段 |

### 2.2 Chat() — Go 直连 LLM

```
┌──────────┐    OpenAI 兼容 API    ┌──────────────┐
│  Go 层   │ ──────────────────── │  LLM 模型    │
│          │                      │  (Xinference  │
│ 1.配额检查│                      │   / 阿里百炼)  │
│ 2.规则合并│                      └──────────────┘
│ 3.Prompt │
│   组装   │
│ 4.直接调用│
│ 5.Token  │
│   统计   │
└──────────┘
```

适用场景：纯规则库模式（`rules_only`）。Go 层把租户规则 + 流程数据拼成 prompt，直接调用 LLM，不需要 Python 参与。这是第一阶段的默认路径。

### 2.3 ChatViaPython() — 经由 Python 服务

```
┌──────────┐   HTTP POST    ┌───────────────┐   LLM API   ┌──────────┐
│  Go 层   │ ─────────────> │  Python 层    │ ──────────> │ LLM 模型 │
│          │                │               │             └──────────┘
│ 1.配额检查│                │ 1.RAG 检索    │
│ 2.数据脱敏│                │   (向量库查询) │
│ 3.构建请求│                │ 2.LangChain   │
│          │                │   链编排       │
│ 4.Token  │  <───────────  │ 3.OCR 解析    │
│   统计   │   响应结果      │   (按需)      │
│ 5.异步   │                │ 4.调用 LLM    │
│   写日志  │                └───────────────┘
└──────────┘
```

适用场景：制度库 RAG 模式（`rag_only`）或混合模式（`hybrid`）。Python 端在调用 LLM 之前需要做 Go 做不了的事情：

| Python 端职责 | 说明 | Go 为什么做不了 |
|--------------|------|----------------|
| RAG 检索 | 从 pgvector 向量库查询制度文档相关片段，拼入 prompt | 需要 embedding 模型 + 向量相似度计算，Python 生态（LangChain、sentence-transformers）成熟 |
| LangChain 编排 | Checklist Chain、Retrieval Chain、并行混合 | LangChain 是 Python 库，Go 没有等价实现 |
| OCR 解析 | 对附件图片/扫描件提取文字 | Tesseract/PaddleOCR 等均为 Python 生态 |

### 2.4 Go 层始终负责的"门卫"工作

无论走哪条链路，以下职责始终在 Go 层完成：

- JWT 鉴权 + 租户隔离
- Token 配额检查（调用前）
- 数据脱敏（`SanitizeText`，调用前）
- Token 用量累加（调用后）
- 异步写入 `tenant_llm_message_logs`（调用后）

### 2.5 路由切换逻辑

未来在审核执行 service 中，按 `process_audit_configs.kb_mode` 字段决定走哪条链路：

```go
switch config.KBMode {
case "rules_only":
    // 第一阶段：Go 直连 LLM
    resp, err = aiCallerService.Chat(c, tenantID, userID, modelCfg, chatReq)
case "rag_only", "hybrid":
    // 第二阶段：经由 Python 服务
    resp, err = aiCallerService.ChatViaPython(c, tenantID, userID, modelCfg, chatReq, auditCtx)
}
```

这就是为什么现在提前写好 `ChatViaPython()` — 等 Python 端实现了，Go 这边只需要加一个 switch case，不用改调用层代码。


---

## 三、OCR 解析设计

### 3.1 触发机制

OCR 不是对所有流程数据都执行的，而是根据字段类型决定是否触发。当流程字段的类型为附件/图片类时，Python 端在 RAG 链路中自动触发 OCR 解析。

### 3.2 字段类型与 OCR 关联

OA 系统（如泛微 E9）的字段定义中包含 `field_type`，不同类型的处理方式不同：

| field_type | 说明 | 处理方式 |
|-----------|------|---------|
| `text` | 单行文本 | 直接使用，无需 OCR |
| `textarea` | 多行文本 | 直接使用，无需 OCR |
| `int` / `float` | 数值 | 直接使用，无需 OCR |
| `date` / `datetime` | 日期时间 | 直接使用，无需 OCR |
| `select` / `checkbox` | 选择项 | 直接使用，无需 OCR |
| `attachment` | 附件文件 | 按文件扩展名判断 → 见下表 |
| `image` | 图片字段 | 触发 OCR |

### 3.3 附件文件的细分处理

当 `field_type = "attachment"` 时，根据文件扩展名进一步判断：

| 文件类型 | 扩展名 | 处理方式 |
|---------|--------|---------|
| PDF（文字版） | `.pdf` | 直接提取文字（PyPDF2 / pdfplumber） |
| PDF（扫描版） | `.pdf` | OCR 提取（先判断是否含可选择文字，无则 OCR） |
| Word 文档 | `.docx` / `.doc` | python-docx 提取文字 |
| Excel 表格 | `.xlsx` / `.xls` | openpyxl / pandas 提取数据 |
| 图片 | `.jpg` / `.png` / `.tiff` / `.bmp` | OCR 提取 |
| 其他 | — | 跳过，记录告警日志 |

### 3.4 OCR 处理流程

```
Python AI 服务收到请求
  │
  ├─ 遍历 process_data 中的每个字段
  │   │
  │   ├─ field_type 为文本/数值/日期类 → 直接拼入 prompt
  │   │
  │   ├─ field_type 为 image → 下载图片 → OCR → 提取文字 → 拼入 prompt
  │   │
  │   └─ field_type 为 attachment → 下载文件 → 按扩展名判断
  │       ├─ PDF/Word/Excel → 文字提取 → 拼入 prompt
  │       └─ 图片类 → OCR → 提取文字 → 拼入 prompt
  │
  └─ 所有字段处理完毕 → 组装最终 prompt → 调用 LLM
```

### 3.5 OCR 技术选型

| 方案 | 说明 | 适用场景 |
|------|------|---------|
| PaddleOCR | 百度开源，中文识别效果好 | 中文为主的 OA 审批附件（推荐） |
| Tesseract | Google 开源，多语言支持 | 英文或混合语言场景 |
| 云端 OCR API | 阿里云/腾讯云 OCR 服务 | 对准确率要求极高、本地资源不足时 |

推荐首选 PaddleOCR，原因：
- OA 审批场景以中文文档为主
- 本地部署，不依赖外部 API，数据不出内网
- Docker 镜像可集成，与 Python AI 服务同容器部署

### 3.6 数据流中的位置

OCR 发生在 Python 端，Go 层完全不感知 OCR 的存在：

```
Go 层                          Python 层
─────                          ─────────
1. 从 OA 拉取流程数据            
   (含附件 URL/ID)              
2. 数据脱敏                     
3. 构建请求体                   
   (process_data 含原始字段)     
4. POST → Python               
                               5. 解析 process_data
                               6. 识别附件/图片字段
                               7. 下载附件 → OCR/文字提取
                               8. 将提取结果替换原始字段值
                               9. RAG 检索（如需要）
                               10. 组装最终 prompt
                               11. 调用 LLM
                               12. 返回结果 → Go
13. Token 统计                  
14. 异步写日志                  
15. 返回前端                    
```

### 3.7 配置项

OCR 相关配置可在 `process_audit_configs.ai_config` 的 JSONB 中扩展：

```json
{
  "audit_strictness": "standard",
  "system_prompt": "...",
  "user_prompt_template": "...",
  "ocr_config": {
    "enabled": true,
    "engine": "paddleocr",
    "language": "ch",
    "max_file_size_mb": 20,
    "supported_formats": ["pdf", "jpg", "png", "docx", "xlsx"],
    "timeout_seconds": 30
  }
}
```

租户管理员可以控制是否启用 OCR、选择引擎、限制文件大小等。

### 3.8 实现阶段

| 阶段 | OCR 状态 |
|------|---------|
| 第一阶段（rules_only） | 不涉及 OCR，Go 直连 LLM，流程数据均为结构化文本 |
| 第二阶段（rag_only / hybrid） | Python 端实现 OCR，按字段类型自动触发 |
