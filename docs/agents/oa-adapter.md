# OA 适配器与智能体系统工具

系统工具**禁止**直连 OA 业务库或拼厂商 SQL。一律通过 [`OAAdapter`](../../go-service/internal/pkg/oa/adapter.go) 及可选扩展接口。连接池、租户绑定、只读原则见 [`docs/oa-integration.md`](../oa-integration.md)。

## 1. 为什么必须走适配器

- 多 OA 类型：今日仅 `weaver_e9` 完整实现；`weaver_ebridge`、`zhiyuan_a8`、`landray_ekp`、`custom` 在工厂中未实现。新类型只加适配器，系统工具 code 不变。
- 只读：适配器文档约定不对 OA 写入。智能体「辅助办」只生成意见与跳转 URL，不调用不存在的 `Approve`/`Reject`。
- 权限：待办按用户 `loginid` 过滤；单流程详情在工作台侧历史上缺少实例级校验，对话场景必须补可见性，否则 `process_id` 可被猜中。

## 2. 租户如何拿到适配器

与审核工作台相同：

1. 系统管理员配置 `oa_database_connections`（`oa_type`、驱动、账号等）。
2. 系统管理员把连接分配给租户：`tenants.oa_db_connection_id`。
3. 运行时 `ConnectionManager` 按连接创建适配器。

智能体不单独选 OA 连接。未分配连接时，所有 `oa_required=true` 的系统工具返回明确业务错误（如未配置 OA），不要调用空指针。

用户身份：JWT `username` 对应泛微 `hrmresource.loginid`；嵌入页用 `ResolveUsernameByOAUserID`。工具入参中的用户不得由模型指定为任意人，只使用当前登录/嵌入用户。

## 3. 适配器方法 → 系统工具

核心接口（所有 OA 类型终态都应实现）：

| 适配器方法 | 系统工具 | 说明 |
|------------|----------|------|
| `FetchTodoListPaged` | `list_my_todos` | 仅当前用户待办；筛选下推 SQL |
| `FetchProcessData` + 可选 `BrowseValueResolver` | `get_process` | 主表/明细/附件元数据；附件识别走现有服务 |
| `FetchProcessFlow` | `get_approval_flow` | 历史只读 |
| `FetchProcessRequestSummary` | 卡片标题/嵌入上下文 | 可与 `get_process` 组合 |
| `IsProcessInTodo` | 可见性辅助 | 不够单独作为「可读」条件 |
| `ResolveUsernameByOAUserID` | 嵌入身份 | 非对话工具 |
| `CheckUserPermission` | 暂不作为对话主路径 | 已实现但粒度是流程类型且业务侧未使用 |

可选接口：

| 可选接口 | 对话是否使用 | 不支持时 |
|----------|----------------|----------|
| `BrowseValueResolver` | `get_process` 显示名 | 保留原始 ID，卡片注明 |
| `WorkflowDefinitionSelector` | 预留检索流程定义 | 工具不可见或 `capability_unsupported` |
| `ModelContextQuerier` | 预留，第一期不挂默认智能体 | 同上 |
| `RecentProcessScanner` | 不进对话 | 定时嵌入用 |
| `ProcessRequestWatermarkResolver` | 不进对话 | 嵌入提交检测 |

**第一期不提供**基于 `FetchArchivedListPaged` 的对话工具：该方法忽略 `username`，返回租户级全量归档，存在越权风险。若未来要做「我归档过的」，必须先在适配器按参与人/申请人过滤。

## 4. 必须新增的适配器能力：流程可见性

对话里模型可能传入任意 `process_id`。在开放 `get_process` / `get_approval_flow` 前，适配器（或紧贴适配器的领域服务）必须提供：

```text
CheckProcessVisibility(ctx, username, processID) (visible bool, err error)
```

`visible` 为真当且仅当（建议）：

- 该流程在用户待办中（`IsProcessInTodo`），或
- 用户是申请人（`workflow_requestbase` 创建人与 `hrmresource.loginid` 对应），或
- 用户出现在审批历史操作人中

嵌入场景同样校验，避免 embed token 泄漏后枚举 requestid。

实现放在各 OA 适配器内（SQL 方言不同），不要只在 Ecology9 的业务 Service 里写死表名。

## 5. OA 跳转（非适配器写操作）

`resolve_oa_url` **不写库**。使用已有 [`BuildOAProcessURL`](../../go-service/internal/service/oa_connection_service.go)，读取连接上的 `oa_base_url`、`process_url_template`。未配置则工具失败并提示管理员补全跳转模板。

## 6. 调用现有审核 / 总结

`run_audit` / `run_summary` / `get_latest_audit` / `get_latest_summary` 不是 OA 写接口：

- 读/写的是 AuraOA 的 `audit_logs`、`process_summary_*` 等。
- 拉 OA 表单仍走适配器（与工作台同一套 `FetchProcessData`）。
- 须满足流程可见性，且用户具备对应工作台权限时是否还要再卡一层：默认 **对话工具授权已包含**，不再强制 `page_permissions` 含 `/dashboard`（避免「能对话辅助办但不能进工作台」的角色卡住）。产品若要强绑定，在租户再分配时不要把 `run_audit` 授给无工作台角色。

## 7. 新 OA 类型接入清单

新增例如致远/蓝凌适配器时：

1. 实现 `OAAdapter` 必选方法 + `CheckProcessVisibility`。
2. 按需实现可选接口；未实现则对应工具在该租户 `oa_type` 下标记 `capability_unsupported`。
3. 工厂 `oa_type` 注册；连接页可选该类型。
4. **不必**为每个 OA 复制一套 `tool_code`。系统工具保持稳定，差异留在适配器。
5. 更新 [`docs/oa-integration.md`](../oa-integration.md) 与本文映射表。
6. 若该 OA 需要原生 HTTP 才能「办」，另开需求；本期仍不允许写操作。

## 8. 工具执行时的适配器上下文

每次系统工具调用应带：

- `tenant_id` → 解析 OA 连接 → `GetAdapter`
- `username`（当前用户）
- `ctx` 取消与超时（对话一轮超时与审核 Job 超时分开：查询类短超时，`run_audit` 走现有 Job）

附件：`FetchProcessData` 已可注入附件识别；对话 `get_process` 默认跟工作台同一套识别开关，避免对话另下一套附件策略。

## 9. 能力探测

租户 `oa_type` 在工厂中无实现时：OA 类工具全部不可用，管理端智能体绑定处标注「当前 OA 类型未适配」。

可选接口用类型断言：

```text
if resolver, ok := adapter.(oa.BrowseValueResolver); ok { ... }
```

与审核配置里建模表查询的做法一致。
