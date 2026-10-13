# AI 调用记录接口

> 权限要求：JWT + TenantContext + `tenant_admin` 角色
>
> 路由前缀：`/api/tenant/llm-logs`

租户管理员在**数据管理 → AI 调用记录**页使用。每次经 `AIModelCallerService` 的 LLM 调用均异步写入 `tenant_llm_message_logs`。

## 分页查询调用流程列表

```
GET /api/tenant/llm-logs/processes
```

按流程维度聚合展示 AI 调用记录。

聚合维度保持不变。审核、总结的 OA 通用嵌入调用会通过 `business_log_id` 关联业务执行记录：
最新操作人展示为“OA 嵌入审核（姓名）”或“OA 嵌入总结（姓名）”，并返回
`latest_trigger_detail` 供页面在同一单元格显示保存、提交、打开、进入表单、重新审核或定时检查动作。
人员无法解析时省略姓名括号。系统内调用仍显示本系统操作人，不额外展示触发动作。

**查询参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| `request_type` | string | 场景：`audit` / `archive` / `summary` / `chat` / `sql_assist` / `rule_assist` / `feedback_memory` |
| `call_type` | string | 类型：`reasoning` / `structured` |
| `keyword` | string | 流程标题关键词 |
| `operator` | string | 操作人 |
| `start_date` | string | 开始日期 YYYY-MM-DD |
| `end_date` | string | 结束日期 YYYY-MM-DD |
| `page` | int | 页码，从 1 开始 |
| `page_size` | int | 每页条数，默认 20 |

**响应**：`PagedResult`（`items` + `total` + `page` + `page_size`）

---

## 获取统计

```
GET /api/tenant/llm-logs/stats
```

按场景分布统计 AI 调用流程数量。

---

## 获取流程调用链

```
GET /api/tenant/llm-logs/:processId/chain
```

返回指定流程的全部 AI 调用记录（时间倒序，含 Token、耗时、提示词及本次调用关联业务执行记录的
`config_version_no`）。OA 通用嵌入调用还返回每次业务执行对应的 `user_name` 和 `trigger_detail`，
因此操作链可分别展示所有实际操作人及动作。同一流程先后使用多个配置版本时，每条调用返回各自实际版本；迁移前无法还原
配置版本的历史调用该字段为空，管理页展示为“历史记录（未记录配置版本）”。

---

## 获取单条调用详情

```
GET /api/tenant/llm-logs/calls/:id
```

返回单条调用详情，含系统提示词、用户提示词、深度思考过程（`reasoning_content`）、模型响应正文与 `config_version_no`。版本号通过
`business_log_id` 关联审核、归档复盘或流程总结执行日志获取，不使用流程当前绑定版本覆盖历史调用。

---

## 相关接口

租户 Token 消耗统计见 [用户设置接口](./user-settings.md)：

```
GET /api/tenant/stats/token-usage
```

全平台统计见 [系统管理接口](./system-admin.md)：

```
GET /api/admin/stats/token-usage
```

## 对话调用链（2026-09-05）

`request_type=chat` 可用于列表筛选。`business_log_id` 关联聊天会话，`process_title` 保存会话标题，真实 OA 流程 ID 不被虚构。聚合列表在 `process_id` 响应字段使用 `chat:<session UUID>` 作为对话链键，原样 URL 编码传入 `/:processId/chain` 即可查看。同一会话多轮调用合并；历史无关联的调用按 `chat:<log UUID>` 单列。此键不能作为 OA 流程 ID 跳转。

统计新增 `chat_count`（对话链数量），`total` 包含流程及对话链。Token 用量、配额结算和仪表盘 Token 总量沿用统一 LLM 日志聚合。聊天链的 `config_version_no` 不关联审核/归档版本。

### 日期筛选边界

日志/快照列表及其导出接口的 `start_date`、`end_date` 按应用配置时区解释，结束日期包含当天全部记录。
实现使用 `时间 >= 开始日期零点 AND 时间 < 结束日期次日零点`，包含当天最后一秒内的微秒记录，次日边界按自然日计算以兼容夏令时。

## SQL 辅助场景

`request_type` 新增 `sql_assist`，`call_type=structured`。统计响应新增 `sql_assist_count`（每次独立辅助业务计数），总调用与 Token/费用包含此场景；数据管理页提供对应筛选。


## 独立 AI 辅助操作

数据记忆 SQL 辅助、AI 梳理规则和反馈记忆分别使用 `sql_assist`、`rule_assist`、`feedback_memory`，均为 `call_type=structured`。三者都通过统一模型入口记录实际提示词、响应、Token 和耗时，并计入配额、用量与仪表盘总量。

新辅助操作使用独立 `business_log_id`；规则文件分片调用共享同次导入的业务ID，下一次导入使用新ID。SQL 辅助每次独立生成，反馈记忆使用生成版本ID。无需伪造 OA `process_id`。

聚合列表新增 `request_type` 和 `business_kind`（`process|chat|operation`）。辅助操作在响应的 `process_id` 字段返回 `场景:业务UUID` 形式的**调用链键**，将其原样 URL 编码传入 chain 接口，不用于 OA 跳转。历史 SQL 辅助按原伪流程键保留分组；历史 `rule-import:` 调用以 `rule_assist` 展示，不再计入审核/归档次数（旧记录无法恢复每次导入边界，仍按原配置标识分组），原始日志不改写。

统计新增 `rule_assist_count`、`feedback_memory_count`；`sql_assist_count` 继续保留。`total` 是不同业务调用链总量，同一 OA 流程包含审核和归档时合为一个流程链，因此分场景数量不要求相加等于总量。筛选场景时保留同一流程的完整调用链，但不同对话或辅助操作不会因空流程ID而串到一起。

页面为辅助操作提供独立详情：结构化条目/SQL、待确认事项、输入输出Token、只读提示词与原始模型输出，不展示不适用的审核配置版本。
