# TODO：技术改造计划

> 文档版本：v1.0 | 创建日期：2026-03-19
> 记录系统后期需要引入的技术改造、新技术评估与基础设施升级计划。

---

## 一、后端分页（优先级：🔴 高）

### 问题背景

当前所有列表查询接口均返回全量数据，前端通过 `usePagination.ts` 做内存分页。随着数据增长，以下场景会出现严重性能问题：

| 场景 | 预估数据量 | 影响 |
|------|-----------|------|
| 组织成员列表 | 数千人 | 首次加载慢、内存占用高 |
| 审核日志 | 每天新增数十~数百条 | 数月后查询缓慢 |
| 归档日志 | 周期性批量写入 | 同上 |
| 定时任务日志 | 每日多次累积 | 同上 |
| 用户个人配置列表 | 等于用户数 | 大型租户下列表卡顿 |
| Token 消耗日志 | 每次 AI 调用一条 | 增长最快的表 |

### 改造方案

#### 1. Repository 层增加分页接口

```go
// 通用分页参数
type PageQuery struct {
    Page     int    `form:"page" binding:"min=1"`
    PageSize int    `form:"page_size" binding:"min=1,max=100"`
    SortBy   string `form:"sort_by"`
    SortDir  string `form:"sort_dir" binding:"oneof=asc desc"`
    Keyword  string `form:"keyword"`
}

// 通用分页结果
type PageResult[T any] struct {
    Items    []T   `json:"items"`
    Total    int64 `json:"total"`
    Page     int   `json:"page"`
    PageSize int   `json:"page_size"`
}
```

#### 2. 前端适配

```typescript
// composable 调用改造
const { data, total, loading } = useServerPagination('/api/tenant/org/members', {
  page: currentPage,
  pageSize: 20,
  keyword: searchText,
})
```

#### 3. 改造优先级

| 优先级 | 接口 | 原因 |
|--------|------|------|
| P0 | 日志查询（audit/cron/archive_logs） | 数据增长最快 |
| P0 | Token 消耗日志 | 每次 AI 调用写入 |
| P1 | 组织成员列表 | 大型租户必需 |
| P1 | 用户个人配置列表 | 等于用户数 |
| P2 | 配置列表（audit/archive configs） | 通常<50条，可后期改造 |
| P2 | 选项/系统配置查询 | 数据量极小 |

### 注意事项

- [ ] 后端分页需同步支持 **排序**（`ORDER BY`）和 **搜索**（`WHERE ... LIKE`）
- [ ] 前端 `usePagination.ts` 保留，低数据量场景仍可使用前端分页
- [ ] 分页 API 的 `total` 字段应使用 `COUNT(*)` 独立查询（避免大表全扫描时考虑缓存 count）
- [ ] 审核日志等大表查询需加 **时间范围** 必传参数，避免无界查询

---

## 二、Redis 场景扩展（优先级：🟡 中）

### 当前 Redis 使用情况

| 用途 | 说明 |
|------|------|
| JWT 黑名单 | 登出时将 Access Token 写入 Redis，TTL = 剩余有效期 |
| 登录锁定 | 连续登录失败次数计数（可能未实际使用 Redis，使用的是 DB 字段） |

**当前 Redis 利用率极低**，仅用于 JWT 黑名单。

### 可扩展场景评估

| 场景 | 必要性 | 复杂度 | 说明 |
|------|--------|--------|------|
| **Token 配额缓存** | 🔴 高 | 中 | 当前 Token 配额检查走 DB 原子 UPDATE，高并发时为瓶颈 |
| **API 限流** | 🔴 高 | 低 | 基于 Redis 令牌桶/滑动窗口实现，保护 AI 调用接口 |
| **OA 数据缓存** | 🟡 中 | 中 | 字段元数据、流程列表等不频繁变化的数据缓存 |
| **仪表盘统计缓存** | 🟡 中 | 低 | 聚合查询结果缓存（TTL=5~10分钟） |
| **分布式锁** | 🟡 中 | 低 | 定时任务执行锁，防止多实例重复执行 |
| **会话存储** | 🟢 低 | 低 | 当前用 JWT 无状态认证，无需 session |
| **消息队列替代** | 🟢 低 | 中 | Redis Stream 可做轻量 MQ，但不如专业 MQ |

### 推荐优先实现

#### 场景 1：Token 配额缓存

```
当前：每次 AI 调用 → UPDATE tenants WHERE token_used + N <= token_quota（DB 行锁）
改造：
  1. 启动时加载配额到 Redis：HSET tenant:{id} quota 10000 used 500
  2. 预扣：HINCRBY tenant:{id} used N → 检查是否超限
  3. 结算：HINCRBY tenant:{id} used (actual-reserved)
  4. 异步同步：定期将 Redis 值刷回 DB（每 30s 或每 100 次调用）
```

优点：大幅降低 DB 锁竞争，提高并发能力。

#### 场景 2：API 限流

```go
// 基于 Redis 的滑动窗口限流
middleware.RateLimiter(rdb, "ai_chat", 10, time.Minute) // 每分钟 10 次
```

推荐保护的接口：
- `POST /api/audit/execute` — AI 审核执行
- `POST /api/auth/login` — 登录（防暴力破解，替代当前的 DB 计数方式）
- `POST /api/*/test` — 连接测试类接口

#### 场景 3：仪表盘统计缓存

```
GET /api/dashboard/audit-summary
  → Redis GET dashboard:{tenant_id}:audit_summary
    → HIT: 返回缓存
    → MISS: 执行 SQL 聚合 → SET 结果，TTL=5min → 返回
```

### 是否需要引入

**结论：需要，但分阶段引入。**

- **第一阶段**（API 限流）：成本低，收益高，建议立即实现
- **第二阶段**（Token 缓存 + 仪表盘缓存）：等 AI 执行链路打通后实现
- **第三阶段**（OA 缓存 + 分布式锁）：等定时任务和多实例部署时实现

---

## 三、消息队列（优先级：🟡 中）

### 问题背景

AI 模型调用是系统中最重的操作（单次 2~10 秒），当前采用同步调用：

```
HTTP 请求 → Handler → Service.Chat() → 等待 AI 响应 → 返回
```

**问题**：
1. 批量审核时，前端长时间等待（N 条 × 5s = 长时间阻塞）
2. HTTP 连接超时风险（Nginx 默认 60s，多条流程可能超时）
3. AI 服务不稳定时，重试逻辑在 HTTP 请求生命周期内执行，影响用户体验

### 改造方案

引入消息队列，将 AI 调用异步化：

```
┌───────────┐    ┌──────────┐    ┌───────────┐    ┌────────────┐
│  Handler  │───►│  MQ      │───►│  Worker   │───►│  AI Model  │
│ (快速返回) │    │ (队列)   │    │ (消费者)  │    │  (调用)    │
└───────────┘    └──────────┘    └─────┬─────┘    └────────────┘
      ▲                               │
      │         ┌──────────┐           │
      └─────────│ WebSocket│◄──────────┘
                │ /SSE 推送│  (结果通知)
                └──────────┘
```

### 技术选型评估

| 方案 | 优点 | 缺点 | 适合场景 |
|------|------|------|---------|
| **Redis Stream** | 无额外依赖、Go 客户端已有 | 持久化能力弱、无死信队列 | 轻量场景、快速落地 |
| **RabbitMQ** | 成熟稳定、功能丰富 | 额外运维成本 | 中大型系统 |
| **NATS** | Go 原生、性能极高 | 生态相对小 | Go 微服务 |
| **Go Channel + goroutine** | 零依赖 | 单实例、无持久化 | MVP 阶段 |

### 推荐方案

**第一阶段**：Go Channel + goroutine pool（零依赖，快速实现）

```go
type AuditJobQueue struct {
    jobs chan AuditJob
    wg   sync.WaitGroup
}

func (q *AuditJobQueue) Submit(job AuditJob) string {
    jobID := uuid.New().String()
    q.jobs <- AuditJob{ID: jobID, ...}
    return jobID // 立即返回 jobID
}

// 客户端轮询 GET /api/audit/jobs/{jobID}/status
```

**第二阶段**：Redis Stream（引入持久化、多消费者组）

**第三阶段**：如果部署规模扩大，考虑 NATS 或 RabbitMQ

### 需要异步化的接口

| 接口 | 原因 |
|------|------|
| 单条审核执行 | AI 调用耗时 2~10s |
| 批量审核执行 | 多条累积耗时 |
| 归档复盘执行 | 同审核 |
| 附件 OCR | 耗时更长 |

---

## 四、OCR 附件识别能力（优先级：🟡 中）

### 需求场景

OA 流程中常有附件（合同扫描件、发票图片、证照文件），当前系统仅审核结构化表单数据，无法识别附件内容。

### 技术方案

```
OA 附件 → 下载到本地/内存 → OCR 识别 → 文本提取 → 合并到审核提示词
```

### OCR 技术选型

| 方案 | 优点 | 缺点 | 推荐度 |
|------|------|------|--------|
| **PaddleOCR（Python）** | 中文识别率高、免费、本地部署 | 需要 Python 服务 | ⭐⭐⭐⭐⭐ |
| **Tesseract（Go/C++）** | 开源、多语言 | 中文识别率一般 | ⭐⭐⭐ |
| **阿里云 OCR API** | 精度高、无需部署 | 收费、依赖网络 | ⭐⭐⭐⭐ |
| **多模态 LLM** | 直接"看"图片 | 成本高、速度慢 | ⭐⭐ |

### 推荐方案

结合 Python AI 服务（`ai-service`），在 Python 端集成 PaddleOCR：

```
Go 后端                        Python AI 服务
   │                                │
   ├─ 下载 OA 附件                   │
   ├─ POST /api/v1/ocr             │
   │   body: {file: base64}  ────►  ├─ PaddleOCR 识别
   │                                ├─ 返回识别文本
   │◄──────────────────────────────  │
   ├─ 合并到审核提示词                │
   ├─ POST /api/v1/chat ──────────► ├─ AI 审核
   └─ 返回结果                       └─
```

### 实现步骤

- [ ] **OA 适配器扩展**：`OAAdapter` 接口新增 `FetchAttachments(ctx, processID) ([]Attachment, error)`
- [ ] **附件下载**：从 OA 数据库中读取附件存储路径或 Blob，下载到临时目录
- [ ] **OCR 服务**：在 Python AI 服务中集成 PaddleOCR，提供 `/api/v1/ocr` 接口
- [ ] **文本合并**：将 OCR 结果合并到用户提示词中（建议放在 `{{attachments}}` 模板变量中）
- [ ] **提示词模板更新**：添加 `{{attachments}}` 变量支持
- [ ] **前端展示**：审核结果中标注「附件识别内容」

### 注意事项

- 附件文件可能很大（合同 PDF 几十 MB），需限制处理大小
- OCR 识别耗时较长（1~5 秒/页），需考虑异步处理
- 敏感附件（如合同）的脱敏处理
- 支持的格式：PDF、图片（JPG/PNG）、Word、Excel

---

## 五、Python AI 服务实现（优先级：🟡 中）

### 当前状态

`docker-compose.yml` 中已定义 `ai-service` 容器，Go 后端 `ChatViaPython()` 已实现 HTTP 调用逻辑，但 `ai-service/` 目录为空。

### 推荐技术栈

| 组件 | 选型 |
|------|------|
| Web 框架 | FastAPI |
| AI 调用 | LangChain / OpenAI SDK |
| OCR | PaddleOCR |
| 向量数据库 | PostgreSQL pgvector（复用现有 DB） |
| 文档解析 | Unstructured / PyPDF2 |

### 核心接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/chat/completions` | 转发 AI 调用（支持 RAG 增强） |
| POST | `/api/v1/ocr` | OCR 文本识别 |
| POST | `/api/v1/embeddings` | 文档向量化（RAG 用） |
| GET | `/health` | 健康检查 |

### 实现步骤

- [ ] 初始化 FastAPI 项目结构
- [ ] 实现 `/api/v1/chat/completions`：接收 Go 后端请求，调用 LLM
- [ ] 实现 OCR 接口
- [ ] 实现 RAG 增强（知识库规则向量检索）
- [ ] Dockerfile 与 docker-compose 集成
- [ ] 健康检查与日志

---

## 六、监控与可观测性（优先级：🟢 低-中）

### 当前状态

- 使用 `zap` 结构化日志
- 无 Metrics 采集
- 无 Distributed Tracing
- `trace_id` 字段已在响应中，但未与 Tracing 系统集成

### 改造建议

| 层面 | 方案 | 说明 |
|------|------|------|
| **Metrics** | Prometheus + Grafana | Go gin-contrib/prometheus 中间件 |
| **Tracing** | OpenTelemetry + Jaeger | trace_id 与日志/API 响应关联 |
| **日志聚合** | ELK / Loki | 集中化日志查询 |
| **健康检查** | `/api/health/detailed` | 返回 DB/Redis/AI Service 连通性 |

### 推荐指标

| 指标 | 类型 | 说明 |
|------|------|------|
| `http_requests_total` | Counter | API 请求总数（按路径/状态码） |
| `http_request_duration_seconds` | Histogram | 请求耗时分布 |
| `ai_call_duration_seconds` | Histogram | AI 调用耗时 |
| `ai_call_total` | Counter | AI 调用次数（按模型/租户） |
| `token_usage_total` | Counter | Token 消耗（按租户/模型） |
| `db_connections_active` | Gauge | 活跃数据库连接数 |
| `redis_connections_active` | Gauge | 活跃 Redis 连接数 |

---

## 七、多实例部署与水平扩展（优先级：🟢 低）

### 当前限制

| 问题 | 说明 |
|------|------|
| 定时任务 | 多实例会重复执行 Cron 任务 |
| Token 配额 | DB 行锁在高并发下成为瓶颈 |
| JWT 黑名单 | 已使用 Redis，多实例安全 ✅ |
| OA 连接 | 每个实例独立创建连接，无连接池共享 |

### 改造方向

- [ ] **分布式锁**：Cron 任务执行前先 `SET NX` 抢锁
- [ ] **Token 配额移至 Redis**：见上文 Redis 章节
- [ ] **无状态化**：确保所有实例无本地状态依赖
- [ ] **负载均衡**：Nginx upstream + health check

---

## 八、数据备份与恢复（优先级：🟢 低）

### 当前状态

`system_configs` 中已有备份相关配置：
- `system.backup_enabled` = false
- `system.backup_cron` = "0 2 * * *"（每天凌晨2点）
- `system.backup_retention_days` = 30

**但备份功能完全未实现。**

### 待实现

- [ ] **PostgreSQL 定期备份**：`pg_dump` 脚本 + Cron
- [ ] **备份存储**：本地目录 / OSS / S3
- [ ] **备份恢复**：提供恢复脚本或管理界面
- [ ] **备份通知**：成功/失败邮件通知
- [ ] **数据保留策略**：根据 `backup_retention_days` 清理过期备份

---

## 九、技术选型总结与优先级排序

| 优先级 | 改造项 | 预估工时 | 依赖 |
|--------|--------|---------|------|
| 🔴 P0 | 后端分页（日志/成员列表） | 2~3天 | 无 |
| 🔴 P0 | Redis API 限流 | 1天 | 无 |
| 🟠 P1 | 审核执行完整链路 | 5~7天 | OA 适配器扩展 |
| 🟠 P1 | 异步任务队列（Go Channel） | 2~3天 | 审核链路 |
| 🟡 P2 | Redis Token 配额缓存 | 1~2天 | 审核链路 |
| 🟡 P2 | 邮件发送服务 | 2天 | SMTP 配置 |
| 🟡 P2 | 定时任务调度引擎 | 3~5天 | 审核链路 + 邮件 |
| 🟡 P2 | Python AI 服务基础框架 | 3~5天 | 无 |
| 🟡 P2 | 仪表盘统计接口 | 3~5天 | 审核/归档链路 |
| 🟢 P3 | OCR 附件识别 | 5~7天 | Python AI 服务 |
| 🟢 P3 | 监控（Prometheus + Grafana） | 2~3天 | 无 |
| 🟢 P3 | 数据备份自动化 | 1~2天 | 无 |
| 🟢 P3 | 多实例分布式锁 | 1天 | Redis |
