# AI 交互接口文档

> OA 智审平台核心功能 — 流程智能审核 AI 交互设计

---

## 1. 设计哲学

### 1.1 为什么需要多轮交互？

LLM 在自由推理时质量最高，但在严格结构化输出时容易丢失推理深度。反过来，如果只要求结构化输出，模型可能"偷懒"跳过深度分析。

因此本平台采用 **"先推理，后结构化"** 的两阶段编排：

| 阶段 | 目的 | 输出形式 | 模型要求 |
|------|------|---------|---------|
| 第一阶段：深度推理 | 让 AI 自由分析表单数据、逐条审视规则、结合审批流上下文进行完整推理 | 自由文本（Markdown） | 推理能力强的主模型 |
| 第二阶段：结构化提取 | 将第一阶段的推理结果 + 原始规则列表一起输入，要求严格按 JSON Schema 输出得分、建议选项、逐条规则校验结果 | 严格 JSON | 可用同一模型的 JSON mode，或更轻量的模型 |

### 1.2 为什么不是三次？

你可能会想把"得分+建议"和"规则校验"拆成两次。但仔细想：

- 得分本身就是规则校验结果的汇总（通过率 × 权重 = 得分）
- 建议选项（通过/退回/驳回）也是基于规则校验的综合判断
- 这三者有强因果关系，放在同一次结构化提取中反而更一致

所以：**2 次交互是最优解**。第一次做深度思考，第二次做精确提取。

### 1.3 什么时候可以只用 1 次？

当模型能力足够强（如 GPT-4o、Qwen2.5-72B）且规则数量较少时，可以在单次调用中同时完成推理和结构化输出。系统应支持配置：

- `interaction_mode: "two_phase"` — 默认，两阶段编排
- `interaction_mode: "single_pass"` — 单次调用，适合强模型 + 简单场景

---

## 2. 交互流程详解

### 2.1 第一阶段：深度推理（Reasoning Phase）

**输入构成**：
```
系统提示词（含审核尺度预设指令，由后端根据当前尺度自动追加）
+ 主表字段数据（{{main_table}}）
+ 明细表数据（{{detail_tables}}，可能多个）
+ 生效规则列表（{{rules}}，含规则内容和优先级）
+ 已过审批流节点（{{flow_history}}，仅当有规则关联审批流时）
+ 流程图完整节点信息（{{flow_graph}}，包含全部审批节点、顺序和状态）
+ 当前审批节点（{{current_node}}）
```

**要求模型输出**：自由格式的分析文本，但需覆盖：
- 对每条规则的判断思路
- 发现的风险点
- 整体合规性评估
- 改进建议

**关键点**：这一步不要求任何格式约束，让模型充分"思考"。

### 2.2 第二阶段：结构化提取（Extraction Phase）

**输入构成**：
```
第一阶段的完整推理文本
+ 原始规则列表（用于对齐 rule_id）
+ 严格的 JSON Schema 定义
```

**要求模型输出**：严格 JSON，包含三部分：

```json
{
  "recommendation": {
    "action": "return",
    "score": 72,
    "confidence": 0.85
  },
  "rule_checks": [
    {
      "rule_id": "R001",
      "passed": true,
      "reasoning": "采购金额 ¥156,000 未超过部门季度预算上限 ¥200,000"
    }
  ],
  "risk_points": ["供应商未在合格名录中", "缺少竞争性比价材料"],
  "suggestions": ["补充供应商资质证明", "提供至少3家供应商报价"]
}
```

**关键点**：
- 使用 JSON mode / function calling / structured output 强制格式
- `rule_checks` 数组长度必须等于输入规则数量（一一对应）
- `score` 为 0-100 整数
- `action` 必须从枚举值中选择

---

## 3. 接口定义

### 3.1 发起审核

**POST** `/api/audit/execute`

前端调用此接口发起审核。后端内部编排 AI 交互，对前端透明。

**请求体**:
```json
{
  "process_id": "WF-2025-001"
}
```

**响应**（审核完成后）:
```json
{
  "trace_id": "TR-20250610-A3F8",
  "process_id": "WF-2025-001",
  "status": "completed",
  "recommendation": {
    "action": "return",
    "action_label": "建议退回",
    "score": 72,
    "confidence": 0.85
  },
  "rule_checks": {
    "total": 5,
    "passed": 3,
    "failed": 2,
    "details": [
      {
        "rule_id": "R001",
        "rule_name": "预算额度校验",
        "passed": true,
        "reasoning": "采购金额 ¥156,000 未超过部门季度预算上限 ¥200,000",
        "is_locked": true,
        "related_flow": false
      }
    ]
  },
  "ai_analysis": {
    "summary": "该采购申请整体合规性尚可，但存在两个关键问题需要修正。",
    "risk_points": [
      "供应商未在合格名录中",
      "缺少竞争性比价材料"
    ],
    "suggestions": [
      "补充供应商资质证明",
      "提供至少3家供应商报价"
    ],
    "full_reasoning": "该采购申请整体合规性尚可...\n\n1. 供应商资质问题：..."
  },
  "meta": {
    "duration_ms": 3850,
    "model_used": "Qwen2.5-72B",
    "interaction_mode": "two_phase",
    "phase1_duration_ms": 2200,
    "phase2_duration_ms": 1650
  }
}
```

### 3.2 建议选项枚举

`recommendation.action` 必须从以下固定选项中选择：

| action | action_label | 说明 | 典型得分区间 |
|--------|-------------|------|-------------|
| `approve` | 建议通过 | 流程合规，建议直接通过 | 80-100 |
| `return` | 建议退回 | 存在问题需修改后重新提交 | 50-79 |
| `reject` | 建议驳回 | 严重违规，建议直接驳回 | 0-49 |
| `review` | 建议人工复核 | AI 置信度不足，建议人工判断 | 任意（confidence < 0.6） |

得分区间仅为参考，最终由 AI 综合判断。`review` 在 AI 对自身判断不确定时触发。

### 3.3 审核进度查询

**GET** `/api/audit/progress/{trace_id}`

审核耗时较长时，前端轮询获取进度。

**响应**:
```json
{
  "trace_id": "TR-20250610-A3F8",
  "status": "in_progress",
  "current_phase": 1,
  "total_phases": 2,
  "phase_label": "AI 深度分析中...",
  "phases": [
    { "phase": 1, "label": "深度推理分析", "status": "in_progress", "duration_ms": null },
    { "phase": 2, "label": "结构化结果提取", "status": "pending", "duration_ms": null }
  ],
  "partial_result": null
}
```

当第一阶段完成后，`partial_result` 可返回推理文本摘要，前端可提前展示"AI 正在整理结论..."。

---

## 4. 批量审核

### 4.1 发起批量审核

**POST** `/api/audit/batch`

**请求体**:
```json
{
  "process_ids": ["WF-2025-001", "WF-2025-002", "WF-2025-003"]
}
```

**响应**:
```json
{
  "batch_id": "BATCH-20250610-001",
  "total": 3,
  "status": "processing",
  "created_at": "2025-06-10 09:30:00"
}
```

### 4.2 查询批量进度

**GET** `/api/audit/batch/{batch_id}`

**响应**:
```json
{
  "batch_id": "BATCH-20250610-001",
  "total": 3,
  "completed": 2,
  "failed": 0,
  "status": "processing",
  "progress_percent": 66,
  "results": [
    {
      "process_id": "WF-2025-001",
      "status": "completed",
      "recommendation": "return",
      "action_label": "建议退回",
      "score": 72
    },
    {
      "process_id": "WF-2025-002",
      "status": "completed",
      "recommendation": "approve",
      "action_label": "建议通过",
      "score": 88
    },
    {
      "process_id": "WF-2025-003",
      "status": "in_progress",
      "recommendation": null,
      "action_label": null,
      "score": null
    }
  ]
}
```

---

## 5. 审核反馈

**POST** `/api/audit/feedback`

```json
{
  "process_id": "WF-2025-001",
  "trace_id": "TR-20250610-A3F8",
  "adopted": true,
  "action_taken": "return",
  "user_comment": "AI建议合理，已退回要求补充材料"
}
```

---

## 6. 提示词变量与配置关联

### 6.1 提示词变量

系统提示词支持以下变量，运行时由后端替换：

| 变量 | 说明 | 来源 |
|------|------|------|
| `{{main_table}}` | 主表字段数据（JSON） | OA 数据库 |
| `{{detail_tables}}` | 明细表数据（JSON 数组，可能多个明细表） | OA 数据库 |
| `{{rules}}` | 当前生效的规则列表（JSON） | 规则配置 + 用户偏好合并 |
| `{{flow_history}}` | 已过审批流节点信息（JSON） | OA 数据库（仅当有规则 `related_flow=true` 时注入） |
| `{{flow_graph}}` | 流程图完整节点信息（JSON），包含流程的全部审批节点、顺序和状态 | OA 数据库 |
| `{{current_node}}` | 当前审批节点名称 | OA 数据库 |

> 注：`{{process_type}}`（流程类型）已从模板变量中移除，流程类型信息由后端在构建上下文时自动注入。审核尺度不再作为模板变量，而是通过预设提示词机制由后端自动追加到提示词末尾（参见 6.2 节）。

### 6.2 审核尺度对提示词的影响

| 尺度 | 追加到系统提示词的指令 |
|------|---------------------|
| `strict` | "请以最严格的标准审核，任何疑点均应建议退回或驳回，宁可误判也不放过" |
| `standard` | "请以常规标准审核，明确违规项建议退回，存疑项说明理由并给出改进建议" |
| `loose` | "请以宽松标准审核，仅明显违规项建议退回，轻微问题可标注但不影响最终建议结果" |

### 6.3 多模型编排配置

租户管理员可为每个流程配置审核尺度和提示词模板（AI 服务商、模型选型等由系统管理员统一配置）：

```json
{
  "ai_config": {
    "audit_strictness": "standard",
    "reasoning_prompt": "你是一个专业的采购审核助手...\n\n主表数据：{{main_table}}\n明细表数据：{{detail_tables}}\n审核规则：{{rules}}\n审批流历史：{{flow_history}}\n流程图：{{flow_graph}}\n当前节点：{{current_node}}",
    "extraction_prompt": "请根据以上推理分析结果，严格按照 JSON Schema 输出结构化审核结论...\n\n原始规则列表：{{rules}}"
  }
}
```

- `audit_strictness`: 审核尺度，影响 AI 建议倾向（通过/退回/驳回）
- `reasoning_prompt`: 第一阶段推理提示词，支持变量插入，让 AI 进行深度分析
- `extraction_prompt`: 第二阶段提取提示词，要求 AI 输出结构化 JSON 结果

> 注：`reasoning_model`、`extraction_model`、`interaction_mode`、`context_window`、`temperature` 等模型参数由系统管理员在全局 AI 模型管理中统一配置，租户管理员无需关心。

### 6.4 规则与审批流关联

规则配置中的 `related_flow` 字段标识该规则是否需要关联已过的审批流信息：

```json
{
  "id": "R001",
  "rule_content": "金额超过50万需总经理审批",
  "related_flow": true
}
```

当任一规则的 `related_flow` 为 `true` 时，后端会额外查询该流程已经过的审批节点信息，注入到 `{{flow_history}}` 变量中。

---

## 7. 错误处理

| 错误码 | 说明 | 前端处理建议 |
|--------|------|-------------|
| `AI_SERVICE_UNAVAILABLE` | AI 服务不可用 | 提示用户稍后重试 |
| `MODEL_TIMEOUT` | 模型响应超时 | 提供重试按钮 |
| `INVALID_RESPONSE` | AI 返回格式异常 | 后端自动重试一次，仍失败则返回此错误 |
| `TOKEN_QUOTA_EXCEEDED` | 租户 Token 配额已用尽 | 提示联系管理员 |
| `PROCESS_NOT_FOUND` | 流程不存在或无权访问 | 刷新列表 |
| `BATCH_LIMIT_EXCEEDED` | 批量审核数量超过上限（默认 50） | 提示减少选择数量 |

---

## 8. 数据隔离与安全

- 每个用户只能审核自己待办列表中的流程（OA 系统中分配给自己的审批任务）
- 敏感字段（薪资、身份证号等）在 Go 层脱敏后再传给 Python AI 层
- 所有审核记录（含 AI 推理原文）持久化到 MongoDB，不可篡改
- `trace_id` 贯穿整个审核链路，用于审计追溯
