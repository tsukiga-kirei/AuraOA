# 分页方式分析文档

> 更新日期：2026-04-10

本文档梳理项目中前后端各模块的分页实现方式，分析其合理性，并给出优化建议。

---

## 一、分页方式总览

项目中存在两种分页策略：

| 策略 | 说明 | 适用场景 |
|------|------|----------|
| **后端分页（Server-side）** | 前端传 `page` + `page_size`，后端 SQL `LIMIT/OFFSET` 或 OA SQL 分页，返回 `{ items, total, page, page_size }` | 数据量大、需要筛选排序的列表 |
| **前端分页（Client-side）** | 后端一次返回全量数据，前端用 `usePagination` composable 在内存中切片 | 数据量小（通常 < 200 条）的管理列表 |

---

## 二、各模块分页方式详情

### 2.1 后端分页模块

#### 2.1.1 审核工作台（Dashboard）

- **页面**：`frontend/pages/dashboard.vue`
- **API**：`GET /api/audit/processes?tab=xxx&page=1&page_size=10`
- **前端 composable**：`useAuditApi.listProcesses()` 传递 `page` / `page_size`
- **后端实现**：
  - `pending_ai` / `ai_audited` tab：从 OA 分批拉取全量待办（`FetchTodoListPaged` 每批 500），在 Go 内存中按 tab 分组、筛选后，用 `normalizeAuditPage()` 做内存切片分页
  - `completed` tab：从 `audit_process_snapshots` 表 DB 真分页（`LIMIT/OFFSET`），排除当前待办
- **默认 pageSize**：10
- **分页组件**：`<a-pagination>` 带 `show-size-changer`，可选 10/20/50

#### 2.1.2 归档复盘工作台（Archive）

- **页面**：`frontend/pages/archive.vue`
- **API**：`GET /api/archive/processes?audit_status=xxx&page=1&page_size=20`
- **前端 composable**：`useArchiveReviewApi.listProcesses()` 传递 `page` / `page_size`
- **后端实现**：
  - `unaudited`：从 OA 分批拉取全量已归档流程（`FetchArchivedListPaged` 每批 500），排除已有 snapshot 的，在 Go 内存中切片分页
  - `compliant` / `partially_compliant` / `non_compliant`：从 `archive_process_snapshots` 表 DB 分页（`LIMIT/OFFSET`）；**当存在日期范围筛选时**，会先从 OA 分批拉取该日期范围内的全量流程 ID（`FetchArchivedListPaged` 每批 500），再用 `process_id IN ?` 与 snapshot 表交叉过滤后分页，以保证与 `GetStats` 统计口径一致
- **默认 pageSize**：20
- **分页组件**：`<a-pagination>` 带 `show-size-changer`，可选 10/20/50

#### 2.1.3 数据管理页（Admin Data）

- **页面**：`frontend/pages/admin/tenant/data.vue`
- **API**：
  - `GET /api/audit/snapshots?page=1&page_size=10` — 审核快照
  - `GET /api/archive/snapshots?page=1&page_size=10` — 归档快照
  - `GET /api/tenant/cron/logs?page=1&page_size=10` — 定时任务日志
- **前端 composable**：`useAdminDataApi` 中 `listAuditSnapshots()` / `listArchiveSnapshots()` / `listCronLogs()` 通过 `buildParams(filter)` 传递分页参数
- **后端实现**：全部为 DB 真分页
  - `AuditProcessSnapshotRepo.ListPagedWithUser()` — `LIMIT/OFFSET` + `COUNT`
  - `ArchiveProcessSnapshotRepo.ListPagedWithUser()` — `LIMIT/OFFSET` + `COUNT`
  - `CronLogRepo.ListPagedByTenant()` — `LIMIT/OFFSET` + `COUNT`
- **默认 pageSize**：10
- **分页组件**：`<a-pagination>` 带 `show-size-changer`，可选 10/20/50


### 2.2 前端分页模块

所有前端分页均使用 `frontend/composables/usePagination.ts`，该 composable 接收一个响应式数组，在内存中做 `slice` 切片：

```ts
// usePagination 核心逻辑
const paged = computed(() => {
  const start = (current.value - 1) * pageSize.value
  return unref(source).slice(start, start + pageSize.value)
})
```

#### 2.2.1 组织架构 — 成员列表（Org）

- **页面**：`frontend/pages/admin/tenant/org.vue`
- **API**：后端一次返回全部成员
- **前端**：`filteredMembers`（搜索 + 角色过滤后）→ `usePagination(filteredMembers, 10)`
- **默认 pageSize**：10
- **数据规模**：通常 < 200 人，前端分页合理

#### 2.2.2 系统管理 — 租户成员（System Tenants）

- **页面**：`frontend/pages/admin/system/tenants.vue`
- **API**：后端一次返回租户成员列表
- **前端**：`usePagination(tenantMembers, 10)`
- **默认 pageSize**：10
- **数据规模**：通常 < 100 人，前端分页合理

#### 2.2.3 用户个人配置管理（User Configs）

- **页面**：`frontend/pages/admin/tenant/user-configs.vue`
- **API**：后端一次返回全部用户配置
- **前端**：`filteredConfigs`（搜索 + 状态过滤后）→ `usePagination(filteredConfigs, 10)`
- **默认 pageSize**：10
- **数据规模**：与用户数一致，通常 < 200，前端分页合理

#### 2.2.4 规则配置 — 字段选择器（Rules）

- **页面**：`frontend/pages/admin/tenant/rules.vue`
- **用途**：审核/归档字段穿梭框中的分页（已选/未选/页面展示字段）
- **前端**：多个 `usePagination` 实例，pageSize = 5
- **数据规模**：字段数通常 < 100，前端分页合理

#### 2.2.5 个人设置 — 字段选择器（Settings）

- **页面**：`frontend/pages/settings.vue`
- **用途**：工作台字段选择穿梭框中的分页
- **前端**：多个 `usePagination` 实例，pageSize = 6 或 10
- **数据规模**：字段数通常 < 100，前端分页合理

### 2.3 无分页模块

| 模块 | 页面 | API | 说明 |
|------|------|-----|------|
| 定时任务管理 | `cron.vue` | `GET /api/tenant/cron/tasks` | 返回全部任务实例（通常 < 20），卡片布局无分页 |
| 任务执行日志 | `cron.vue` 抽屉 | `GET /api/tenant/cron/tasks/:id/logs` | 返回最近 50 条，无分页 |
| 审核/归档配置 | `rules.vue` | 各配置 API | 配置数量少，无需分页 |
| 仪表盘概览 | `overview.vue` | 统计 API | 纯统计数据，无列表 |

---

## 三、后端分页响应格式

### 3.1 统一格式（数据管理页 + 工作台）

```json
{
  "items": [...],
  "total": 150,
  "page": 1,
  "page_size": 20
}
```

对应 DTO：
- `dto.AuditProcessListResponse`
- `dto.ArchiveProcessListResponse`
- 数据管理页使用前端类型 `PagedResult<T>`（`{ items, total, page, page_size }`）

### 3.2 参数规范

| 参数 | 说明 | 默认值 | 最大值 |
|------|------|--------|--------|
| `page` | 页码，1-indexed | 1 | 无限制 |
| `page_size` | 每页条数 | 20 | 100（工作台）/ 200（数据管理页） |

---

## 四、问题与不一致

### 4.1 审核工作台 pending/ai_audited 的"伪分页"

**现状**：`pending_ai` 和 `ai_audited` tab 的分页流程为：
1. 从 OA 数据库**分批拉取全量**待办——每次向 OA 请求 500 条（`batchSize = 500`），循环翻页直到拉完所有数据
2. 在 Go 内存中按 tab 分组（pending_ai / ai_audited）、合并本地审核状态
3. 用 `normalizeAuditPage()` 对筛选后的结果数组做 `slice(start, end)` 取出用户请求的那一页（如 20 条）

**问题**：
- 用户只想看 20 条，后端却每次都要从 OA 拉取全量数据（可能数千条），然后丢弃绝大部分
- 每次翻页、切换筛选条件都会重复执行全量拉取，响应慢且对 OA 系统压力大
- OA SQL 本身支持 `LIMIT/OFFSET`（`FetchTodoListPaged` 已实现），但因为需要按 tab 分组（pending_ai vs ai_audited），无法在 OA 层直接分页——OA 不知道哪些流程已被 AI 审核过
- 内存开销随待办数量线性增长

### 4.2 归档复盘 unaudited 的"伪分页"

**现状**：与审核工作台类似，`unaudited` tab 需要：
1. 从 OA 数据库分批拉取全量已归档流程（每批 500 条，循环直到拉完）
2. 排除已有 snapshot（已审核过）的流程
3. 对剩余流程在 Go 内存中做 `slice` 切片分页

**问题**：同上，用户只想看一页数据，后端却每次都要拉取全量 OA 数据再丢弃。

### 4.3 默认 pageSize 不统一

| 模块 | 默认 pageSize |
|------|--------------|
| 审核工作台 | 10 |
| 归档复盘工作台 | 20 |
| 数据管理页 | 10 |
| 前端分页（成员/配置） | 10 |
| 字段选择器 | 5 或 6 |

### 4.4 pageSize 上限不统一

- 工作台 handler：`normalizeAuditPage` 限制 `pageSize <= 100`
- 数据管理页 repository：`ListPagedWithUser` 限制 `pageSize <= 200`

---

## 五、优化建议

### 5.1 高优先级：缓存 OA 全量数据减少重复拉取

**涉及模块**：审核工作台（pending_ai / ai_audited）、归档复盘（unaudited）

**现状问题**：用户每次翻页、切换筛选条件都会触发 OA 全量拉取，响应慢且对 OA 系统压力大。

**建议方案**：
- 在 Service 层增加短时缓存（如 Redis，TTL 30-60s），key 为 `tenant:{id}:user:{username}:todo_filtered:{filterHash}`
- 首次请求拉取全量并缓存，后续翻页直接从缓存切片
- 筛选条件变化时缓存 miss，重新拉取
- 手动刷新按钮可强制清除缓存

**预期收益**：翻页响应从秒级降到毫秒级，OA 查询量减少 80%+。

### 5.2 中优先级：统一默认 pageSize

**建议**：全局统一默认 `pageSize = 20`，前端可在 `usePagination` 和各 composable 中统一配置。

### 5.3 中优先级：统一 pageSize 上限

**建议**：统一为 `100`，数据管理页的 200 上限过大，可能导致单次查询过慢。

### 5.4 低优先级：前端分页模块暂不需要改动

以下模块使用前端分页是合理的，数据量小（< 200 条），无需改为后端分页：
- 组织架构成员列表
- 租户成员列表
- 用户个人配置列表
- 字段选择器穿梭框

**但需注意**：如果未来用户数增长到 500+ 以上，`org.vue` 和 `user-configs.vue` 应考虑迁移到后端分页。

### 5.5 低优先级：定时任务日志（cron 抽屉）

**现状**：`listTaskLogs` 返回最近 50 条，无分页。

**建议**：当前数据量可控，暂不需要分页。如果日志量增长，可增加后端分页支持。

---

## 六、分页架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                        前端 (Nuxt/Vue)                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  后端分页页面                        前端分页页面                │
│  ┌──────────────┐                   ┌──────────────┐            │
│  │ dashboard.vue│ ← page/page_size  │   org.vue    │            │
│  │ archive.vue  │    via API        │  tenants.vue │            │
│  │ data.vue     │                   │ user-cfg.vue │            │
│  └──────┬───────┘                   │  rules.vue   │            │
│         │                           │ settings.vue │            │
│         │ useAuditApi               └──────┬───────┘            │
│         │ useArchiveReviewApi              │                    │
│         │ useAdminDataApi           usePagination(source, size) │
│         │                           → computed slice            │
│         ▼                                                       │
├─────────────────────────────────────────────────────────────────┤
│                        后端 (Go/Gin)                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Handler 层：解析 page/page_size query 参数                     │
│       ↓                                                         │
│  Service 层：                                                   │
│  ┌─────────────────────────┐  ┌──────────────────────────┐      │
│  │ 工作台（OA 数据源）     │  │ 数据管理页（DB 数据源）  │      │
│  │                         │  │                          │      │
│  │ OA 分批拉取全量         │  │ Repository 层            │      │
│  │ → Go 内存筛选/分组      │  │ → SQL LIMIT/OFFSET      │      │
│  │ → 内存切片分页          │  │ → COUNT(*) 获取 total    │      │
│  │                         │  │                          │      │
│  │ ⚠ 伪分页，每次翻页     │  │ ✅ 真分页               │      │
│  │   重新拉取全量          │  │                          │      │
│  └─────────────────────────┘  └──────────────────────────┘      │
│                                                                 │
│  归档复盘 snapshot 分页（有日期筛选时）：                        │
│  OA 分批拉取日期范围内流程 ID → process_id IN ? 交叉过滤        │
│  → DB LIMIT/OFFSET 分页（混合模式，保证与 GetStats 口径一致）   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```
