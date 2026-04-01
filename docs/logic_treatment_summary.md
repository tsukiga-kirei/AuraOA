# Dashboard vs. Archive 逻辑处理梳理

本文档概述了 **审核工作台 (Dashboard)** 和 **归档复盘 (Archive)** 模块中应用的逻辑流程及优化项。

## 1. 逻辑对比概览

| 功能特性 | 审核工作台 (`dashboard.vue`) | 归档复盘 (`archive.vue`) |
| :--- | :--- | :--- |
| **数据源** | OA 待办/已办表 (`audit_logs`) | OA 已归档表 (`archive_logs`) |
| **核心操作** | `executeAudit` (实时审核) | `executeReview` (定期/手动复盘) |
| **实时反馈** | SSE 流式 + 轮询 (`waitAuditJob`) | SSE 流式 + 轮询 (`waitArchiveJob`) |
| **最新结论** | 初次审核结论 (主表字段) | 快照结论 (`archive_process_snapshots`) |

## 2. 持久化策略 (Persistence)

### 日期范围持久化
两个模块现在都通过 `sessionStorage` 实现了筛选日期的持久化：
- **Dashboard Key**: `oa-smart-audit:dashboard:list-date-range`
- **Archive Key**: `oa-smart-audit:archive:list-date-range`
- **逻辑**: 修改选择后立即保存。如果当前系统时间发生变化（跨天），持久化内容将自动失效，以确保用户看到的是最新的数据。

### 批量状态持久化 (Batch State)
为了支持在批量执行过程中刷新页面：
- **队列存储**: `ids` (任务ID列表), `queueMeta` (元数据), `nextIndex` (进度索引) 均存入 `sessionStorage`。
- **正在执行任务追踪**: `inflightJobId` 被绑定到当前子任务。即使列表重新加载后未能及时获取到 Job ID，中止操作依然可以通过此持久化的 ID 成功发送至后端。

## 3. 核心优化项说明

### 解决“中止失败，任务 ID 缺失”
该错误以往常发生在用户刷新页面后，UI 失去了对异步任务 `audit_logs.id` (或 `archive_logs.id`) 的追踪。
- **解决方案**: `resolveArchiveJobIdForCancel` 工具函数现在优先从当前列表行中寻找 ID，若未获取到（如列表尚未同步），则从持久化的批量状态中回溯查找。

### 失败日志与快照表的一致性
针对“日志状态为 Failed 但结果仍记录入快照表”的问题：
- **前端优化**: 当触发中止时，前端状态立刻本地更新为 `failed`，提供即时反馈。
- **后端保证**: 在 `archive_review_service.go` 中，快照更新严格限定在解析成功 (`parseErr == nil`) 之后。中止操作会通过 CancelContext 阻断解析流程，从而防止产生错误的快照。

## 4. 关键执行流程 (Execution Flow)

### 批量审核/复盘流程
1. **选择**: 用户选择最多 10 条数据。
2. **持久化**: 队列信息存入 `sessionStorage`。
3. **循环执行**:
   - `runBatchLoop` 逐条处理。
   - 每次调用后，第一时间将后端返回的 `jobId` 存入 `inflightJobId`。
4. **自动恢复**: `onMounted` 钩子会检测 `sessionStorage`。若发现未完成的任务 (`nextIndex < ids.length`)，则自动恢复执行。

### 详情追踪逻辑
- **主动发现**: 列表加载后，若项状态本身处于异步状态（如 `assembling`, `reasoning` 等），系统会自动启动 SSE 追踪，无需用户手动点击。
- **资源利用**: 切换选择项时会断开旧的选择项 SSE 连接，确保浏览器端不产生多余的任务积压。
