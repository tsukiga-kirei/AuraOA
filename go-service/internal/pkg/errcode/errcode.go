// Package errcode 定义全局业务错误码。
// 错误码规则：
//   - 0       成功
//   - 400xx   客户端请求错误
//   - 401xx   身份认证错误
//   - 403xx   权限不足
//   - 404xx   资源不存在
//   - 409xx   资源冲突
//   - 500xx   服务端内部错误
//   - 502xx   OA 集成错误
//   - 503xx   AI 模型错误
package errcode

const (
	Success = 0 // 成功

	// 400xx - 客户端请求错误
	ErrParamValidation    = 40001 // 参数校验失败
	ErrBatchLimitExceeded = 40002 // 批量审核超过上限
	ErrNoAIModelConfig    = 40003 // 租户未配置 AI 模型
	ErrNoProcessConfig    = 40004 // 流程审核配置不存在

	// 401xx - 身份认证错误
	ErrNoAuthToken      = 40100 // 未提供认证令牌
	ErrTokenInvalid     = 40101 // 令牌无效或已过期
	ErrTokenRevoked     = 40102 // 令牌已被吊销
	ErrWrongPassword    = 40103 // 用户名或密码错误
	ErrAccountLocked    = 40104 // 账户被锁定
	ErrAccountDisabled  = 40105 // 账户已被禁用
	ErrTenantNotFound   = 40106 // 租户不存在或已停用
	ErrNoRoleInTenant   = 40107 // 用户在该租户无角色分配
	ErrRoleSwitchFailed = 40108 // 角色切换失败

	// 403xx - 权限不足
	ErrInsufficientPerms   = 40300 // 权限不足
	ErrCrossTenantAccess   = 40301 // 不允许跨租户访问
	ErrPermissionDenied    = 40302 // 操作被拒绝（如尝试修改被锁定的配置）
	ErrMandatoryRuleLocked = 40303 // 强制规则不可修改

	// 404xx - 资源不存在
	ErrResourceNotFound = 40400 // 资源不存在
	ErrProcessNotFound  = 40401 // 流程在 OA 系统中不存在
	ErrConfigNotFound   = 40402 // 流程审核配置不存在
	ErrRuleNotFound     = 40403 // 审核规则不存在

	// 409xx - 资源冲突
	ErrResourceConflict     = 40900 // 资源冲突
	ErrDuplicateProcessType = 40901 // 同一租户下流程类型重复
	ErrBootstrapForbidden   = 40910 // 已有用户，禁止再次执行初始化

	// 500xx - 服务端内部错误
	ErrInternalServer = 50000 // 服务器内部错误
	ErrDatabase       = 50001 // 数据库操作失败
	ErrRedisConn      = 50002 // Redis 连接失败

	// 502xx - OA 集成错误
	ErrOAConnectionFailed = 50201 // OA 数据库连接失败
	ErrOAQueryFailed      = 50202 // OA 数据库查询失败
	ErrOATypeUnsupported  = 50203 // 不支持的 OA 类型

	// 503xx - AI 模型错误
	ErrAIConnectionFailed      = 50301 // AI 模型连接失败
	ErrAICallFailed            = 50302 // AI 模型调用失败
	ErrAIDeployTypeUnsupported = 50303 // 不支持的 AI 部署类型
	ErrTokenQuotaExceeded      = 50304 // 租户 Token 配额已用尽
	ErrAuditParseFailed        = 50305 // AI 审核结果解析失败
)
