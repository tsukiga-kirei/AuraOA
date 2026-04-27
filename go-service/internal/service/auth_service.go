package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"go.uber.org/zap"

	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/pkg/hash"
	jwtpkg "oa-smart-audit/go-service/internal/pkg/jwt"
	pkglogger "oa-smart-audit/go-service/internal/pkg/logger"
	"oa-smart-audit/go-service/internal/repository"
)

// AuthService 负责身份验证、令牌管理、角色切换和菜单权限检索。
type AuthService struct {
	userRepo         *repository.UserRepo
	rdb              *redis.Client
	db               *gorm.DB
	systemConfigRepo *repository.SystemConfigRepo
}

// NewAuthService 构造 AuthService，注入用户仓储、Redis 客户端、数据库连接和系统配置仓储。
func NewAuthService(userRepo *repository.UserRepo, rdb *redis.Client, db *gorm.DB, systemConfigRepo *repository.SystemConfigRepo) *AuthService {
	return &AuthService{
		userRepo:         userRepo,
		rdb:              rdb,
		db:               db,
		systemConfigRepo: systemConfigRepo,
	}
}

// getSessionCacheTTL 返回 session 缓存的 TTL，与 refresh_token 有效期一致。
// 优先从数据库 system_configs 读取 auth.refresh_token_ttl_days，降级使用 config.yaml。
func (s *AuthService) getSessionCacheTTL() time.Duration {
	if val, err := s.systemConfigRepo.FindByKey("auth.refresh_token_ttl_days"); err == nil && val != "" {
		var days int
		if _, parseErr := fmt.Sscanf(val, "%d", &days); parseErr == nil && days > 0 {
			return time.Duration(days) * 24 * time.Hour
		}
	}
	return jwtpkg.GetRefreshTokenTTL()
}

// getAccessTokenTTL 返回 access_token 有效期。
// 优先从数据库 system_configs 读取 auth.access_token_ttl_hours，降级使用 config.yaml。
func (s *AuthService) getAccessTokenTTL() time.Duration {
	if val, err := s.systemConfigRepo.FindByKey("auth.access_token_ttl_hours"); err == nil && val != "" {
		var hours int
		if _, parseErr := fmt.Sscanf(val, "%d", &hours); parseErr == nil && hours > 0 {
			return time.Duration(hours) * time.Hour
		}
	}
	return jwtpkg.GetAccessTokenTTL()
}

// getRefreshTokenTTL 返回 refresh_token 有效期。
// 优先从数据库 system_configs 读取 auth.refresh_token_ttl_days，降级使用 config.yaml。
func (s *AuthService) getRefreshTokenTTL() time.Duration {
	if val, err := s.systemConfigRepo.FindByKey("auth.refresh_token_ttl_days"); err == nil && val != "" {
		var days int
		if _, parseErr := fmt.Sscanf(val, "%d", &days); parseErr == nil && days > 0 {
			return time.Duration(days) * 24 * time.Hour
		}
	}
	return jwtpkg.GetRefreshTokenTTL()
}

// ServiceError 业务层错误，携带错误码和用户可读消息，供 handler 层转换为 HTTP 响应。
type ServiceError struct {
	Code    int
	Message string
}

func (e *ServiceError) Error() string {
	return e.Message
}

func newServiceError(code int, msg string) *ServiceError {
	return &ServiceError{Code: code, Message: msg}
}

// ---------------------------------------------------------------------------
// 首次部署初始化（无任何用户时创建系统管理员）
// ---------------------------------------------------------------------------

var bootstrapUsernameRe = regexp.MustCompile(`^[a-zA-Z0-9_]{3,100}$`)

// BootstrapStatus 返回是否需要展示初始化向导（users 表为空）。
func (s *AuthService) BootstrapStatus() (*dto.BootstrapStatusResponse, error) {
	n, err := s.userRepo.CountUsers()
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	return &dto.BootstrapStatusResponse{NeedsSetup: n == 0}, nil
}

// BootstrapAdmin 在零用户时创建首个系统管理员账号及 system_admin 角色分配。
func (s *AuthService) BootstrapAdmin(req *dto.BootstrapAdminRequest) error {
	username := strings.TrimSpace(req.Username)
	displayName := strings.TrimSpace(req.DisplayName)
	if username == "" || displayName == "" {
		return newServiceError(errcode.ErrParamValidation, "用户名或显示名称不能为空")
	}
	if !bootstrapUsernameRe.MatchString(username) {
		return newServiceError(errcode.ErrParamValidation, "用户名需为 3–100 位字母、数字、下划线")
	}
	if len(req.Password) < 8 {
		return newServiceError(errcode.ErrParamValidation, "密码至少 8 位")
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&model.User{}).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return newServiceError(errcode.ErrBootstrapForbidden, "系统已初始化")
		}
		hashStr, err := hash.HashPassword(req.Password)
		if err != nil {
			return newServiceError(errcode.ErrInternalServer, "服务器内部错误")
		}
		user := &model.User{
			ID:                uuid.New(),
			Username:          username,
			PasswordHash:      hashStr,
			DisplayName:       displayName,
			Status:            "active",
			PasswordChangedAt: time.Now(),
		}
		if err := tx.Create(user).Error; err != nil {
			return newServiceError(errcode.ErrDatabase, "创建用户失败，用户名可能已存在")
		}
		assign := &model.UserRoleAssignment{
			ID:       uuid.New(),
			UserID:   user.ID,
			Role:     "system_admin",
			TenantID: nil,
			Label:    "系统管理员 - " + displayName,
		}
		if err := tx.Create(assign).Error; err != nil {
			return newServiceError(errcode.ErrDatabase, "数据库错误")
		}
		return nil
	})
	if err != nil {
		if se, ok := err.(*ServiceError); ok {
			return se
		}
		return newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	pkglogger.Global().Info("系统管理员初始化成功", zap.String("username", username))
	return nil
}

// ---------------------------------------------------------------------------
// 登录
// ---------------------------------------------------------------------------

// Login 验证用户身份，返回访问令牌、刷新令牌、用户信息及角色列表。
// 登录流程：查找用户 → 检查禁用/锁定 → 验证密码 → 校验租户 → 选择活跃角色 → 生成令牌 → 写入登录历史 → 缓存 session。
func (s *AuthService) Login(req *dto.LoginRequest, clientIP string, userAgent string) (*dto.LoginResponse, error) {
	// IPv6 回环地址统一转为 IPv4 格式，便于日志展示
	if clientIP == "::1" {
		clientIP = "127.0.0.1"
	}

	// 1. 通过用户名查找用户
	user, err := s.userRepo.FindByUsername(req.Username)
	if err != nil {
		return nil, newServiceError(errcode.ErrWrongPassword, "用户名或密码错误")
	}

	// 2. 检查账户禁用状态
	if user.Status == "disabled" {
		pkglogger.Global().Warn("登录失败：账户已被禁用", zap.String("username", req.Username))
		return nil, newServiceError(errcode.ErrAccountDisabled, "账户已被禁用")
	}

	// 3. 检查账户锁定：连续失败 5 次且锁定期未过
	if user.LoginFailCount >= 5 && user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		pkglogger.Global().Warn("登录失败：账户被锁定", zap.String("username", req.Username))
		return nil, newServiceError(errcode.ErrAccountLocked, "账户被锁定")
	}

	// 4. 验证密码（bcrypt 比对）
	if !hash.CheckPassword(req.Password, user.PasswordHash) {
		_ = s.userRepo.UpdateLoginFail(user)
		pkglogger.Global().Warn("登录失败：密码错误", zap.String("username", req.Username))
		return nil, newServiceError(errcode.ErrWrongPassword, "用户名或密码错误")
	}

	// 5. 若指定了 tenant_id 且非 system_admin，校验租户是否存在且处于活跃状态
	var tenant *model.Tenant
	if req.TenantID != "" && req.PreferredRole != "system_admin" {
		tenantUUID, parseErr := uuid.Parse(req.TenantID)
		if parseErr != nil {
			return nil, newServiceError(errcode.ErrTenantNotFound, "租户不存在或已停用")
		}
		tenant, err = s.userRepo.FindTenantByID(tenantUUID)
		if err != nil || tenant.Status != "active" {
			return nil, newServiceError(errcode.ErrTenantNotFound, "租户不存在或已停用")
		}
	}

	// 6. 查找用户的全部角色分配
	assignments, err := s.userRepo.FindRoleAssignments(user.ID)
	if err != nil || len(assignments) == 0 {
		return nil, newServiceError(errcode.ErrNoRoleInTenant, "用户在该租户无角色分配")
	}

	// 7. 若指定了 tenant_id，过滤出该租户下的角色分配
	filtered := assignments
	if req.TenantID != "" && req.PreferredRole != "system_admin" {
		tenantUUID, _ := uuid.Parse(req.TenantID)
		filtered = filterAssignmentsByTenant(assignments, &tenantUUID, false)
		if len(filtered) == 0 {
			return nil, newServiceError(errcode.ErrNoRoleInTenant, "用户在该租户无角色分配")
		}
	}

	// 8. 按优先级选择活跃角色（preferred_role > system_admin > tenant_admin > business）
	activeAssignment := selectActiveRole(filtered, req.PreferredRole)

	// 若指定了 preferred_role 且不是 system_admin，但最终选中的角色不匹配，说明该租户下没有对应角色
	if req.PreferredRole != "" && req.PreferredRole != "system_admin" && activeAssignment.Role != req.PreferredRole {
		return nil, newServiceError(errcode.ErrNoRoleInTenant, "用户在该租户下没有对应角色")
	}

	// 9. 重置登录失败次数
	if err := s.userRepo.ResetLoginFail(user.ID); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	// 构建活跃角色声明
	activeRoleClaim := buildActiveRoleClaim(activeAssignment, tenant)

	// 收集所有角色 ID，用于令牌中携带
	allRoleIDs := make([]string, len(assignments))
	for i, a := range assignments {
		allRoleIDs[i] = a.ID.String()
	}

	// 权限列表（业务用户由 GetMenu 填充，管理员角色为空）
	permissions := []string{}

	// 10. 生成 access_token（优先使用数据库配置的 TTL）
	claims := &jwtpkg.JWTClaims{
		Sub:         user.ID.String(),
		Username:    user.Username,
		DisplayName: user.DisplayName,
		ActiveRole:  activeRoleClaim,
		Permissions: permissions,
		AllRoleIDs:  allRoleIDs,
	}
	accessToken, err := jwtpkg.GenerateAccessTokenWithTTL(claims, s.getAccessTokenTTL())
	if err != nil {
		return nil, newServiceError(errcode.ErrInternalServer, "服务器内部错误")
	}

	// 11. 生成 refresh_token（优先使用数据库配置的 TTL）
	refreshToken, refreshJTI, err := jwtpkg.GenerateRefreshTokenWithTTL(user.ID.String(), "", s.getRefreshTokenTTL())
	if err != nil {
		return nil, newServiceError(errcode.ErrInternalServer, "服务器内部错误")
	}

	// 12. 写入登录历史记录
	var loginTenantID *uuid.UUID
	if req.TenantID != "" {
		tid, _ := uuid.Parse(req.TenantID)
		loginTenantID = &tid
	}
	history := &model.LoginHistory{
		ID:        uuid.New(),
		UserID:    user.ID,
		TenantID:  loginTenantID,
		IP:        clientIP,
		UserAgent: userAgent,
		LoginAt:   time.Now(),
	}
	_ = s.userRepo.CreateLoginHistory(history)

	// 13. 缓存 session 到 Redis，key 为 "session:{user_id}"，TTL 2h
	sessionData := map[string]interface{}{
		"user_id":      user.ID.String(),
		"username":     user.Username,
		"display_name": user.DisplayName,
		"active_role":  activeRoleClaim,
		"all_role_ids": allRoleIDs,
		"permissions":  permissions,
		"refresh_jti":  refreshJTI,
	}
	sessionJSON, _ := json.Marshal(sessionData)
	sessionKey := fmt.Sprintf("session:%s", user.ID.String())
	s.rdb.Set(context.Background(), sessionKey, string(sessionJSON), s.getSessionCacheTTL())

	// 14. 批量查询所有角色分配对应的租户名称和状态
	tenantNameCache := make(map[string]string)
	tenantStatusCache := make(map[string]string)
	if tenant != nil {
		tenantNameCache[tenant.ID.String()] = tenant.Name
		tenantStatusCache[tenant.ID.String()] = tenant.Status
	}
	for _, a := range assignments {
		if a.TenantID != nil {
			tidStr := a.TenantID.String()
			if _, exists := tenantNameCache[tidStr]; !exists {
				if t, err := s.userRepo.FindTenantByID(*a.TenantID); err == nil {
					tenantNameCache[tidStr] = t.Name
					tenantStatusCache[tidStr] = t.Status
				}
			}
		}
	}

	// 15. 构建响应，过滤掉已停用租户的角色
	roles := make([]dto.RoleInfo, 0, len(assignments))
	for _, a := range assignments {
		var tid *string
		var tname *string
		if a.TenantID != nil {
			s := a.TenantID.String()
			// 跳过已停用租户的角色
			if status, ok := tenantStatusCache[s]; ok && status != "active" {
				continue
			}
			tid = &s
			if name, ok := tenantNameCache[s]; ok {
				tname = &name
			}
		}
		roles = append(roles, dto.RoleInfo{
			ID:         a.ID.String(),
			Role:       a.Role,
			TenantID:   tid,
			TenantName: tname,
			Label:      a.Label,
		})
	}

	resp := &dto.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: dto.UserInfo{
			ID:          user.ID.String(),
			Username:    user.Username,
			DisplayName: user.DisplayName,
			Email:       user.Email,
			Phone:       user.Phone,
			Locale:      user.Locale,
		},
		Roles: roles,
		ActiveRole: dto.RoleInfo{
			ID:         activeAssignment.ID.String(),
			Role:       activeAssignment.Role,
			TenantID:   activeRoleClaim.TenantID,
			TenantName: activeRoleClaim.TenantName,
			Label:      activeAssignment.Label,
		},
		Permissions: permissions,
	}

	logFields := []zap.Field{
		zap.String("username", user.Username),
		zap.String("role", activeAssignment.Role),
	}
	if activeRoleClaim.TenantID != nil {
		logFields = append(logFields, zap.String("tenantID", *activeRoleClaim.TenantID))
	}
	pkglogger.Global().Info("登录成功", logFields...)

	return resp, nil
}

// ---------------------------------------------------------------------------
// 退出登录
// ---------------------------------------------------------------------------

// LogoutRequest 注销所需的令牌 JTI 和用户 ID
type LogoutRequest struct {
	AccessJTI  string
	RefreshJTI string
	UserID     string
}

// Logout 使访问令牌和刷新令牌失效，并清除 Redis 中的 session 缓存。
func (s *AuthService) Logout(req *LogoutRequest) error {
	ctx := context.Background()

	// 1. 将 access_token JTI 加入黑名单，TTL 与令牌有效期一致（2h）
	if req.AccessJTI != "" {
		blacklistKey := fmt.Sprintf("blacklist:%s", req.AccessJTI)
		s.rdb.Set(ctx, blacklistKey, "1", 2*time.Hour)
	}

	// 2. 将 refresh_token JTI 加入黑名单，TTL 7 天
	if req.RefreshJTI != "" {
		blacklistKey := fmt.Sprintf("blacklist:%s", req.RefreshJTI)
		s.rdb.Set(ctx, blacklistKey, "1", 7*24*time.Hour)
	}

	// 3. 删除 session 缓存
	if req.UserID != "" {
		sessionKey := fmt.Sprintf("session:%s", req.UserID)
		s.rdb.Del(ctx, sessionKey)
	}

	pkglogger.Global().Info("用户已退出登录", zap.String("userID", req.UserID))
	return nil
}

// ---------------------------------------------------------------------------
// 刷新令牌
// ---------------------------------------------------------------------------

// Refresh 校验刷新令牌并签发新的访问令牌。
// 优先从 Redis session 缓存重建 claims，缓存失效时降级查询数据库。
func (s *AuthService) Refresh(req *dto.RefreshRequest) (*dto.RefreshResponse, error) {
	ctx := context.Background()

	// 1. 解析 refresh_token（标准 JWT，含 RegisteredClaims）
	claims, err := parseRefreshToken(req.RefreshToken)
	if err != nil {
		return nil, newServiceError(errcode.ErrTokenInvalid, "认证令牌无效或已过期")
	}

	// 2. 检查 refresh_token JTI 是否已被吊销
	blacklistKey := fmt.Sprintf("blacklist:%s", claims.ID)
	exists, err := s.rdb.Exists(ctx, blacklistKey).Result()
	if err != nil {
		return nil, newServiceError(errcode.ErrRedisConn, "Redis 连接错误")
	}
	if exists > 0 {
		return nil, newServiceError(errcode.ErrTokenRevoked, "令牌已被吊销")
	}

	// 3. 解析用户 ID
	userID, parseErr := uuid.Parse(claims.Subject)
	if parseErr != nil {
		return nil, newServiceError(errcode.ErrTokenInvalid, "认证令牌无效或已过期")
	}

	sessionKey := fmt.Sprintf("session:%s", claims.Subject)
	sessionJSON, err := s.rdb.Get(ctx, sessionKey).Result()

	var accessToken string

	if err == nil && sessionJSON != "" {
		// 从缓存的 session 重建 claims，避免查库
		var sessionData map[string]interface{}
		if jsonErr := json.Unmarshal([]byte(sessionJSON), &sessionData); jsonErr == nil {
			jwtClaims := rebuildClaimsFromSession(sessionData)
			accessToken, err = jwtpkg.GenerateAccessTokenWithTTL(jwtClaims, s.getAccessTokenTTL())
			if err != nil {
				return nil, newServiceError(errcode.ErrInternalServer, "服务器内部错误")
			}
		}
	}

	// 降级：session 缓存失效时重新查询用户和角色分配
	if accessToken == "" {
		user, findErr := s.userRepo.FindByID(userID)
		if findErr != nil {
			return nil, newServiceError(errcode.ErrTokenInvalid, "认证令牌无效或已过期")
		}

		assignments, findErr := s.userRepo.FindRoleAssignments(user.ID)
		if findErr != nil || len(assignments) == 0 {
			return nil, newServiceError(errcode.ErrNoRoleInTenant, "用户在该租户无角色分配")
		}

		activeAssignment := selectActiveRole(assignments, "")
		activeRoleClaim := buildActiveRoleClaim(activeAssignment, nil)

		allRoleIDs := make([]string, len(assignments))
		for i, a := range assignments {
			allRoleIDs[i] = a.ID.String()
		}

		jwtClaims := &jwtpkg.JWTClaims{
			Sub:         user.ID.String(),
			Username:    user.Username,
			DisplayName: user.DisplayName,
			ActiveRole:  activeRoleClaim,
			Permissions: []string{},
			AllRoleIDs:  allRoleIDs,
		}
		accessToken, err = jwtpkg.GenerateAccessTokenWithTTL(jwtClaims, s.getAccessTokenTTL())
		if err != nil {
			return nil, newServiceError(errcode.ErrInternalServer, "服务器内部错误")
		}

		// 刷新成功后重建 session 缓存，避免后续刷新再次查库
		sessionData := map[string]interface{}{
			"user_id":      user.ID.String(),
			"username":     user.Username,
			"display_name": user.DisplayName,
			"active_role":  activeRoleClaim,
			"all_role_ids": allRoleIDs,
			"permissions":  []string{},
		}
		rebuildJSON, _ := json.Marshal(sessionData)
		s.rdb.Set(ctx, sessionKey, string(rebuildJSON), s.getSessionCacheTTL())
	}

	return &dto.RefreshResponse{AccessToken: accessToken}, nil
}

// ---------------------------------------------------------------------------
// 切换角色
// ---------------------------------------------------------------------------

// SwitchRole 验证目标角色归属、生成新令牌、将旧令牌加入黑名单并更新 session 缓存。
func (s *AuthService) SwitchRole(userID uuid.UUID, roleID string, oldJTI string) (*dto.SwitchRoleResponse, error) {
	ctx := context.Background()

	// 1. 解析 roleID 并验证该角色分配属于当前用户
	roleUUID, parseErr := uuid.Parse(roleID)
	if parseErr != nil {
		return nil, newServiceError(errcode.ErrRoleSwitchFailed, "角色切换失败")
	}

	assignment, err := s.userRepo.FindRoleAssignmentByID(roleUUID)
	if err != nil {
		return nil, newServiceError(errcode.ErrRoleSwitchFailed, "角色切换失败")
	}
	if assignment.UserID != userID {
		pkglogger.Global().Warn("角色切换失败：角色不属于该用户",
			zap.String("userID", userID.String()),
			zap.String("roleID", roleID),
		)
		return nil, newServiceError(errcode.ErrRoleSwitchFailed, "角色切换失败")
	}

	// 2. 构建新的 ActiveRoleClaim，同时校验目标租户是否处于活跃状态
	var tenant *model.Tenant
	if assignment.TenantID != nil {
		tenant, _ = s.userRepo.FindTenantByID(*assignment.TenantID)
		if tenant == nil || tenant.Status != "active" {
			return nil, newServiceError(errcode.ErrTenantNotFound, "租户不存在或已停用")
		}
	}
	activeRoleClaim := buildActiveRoleClaim(assignment, tenant)

	// 3. 查询用户基本信息用于生成令牌
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, newServiceError(errcode.ErrInternalServer, "服务器内部错误")
	}

	// 获取所有角色 ID
	assignments, err := s.userRepo.FindRoleAssignments(userID)
	if err != nil {
		return nil, newServiceError(errcode.ErrInternalServer, "服务器内部错误")
	}
	allRoleIDs := make([]string, len(assignments))
	for i, a := range assignments {
		allRoleIDs[i] = a.ID.String()
	}

	permissions := []string{}

	// 4. 使用新的 activeRole 生成 access_token（优先使用数据库配置的 TTL）
	claims := &jwtpkg.JWTClaims{
		Sub:         user.ID.String(),
		Username:    user.Username,
		DisplayName: user.DisplayName,
		ActiveRole:  activeRoleClaim,
		Permissions: permissions,
		AllRoleIDs:  allRoleIDs,
	}
	accessToken, err := jwtpkg.GenerateAccessTokenWithTTL(claims, s.getAccessTokenTTL())
	if err != nil {
		return nil, newServiceError(errcode.ErrInternalServer, "服务器内部错误")
	}

	// 5. 将旧 JTI 加入黑名单，使旧令牌立即失效
	if oldJTI != "" {
		blacklistKey := fmt.Sprintf("blacklist:%s", oldJTI)
		s.rdb.Set(ctx, blacklistKey, "1", 2*time.Hour)
	}

	// 6. 更新 session 缓存
	sessionData := map[string]interface{}{
		"user_id":      user.ID.String(),
		"username":     user.Username,
		"display_name": user.DisplayName,
		"active_role":  activeRoleClaim,
		"all_role_ids": allRoleIDs,
		"permissions":  permissions,
	}
	sessionJSON, _ := json.Marshal(sessionData)
	sessionKey := fmt.Sprintf("session:%s", user.ID.String())
	s.rdb.Set(ctx, sessionKey, string(sessionJSON), s.getSessionCacheTTL())

	// 7. 获取新角色对应的菜单
	menuResp, _ := s.GetMenu(activeRoleClaim, user.ID.String(), func() string {
		if assignment.TenantID != nil {
			return assignment.TenantID.String()
		}
		return ""
	}())
	var menuItems []dto.MenuItem
	if menuResp != nil {
		menuItems = menuResp.Menus
	}

	// 8. 构建并返回切换角色响应
	var tid *string
	var tname *string
	if assignment.TenantID != nil {
		s := assignment.TenantID.String()
		tid = &s
		if tenant != nil {
			tname = &tenant.Name
		}
	}

	pkglogger.Global().Info("角色切换成功",
		zap.String("userID", userID.String()),
		zap.String("role", assignment.Role),
	)
	return &dto.SwitchRoleResponse{
		AccessToken: accessToken,
		ActiveRole: dto.RoleInfo{
			ID:         assignment.ID.String(),
			Role:       assignment.Role,
			TenantID:   tid,
			TenantName: tname,
			Label:      assignment.Label,
		},
		Permissions: permissions,
		Menus:       menuItems,
	}, nil
}
// ---------------------------------------------------------------------------

// GetMenu 根据用户的活跃角色返回菜单项。
// system_admin 返回固定的系统管理菜单；tenant_admin 和 business 从 org_roles.page_permissions 动态读取。
func (s *AuthService) GetMenu(activeRole jwtpkg.ActiveRoleClaim, userID string, tenantID string) (*dto.MenuResponse, error) {
	switch activeRole.Role {
	case "system_admin":
		// system_admin 无 org_member 记录，使用硬编码菜单
		return &dto.MenuResponse{
			Menus: []dto.MenuItem{
				{Key: "tenant-management", Label: "租户管理", Path: "/admin/system/tenants"},
				{Key: "system-settings", Label: "系统设置", Path: "/admin/system/settings"},
			},
		}, nil

	case "tenant_admin", "business":
		// tenant_admin 和 business 统一从 org_roles.page_permissions 读取，按系统角色过滤
		return s.getMenuFromOrgRoles(userID, tenantID, activeRole.Role)

	default:
		return &dto.MenuResponse{Menus: []dto.MenuItem{}}, nil
	}
}

// getMenuFromOrgRoles 查询用户的 OrgMember + OrgRoles 并合并 page_permissions。
// activeSystemRole 用于过滤：tenant_admin 只返回后台管理路径，business 只返回前台业务路径。
func (s *AuthService) getMenuFromOrgRoles(userID string, tenantID string, activeSystemRole string) (*dto.MenuResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return &dto.MenuResponse{Menus: []dto.MenuItem{}}, nil
	}
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return &dto.MenuResponse{Menus: []dto.MenuItem{}}, nil
	}

	// 查询 org_members，预加载角色信息
	var members []model.OrgMember
	if err := s.db.Where("user_id = ? AND tenant_id = ?", uid, tid).
		Preload("Roles").
		Find(&members).Error; err != nil {
		return &dto.MenuResponse{Menus: []dto.MenuItem{}}, nil
	}

	// 合并所有角色的 page_permissions，去重
	seen := make(map[string]bool)
	var menus []dto.MenuItem

	// 路径到菜单项的映射（业务页面 + 租户管理页面）
	pathLabels := map[string]struct{ key, label string }{
		"/overview":                  {key: "overview", label: "概览"},
		"/dashboard":                 {key: "dashboard", label: "审核工作台"},
		"/cron":                      {key: "cron", label: "定时任务"},
		"/archive":                   {key: "archive", label: "归档复盘"},
		"/settings":                  {key: "settings", label: "个人设置"},
		"/admin/tenant/rules":        {key: "rules-management", label: "规则管理"},
		"/admin/tenant/org":          {key: "org-management", label: "组织管理"},
		"/admin/tenant/data":         {key: "data-management", label: "数据管理"},
		"/admin/tenant/user-configs": {key: "user-configs", label: "用户配置"},
	}

	for _, member := range members {
		for _, role := range member.Roles {
			// 优先尝试解析为字符串数组（标准格式）
			var paths []string
			if err := json.Unmarshal(role.PagePermissions, &paths); err == nil {
				for _, p := range paths {
					if !seen[p] {
						seen[p] = true
						info, ok := pathLabels[p]
						if !ok {
							info = struct{ key, label string }{key: p, label: p}
						}
						menus = append(menus, dto.MenuItem{Key: info.key, Label: info.label, Path: p})
					}
				}
				continue
			}
			// 兼容旧格式：尝试解析为 []dto.MenuItem
			var items []dto.MenuItem
			if err := json.Unmarshal(role.PagePermissions, &items); err != nil {
				continue
			}
			for _, item := range items {
				if !seen[item.Key] {
					seen[item.Key] = true
					menus = append(menus, item)
				}
			}
		}
	}

	if menus == nil {
		menus = []dto.MenuItem{}
	}

	// 按系统角色过滤：tenant_admin 只看后台管理页面，business 只看前台业务页面
	// /overview 和 /settings 对所有角色通用
	if activeSystemRole == "tenant_admin" {
		var filtered []dto.MenuItem
		for _, m := range menus {
			if strings.HasPrefix(m.Path, "/admin/tenant/") || m.Path == "/overview" || m.Path == "/settings" {
				filtered = append(filtered, m)
			}
		}
		if filtered == nil {
			filtered = []dto.MenuItem{}
		}
		menus = filtered
	} else if activeSystemRole == "business" {
		var filtered []dto.MenuItem
		for _, m := range menus {
			if !strings.HasPrefix(m.Path, "/admin/") {
				filtered = append(filtered, m)
			}
		}
		if filtered == nil {
			filtered = []dto.MenuItem{}
		}
		menus = filtered
	}

	return &dto.MenuResponse{Menus: menus}, nil
}

// ---------------------------------------------------------------------------
// 辅助函数
// ---------------------------------------------------------------------------

// filterAssignmentsByTenant 返回与给定租户 ID 匹配的角色分配。
// includeSystemAdmin 控制是否保留 system_admin 分配（TenantID 为空）。
func filterAssignmentsByTenant(assignments []model.UserRoleAssignment, tenantID *uuid.UUID, includeSystemAdmin bool) []model.UserRoleAssignment {
	var result []model.UserRoleAssignment
	for _, a := range assignments {
		if a.Role == "system_admin" {
			if includeSystemAdmin {
				result = append(result, a)
			}
			continue
		}
		if a.TenantID != nil && tenantID != nil && *a.TenantID == *tenantID {
			result = append(result, a)
		}
	}
	return result
}

// selectActiveRole 按优先级选择最佳角色：
// 首选角色匹配 > system_admin > tenant_admin > business
func selectActiveRole(assignments []model.UserRoleAssignment, preferredRole string) *model.UserRoleAssignment {
	// 优先匹配 preferred_role
	if preferredRole != "" {
		for i := range assignments {
			if assignments[i].Role == preferredRole {
				return &assignments[i]
			}
		}
	}

	// 按优先级回退
	priorities := []string{"system_admin", "tenant_admin", "business"}
	for _, role := range priorities {
		for i := range assignments {
			if assignments[i].Role == role {
				return &assignments[i]
			}
		}
	}

	// 兜底返回第一个分配
	if len(assignments) > 0 {
		return &assignments[0]
	}
	return nil
}

// buildActiveRoleClaim 根据角色分配和可选租户构造 ActiveRoleClaim。
func buildActiveRoleClaim(assignment *model.UserRoleAssignment, tenant *model.Tenant) jwtpkg.ActiveRoleClaim {
	claim := jwtpkg.ActiveRoleClaim{
		ID:    assignment.ID.String(),
		Role:  assignment.Role,
		Label: assignment.Label,
	}
	if assignment.TenantID != nil {
		tid := assignment.TenantID.String()
		claim.TenantID = &tid
	}
	if tenant != nil {
		claim.TenantName = &tenant.Name
	}
	return claim
}

// parseRefreshToken 解析刷新令牌（标准 JWT，含 RegisteredClaims）。
func parseRefreshToken(tokenString string) (*jwtpkg.JWTClaims, error) {
	return jwtpkg.ParseRefreshToken(tokenString)
}

// rebuildClaimsFromSession 从 Redis 缓存的 session 数据重建 JWTClaims。
func rebuildClaimsFromSession(data map[string]interface{}) *jwtpkg.JWTClaims {
	claims := &jwtpkg.JWTClaims{}

	if v, ok := data["user_id"].(string); ok {
		claims.Sub = v
	}
	if v, ok := data["username"].(string); ok {
		claims.Username = v
	}
	if v, ok := data["display_name"].(string); ok {
		claims.DisplayName = v
	}

	// 从 map 重建 ActiveRole
	if ar, ok := data["active_role"].(map[string]interface{}); ok {
		claims.ActiveRole = jwtpkg.ActiveRoleClaim{}
		if v, ok := ar["id"].(string); ok {
			claims.ActiveRole.ID = v
		}
		if v, ok := ar["role"].(string); ok {
			claims.ActiveRole.Role = v
		}
		if v, ok := ar["tenant_id"].(string); ok {
			claims.ActiveRole.TenantID = &v
		}
		if v, ok := ar["tenant_name"].(string); ok {
			claims.ActiveRole.TenantName = &v
		}
		if v, ok := ar["label"].(string); ok {
			claims.ActiveRole.Label = v
		}
	}

	// 重建所有角色 ID
	if ids, ok := data["all_role_ids"].([]interface{}); ok {
		for _, id := range ids {
			if s, ok := id.(string); ok {
				claims.AllRoleIDs = append(claims.AllRoleIDs, s)
			}
		}
	}

	// 重建权限列表
	if perms, ok := data["permissions"].([]interface{}); ok {
		for _, p := range perms {
			if s, ok := p.(string); ok {
				claims.Permissions = append(claims.Permissions, s)
			}
		}
	}

	return claims
}

// ---------------------------------------------------------------------------
// ChangePassword
// ---------------------------------------------------------------------------

// ChangePassword 校验当前密码后更新为新密码，新旧密码不能相同。
func (s *AuthService) ChangePassword(userID uuid.UUID, req *dto.ChangePasswordRequest) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return newServiceError(errcode.ErrWrongPassword, "用户不存在")
	}

	if !hash.CheckPassword(req.CurrentPassword, user.PasswordHash) {
		pkglogger.Global().Warn("修改密码失败：当前密码错误", zap.String("userID", userID.String()))
		return newServiceError(errcode.ErrWrongPassword, "当前密码错误")
	}

	// 新密码不能与当前密码相同
	if req.CurrentPassword == req.NewPassword {
		return newServiceError(errcode.ErrParamValidation, "新密码不能与当前密码相同")
	}

	newHash, err := hash.HashPassword(req.NewPassword)
	if err != nil {
		return newServiceError(errcode.ErrInternalServer, "服务器内部错误")
	}

	if err := s.userRepo.UpdatePasswordHashAndTime(userID, newHash); err != nil {
		return newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	pkglogger.Global().Info("密码修改成功", zap.String("userID", userID.String()))
	return nil
}

// ---------------------------------------------------------------------------
// GetMe
// ---------------------------------------------------------------------------

// GetMe 返回当前用户的完整个人信息，包含组织层级信息、角色列表和近期登录历史。
func (s *AuthService) GetMe(userID uuid.UUID, activeRole jwtpkg.ActiveRoleClaim, allRoleIDs []string) (*dto.MeResponse, error) {
	// 1. 查询用户基本信息
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, newServiceError(errcode.ErrTokenInvalid, "用户不存在")
	}

	// 2. 查询所有角色分配，用于构建角色列表
	assignments, err := s.userRepo.FindRoleAssignments(userID)
	if err != nil {
		assignments = []model.UserRoleAssignment{}
	}

	// 批量查询租户名称和状态
	tenantNameCache := make(map[string]string)
	tenantStatusCache := make(map[string]string)
	for _, a := range assignments {
		if a.TenantID != nil {
			tidStr := a.TenantID.String()
			if _, exists := tenantNameCache[tidStr]; !exists {
				if t, tErr := s.userRepo.FindTenantByID(*a.TenantID); tErr == nil {
					tenantNameCache[tidStr] = t.Name
					tenantStatusCache[tidStr] = t.Status
				}
			}
		}
	}

	roles := make([]dto.RoleInfo, 0, len(assignments))
	for _, a := range assignments {
		var tid *string
		var tname *string
		if a.TenantID != nil {
			s := a.TenantID.String()
			// 跳过已停用租户的角色
			if status, ok := tenantStatusCache[s]; ok && status != "active" {
				continue
			}
			tid = &s
			if name, ok := tenantNameCache[s]; ok {
				tname = &name
			}
		}
		roles = append(roles, dto.RoleInfo{
			ID:         a.ID.String(),
			Role:       a.Role,
			TenantID:   tid,
			TenantName: tname,
			Label:      a.Label,
		})
	}

	activeRoleDTO := dto.RoleInfo{
		ID:         activeRole.ID,
		Role:       activeRole.Role,
		TenantID:   activeRole.TenantID,
		TenantName: activeRole.TenantName,
		Label:      activeRole.Label,
	}

	resp := &dto.MeResponse{
		User: dto.UserInfo{
			ID:          user.ID.String(),
			Username:    user.Username,
			DisplayName: user.DisplayName,
			Email:       user.Email,
			Phone:       user.Phone,
			Locale:      user.Locale,
		},
		Roles:             roles,
		ActiveRole:        activeRoleDTO,
		PasswordChangedAt: user.PasswordChangedAt.Format("2006-01-02 15:04:05"),
	}

	// 3. 若活跃角色关联了租户，查询组织层级信息（部门、职位、组织角色、页面权限）
	if activeRole.TenantID != nil && *activeRole.TenantID != "" {
		tid, parseErr := uuid.Parse(*activeRole.TenantID)
		if parseErr == nil {
			if tname, ok := tenantNameCache[tid.String()]; ok {
				resp.TenantName = tname
			} else if t, tErr := s.userRepo.FindTenantByID(tid); tErr == nil {
				resp.TenantName = t.Name
			}

			// 查询 org_members，预加载部门和角色
			var members []model.OrgMember
			if err := s.db.Where("user_id = ? AND tenant_id = ?", userID, tid).
				Preload("Department").
				Preload("Roles").
				Find(&members).Error; err == nil && len(members) > 0 {

				member := members[0]
				resp.DepartmentName = member.Department.Name
				resp.Position = member.Position

				// 合并所有组织角色的 page_permissions
				permSet := make(map[string]bool)
				var orgRoles []dto.MeOrgRole
				for _, role := range member.Roles {
					var perms []string
					if err := json.Unmarshal(role.PagePermissions, &perms); err != nil {
						perms = []string{}
					}
					for _, p := range perms {
						permSet[p] = true
					}
					orgRoles = append(orgRoles, dto.MeOrgRole{
						ID:              role.ID.String(),
						Name:            role.Name,
						Description:     role.Description,
						PagePermissions: perms,
						IsSystem:        role.IsSystem,
					})
				}
				resp.OrgRoles = orgRoles

				allPerms := make([]string, 0, len(permSet))
				for p := range permSet {
					allPerms = append(allPerms, p)
				}
				resp.PagePermissions = allPerms
			}
		}
	}

	// 确保切片非 nil
	if resp.OrgRoles == nil {
		resp.OrgRoles = []dto.MeOrgRole{}
	}
	if resp.PagePermissions == nil {
		resp.PagePermissions = []string{}
	}

	// 4. 查询近期登录历史（最近 5 条）
	loginHistories, _ := s.userRepo.FindRecentLoginHistory(userID, 5)
	loginItems := make([]dto.LoginHistoryItem, 0, len(loginHistories))
	for _, h := range loginHistories {
		loginItems = append(loginItems, dto.LoginHistoryItem{
			Time:   h.LoginAt.Format("2006-01-02 15:04:05"),
			IP:     h.IP,
			Device: h.UserAgent,
		})
	}
	resp.LoginHistory = loginItems

	return resp, nil
}

// ---------------------------------------------------------------------------
// UpdateLocale
// ---------------------------------------------------------------------------

// UpdateLocale 更新用户的语言偏好设置，仅支持 zh-CN 和 en-US。
func (s *AuthService) UpdateLocale(userID uuid.UUID, locale string) error {
	// 校验语言设置合法性
	switch locale {
	case "zh-CN", "en-US":
		// 合法值
	default:
		return newServiceError(errcode.ErrParamValidation, "不支持的语言设置")
	}

	if err := s.userRepo.UpdateLocale(userID, locale); err != nil {
		return newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	return nil
}

// ---------------------------------------------------------------------------
// UpdateProfile
// ---------------------------------------------------------------------------

// UpdateProfile 更新用户的显示名称、邮箱和手机号，更新前进行格式校验。
func (s *AuthService) UpdateProfile(userID uuid.UUID, req *dto.UpdateProfileRequest) error {
	// 校验邮箱格式（若提供）
	if req.Email != "" {
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(req.Email) {
			return newServiceError(errcode.ErrParamValidation, "邮箱格式不正确")
		}
	}
	// 校验手机号格式：必须为 11 位数字（若提供）
	if req.Phone != "" {
		phoneRegex := regexp.MustCompile(`^\d{11}$`)
		if !phoneRegex.MatchString(req.Phone) {
			return newServiceError(errcode.ErrParamValidation, "手机号必须为11位数字")
		}
	}

	updates := map[string]interface{}{}
	if req.DisplayName != "" {
		updates["display_name"] = req.DisplayName
	}
	// 邮箱和手机号允许传空字符串以清空，直接覆盖写入
	updates["email"] = req.Email
	updates["phone"] = req.Phone

	if len(updates) == 0 {
		return nil
	}

	if err := s.userRepo.UpdateProfile(userID, updates); err != nil {
		return newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	return nil
}
