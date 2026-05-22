# [#005] `{{flow_graph}}` 未输出完整流程定义图

## 状态

**未修复**。当前 `{{flow_graph}}` 在部分场景下仍会退化为当前节点或已发生审批路径，不能稳定表达完整流程定义图。

## 问题描述

AI 推理模板中的 `{{flow_graph}}` 预期应提供**完整流程图**：包含流程定义中的全部节点、节点连接关系、出口名称、分支条件 / 路由条件等信息。这样模型才能判断“当前节点之后可能流向哪里”“某个金额 / 部门 / 条件会进入哪条分支”。

当前实现中，`FetchProcessFlow` 会先查询审批历史，再尝试通过 `workflow_nodelink`、`workflow_nodebase`、`rule_base` 拼接路由图。但如果历史查询失败、路由图查询失败、条件表关联不完整，或者当前 E9 版本的条件配置不在已查询的字段中，`GraphText` 会退化为简单节点路径，最差情况下只输出当前节点。

因此，`{{flow_graph}}` 的实际内容可能是：

```text
部门负责人审批
```

或：

```text
提交 → 部门负责人审批 → 财务审批
```

而不是完整流程定义图：

```text
提交 → 部门负责人审批
部门负责人审批 → 财务审批 [金额 <= 5000]
部门负责人审批 → 分管领导审批 [金额 > 5000]
分管领导审批 → 财务审批
财务审批 → 归档
```

## 影响范围

| 场景 | 影响 |
|------|------|
| 审核规则依赖后续节点 | AI 可能只知道当前节点，不知道后续审批链路 |
| 审核规则依赖分支条件 | AI 无法准确判断金额、部门、合同类型等条件会触发哪条路径 |
| 归档复盘 | 只能看到实际历史路径时，难以对比“应走路径”和“实际路径” |
| 提示词变量说明 | 前端描述为“完整流程图节点信息”，但后端输出可能不完整 |

## 期望行为

`{{flow_graph}}` 应输出流程定义级别的完整路由图，而不只是当前实例已走过的节点：

1. 包含全部节点。
2. 包含全部节点连接关系。
3. 包含出口名称 / 出口条件。
4. 能区分“完整定义图”和“当前实例审批历史”。
5. 路由图获取失败时，应明确输出“未能获取完整流程图”，而不是静默伪装成完整图。

## 初步实现方向

1. 继续以 `workflow_requestbase.workflowid` 定位流程定义。
2. 以 `workflow_nodebase` 获取流程全部节点。
3. 以 `workflow_nodelink` 获取节点连接关系。
4. 补齐不同 E9 版本中条件配置的来源表，不能只依赖当前 `rule_base.condit` 关联。
5. 将 `GraphText` 明确标记为“流程定义图”，将 `HistoryText` 保持为“当前实例审批历史”。
6. 若条件解析失败，仍输出节点连接，但在边上标记“条件未解析”。

## 参考代码

- 占位符替换：`go-service/internal/service/audit_prompt_builder.go`、`go-service/internal/service/archive_prompt_builder.go`
- 流程快照：`go-service/internal/pkg/oa/ecology9.go` — `FetchProcessFlow`
- 路由图查询：`go-service/internal/pkg/oa/ecology9.go` — `fetchFlowRouteGraph`
- 当前兜底：`go-service/internal/pkg/oa/ecology9.go` — `fetchCurrentNodeSnapshot`

## 相关文档

- [AI 系统对接说明](../ai-integration.md)
- [OA 系统对接说明](../oa-integration.md)
