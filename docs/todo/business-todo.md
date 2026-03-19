# TODO：业务功能待办

> 文档版本：v1.0 | 创建日期：2026-03-19
> 记录当前尚未集成到后端的核心业务模块，以及需要完善的业务功能。

---

## 一、审核工作台（overview.vue）— 待后端集成

### 当前状态

前端 `overview.vue` 已完成 UI 开发（约 46KB 代码），包含：
- 待审核流程列表（含搜索/筛选/分页）
- 单条审核触发与结果展示
- 批量审核功能
- 审核结果详情面板（含规则结果、风险点、改进建议）
- 已审核快照列表
- 审核历史标签页（已通过/已退回）

**但所有数据来源均为 `useMockData.ts` 中的 Mock 数据，未与后端对接。**

### 待实现项

- [ ] **OA 流程列表接口**：后端需通过 OA 适配器拉取待审核流程列表（当前 OA 适配器已实现 `FetchProcessData`，但未暴露为 API）
- [ ] **单条审核执行接口**：串联完整链路 —— 拉取 OA 数据 → 构建提示词 → 调用 AI → 解析结果 → 写入 `audit_logs`
- [ ] **批量审核执行接口**：批量调用审核链路，支持进度反馈
- [ ] **审核历史查询接口**：从 `audit_logs` 表读取历史审核记录
- [ ] **审核快照接口**：审核结果的快照保存与查询
- [ ] **前端对接**：用 `authFetch` 替换 Mock 数据调用

### 关键依赖

1. OA 适配器需新增「拉取待审核流程列表」能力（当前仅支持单条 `FetchProcessData`）
2. 提示词完整构建（`BuildPrompt` 当前仅覆盖推理阶段，提取阶段未实现）
3. AI 调用结果的 JSON 解析与容错

---

## 二、定时任务执行 — 待实现

### 当前状态

- ✅ 定时任务配置管理：前后端已完整对接（`cron_config_handler.go` + `useCronApi.ts`）
- ✅ 任务类型预设（6 种）和租户覆盖配置已就绪
- ✅ 前端定时任务页面 `cron.vue` 已具备任务列表展示、Cron 表达式配置等 UI
- ❌ **后端无 Cron 调度引擎** — 未引入 `robfig/cron` 等调度库
- ❌ **任务执行逻辑未实现** — 批量审核/日报/周报的执行代码为空
- ❌ **邮件推送未实现** — SMTP 配置字段已有，但发送功能未编码

### 待实现项

- [ ] **引入 Cron 调度库**：考虑使用 `robfig/cron/v3` 或 `go-co-op/gocron`
- [ ] **批量审核任务**：从 OA 拉取待审核流程 → 逐条/并发调用 AI → 写入 `audit_logs` → 邮件推送结果
- [ ] **日报推送任务**：统计当日审核数据 → 渲染模板 → 发送邮件
- [ ] **周报推送任务**：统计本周审核数据 → 渲染模板 → 发送邮件
- [ ] **归档批量处理**：从 OA 拉取已归档流程 → AI 合规复核 → 写入 `archive_logs`
- [ ] **归档日报/周报推送**：同审核日报/周报
- [ ] **任务实例管理**：创建/激活/停用 `cron_tasks` 记录
- [ ] **执行日志写入**：写入 `cron_logs` 表
- [ ] **邮件发送能力**：基于 `system_configs` 中的 SMTP 配置实现邮件发送

---

## 三、归档复盘（archive.vue）— 待后端集成

### 当前状态

- ✅ 前端 `archive.vue` 已完成 UI 开发（约 42KB）
- ✅ 归档配置管理（字段/规则/访问控制）已完整对接后端
- ❌ **归档复盘执行链路未实现** — 前端展示的已归档流程和复核结果均为 Mock
- ❌ **归档流程列表接口未实现** — 后端无从 OA 拉取已归档流程的 API
- ❌ **归档执行结果存储** — `archive_logs` 表已创建，但无写入逻辑

### 待实现项

- [ ] **已归档流程列表接口**：OA 适配器需新增拉取已归档流程列表能力
- [ ] **归档复盘执行接口**：拉取流程 → 构建复核提示词 → 调用 AI → 解析结果 → 写入 `archive_logs`
- [ ] **归档历史查询接口**：从 `archive_logs` 读取历史
- [ ] **审批流分析**：拉取流程审批流信息（节点完整性检查）
- [ ] **前端对接**：替换 Mock 数据

---

## 四、仪表盘（dashboard.vue）— 待后端集成

### 当前状态

- ✅ 前端 `dashboard.vue` 已完成 UI 开发（约 60KB，最大的页面文件）
- ✅ 支持 15 种可配置的仪表板小部件
- ✅ 仪表板偏好设置（启用/禁用组件、调整大小）已对接后端
- ❌ **所有仪表板数据均为 Mock** — 包括统计数字、趋势图、排行榜

### 待实现项

按角色分组：

**业务用户组件**：
- [ ] `audit_summary` — 审核概览统计（从 `audit_logs` 聚合）
- [ ] `pending_tasks` — 待办任务数（从 OA 拉取）
- [ ] `weekly_trend` — 周审核趋势图（从 `audit_logs` 按日聚合）
- [ ] `cron_tasks` — 定时任务执行概览（从 `cron_tasks`/`cron_logs` 读取）
- [ ] `archive_review` — 归档复盘概览（从 `archive_logs` 聚合）
- [ ] `recent_activity` — 最近动态（从多个日志表合并）

**租户管理员组件**：
- [ ] `dept_distribution` — 部门分布（从 `audit_logs` + `org_members` 联查）
- [ ] `ai_performance` — AI 模型表现（从 `tenant_llm_message_logs` 统计）
- [ ] `tenant_usage` — 资源用量（Token/存储）
- [ ] `user_activity` — 用户活跃排行

**系统管理员组件**：
- [ ] `system_health` — 系统健康状态（需引入监控机制）
- [ ] `tenant_overview` — 租户总览
- [ ] `api_metrics` — API 调用指标（需引入 Prometheus 或自定义统计）
- [ ] `monitor_metrics` — 运行指标
- [ ] `monitor_alerts` — 告警事件

### 推荐实现策略

1. **优先实现业务用户组件**：audit_summary / weekly_trend / recent_activity 可直接从现有日志表聚合
2. **AI/Token 统计**：`tenant_llm_message_logs` 表已设计好聚合索引
3. **系统监控类**：可后期引入 Prometheus + Grafana，初期可用简单的 `/api/health/detailed` 接口

---

## 五、消息与通知 — 待实现

### 当前状态

系统配置中已有 SMTP 相关字段：
- `system.notification_email` — 系统通知发件邮箱
- `system.smtp_host` / `system.smtp_port` / `system.smtp_username` / `system.smtp_ssl`

定时任务模板中已有 `push_email` 字段和邮件内容模板。

**但实际的邮件发送能力完全未实现。**

### 待实现项

- [ ] **邮件发送服务**：基于 `net/smtp` 或第三方库（如 `go-gomail`）实现
- [ ] **邮件模板渲染**：基于 `content_template` 中的模板变量渲染 HTML/Markdown
- [ ] **通知触发机制**：
  - 审核完成通知（发给流程申请人/审批人）
  - 定时任务结果推送（日报/周报）
  - 系统告警通知（管理员）
- [ ] **站内消息系统**：考虑后期新增 `notifications` 表，实现站内消息
- [ ] **前端消息中心**：AppHeader 中的通知图标已预留，需实现消息列表/已读/未读

---

## 六、数据管理 — 待完善

### 当前状态

前端 `admin/tenant/` 页面中有数据管理相关 UI（审核日志/定时日志/归档日志的查看与导出），但：

- 审核/归档日志的查看依赖上述审核/归档执行链路的实现
- 数据导出功能（Excel）前端已引入 `xlsx` 库

### 待实现项

- [ ] **审核日志查询 API**：分页、筛选、时间范围查询 `audit_logs`
- [ ] **归档日志查询 API**：分页、筛选 `archive_logs`
- [ ] **定时任务日志查询 API**：分页、筛选 `cron_logs`
- [ ] **数据导出 API**：后端渲染 Excel/CSV 导出（前端导出在数据量大时性能差）
- [ ] **数据保留策略执行**：根据 `tenants.data_retention_days` / `log_retention_days` 定期清理过期数据
