package service

import (
	"context"
	"encoding/json"
	"fmt"
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

// AuthService handles authentication, token management, role switching, and menu retrieval.
type AuthService struct {
	userRepo *repository.UserRepo
	rdb      *redis.Client
	db       *gorm.DB
}

// NewAuthService creates a new AuthService instance.
func NewAuthService(userRepo *repository.UserRepo, rdb *redis.Client, db *gorm.DB) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		rdb:      rdb,
		db:       db,
	}
}

// ServiceError carries a business error code and message for the handler layer.
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
// Login
// ---------------------------------------------------------------------------

// Login authenticates a user and returns tokens, user info, roles, and active role.
func (s *AuthService) Login(req *dto.LoginRequest) (*dto.LoginResponse, error) {
	// 1. Find user by username
	user, err := s.userRepo.FindByUsername(req.Username)
	if err != nil {
		return nil, newServiceError(errcode.ErrWrongPassword, "用户名或密码错误")
	}

	// 2. Check disabled status
	if user.Status == "disabled" {
		return nil, newServiceError(errcode.ErrAccountDisabled, "账户已被禁用")
	}

	// 3. Check locked: login_fail_count >= 5 AND locked_until > now
	if user.LoginFailCount >= 5 && user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return nil, newServiceError(errcode.ErrAccountLocked, "账户被锁定")
	}

	// 4. Verify password
	if !hash.CheckPassword(req.Password, user.PasswordHash) {
		_ = s.userRepo.UpdateLoginFail(user)
		return nil, newServiceError(errcode.ErrWrongPassword, "用户名或密码错误")
	}

	// 5. If tenant_id provided and preferred_role != system_admin, validate tenant
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

	// 6. Find role assignments for user
	assignments, err := s.userRepo.FindRoleAssignments(user.ID)
	if err != nil || len(assignments) == 0 {
		return nil, newServiceError(errcode.ErrNoRoleInTenant, "用户在该租户无角色分配")
	}

	// 7. Filter assignments by tenant_id if provided
	filtered := assignments
	if req.TenantID != "" && req.PreferredRole != "system_admin" {
		tenantUUID, _ := uuid.Parse(req.TenantID)
		filtered = filterAssignmentsByTenant(assignments, &tenantUUID)
		if len(filtered) == 0 {
			return nil, newServiceError(errcode.ErrNoRoleInTenant, "用户在该租户无角色分配")
		}
	}

	// 8. Select activeRole by priority
	activeAssignment := selectActiveRole(filtered, req.PreferredRole)

	// 9. Reset login fail count
	if err := s.userRepo.ResetLoginFail(user.ID); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	// Build active role claim
	activeRoleClaim := buildActiveRoleClaim(activeAssignment, tenant)

	// Collect all role IDs
	allRoleIDs := make([]string, len(assignments))
	for i, a := range assignments {
		allRoleIDs[i] = a.ID.String()
	}

	// Build permissions (for business users, will be populated by GetMenu; for admin roles, empty)
	permissions := []string{}

	// 10. Generate access_token
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

	// 11. Generate refresh_token
	refreshToken, refreshJTI, err := jwtpkg.GenerateRefreshToken(user.ID.String(), "")
	if err != nil {
		return nil, newServiceError(errcode.ErrInternalServer, "服务器内部错误")
	}

	// 12. Create login history record
	var loginTenantID *uuid.UUID
	if req.TenantID != "" {
		tid, _ := uuid.Parse(req.TenantID)
		loginTenantID = &tid
	}
	history := &model.LoginHistory{
		UserID:   user.ID,
		TenantID: loginTenantID,
		LoginAt:  time.Now(),
	}
	_ = s.userRepo.CreateLoginHistory(history)

	// 13. Cache session in Redis: key "session:{user_id}", TTL 2h
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

	// 14. Build response
	roles := make([]dto.RoleInfo, len(assignments))
	for i, a := range assignments {
		var tid *string
		if a.TenantID != nil {
			s := a.TenantID.String()
			tid = &s
		}
		roles[i] = dto.RoleInfo{
			ID:       a.ID.String(),
			Role:     a.Role,
			TenantID: tid,
			Label:    a.Label,
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
			AvatarURL:   user.AvatarURL,
			Locale:      user.Locale,
		},
		Roles: roles,
		ActiveRole: dto.RoleInfo{
			ID:       activeAssignment.ID.String(),
			Role:     activeAssignment.Role,
			TenantID: activeRoleClaim.TenantID,
			Label:    activeAssignment.Label,
		},
		Permissions: permissions,
	}

	return resp, nil
}

// ---------------------------------------------------------------------------
// Logout
// ---------------------------------------------------------------------------

// LogoutRequest holds the JTIs and user ID needed for logout.
type LogoutRequest struct {
	AccessJTI  string
	RefreshJTI string
	UserID     string
}

// Logout invalidates both tokens and removes the session cache.
func (s *AuthService) Logout(req *LogoutRequest) error {
	ctx := context.Background()

	// 1. Add access_token JTI to blacklist (TTL = 2h default)
	if req.AccessJTI != "" {
		blacklistKey := fmt.Sprintf("blacklist:%s", req.AccessJTI)
		s.rdb.Set(ctx, blacklistKey, "1", 2*time.Hour)
	}

	// 2. Add refresh_token JTI to blacklist (TTL = 7d)
	if req.RefreshJTI != "" {
		blacklistKey := fmt.Sprintf("blacklist:%s", req.RefreshJTI)
		s.rdb.Set(ctx, blacklistKey, "1", 7*24*time.Hour)
	}

	// 3. Delete session cache
	if req.UserID != "" {
		sessionKey := fmt.Sprintf("session:%s", req.UserID)
		s.rdb.Del(ctx, sessionKey)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Refresh
// ---------------------------------------------------------------------------

// Refresh validates a refresh token and returns a new access token.
func (s *AuthService) Refresh(req *dto.RefreshRequest) (*dto.RefreshResponse, error) {
	ctx := context.Background()

	// 1. Parse refresh_token (standard JWT with RegisteredClaims)
	secret := ""
	_ = secret // parsed via jwtpkg which reads viper config
	claims, err := parseRefreshToken(req.RefreshToken)
	if err != nil {
		return nil, newServiceError(errcode.ErrTokenInvalid, "认证令牌无效或已过期")
	}

	// 2. Check blacklist for refresh_token JTI
	blacklistKey := fmt.Sprintf("blacklist:%s", claims.ID)
	exists, err := s.rdb.Exists(ctx, blacklistKey).Result()
	if err != nil {
		return nil, newServiceError(errcode.ErrRedisConn, "Redis 连接错误")
	}
	if exists > 0 {
		return nil, newServiceError(errcode.ErrTokenRevoked, "令牌已被吊销")
	}

	// 3. Try to get session from cache, otherwise re-query user
	userID, parseErr := uuid.Parse(claims.Subject)
	if parseErr != nil {
		return nil, newServiceError(errcode.ErrTokenInvalid, "认证令牌无效或已过期")
	}

	sessionKey := fmt.Sprintf("session:%s", claims.Subject)
	sessionJSON, err := s.rdb.Get(ctx, sessionKey).Result()

	var accessToken string

	if err == nil && sessionJSON != "" {
		// Rebuild claims from cached session
		var sessionData map[string]interface{}
		if jsonErr := json.Unmarshal([]byte(sessionJSON), &sessionData); jsonErr == nil {
			jwtClaims := rebuildClaimsFromSession(sessionData)
			accessToken, err = jwtpkg.GenerateAccessToken(jwtClaims)
			if err != nil {
				return nil, newServiceError(errcode.ErrInternalServer, "服务器内部错误")
			}
		}
	}

	// Fallback: re-query user and assignments
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
// SwitchRole
// ---------------------------------------------------------------------------

// SwitchRole validates the target role, generates a new token, blacklists the old one, and updates the session.
func (s *AuthService) SwitchRole(userID uuid.UUID, roleID string, oldJTI string) (*dto.SwitchRoleResponse, error) {
	ctx := context.Background()

	// 1. Find role assignment by roleID, verify it belongs to current user
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

	// 2. Build new ActiveRoleClaim from the assignment
	var tenant *model.Tenant
	if assignment.TenantID != nil {
		tenant, _ = s.userRepo.FindTenantByID(*assignment.TenantID)
	}
	activeRoleClaim := buildActiveRoleClaim(assignment, tenant)

	// 3. Get user info for token generation
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, newServiceError(errcode.ErrInternalServer, "服务器内部错误")
	}

	// Get all role IDs
	assignments, err := s.userRepo.FindRoleAssignments(userID)
	if err != nil {
		return nil, newServiceError(errcode.ErrInternalServer, "服务器内部错误")
	}
	allRoleIDs := make([]string, len(assignments))
	for i, a := range assignments {
		allRoleIDs[i] = a.ID.String()
	}

	permissions := []string{}

	// 4. Generate new access_token with updated activeRole
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

	// 5. Add old JTI to blacklist
	if oldJTI != "" {
		blacklistKey := fmt.Sprintf("blacklist:%s", oldJTI)
		s.rdb.Set(ctx, blacklistKey, "1", 2*time.Hour)
	}

	// 6. Update session cache
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

	// 7. Return SwitchRoleResponse
	var tid *string
	if assignment.TenantID != nil {
		s := assignment.TenantID.String()
		tid = &s
	}

	return &dto.SwitchRoleResponse{
		AccessToken: accessToken,
		ActiveRole: dto.RoleInfo{
			ID:       assignment.ID.String(),
			Role:     assignment.Role,
			TenantID: tid,
			Label:    assignment.Label,
		},
		Permissions: permissions,
	}, nil
}

// ---------------------------------------------------------------------------
// GetMenu
// ---------------------------------------------------------------------------

// GetMenu returns menu items based on the user's active role.
// system_admin and tenant_admin get fixed menus; business users get merged OrgRole page_permissions.
func (s *AuthService) GetMenu(activeRole jwtpkg.ActiveRoleClaim, userID string, tenantID string) (*dto.MenuResponse, error) {
	switch activeRole.Role {
	case "system_admin":
		return &dto.MenuResponse{
			Menus: []dto.MenuItem{
				{Key: "tenant-management", Label: "租户管理", Path: "/admin/tenants"},
				{Key: "system-settings", Label: "系统设置", Path: "/admin/settings"},
				{Key: "system-monitor", Label: "系统监控", Path: "/admin/monitor"},
			},
		}, nil

	case "tenant_admin":
		return &dto.MenuResponse{
			Menus: []dto.MenuItem{
				{Key: "org-management", Label: "组织管理", Path: "/admin/tenant/org"},
				{Key: "rules-management", Label: "规则管理", Path: "/admin/tenant/rules"},
				{Key: "data-management", Label: "数据管理", Path: "/admin/tenant/data"},
				{Key: "tenant-settings", Label: "租户设置", Path: "/admin/tenant/settings"},
			},
		}, nil

	case "business":
		return s.getBusinessMenu(userID, tenantID)

	default:
		return &dto.MenuResponse{Menus: []dto.MenuItem{}}, nil
	}
}

// getBusinessMenu queries OrgMember + OrgRoles for a business user and merges page_permissions.
func (s *AuthService) getBusinessMenu(userID string, tenantID string) (*dto.MenuResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return &dto.MenuResponse{Menus: []dto.MenuItem{}}, nil
	}
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return &dto.MenuResponse{Menus: []dto.MenuItem{}}, nil
	}

	// Query org_members WHERE user_id = ? AND tenant_id = ?, preload Roles
	var members []model.OrgMember
	if err := s.db.Where("user_id = ? AND tenant_id = ?", uid, tid).
		Preload("Roles").
		Find(&members).Error; err != nil {
		return &dto.MenuResponse{Menus: []dto.MenuItem{}}, nil
	}

	// Merge and deduplicate page_permissions from all roles
	seen := make(map[string]bool)
	var menus []dto.MenuItem

	for _, member := range members {
		for _, role := range member.Roles {
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

	return &dto.MenuResponse{Menus: menus}, nil
}

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

// filterAssignmentsByTenant returns assignments that match the given tenant ID,
// plus any system_admin assignments (which have nil TenantID).
func filterAssignmentsByTenant(assignments []model.UserRoleAssignment, tenantID *uuid.UUID) []model.UserRoleAssignment {
	var result []model.UserRoleAssignment
	for _, a := range assignments {
		if a.Role == "system_admin" {
			result = append(result, a)
			continue
		}
		if a.TenantID != nil && tenantID != nil && *a.TenantID == *tenantID {
			result = append(result, a)
		}
	}
	return result
}

// selectActiveRole picks the best role by priority:
// preferred_role match > system_admin > tenant_admin > business
func selectActiveRole(assignments []model.UserRoleAssignment, preferredRole string) *model.UserRoleAssignment {
	// Try preferred_role match first
	if preferredRole != "" {
		for i := range assignments {
			if assignments[i].Role == preferredRole {
				return &assignments[i]
			}
		}
	}

	// Priority order fallback
	priorities := []string{"system_admin", "tenant_admin", "business"}
	for _, role := range priorities {
		for i := range assignments {
			if assignments[i].Role == role {
				return &assignments[i]
			}
		}
	}

	// Fallback to first assignment
	if len(assignments) > 0 {
		return &assignments[0]
	}
	return nil
}

// buildActiveRoleClaim constructs an ActiveRoleClaim from a role assignment and optional tenant.
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

// parseRefreshToken parses a refresh token (standard JWT with RegisteredClaims only).
func parseRefreshToken(tokenString string) (*jwtpkg.JWTClaims, error) {
	// Refresh tokens use the same signing key; try parsing as JWTClaims first,
	// then fall back to RegisteredClaims if needed.
	claims, err := jwtpkg.ParseToken(tokenString)
	if err != nil {
		return nil, err
	}
	return claims, nil
}

// rebuildClaimsFromSession reconstructs JWTClaims from cached session data.
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

	// Rebuild ActiveRole from map
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

	// Rebuild AllRoleIDs
	if ids, ok := data["all_role_ids"].([]interface{}); ok {
		for _, id := range ids {
			if s, ok := id.(string); ok {
				claims.AllRoleIDs = append(claims.AllRoleIDs, s)
			}
		}
	}

	// Rebuild Permissions
	if perms, ok := data["permissions"].([]interface{}); ok {
		for _, p := range perms {
			if s, ok := p.(string); ok {
				claims.Permissions = append(claims.Permissions, s)
			}
		}
	}

	return claims
}
