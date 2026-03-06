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

	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/pkg/hash"
	jwtpkg "oa-smart-audit/go-service/internal/pkg/jwt"
	"oa-smart-audit/go-service/internal/repository"
)

//AuthService 处理身份验证、令牌管理、角色切换和菜单检索。
type AuthService struct {
	userRepo *repository.UserRepo
	rdb      *redis.Client
	db       *gorm.DB
}

//NewAuthService 创建一个新的 AuthService 实例。
func NewAuthService(userRepo *repository.UserRepo, rdb *redis.Client, db *gorm.DB) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		rdb:      rdb,
		db:       db,
	}
}

//ServiceError携带了handler层的业务错误码和消息。
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
//登录
// ---------------------------------------------------------------------------

//登录对用户进行身份验证并返回令牌、用户信息、角色和活动角色。
func (s *AuthService) Login(req *dto.LoginRequest, clientIP string, userAgent string) (*dto.LoginResponse, error) {
	// Normalize IPv6 loopback to IPv4 for readability
	if clientIP == "::1" {
		clientIP = "127.0.0.1"
	}

	//1.通过用户名查找用户
	user, err := s.userRepo.FindByUsername(req.Username)
	if err != nil {
		return nil, newServiceError(errcode.ErrWrongPassword, "用户名或密码错误")
	}

	//2. 检查禁用状态
	if user.Status == "disabled" {
		return nil, newServiceError(errcode.ErrAccountDisabled, "账户已被禁用")
	}

	//3. 检查锁定：login_fail_count >= 5 AND Locked_until > now
	if user.LoginFailCount >= 5 && user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return nil, newServiceError(errcode.ErrAccountLocked, "账户被锁定")
	}

	//4. 验证密码
	if !hash.CheckPassword(req.Password, user.PasswordHash) {
		_ = s.userRepo.UpdateLoginFail(user)
		return nil, newServiceError(errcode.ErrWrongPassword, "用户名或密码错误")
	}

	//5. 如果提供了tenant_id并且preferred_role != system_admin，则验证租户
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

	//6. 查找用户的角色分配
	assignments, err := s.userRepo.FindRoleAssignments(user.ID)
	if err != nil || len(assignments) == 0 {
		return nil, newServiceError(errcode.ErrNoRoleInTenant, "用户在该租户无角色分配")
	}

	//7. 按tenant_id（如果提供）过滤分配
	filtered := assignments
	if req.TenantID != "" && req.PreferredRole != "system_admin" {
		tenantUUID, _ := uuid.Parse(req.TenantID)
		filtered = filterAssignmentsByTenant(assignments, &tenantUUID, false)
		if len(filtered) == 0 {
			return nil, newServiceError(errcode.ErrNoRoleInTenant, "用户在该租户无角色分配")
		}
	}

	//8.按优先级选择activeRole
	activeAssignment := selectActiveRole(filtered, req.PreferredRole)

	// 如果指定了 preferred_role 且不是 system_admin，但最终选中的角色不匹配，说明该租户下没有对应角色
	if req.PreferredRole != "" && req.PreferredRole != "system_admin" && activeAssignment.Role != req.PreferredRole {
		return nil, newServiceError(errcode.ErrNoRoleInTenant, "用户在该租户下没有对应角色")
	}

	//9.重置登录失败次数
	if err := s.userRepo.ResetLoginFail(user.ID); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	//建立积极的角色主张
	activeRoleClaim := buildActiveRoleClaim(activeAssignment, tenant)

	//收集所有角色 ID
	allRoleIDs := make([]string, len(assignments))
	for i, a := range assignments {
		allRoleIDs[i] = a.ID.String()
	}

	//构建权限（对于业务用户，将由 GetMenu 填充；对于管理员角色，为空）
	permissions := []string{}

	//10.生成access_token
	claims := &jwtpkg.JWTClaims{
		Sub:         user.ID.String(),
		Username:    user.Username,
		DisplayName: user.DisplayName,
		ActiveRole:  activeRoleClaim,
		Permissions: permissions,
		AllRoleIDs:  allRoleIDs,
	}
	accessToken, err := jwtpkg.GenerateAccessToken(claims)
	if err != nil {
		return nil, newServiceError(errcode.ErrInternalServer, "服务器内部错误")
	}

	//11.生成refresh_token
	refreshToken, refreshJTI, err := jwtpkg.GenerateRefreshToken(user.ID.String(), "")
	if err != nil {
		return nil, newServiceError(errcode.ErrInternalServer, "服务器内部错误")
	}

	//12.创建登录历史记录
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

	//13.Redis中缓存session：key "session:{user_id}", TTL 2h
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
	s.rdb.Set(context.Background(), sessionKey, string(sessionJSON), 2*time.Hour)

	//14. Batch-fetch tenant names for all assignments
	tenantNameCache := make(map[string]string) // tenantID string -> tenant name
	if tenant != nil {
		tenantNameCache[tenant.ID.String()] = tenant.Name
	}
	for _, a := range assignments {
		if a.TenantID != nil {
			tidStr := a.TenantID.String()
			if _, exists := tenantNameCache[tidStr]; !exists {
				if t, err := s.userRepo.FindTenantByID(*a.TenantID); err == nil {
					tenantNameCache[tidStr] = t.Name
				}
			}
		}
	}

	//15. 建立响应
	roles := make([]dto.RoleInfo, len(assignments))
	for i, a := range assignments {
		var tid *string
		var tname *string
		if a.TenantID != nil {
			s := a.TenantID.String()
			tid = &s
			if name, ok := tenantNameCache[s]; ok {
				tname = &name
			}
		}
		roles[i] = dto.RoleInfo{
			ID:         a.ID.String(),
			Role:       a.Role,
			TenantID:   tid,
			TenantName: tname,
			Label:      a.Label,
		}
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

	return resp, nil
}

// ---------------------------------------------------------------------------
//退出
// ---------------------------------------------------------------------------

//LogoutRequest 保存注销所需的 JTI 和用户 ID。
type LogoutRequest struct {
	AccessJTI  string
	RefreshJTI string
	UserID     string
}

//注销会使两个令牌失效并删除会话缓存。
func (s *AuthService) Logout(req *LogoutRequest) error {
	ctx := context.Background()

	//1.将access_token JTI添加到黑名单（默认TTL = 2h）
	if req.AccessJTI != "" {
		blacklistKey := fmt.Sprintf("blacklist:%s", req.AccessJTI)
		s.rdb.Set(ctx, blacklistKey, "1", 2*time.Hour)
	}

	//2.将refresh_token JTI添加到黑名单（TTL = 7d）
	if req.RefreshJTI != "" {
		blacklistKey := fmt.Sprintf("blacklist:%s", req.RefreshJTI)
		s.rdb.Set(ctx, blacklistKey, "1", 7*24*time.Hour)
	}

	//3.删除会话缓存
	if req.UserID != "" {
		sessionKey := fmt.Sprintf("session:%s", req.UserID)
		s.rdb.Del(ctx, sessionKey)
	}

	return nil
}

// ---------------------------------------------------------------------------
//刷新
// ---------------------------------------------------------------------------

//刷新验证刷新令牌并返回新的访问令牌。
func (s *AuthService) Refresh(req *dto.RefreshRequest) (*dto.RefreshResponse, error) {
	ctx := context.Background()

	//1.解析refresh_token（带有RegisteredClaims的标准JWT）
	secret := ""
	_ = secret //通过 jwtpkg 解析，读取 viper 配置
	claims, err := parseRefreshToken(req.RefreshToken)
	if err != nil {
		return nil, newServiceError(errcode.ErrTokenInvalid, "认证令牌无效或已过期")
	}

	//2.检查refresh_token JTI黑名单
	blacklistKey := fmt.Sprintf("blacklist:%s", claims.ID)
	exists, err := s.rdb.Exists(ctx, blacklistKey).Result()
	if err != nil {
		return nil, newServiceError(errcode.ErrRedisConn, "Redis 连接错误")
	}
	if exists > 0 {
		return nil, newServiceError(errcode.ErrTokenRevoked, "令牌已被吊销")
	}

	//3.尝试从缓存中获取session，否则重新查询user
	userID, parseErr := uuid.Parse(claims.Subject)
	if parseErr != nil {
		return nil, newServiceError(errcode.ErrTokenInvalid, "认证令牌无效或已过期")
	}

	sessionKey := fmt.Sprintf("session:%s", claims.Subject)
	sessionJSON, err := s.rdb.Get(ctx, sessionKey).Result()

	var accessToken string

	if err == nil && sessionJSON != "" {
		//从缓存的会话重建声明
		var sessionData map[string]interface{}
		if jsonErr := json.Unmarshal([]byte(sessionJSON), &sessionData); jsonErr == nil {
			jwtClaims := rebuildClaimsFromSession(sessionData)
			accessToken, err = jwtpkg.GenerateAccessToken(jwtClaims)
			if err != nil {
				return nil, newServiceError(errcode.ErrInternalServer, "服务器内部错误")
			}
		}
	}

	//后备：重新查询用户和分配
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
		accessToken, err = jwtpkg.GenerateAccessToken(jwtClaims)
		if err != nil {
			return nil, newServiceError(errcode.ErrInternalServer, "服务器内部错误")
		}
	}

	return &dto.RefreshResponse{AccessToken: accessToken}, nil
}

// ---------------------------------------------------------------------------
//切换角色
// ---------------------------------------------------------------------------

//SwitchRole 验证目标角色、生成新令牌、将旧令牌列入黑名单并更新会话。
func (s *AuthService) SwitchRole(userID uuid.UUID, roleID string, oldJTI string) (*dto.SwitchRoleResponse, error) {
	ctx := context.Background()

	//1.通过roleID查找角色分配，验证其属于当前用户
	roleUUID, parseErr := uuid.Parse(roleID)
	if parseErr != nil {
		return nil, newServiceError(errcode.ErrRoleSwitchFailed, "角色切换失败")
	}

	assignment, err := s.userRepo.FindRoleAssignmentByID(roleUUID)
	if err != nil {
		return nil, newServiceError(errcode.ErrRoleSwitchFailed, "角色切换失败")
	}
	if assignment.UserID != userID {
		return nil, newServiceError(errcode.ErrRoleSwitchFailed, "角色切换失败")
	}

	//2. 根据分配构建新的 ActiveRoleClaim
	var tenant *model.Tenant
	if assignment.TenantID != nil {
		tenant, _ = s.userRepo.FindTenantByID(*assignment.TenantID)
	}
	activeRoleClaim := buildActiveRoleClaim(assignment, tenant)

	//3. 获取用户信息以生成token
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, newServiceError(errcode.ErrInternalServer, "服务器内部错误")
	}

	//获取所有角色ID
	assignments, err := s.userRepo.FindRoleAssignments(userID)
	if err != nil {
		return nil, newServiceError(errcode.ErrInternalServer, "服务器内部错误")
	}
	allRoleIDs := make([]string, len(assignments))
	for i, a := range assignments {
		allRoleIDs[i] = a.ID.String()
	}

	permissions := []string{}

	//4. 使用更新的 activeRole 生成新的 access_token
	claims := &jwtpkg.JWTClaims{
		Sub:         user.ID.String(),
		Username:    user.Username,
		DisplayName: user.DisplayName,
		ActiveRole:  activeRoleClaim,
		Permissions: permissions,
		AllRoleIDs:  allRoleIDs,
	}
	accessToken, err := jwtpkg.GenerateAccessToken(claims)
	if err != nil {
		return nil, newServiceError(errcode.ErrInternalServer, "服务器内部错误")
	}

	//5. 将旧JTI添加到黑名单
	if oldJTI != "" {
		blacklistKey := fmt.Sprintf("blacklist:%s", oldJTI)
		s.rdb.Set(ctx, blacklistKey, "1", 2*time.Hour)
	}

	//6. 更新会话缓存
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
	s.rdb.Set(ctx, sessionKey, string(sessionJSON), 2*time.Hour)

	//7. Get menus for the new active role
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

	//8.返回SwitchRoleResponse
	var tid *string
	var tname *string
	if assignment.TenantID != nil {
		s := assignment.TenantID.String()
		tid = &s
		if tenant != nil {
			tname = &tenant.Name
		}
	}

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
//获取菜单
// ---------------------------------------------------------------------------

//GetMenu 根据用户的活动角色返回菜单项。
//system_admin 和tenant_admin 获得固定菜单；业务用户获得合并 OrgRole page_permissions。
func (s *AuthService) GetMenu(activeRole jwtpkg.ActiveRoleClaim, userID string, tenantID string) (*dto.MenuResponse, error) {
	switch activeRole.Role {
	case "system_admin":
		// system_admin 没有 org_member 记录，保持硬编码
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

//getMenuFromOrgRoles 查询用户的 OrgMember + OrgRoles 并合并 page_permissions。
//activeSystemRole 用于过滤：tenant_admin 只返回后台管理路径，business 只返回前台业务路径。
func (s *AuthService) getMenuFromOrgRoles(userID string, tenantID string, activeSystemRole string) (*dto.MenuResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return &dto.MenuResponse{Menus: []dto.MenuItem{}}, nil
	}
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return &dto.MenuResponse{Menus: []dto.MenuItem{}}, nil
	}

	//查询 org_members WHERE user_id = ? AND tenant_id = ?, 预加载角色
	var members []model.OrgMember
	if err := s.db.Where("user_id = ? AND tenant_id = ?", uid, tid).
		Preload("Roles").
		Find(&members).Error; err != nil {
		return &dto.MenuResponse{Menus: []dto.MenuItem{}}, nil
	}

	//合并和删除所有角色的 page_permissions
	// page_permissions is stored as a JSON string array like ["/overview", "/dashboard"]
	seen := make(map[string]bool)
	var menus []dto.MenuItem

	// Path → label mapping for all menu items (business + tenant admin)
	pathLabels := map[string]struct{ key, label string }{
		"/overview":                 {key: "overview", label: "概览"},
		"/dashboard":                {key: "dashboard", label: "审核工作台"},
		"/cron":                     {key: "cron", label: "定时任务"},
		"/archive":                  {key: "archive", label: "归档复盘"},
		"/settings":                 {key: "settings", label: "个人设置"},
		"/admin/tenant/rules":       {key: "rules-management", label: "规则管理"},
		"/admin/tenant/org":         {key: "org-management", label: "组织管理"},
		"/admin/tenant/data":        {key: "data-management", label: "数据管理"},
		"/admin/tenant/user-configs": {key: "user-configs", label: "用户配置"},
	}

	for _, member := range members {
		for _, role := range member.Roles {
			// Try to unmarshal as string array first (correct format)
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
			// Fallback: try to unmarshal as []dto.MenuItem (legacy format)
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
//辅助函数
// ---------------------------------------------------------------------------

//filterAssignmentsByTenant 返回与给定租户 ID 匹配的分配。
//includeSystemAdmin 控制是否保留 system_admin 分配（TenantID 为空）。
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

//selectActiveRole 按优先级选择最佳角色：
//首选角色匹配 > 系统管理员 > 租户管理员 > 业务
func selectActiveRole(assignments []model.UserRoleAssignment, preferredRole string) *model.UserRoleAssignment {
	//首先尝试 Preferred_role 匹配
	if preferredRole != "" {
		for i := range assignments {
			if assignments[i].Role == preferredRole {
				return &assignments[i]
			}
		}
	}

	//优先顺序回退
	priorities := []string{"system_admin", "tenant_admin", "business"}
	for _, role := range priorities {
		for i := range assignments {
			if assignments[i].Role == role {
				return &assignments[i]
			}
		}
	}

	//回退到第一个任务
	if len(assignments) > 0 {
		return &assignments[0]
	}
	return nil
}

//buildActiveRoleClaim 根据角色分配和可选租户构造 ActiveRoleClaim。
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

//parseRefreshToken 解析刷新令牌（仅带有 RegisteredClaims 的标准 JWT）。
func parseRefreshToken(tokenString string) (*jwtpkg.JWTClaims, error) {
	return jwtpkg.ParseRefreshToken(tokenString)
}

//rebuildClaimsFromSession 从缓存的会话数据重建 JWTClaims。
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

	//从地图重建 ActiveRole
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

	//重建所有RoleID
	if ids, ok := data["all_role_ids"].([]interface{}); ok {
		for _, id := range ids {
			if s, ok := id.(string); ok {
				claims.AllRoleIDs = append(claims.AllRoleIDs, s)
			}
		}
	}

	//重建权限
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

// ChangePassword verifies the current password and updates to the new one.
func (s *AuthService) ChangePassword(userID uuid.UUID, req *dto.ChangePasswordRequest) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return newServiceError(errcode.ErrWrongPassword, "用户不存在")
	}

	if !hash.CheckPassword(req.CurrentPassword, user.PasswordHash) {
		return newServiceError(errcode.ErrWrongPassword, "当前密码错误")
	}

	// New password must differ from current password
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

	return nil
}

// ---------------------------------------------------------------------------
// GetMe
// ---------------------------------------------------------------------------

// GetMe returns the full user profile including org-level info for the current tenant.
func (s *AuthService) GetMe(userID uuid.UUID, activeRole jwtpkg.ActiveRoleClaim, allRoleIDs []string) (*dto.MeResponse, error) {
	// 1. Fetch user basic info
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, newServiceError(errcode.ErrTokenInvalid, "用户不存在")
	}

	// 2. Fetch all role assignments for the roles list
	assignments, err := s.userRepo.FindRoleAssignments(userID)
	if err != nil {
		assignments = []model.UserRoleAssignment{}
	}

	// Batch-fetch tenant names
	tenantNameCache := make(map[string]string)
	for _, a := range assignments {
		if a.TenantID != nil {
			tidStr := a.TenantID.String()
			if _, exists := tenantNameCache[tidStr]; !exists {
				if t, tErr := s.userRepo.FindTenantByID(*a.TenantID); tErr == nil {
					tenantNameCache[tidStr] = t.Name
				}
			}
		}
	}

	roles := make([]dto.RoleInfo, len(assignments))
	for i, a := range assignments {
		var tid *string
		var tname *string
		if a.TenantID != nil {
			s := a.TenantID.String()
			tid = &s
			if name, ok := tenantNameCache[s]; ok {
				tname = &name
			}
		}
		roles[i] = dto.RoleInfo{
			ID:         a.ID.String(),
			Role:       a.Role,
			TenantID:   tid,
			TenantName: tname,
			Label:      a.Label,
		}
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

	// 3. If active role has a tenant, fetch org-level info
	if activeRole.TenantID != nil && *activeRole.TenantID != "" {
		tid, parseErr := uuid.Parse(*activeRole.TenantID)
		if parseErr == nil {
			if tname, ok := tenantNameCache[tid.String()]; ok {
				resp.TenantName = tname
			} else if t, tErr := s.userRepo.FindTenantByID(tid); tErr == nil {
				resp.TenantName = t.Name
			}

			// Query org_members with department and roles preloaded
			var members []model.OrgMember
			if err := s.db.Where("user_id = ? AND tenant_id = ?", userID, tid).
				Preload("Department").
				Preload("Roles").
				Find(&members).Error; err == nil && len(members) > 0 {

				member := members[0]
				resp.DepartmentName = member.Department.Name
				resp.Position = member.Position

				// Collect all org roles and merge page_permissions
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

	// Ensure non-nil slices
	if resp.OrgRoles == nil {
		resp.OrgRoles = []dto.MeOrgRole{}
	}
	if resp.PagePermissions == nil {
		resp.PagePermissions = []string{}
	}

	// 4. Fetch recent login history (last 10 entries)
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

// UpdateLocale updates the user's locale preference.
func (s *AuthService) UpdateLocale(userID uuid.UUID, locale string) error {
	// Validate locale value
	switch locale {
	case "zh-CN", "en-US":
		// ok
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

// UpdateProfile updates the user's display_name, email, and phone with validation.
func (s *AuthService) UpdateProfile(userID uuid.UUID, req *dto.UpdateProfileRequest) error {
	// Validate email format (if provided)
	if req.Email != "" {
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(req.Email) {
			return newServiceError(errcode.ErrParamValidation, "邮箱格式不正确")
		}
	}
	// Validate phone format: must be 11 digits (if provided)
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
	// Allow clearing email/phone by sending empty string — only update if key is present
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
