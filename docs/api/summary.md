# 流程总结接口

## 流程总结工作台（JWT + TenantContext + `/summary` 页面权限）

> 路由前缀：`/api/summary`。服务端会校验用户组织角色中的 `/summary` 页面权限，不能通过直接调用接口绕过。

工作台汇总当前 OA 用户可见的待办与已办流程，只保留租户已启用流程总结配置的流程类型。泛微 E9 已办范围限定为当前用户发起或在审批历史中实际处理过的已归档流程。
列表会合并 OA 嵌入已生成的标准总结和个人工作台总结；同一流程优先显示当前用户的个人总结，没有个人总结时显示 OA 嵌入总结。前台对所有角色统一按当前 OA 用户判断，租户管理员也仅能看到本人可见的流程。`summary_result.result_source` 为 `personal` 或 `embed`。
列表和历史查询均按 OA 当前可见性校验，任务状态与流式输出只允许任务发起人读取。

### 获取工作台流程列表

```
GET /api/summary/processes
```

**查询参数**：

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `keyword` | string | - | 按流程标题模糊查询 |
| `applicant` | string | - | 按申请人模糊查询 |
| `department` | string | - | 按部门精确查询 |
| `process_type` | string | - | 流程类型，多个值用逗号分隔 |
| `source` | string | - | `todo` 仅当前待办、`archived` 仅本人归档流程；省略为全部个人可见流程 |
| `summary_status` | string | - | `pending`、`summarized`、`running`、`failed` |
| `start_date` | date | - | 提交/归档开始日期 |
| `end_date` | date | - | 提交/归档结束日期（包含当天） |
| `page` | integer | `1` | 页码，从 1 开始 |
| `page_size` | integer | `20` | 每页条数，范围 1–100 |

响应 `data` 使用统一分页结构 `items`、`total`、`page`、`page_size`。列表项包含流程信息、
`source`（`todo` / `archived`）、`has_summary`、`summary_status`、当前用户的
`visible_block_ids`，以及存在有效快照时的 `summary_result`。
嵌入上下文在能识别当前 OA 用户时会额外返回 `personal_result` 和 `visible_block_ids`；嵌入标准结果与个人工作台结果同时存在时，前端可切换查看，工作台默认优先个人结果。

### 获取工作台统计

```
GET /api/summary/stats
```

筛选参数与列表一致（忽略分页、`summary_status` 与 `source`），返回 `total_count`、`summarized_count`、
`pending_count`、`running_count`、`failed_count`、`todo_count`。其中 `todo_count` 包含已有通用嵌入总结的待办；“我的待办”卡片使用 `source=todo` 获取对应列表。

### 发起流程总结

```
POST /api/summary/execute
```

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `process_id` | string | 是 | OA 流程 ID |
| `process_type` | string | 否 | 前端提示值；服务端以 OA 实际流程类型为准 |
| `title` | string | 否 | 列表标题；为空时使用 OA 标题 |
| `use_latest_config` | boolean | 否 | 是否升级到租户当前最新总结配置，默认沿用已绑定版本 |

任务使用统一 AI 调用入口，日志 `request_type=summary`、`trigger_source=summary_workbench`。
返回 HTTP `202` 时，前端通过任务接口轮询。

### 查询任务与流式输出

```
GET /api/summary/jobs/:id
GET /api/summary/stream/:id
```

只允许任务发起人访问。

### 获取流程总结历史

```
GET /api/summary/history/:processId
```

服务端重新校验当前用户仍可在 OA 待办或已办中访问该流程，然后仅返回本人系统内总结与通用嵌入总结的有效历史。

## 流程总结配置（JWT + TenantContext + `tenant_admin`）

> 路由前缀：`/api/tenant/summary`

### 获取配置列表

```
GET /api/tenant/summary/configs
```

---

### 创建配置

```
POST /api/tenant/summary/configs
```

---

### 获取配置详情

```
GET /api/tenant/summary/configs/:id
```

---

### 更新配置

```
PUT /api/tenant/summary/configs/:id
```

每个总结块配置项中包含 `enable_thinking`（boolean），控制生成该总结块时是否开启深度思考（需模型支持）。
`embed_config` 控制 OA 嵌入总结的自动刷新策略：

| 字段 | 默认值 | 说明 |
|------|--------|------|
| `auto_summary_on_open` | `true` | 没有历史总结时自动生成 |
| `auto_summary_on_data_change` | `true` | 按总结块实际使用的字段和附件版本判断并增量刷新 |
| `auto_summary_on_return_resubmit` | `true` | 退回或重新提交后刷新依赖流程信息的总结块 |
| `auto_summary_on_flow_change` | `false` | 普通审批推进后刷新使用流程基础信息/审批历史的总结块 |
| `scheduled_refresh_enabled` | `false` | 是否定时拉取该流程类型的近期实例并检查变化 |
| `scheduled_refresh_lookback_days` | `3` | 拉取近几天创建的流程，范围 1–30 |
| `scheduled_refresh_interval_minutes` | `5` | 检查频率，支持 5、10、15、30、60 分钟 |

保存配置后，系统会立即同步对应的 `embed_refresh_schedules` 持久化调度记录；关闭定时检查
会停用并移除内存 Cron、清除该配置尚未触发的 Redis 检查，并将已入库但未领取的总结任务标记为
`cancelled`。定时任务执行前还会重新核验总结配置，源配置未明确开启时不会访问 OA 数据库。
已经开始执行的任务不会被强制中断。

自动刷新只调用发生变化的启用总结块；未变化的块沿用最近一次有效结果。手动“重新总结”
仍会执行全部启用块。
首次总结会把启用块及其提示词、字段和外部关联配置保存为不可变执行配置版本并绑定流程。
租户流程配置在首次发布前仅为草稿，不会进入流程总结工作台或 OA 嵌入执行入口；首次发布生成基础配置 v1。
修改总结块不会借用“业务数据变化”开关自动刷新老流程；普通重新总结沿用已绑定版本，只有请求
明确传 `use_latest_config=true` 时才升级到当前配置。响应中的 `config_version_no` 为实际使用版本。
流程级定时检查只发现候选流程，所有总结块均未变化时不会创建总结或 LLM 日志。
任务日志通过 `trigger_detail` 区分 `visible_open`、`form_open`、`manual`、`save_requested`、`submit_requested` 和
`scheduled_scan`；自动任务失败后会保存 `attempt_fingerprint`，相同指纹不会被自动来源反复执行。
worker 会定期接管旧容器遗留的 Redis pending 消息；数据库原子领取保证已完成、失败或取消的记录
只会被确认清理，不会再次调用 AI。

---

### 删除配置

```
DELETE /api/tenant/summary/configs/:id
```

---

### 测试 OA 连接

```
POST /api/tenant/summary/configs/test-connection
```

---

### 拉取流程字段

```
POST /api/tenant/summary/configs/:id/fetch-fields
```

---

### 测试外部关联数据

```
POST /api/tenant/summary/context/test
```

建模表挂载会在 `context_text` 中显示“建模表：中文名（英文表名）”；`mode=rows` 时，`return_fields` 必须填写英文物理列名，服务端会基于泛微字段元数据将返回结果的列名转换为中文显示名。`max_rows=-1` 表示返回全部匹配行，正整数表示行数上限。

---

## 流程总结快照（JWT + TenantContext + `tenant_admin`）

> 路由前缀：`/api/summary/snapshots`

数据管理页使用，从有效总结日志聚合历史数据：嵌入按流程汇总为一组，系统内按流程和操作人汇总为一组。
每组的展示字段、总结块数与 `latest_valid_log_id` 对应最新有效记录；`valid_log_ids` 仅包含该组历史。
`channel` 为 `embed` / `workbench`，系统内行包含 `user_id`；列表按最新记录时间倒序。
嵌入行同时返回最新有效记录的 `trigger_detail`。若 OA 当前操作人员可解析，`operator` 展示为
“OA 嵌入总结（姓名）”，`department` 返回其 OA 部门；无法解析时展示“OA 嵌入总结”且部门为空。
统计按相同分组计算，仍只按渠道筛选，不新增人员、部门等筛选联动。现有共享快照保留给运行时使用，无需清空或回填数据库。

### 获取快照列表

```
GET /api/summary/snapshots
```

**查询参数**：`channel`、`keyword`、`process_type`、`operator`、`department`、`start_date`、`end_date`、`page`、`page_size`

---

### 获取快照统计

```
GET /api/summary/snapshots/stats
```

---

### 导出快照

```
GET /api/summary/snapshots/export
```

按筛选条件导出 Excel。

---

### 获取总结链

```
GET /api/summary/snapshots/:processId/chain
```

支持 `channel=embed|workbench`、`user_id`（UUID，仅系统内渠道使用）查询参数。
数据管理页传入所选行的渠道与操作人，返回对应分组的有效总结链，按时间倒序；省略筛选兼容全流程历史。
嵌入日志的 `user_name` 按当次 OA 当前操作人展示为“OA 嵌入总结（姓名）”；无法解析时为“OA 嵌入总结”。
系统内日志保留实际操作人。每条嵌入记录返回 `trigger_detail`，并包含实际使用的 `config_version_no`；迁移前历史记录可能为空，
数据管理页在“查看详情”抽屉中明确展示为未记录版本。

### 日期筛选边界

日志/快照列表及其导出接口的 `start_date`、`end_date` 按应用配置时区解释，结束日期包含当天全部记录。
实现使用 `时间 >= 开始日期零点 AND 时间 < 结束日期次日零点`，包含当天最后一秒内的微秒记录，次日边界按自然日计算以兼容夏令时。
