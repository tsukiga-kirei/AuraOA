package errcode

//业务错误代码 (40001–50002)。
const (
	Success = 0 // 成功

	//400xx - 客户端错误
	ErrParamValidation = 40001 // 参数校验失败

	//401xx - 身份验证错误
	ErrNoAuthToken       = 40100 // 未提供认证令牌
	ErrTokenInvalid      = 40101 // 令牌无效或已过期
	ErrTokenRevoked      = 40102 // 令牌已被吊销
	ErrWrongPassword     = 40103 // 用户名或密码错误
	ErrAccountLocked     = 40104 // 账户被锁定
	ErrAccountDisabled   = 40105 // 账户已被禁用
	ErrTenantNotFound    = 40106 // 租户不存在或已停用
	ErrNoRoleInTenant    = 40107 // 用户在该租户无角色分配
	ErrRoleSwitchFailed  = 40108 // 角色切换失败

	//403xx - 授权错误
	ErrInsufficientPerms = 40300 // 权限不足
	ErrCrossTenantAccess = 40301 // 不允许跨租户访问

	//404xx - 未找到
	ErrResourceNotFound = 40400 // 资源不存在

	//409xx - 冲突
	ErrResourceConflict = 40900 // 资源冲突

	//500xx - 服务器错误
	ErrInternalServer = 50000 // 服务器内部错误
	ErrDatabase       = 50001 // 数据库错误
	ErrRedisConn      = 50002 //Redis连接错误
)
