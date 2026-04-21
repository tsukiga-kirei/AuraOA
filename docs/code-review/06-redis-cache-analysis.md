# Redis 缓存使用分析与优化建议

## 概述

本文档梳理项目中所有 Redis 缓存使用点，分析每个缓存的合理性，并提出优化方向。

---

## 一、缓存使用清单

### 1. 归档列表缓存 — `archive:list:{tenant_id}:{user_id}:{filter_hash}`

| 属性 | 值 |
|------|-----|
| 位置 | `ArchiveReviewService.ListProcessesPaged` |
| TTL | 5 分钟 |
| 缓存内容 | 单页列表结果（items + total + page + pageSize） |
| 缓存键维度 | tenant_id + user_id + keyword + applicant + process_type + department + audit_status + page + page_size + 日期范围 |

**问题：不合理 ❌**

- 缓存键包含 `page`、`audit_status`、`keyword` 等高变参数，用户每次切换页签、翻页、搜索都会产生全新的缓存键，几乎不可能命中。
- `json:"-"` bug：`ArchiveDateStart` 和 `ArchiveDateEndExclusive` 标记了 `json:"-"`，`ComputeFilterHash(params)` 会丢弃日期字段，导致不同日期范围生成相同缓存键，返回错误数据。（已修复为显式构建 map）
- `listArchiveUnauditedPaged` 每次都从 OA 拉全量数据再内存分页，缓存只缓存了最终的一页结果，但真正昂贵的 OA 全量查询没有被缓存。

### 2. 审核待办列表缓存 — `audit:todo:{tenant_id}:{user_id}:{filter_hash}`

| 属性 | 值 |
|------|-----|
| 位置 | `AuditExecuteService.ListProcessesPaged` |
| TTL | 3 分钟 |
| 缓存内容 | 单页列表结果（items + total） |
| 缓存键维度 | tenant_id + user_id + tab + keyword + applicant + process_type + department + audit_status + page + page_size |

**问题：不合理 ❌**

- 与归档列表相同的问题：缓存键维度过多，命中率极低。
- 同样存在 `json:"-"` bug：`SubmitDateStart` 和 `SubmitDateEndExclusive` 被 `ComputeFilterHash` 忽略，不同日期范围会碰撞到同一个缓存键。
- 底层 `fetchTodoListFiltered` 已被移除，OA 待办查询不再通过该方法分批拉取全量数据再内存分组。

### 3. 归档统计缓存 — `archive:stats:{tenant_id}:{user_id}:{date_range_hash}`

| 属性 | 值 |
|------|-----|
| 位置 | `ArchiveReviewService.GetStats` |
| TTL | 5 分钟 |
| 缓存内容 | 统计计数（total_count, compliant_count 等） |
| 缓存键维度 | tenant_id + user_id + 日期范围 |

**评价：部分合理 ⚠️**

- 缓存键维度合理（只有日期范围），同一用户在同一日期范围内多次请求可以命中。
- 但缓存未命中时，`GetStats` 会分批拉取 OA 全量数据（每批 500 条循环），这个全量拉取本身没有被缓存，只缓存了最终的统计结果。
- 前端每次切换页签都会调用 `loadStats()`，如果日期范围不变，可以命中缓存，这是有效的。

### 4. 审核统计缓存 — `audit:stats:{tenant_id}:{user_id}:{date_range_hash}`

| 属性 | 值 |
|------|-----|
| 位置 | `AuditExecuteService.GetStatsWithParams` |
| TTL | 5 分钟 |
| 缓存内容 | 统计计数（pending_ai_count, ai_done_count 等） |
| 缓存键维度 | tenant_id + user_id + 日期范围 |

**评价：部分合理 ⚠️**

- 与归档统计相同的分析。缓存键维度合理，但底层 OA 全量查询没有被缓存。

### 5. 归档快照缓存 — `archive:snapshot:{tenant_id}:{process_ids_hash}`

| 属性 | 值 |
|------|-----|
| 位置 | `ArchiveReviewService.getArchiveSnapshotMapCached` |
| TTL | 5 分钟 |
| 缓存内容 | processID → ArchiveProcessSnapshot 映射 |
| 缓存键维度 | tenant_id + processIDs 列表的哈希 |

**问题：不合理 ❌**

- `process_ids_hash` 是对整个 processIDs 列表做哈希。不同的 OA 查询结果（不同页、不同筛选）会产生不同的 processIDs 列表，导致不同的哈希值。
- 快照数据来自本地 PostgreSQL，查询本身很快（有索引），缓存收益很小。
- 缓存键与 OA 查询结果强耦合，OA 数据变化（新流程归档）就会导致 processIDs 列表变化，缓存失效。

### 6. 审核快照缓存 — `audit:snapshot:{tenant_id}:{process_ids_hash}`

| 属性 | 值 |
|------|-----|
| 位置 | `AuditExecuteService.getSnapshotMapCached` |
| TTL | 5 分钟 |
| 缓存内容 | processID → AuditProcessSnapshot 映射 |
| 缓存键维度 | tenant_id + processIDs 列表的哈希 |

**问题：不合理 ❌**

- 与归档快照相同的问题。本地 DB 查询快，缓存键与 OA 结果耦合，命中率低。

### 7. 归档配置缓存 — `archive:config:{tenant_id}:{process_type}`

| 属性 | 值 |
|------|-----|
| 位置 | `ArchiveReviewService.getArchiveConfigCached` |
| TTL | 10 分钟 |
| 缓存内容 | ProcessArchiveConfig + ArchiveRule 列表 |
| 缓存键维度 | tenant_id + process_type |

**评价：合理 ✅**

- 配置数据变更频率低，缓存键维度少且稳定，命中率高。
- 配置变更时有 `InvalidateConfigCache` 主动失效。
- 10 分钟 TTL 合理。

### 8. 审核配置缓存 — `audit:config:{tenant_id}:{process_type}`

| 属性 | 值 |
|------|-----|
| 位置 | `AuditExecuteService.getProcessConfigCached` |
| TTL | 10 分钟 |
| 缓存内容 | ProcessAuditConfig + AuditRule 列表 |
| 缓存键维度 | tenant_id + process_type |

**评价：合理 ✅**

- 与归档配置相同，缓存键稳定，命中率高。

### 10. OA 全量数据缓存 — `oa:data:{tenant_id}:{user_id}:{date_range_hash}`

| 属性 | 值 |
|------|-----|
| 位置 | 待接入（`DefaultTTLOAData` 常量已定义于 `cache/config.go`） |
| TTL | 5 分钟 |
| 缓存内容 | OA 跨库查询的全量结果 |
| 缓存键维度 | tenant_id + user_id + 日期范围 |

**评价：方向 1 实施中 🚧**

- 对应优化方向 1「将 OA 全量查询结果缓存为中间层」。
- `DefaultTTLOAData = 5min` 已在 `cache/config.go` 中声明，供翻页/筛选/统计复用。
- 缓存键维度少且稳定，预期命中率高。

### 11. 仪表盘缓存 — `dashboard:{tenant_id}:{user_id}:{role}`

| 属性 | 值 |
|------|-----|
| 位置 | `DashboardOverviewService.BuildOverview` |
| TTL | 2 分钟 |
| 缓存内容 | 完整的 DashboardOverviewResponse |
| 缓存键维度 | tenant_id + user_id + role |

**评价：合理 ✅**

- 仪表盘聚合了多个数据源（快照表、cron_logs、活动记录等），计算成本高。
- 缓存键维度少且稳定（同一用户同一角色），命中率高。
- 2 分钟 TTL 合理，仪表盘数据不需要实时。

---

## 二、核心问题总结

### 问题 1：缓存了错误的层级

列表接口（归档列表、审核待办）的真正瓶颈是 OA 数据库的全量查询（7 表 JOIN，跨库），但缓存放在了最外层（单页结果）。用户每次翻页、切换页签、搜索都会穿透缓存，重新执行昂贵的 OA 查询。

**应该缓存的是 OA 全量查询结果**（按日期范围 + 租户维度），而不是最终的单页结果。这样翻页、页签切换、搜索都可以在缓存的 OA 数据上做内存操作，不需要重新查 OA。

### 问题 2：`json:"-"` 导致缓存键碰撞

`ArchiveListParams` 和 `AuditListParams` 的日期字段标记了 `json:"-"`，`ComputeFilterHash` 使用 `json.Marshal` 计算哈希时会丢弃这些字段。导致：
- 不同日期范围的请求生成相同的缓存键
- 返回错误的缓存数据（数据错乱）

**影响范围**：
- `ArchiveReviewService.ListProcessesPaged`（已修复）
- `AuditExecuteService.ListProcessesPaged`（未修复，存在同样问题）

### 问题 3：快照缓存收益低

快照数据存储在本地 PostgreSQL，有索引，查询本身很快（< 10ms）。但缓存键与 OA 查询结果的 processIDs 列表耦合，不同查询条件产生不同的 processIDs 列表，导致缓存键不稳定，命中率低。用 Redis 缓存本地 DB 的快速查询，增加了复杂度但收益很小。

---

## 三、优化方向

### 方向 1：将 OA 全量查询结果缓存为中间层（进行中 🚧）

> `DefaultTTLOAData = 5min` 已在 `cache/config.go` 中定义。

将 OA 的全量归档/待办流程列表按 `{tenant_id}:{user_id}:{date_range_hash}` 缓存，所有页签切换、翻页、搜索都在这个缓存上做内存过滤和分页。

**预期效果**：
- 同一日期范围内，无论怎么切换页签、翻页、搜索，OA 只查一次
- 消除 `SLOW SQL >= 200ms` 的反复出现
- 缓存命中率从接近 0% 提升到 80%+

### 方向 2：移除低价值缓存

- 移除列表单页结果缓存（被方向 1 替代）
- 移除快照缓存（本地 DB 查询足够快，缓存键不稳定）
- 保留：配置缓存、统计缓存、仪表盘缓存

### 方向 3：修复 `json:"-"` 缓存键 bug

- 审核待办列表 `AuditExecuteService.ListProcessesPaged` 需要与归档列表相同的修复
- 或者在方向 1 实施后，这个问题自然消失（不再按 params 全量做 hash）

---

## 四、缓存合理性评分

| 缓存点 | 合理性 | 命中率预估 | 建议 |
|--------|--------|-----------|------|
| 归档列表（单页） | ❌ | < 5% | 替换为 OA 全量结果缓存 |
| 审核待办（单页） | ❌ | < 5% | 替换为 OA 全量结果缓存 |
| 归档统计 | ⚠️ | ~50% | 保留，但底层 OA 查询需缓存 |
| 审核统计 | ⚠️ | ~50% | 保留，但底层 OA 查询需缓存 |
| 归档快照 | ❌ | < 10% | 移除，本地 DB 查询够快 |
| 审核快照 | ❌ | < 10% | 移除，本地 DB 查询够快 |
| 归档配置 | ✅ | > 90% | 保留 |
| 审核配置 | ✅ | > 90% | 保留 |
| OA 全量数据 | 🚧 | > 80%（预期） | 方向 1 实施中，TTL 常量已定义 |
| 仪表盘 | ✅ | > 80% | 保留 |


---

## 五、已完成的优化

### 优化 1：OA 全量数据中间层缓存

新增两个缓存方法，将 OA 跨库查询结果按日期范围缓存：

- `ArchiveReviewService.fetchOAArchivedDataCached` — 缓存键 `archive:oadata:{tenant_id}:{user_id}:{date_range_hash}`
- `AuditExecuteService.fetchOATodoDataCached` — 缓存键 `audit:oadata:{tenant_id}:{user_id}:{date_range_hash}`

同一日期范围内，`GetStats`、`ListProcessesPaged`（所有页签）、翻页、搜索都复用同一份 OA 数据，OA 只查一次。

### 优化 2：移除低价值的单页结果缓存

- 移除 `ListProcessesPaged`（归档）的 `archive:list:*` 单页缓存
- 移除 `ListProcessesPaged`（审核）的 `audit:todo:*` 单页缓存
- 这些缓存因 key 维度过多（含 page/keyword/audit_status），命中率接近 0%

### 优化 3：快照查询改为直接查 DB

- 新增 `getArchiveSnapshotMapDirect` 和 `getSnapshotMapDirect`
- `ListProcessesPaged` 和 `GetStats` 改用直接 DB 查询（本地 PostgreSQL，< 10ms）
- 旧的 `getArchiveSnapshotMapCached` / `getSnapshotMapCached` 保留给 `ListPendingForBatch` 等低频接口

### 优化 4：内存过滤替代重复 OA 查询

- `listArchiveUnauditedPaged` 改为从缓存的 OA 全量数据中做 keyword/applicant/department 内存过滤
- `ListProcessesPaged`（审核）新增 `filterTodoItemsInMemory` 做内存过滤
- `listArchiveBySnapshotPaged` 改为从缓存的 OA 全量数据获取 processID 列表

### 优化 5：缓存失效策略更新

- 归档/审核任务完成后，新增 `InvalidateOADataCache` 清除 OA 全量数据缓存
- `TenantPrefixes` 更新，包含 `archive:oadata:*` 和 `audit:oadata:*`

### 保留不变的缓存

- ✅ 配置缓存（`audit:config:*`、`archive:config:*`）— 命中率 > 90%
- ✅ 统计缓存（`audit:stats:*`、`archive:stats:*`）— 日期范围维度，命中率合理
- ✅ 仪表盘缓存（`dashboard:*`）— 聚合计算成本高，命中率 > 80%
