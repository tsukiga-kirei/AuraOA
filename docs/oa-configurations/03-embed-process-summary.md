# OA 嵌入流程总结侧边栏

在泛微 E9 审批页右侧通过 **iframe** 嵌入 AuraOA 固定页 `/embed/summary`。嵌入页通过 OA 自定义 JS 获取当前流程 `requestid`，再按租户配置自动或手动生成流程总结。

> 该能力只读取 OA 流程数据、附件识别结果和审批流快照，不修改 OA 流程状态。

---

## 1. 能力概览

| 项目 | 说明 |
|------|------|
| 嵌入地址 | `https://<aura-frontend>/embed/summary`（固定，不带参数） |
| 流程编号 | OA 脚本 `WfForm.getBaseInfo().requestid` → `postMessage` |
| 配置入口 | 租户管理 → 规则配置 → 流程总结 |
| 展示入口 | 租户管理 → 数据信息 → 流程总结 |
| 鉴权方式 | Nuxt 服务端代理携带 `X-Embed-Token` + `X-Tenant-Code`，浏览器无需 AuraOA 登录 |

---

## 2. 数据库设计

基础表迁移：`db/migrations/000044_process_summary.up.sql`
细分自动刷新策略：`db/migrations/000051_embed_change_detection.up.sql`

| 表 | 用途 |
|----|------|
| `process_summary_configs` | 租户级流程总结配置，保存流程类型、字段清单、总结块、嵌入开关 |
| `process_summary_logs` | 每次总结执行日志，保存状态、结构化结果、原始模型输出、解析错误、流程快照 |
| `process_summary_snapshots` | 每个流程的有效总结快照，用于数据页主表和历史链路 |

关键字段：

| 字段 | 说明 |
|------|------|
| `summary_blocks` | 总结块数组，每块独立配置标题、字段范围、用户提示词、排序和启用状态 |
| `embed_enabled` | 是否允许 `/embed/summary` 处理该流程类型 |
| `embed_config.auto_summary_on_open` | 第一次打开嵌入页时自动总结 |
| `embed_config.auto_summary_on_data_change` | 总结块使用的字段或附件版本变化后自动刷新 |
| `embed_config.auto_summary_on_return_resubmit` | 退回或重新提交后刷新流程相关总结块 |
| `embed_config.auto_summary_on_flow_change` | 普通审批推进后刷新流程相关总结块，默认关闭 |
| `raw_content` / `parse_error` | 模型未严格返回 JSON 时保留原文并记录兜底解析原因 |

---

## 3. 租户规则配置

进入 **租户管理 → 规则配置 → 流程总结**：

1. 新增流程类型并测试 OA 连接。
2. 同步字段，系统会读取主表字段、明细表字段，并复用附件识别能力。
3. 开启 **OA 嵌入总结**。
4. 配置总结块。

总结块支持两种字段范围：

| 模式 | 说明 |
|------|------|
| 全部字段 | 主表、明细表、附件识别文本全部进入该块上下文 |
| 指定字段 | 仅选择的字段进入该块上下文，适合附件摘要、金额/供应商摘要等局部总结 |

每个总结块的 **系统提示词固定不可编辑**，后端统一要求模型返回：

```json
{
  "title": "总结块标题",
  "content": "Markdown 正文",
  "points": ["要点 1", "要点 2"]
}
```

用户提示词由租户管理员维护，用来描述该块要总结的目标、判断内容或输出侧重点。多块总结会逐块调用模型，最终组合成 `blocks` 数组返回前台。

---

## 4. 附件与字段处理

总结执行时会复用通用的 OA 数据装配链路：

1. 从 OA 只读库按 `requestid` 读取流程主表、明细表。
2. 根据每个总结块的字段范围裁剪上下文；如果任一启用块选择“全部字段”，本次 OA 数据会按全量字段拉取。
3. 遇到附件字段时调用附件识别服务，优先使用已有 OCR / MinerU 识别结果。
4. 浏览按钮字段会解析为可读显示值。
5. 同步保存流程字段、审批流节点与附件文本到 `process_snapshot`，便于数据页追溯。

自动刷新前先执行轻量比较：

- 主表/明细按每个总结块的 `selected_fields` 计算指纹；
- 附件按字段保存 `docId + versionid + imagefileid + 文件名` 指纹；
- 普通批准、批注、转发和节点推进与退回/重提分开判断；
- 只重新调用发生变化的总结块，其他块沿用最近有效结果。

判断附件版本时不下载文件、不调用 MinerU；只有确认附件块需要刷新后才进入附件识别。

---

## 5. 模型返回兜底

后端解析器会处理常见格式偏差：

| 模型输出 | 处理 |
|----------|------|
| 标准 JSON | 正常解析为总结块 |
| ```json 代码块 | 自动剥离 fence 后解析 |
| JSON 前后有说明文字 | 尝试截取第一个 JSON 对象 |
| 缺少 `content` / `points` | 用可识别字段补齐 |
| 完全无法解析 | 将模型原文整体作为该总结块 `content`，并写入 `parse_error` |

因此前台始终可以展示结果；当触发兜底时，嵌入页和数据详情会显示“已使用兜底解析”。

---

## 6. 嵌入流程

### 6.1 iframe

```html
<iframe
  id="aura-embed-summary"
  title="AI 总结"
  src="https://aura.example.com/embed/summary"
  style="width:380px;height:100%;border:0;"
></iframe>
```

### 6.2 OA 自定义 JS

复用示例脚本：[assets/aura-embed-notify.js](./assets/aura-embed-notify.js)

```javascript
var AURA_EMBED_ORIGIN = 'https://aura.example.com';
var IFRAME_IDS = ['aura-embed-audit', 'aura-embed-summary'];
```

脚本会响应两个方向的消息：

| 方向 | type | 说明 |
|------|------|------|
| iframe → OA | `aura-oa-request-requestid` | 嵌入页请求当前 requestid |
| OA → iframe | `aura-oa-requestid` | `{ requestid: '598488' }` |
| runner → OA | `aura-runner-ready` | 隐藏 runner 已完成嵌入会话认证 |
| OA → runner | `aura-oa-refresh-event` | OA 点击保存或提交时安排后台检查 |
| runner → OA | `aura-runner-event-ack` | 返回对应 `event_id`，确认事件请求已完成 |

runner 认证完成后父页才注册 OA 保存/提交事件；未就绪期间不缓存或补发旧操作。已注册事件最多
等待 400ms；收到确认立即放行，超时或 AuraOA 不可用同样放行。确认只表示后台事件请求已完成，
不等待审核或总结的 AI 任务。

### 6.3 执行时序

```text
OA 页面加载
  → 创建隐藏 /embed/runner 并完成嵌入会话认证
  → 父页才注册保存/提交事件；未就绪期间不缓存、不补发旧操作
OA 点击保存或提交
  → 立即冻结 requestid/workflow_id/人员标识/occurred_at_ms
  → 隐藏 /embed/runner 调用 /api/embed/events
  → 延迟读取 OA，按总结块依赖指纹决定是否入队
/embed/summary 可见页加载
  → postMessage 请求 requestid
  → Nuxt /api/embed/summary/context
  → Go /api/embed/summary/context
  → 在展示旧结果前轻量比较历史快照与 OA 上下文（不识别附件正文、不调用 AI）
  → 未变化则直接展示已有结果，已变化或已有后台任务则进入交互队列
  → Redis Stream 异步执行总结
  → 轮询 /api/embed/summary/jobs/:id，SSE 展示原始模型流
  → 展示结构化 blocks
```

---

## 7. 前台页面

| 页面 | 说明 |
|------|------|
| `/embed/summary` | OA 右侧嵌入页，展示流程信息、总结块、自动刷新状态、手动重新总结按钮 |
| `/admin/tenant/rules` | 租户规则配置页，新增“流程总结”tab |
| `/admin/tenant/data` | 数据管理页，新增“流程总结”tab，可筛选、查看总结历史链 |

嵌入页保持窄侧栏设计：流程信息卡、状态徽标、总结块卡片和原始模型输出折叠区，适配 OA 右侧区域。

---

## 8. 接口速查

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/tenant/summary/configs` | 租户总结配置列表 |
| POST | `/api/tenant/summary/configs` | 新增总结配置 |
| PUT | `/api/tenant/summary/configs/:id` | 保存总结配置 |
| POST | `/api/tenant/summary/configs/:id/fetch-fields` | 同步 OA 字段 |
| GET | `/api/embed/summary/context` | 嵌入页上下文 |
| POST | `/api/embed/summary/execute` | 发起嵌入总结 |
| GET | `/api/embed/summary/jobs/:id` | 查询总结任务 |
| GET | `/api/embed/summary/stream/:id` | SSE 原始模型输出 |
| GET | `/api/summary/snapshots` | 数据页总结快照列表 |
| GET | `/api/summary/snapshots/stats` | 数据页统计 |
| GET | `/api/summary/snapshots/:processId/chain` | 总结历史链 |

---

## 9. 本版暂不开放的功能

为保证嵌入流程先稳定上线，本版没有新增：

| 功能 | 处理 |
|------|------|
| 个人偏好 | 不接入 `/api/tenant/settings`，总结规则完全由租户管理员统一配置 |
| 通用 Cron 总结 | 不新增独立 Cron 任务类型；流程配置内可开启近 N 天、每 5/10 分钟等流程级定时扫描 |
| 批量总结工作台 | 暂无独立工作台，仅支持 OA 嵌入页触发 |
| 导出 Excel | 数据页先查看历史链，后续按运营需求再补导出 |

---

## 10. 相关文档

- [OA 嵌入审核侧边栏](./02-embed-audit-sidebar.md)
- [嵌入接口](../api/embed.md)
- [附件识别](./01-attachment-recognition.md)
