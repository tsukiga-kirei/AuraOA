package service

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/pkg/hash"
	"oa-smart-audit/go-service/internal/repository"
)

// TenantService 处理 system_admin 的租户 CRUD 和统计信息。
type TenantService struct {
	tenantRepo       *repository.TenantRepo
	systemConfigRepo *repository.SystemConfigRepo
	userRepo         *repository.UserRepo
	db               *gorm.DB
}

// NewTenantService 创建一个新的 TenantService 实例。
func NewTenantService(tenantRepo *repository.TenantRepo, systemConfigRepo *repository.SystemConfigRepo, userRepo *repository.UserRepo, db *gorm.DB) *TenantService {
	return &TenantService{
		tenantRepo:       tenantRepo,
		systemConfigRepo: systemConfigRepo,
		userRepo:         userRepo,
		db:               db,
	}
}

// ListTenants 返回所有租户。
func (s *TenantService) ListTenants() ([]dto.TenantResponse, error) {
	tenants, err := s.tenantRepo.List()
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	result := make([]dto.TenantResponse, len(tenants))
	for i := range tenants {
		result[i] = toTenantResponse(&tenants[i])
	}
	return result, nil
}

// ListPublicTenants 返回活跃租户的轻量级列表（公共接口，无需鉴权）。
func (s *TenantService) ListPublicTenants() ([]dto.PublicTenantItem, error) {
	tenants, err := s.tenantRepo.ListActive()
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	result := make([]dto.PublicTenantItem, len(tenants))
	for i := range tenants {
		result[i] = dto.PublicTenantItem{
			ID:   tenants[i].ID.String(),
			Name: tenants[i].Name,
			Code: tenants[i].Code,
		}
	}
	return result, nil
}

// generateTenantCode 自动生成租户编码，格式：T-YYYYMMDD-XXXX。
func (s *TenantService) generateTenantCode() string {
	dateStr := time.Now().Format("20060102")
	prefix := fmt.Sprintf("T-%s-", dateStr)

	// 查询当天已有的最大编号
	var count int64
	s.db.Model(&model.Tenant{}).Where("code LIKE ?", prefix+"%").Count(&count)

	return fmt.Sprintf("%s%04d", prefix, count+1)
}

// getSystemConfigInt 从系统配置表获取整数配置值，不存在则返回默认值。
func (s *TenantService) getSystemConfigInt(key string, defaultVal int) int {
	configs, err := s.systemConfigRepo.ListAll()
	if err != nil {
		return defaultVal
	}
	for _, c := range configs {
		if c.Key == key {
			v, err := strconv.Atoi(c.Value)
			if err != nil {
				return defaultVal
			}
			return v
		}
	}
	return defaultVal
}

// CreateTenant 创建新租户，在事务中完成：校验参数 → 创建租户 → 创建默认角色 → 创建默认部门 → 创建管理员账号。
// 任意步骤失败时整个事务回滚。
func (s *TenantService) CreateTenant(req *dto.CreateTenantRequest) (*dto.TenantResponse, error) {
	// 0. 管理员参数校验
	usernameRegex := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
	if !usernameRegex.MatchString(req.AdminUsername) {
		return nil, newServiceError(errcode.ErrParamValidation, "管理员用户名只能包含英文字母、数字和下划线，且以字母开头")
	}
	if req.AdminEmail != "" {
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(req.AdminEmail) {
			return nil, newServiceError(errcode.ErrParamValidation, "管理员邮箱格式不正确")
		}
	}
	if req.AdminPhone != "" {
		phoneRegex := regexp.MustCompile(`^\d{11}$`)
		if !phoneRegex.MatchString(req.AdminPhone) {
			return nil, newServiceError(errcode.ErrParamValidation, "管理员手机号必须为11位数字")
		}
	}

	// 租户编码校验（如果手动填写）
	if req.Code != "" {
		codeRegex := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
		if !codeRegex.MatchString(req.Code) {
			return nil, newServiceError(errcode.ErrParamValidation, "租户编码只能包含英文字母、数字和下划线")
		}
	}

	// 检查管理员用户名是否已存在
	existingUser, _ := s.userRepo.FindByUsername(req.AdminUsername)
	if existingUser != nil {
		return nil, newServiceError(errcode.ErrResourceConflict, "管理员用户名已存在")
	}

	// 自动生成编码（如果未提供）
	code := req.Code
	if code == "" {
		code = s.generateTenantCode()
	}

	// 检查代码唯一性
	existing, _ := s.tenantRepo.FindByCode(code)
	if existing != nil {
		return nil, newServiceError(errcode.ErrResourceConflict, "租户编码已存在")
	}

	// 从系统配置获取默认值
	defaultTokenQuota := s.getSystemConfigInt("tenant.default_token_quota", 10000)
	defaultMaxConcurrency := s.getSystemConfigInt("tenant.default_max_concurrency", 10)
	defaultLogRetention := s.getSystemConfigInt("tenant.default_log_retention_days", 365)
	defaultDataRetention := s.getSystemConfigInt("tenant.default_data_retention_days", 1095)

	// 联系人信息自动填充：使用管理员信息作为联系人
	contactName := req.ContactName
	if contactName == "" {
		contactName = req.AdminDisplayName
	}
	contactEmail := req.ContactEmail
	if contactEmail == "" {
		contactEmail = req.AdminEmail
	}
	contactPhone := req.ContactPhone
	if contactPhone == "" {
		contactPhone = req.AdminPhone
	}

	tenant := &model.Tenant{
		ID:                  uuid.New(),
		Name:                req.Name,
		Code:                code,
		Description:         req.Description,
		TokenQuota:          req.TokenQuota,
		MaxConcurrency:      req.MaxConcurrency,
		MaxTokensPerRequest: req.MaxTokensPerRequest,
		TimeoutSeconds:      req.TimeoutSeconds,
		RetryCount:          req.RetryCount,
		ContactName:         contactName,
		ContactEmail:        contactEmail,
		ContactPhone:        contactPhone,
		LogRetentionDays:    req.LogRetentionDays,
		DataRetentionDays:   req.DataRetentionDays,
	}

	// 设置 OA 连接 ID
	if req.OADBConnectionID != "" {
		connID, err := uuid.Parse(req.OADBConnectionID)
		if err == nil {
			tenant.OADBConnectionID = &connID
		}
	}

	// 设置主模型 ID
	if req.PrimaryModelID != "" {
		modelID, err := uuid.Parse(req.PrimaryModelID)
		if err == nil {
			tenant.PrimaryModelID = &modelID
		}
	}

	// 设置备用模型 ID
	if req.FallbackModelID != "" {
		modelID, err := uuid.Parse(req.FallbackModelID)
		if err == nil {
			tenant.FallbackModelID = &modelID
		}
	}

	// 设置温度
	if req.Temperature > 0 {
		tenant.Temperature = req.Temperature
	} else {
		tenant.Temperature = 0.3
	}

	// 应用系统默认值（如果未提供）
	if tenant.TokenQuota == 0 {
		tenant.TokenQuota = defaultTokenQuota
	}
	if tenant.MaxConcurrency == 0 {
		tenant.MaxConcurrency = defaultMaxConcurrency
	}
	if tenant.LogRetentionDays == 0 {
		tenant.LogRetentionDays = defaultLogRetention
	}
	if tenant.DataRetentionDays == 0 {
		tenant.DataRetentionDays = defaultDataRetention
	}
	if tenant.MaxTokensPerRequest == 0 {
		tenant.MaxTokensPerRequest = 8192
	}
	if tenant.TimeoutSeconds == 0 {
		tenant.TimeoutSeconds = 60
	}
	if tenant.RetryCount == 0 {
		tenant.RetryCount = 3
	}

	// 启动事务：自动创建租户+默认角色+默认部门+管理员账号
	tx := s.db.Begin()
	defer tx.Rollback() // 提交后无操作

	// 1. 创建租户
	if err := tx.Create(tenant).Error; err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "创建租户失败")
	}

	// 2. 创建默认角色
	businessPerms := []byte(`["/overview","/dashboard","/settings"]`)
	auditPerms := []byte(`["/overview","/dashboard","/cron","/archive","/settings"]`)
	adminPerms := []byte(`["/overview","/dashboard","/cron","/archive","/settings","/admin/tenant/rules","/admin/tenant/org","/admin/tenant/data","/admin/tenant/user-configs"]`)

	defaultRoles := []model.OrgRole{
		{
			ID:              uuid.New(),
			TenantID:        tenant.ID,
			Name:            "业务用户",
			Description:     "普通业务人员，可使用审核工作台等前台功能。仪表盘为所有角色默认拥有。",
			PagePermissions: businessPerms,
			IsSystem:        true,
		},
		{
			ID:              uuid.New(),
			TenantID:        tenant.ID,
			Name:            "审计管理员",
			Description:     "在业务用户基础上，额外拥有归档复盘权限，可进行合规复核。",
			PagePermissions: auditPerms,
			IsSystem:        true,
		},
		{
			ID:              uuid.New(),
			TenantID:        tenant.ID,
			Name:            "租户管理员",
			Description:     "可进入后台管理，配置规则、组织人员、数据信息、用户偏好。",
			PagePermissions: adminPerms,
			IsSystem:        true,
		},
	}

	if err := tx.Create(&defaultRoles).Error; err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "创建默认角色失败")
	}

	// 3. 创建默认部门
	defaultDept := &model.Department{
		ID:        uuid.New(),
		TenantID:  tenant.ID,
		Name:      req.AdminDeptName,
		SortOrder: 0,
	}
	if err := tx.Create(defaultDept).Error; err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "创建默认部门失败")
	}

	// 4. 创建管理员用户
	password := req.AdminPassword
	if password == "" {
		password = "123456"
	}
	passwordHash, err := hash.HashPassword(password)
	if err != nil {
		return nil, newServiceError(errcode.ErrInternalServer, "服务器内部错误")
	}
	adminUser := &model.User{
		ID:                uuid.New(),
		Username:          req.AdminUsername,
		PasswordHash:      passwordHash,
		DisplayName:       req.AdminDisplayName,
		Email:             req.AdminEmail,
		Phone:             req.AdminPhone,
		Status:            "active",
		PasswordChangedAt: time.Now(),
	}
	if err := tx.Create(adminUser).Error; err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "创建管理员用户失败")
	}

	// 4.1 回写 admin_user_id 到租户记录
	if err := tx.Model(tenant).Update("admin_user_id", adminUser.ID).Error; err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "更新租户管理员关联失败")
	}
	tenant.AdminUserID = &adminUser.ID

	// 5. 创建组织成员记录（关联管理员到默认部门）
	adminMember := &model.OrgMember{
		ID:           uuid.New(),
		TenantID:     tenant.ID,
		UserID:       adminUser.ID,
		DepartmentID: defaultDept.ID,
		Position:     "租户管理员",
		Status:       "active",
	}
	if err := tx.Create(adminMember).Error; err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "创建管理员成员记录失败")
	}

	// 6. 关联管理员到"租户管理员"角色（org_member_roles）
	adminRole := defaultRoles[2] // 租户管理员
	if err := tx.Model(adminMember).Association("Roles").Append(&adminRole); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "分配管理员角色失败")
	}

	// 7. 创建 user_role_assignment（系统级角色）
	tenantAdminAssignment := &model.UserRoleAssignment{
		ID:       uuid.New(),
		UserID:   adminUser.ID,
		Role:     "tenant_admin",
		TenantID: &tenant.ID,
		Label:    "租户管理员 - " + req.AdminDisplayName,
	}
	if err := tx.Create(tenantAdminAssignment).Error; err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "创建角色分配失败")
	}

	if err := tx.Commit().Error; err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	resp := toTenantResponse(tenant)
	return &resp, nil
}

// UpdateTenant 更新现有租户字段，使用 map 构建更新字段以避免零值覆盖已有数据。
// 同步更新关联管理员用户的联系人信息。
func (s *TenantService) UpdateTenant(id uuid.UUID, req *dto.UpdateTenantRequest) (*dto.TenantResponse, error) {
	tenant, err := s.tenantRepo.FindByID(id)
	if err != nil {
		return nil, newServiceError(errcode.ErrResourceNotFound, "租户不存在")
	}

	// 联系人邮箱格式校验
	if req.ContactEmail != "" {
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(req.ContactEmail) {
			return nil, newServiceError(errcode.ErrParamValidation, "联系人邮箱格式不正确")
		}
	}
	// 联系人手机号校验
	if req.ContactPhone != "" {
		phoneRegex := regexp.MustCompile(`^\d{11}$`)
		if !phoneRegex.MatchString(req.ContactPhone) {
			return nil, newServiceError(errcode.ErrParamValidation, "联系人手机号必须为11位数字")
		}
	}

	// 构建更新字段 map，只包含实际传入的字段
	fields := make(map[string]interface{})

	if req.Name != "" {
		fields["name"] = req.Name
	}
	if req.Status != "" {
		fields["status"] = req.Status
	}
	if req.Description != "" {
		fields["description"] = req.Description
	}
	if req.OADBConnectionID != nil {
		if *req.OADBConnectionID == "" {
			fields["oa_db_connection_id"] = nil
		} else {
			connID, err := uuid.Parse(*req.OADBConnectionID)
			if err == nil {
				fields["oa_db_connection_id"] = connID
			}
		}
	}
	if req.TokenQuota != 0 {
		fields["token_quota"] = req.TokenQuota
	}
	if req.MaxConcurrency != 0 {
		fields["max_concurrency"] = req.MaxConcurrency
	}
	if req.PrimaryModelID != nil {
		if *req.PrimaryModelID == "" {
			fields["primary_model_id"] = nil
		} else {
			modelID, err := uuid.Parse(*req.PrimaryModelID)
			if err == nil {
				fields["primary_model_id"] = modelID
			}
		}
	}
	if req.FallbackModelID != nil {
		if *req.FallbackModelID == "" {
			fields["fallback_model_id"] = nil
		} else {
			modelID, err := uuid.Parse(*req.FallbackModelID)
			if err == nil {
				fields["fallback_model_id"] = modelID
			}
		}
	}
	if req.MaxTokensPerRequest != 0 {
		fields["max_tokens_per_request"] = req.MaxTokensPerRequest
	}
	if req.Temperature != nil {
		fields["temperature"] = *req.Temperature
	}
	if req.TimeoutSeconds != 0 {
		fields["timeout_seconds"] = req.TimeoutSeconds
	}
	if req.RetryCount != 0 {
		fields["retry_count"] = req.RetryCount
	}
	if req.SSOEnabled != nil {
		fields["sso_enabled"] = *req.SSOEnabled
	}
	if req.SSOEndpoint != "" {
		fields["sso_endpoint"] = req.SSOEndpoint
	}
	if req.LogRetentionDays != 0 {
		fields["log_retention_days"] = req.LogRetentionDays
	}
	if req.DataRetentionDays != 0 {
		fields["data_retention_days"] = req.DataRetentionDays
	}
	if req.ContactName != "" {
		fields["contact_name"] = req.ContactName
	}
	if req.ContactEmail != "" {
		fields["contact_email"] = req.ContactEmail
	}
	if req.ContactPhone != "" {
		fields["contact_phone"] = req.ContactPhone
	}

	// 如果没有任何字段需要更新，直接返回当前数据
	if len(fields) == 0 {
		resp := toTenantResponse(tenant)
		return &resp, nil
	}

	if err := s.tenantRepo.UpdateFields(id, fields); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	// 同步联系人信息到关联的管理员 user 记录
	if tenant.AdminUserID != nil {
		userUpdates := map[string]interface{}{}
		if req.ContactName != "" {
			userUpdates["display_name"] = req.ContactName
		}
		if req.ContactEmail != "" {
			userUpdates["email"] = req.ContactEmail
		}
		if req.ContactPhone != "" {
			userUpdates["phone"] = req.ContactPhone
		}
		if len(userUpdates) > 0 {
			if err := s.db.Model(&model.User{}).Where("id = ?", *tenant.AdminUserID).Updates(userUpdates).Error; err != nil {
				return nil, newServiceError(errcode.ErrDatabase, "同步管理员信息失败")
			}
		}
	}

	// 重新查询最新数据返回
	tenant, err = s.tenantRepo.FindByID(id)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	resp := toTenantResponse(tenant)
	return &resp, nil
}

// DeleteTenant 彻底删除租户及其所有关联数据，需要操作者密码确认。
// 在事务中按依赖顺序清理：org_member_roles → org_members → org_roles → departments → user_role_assignments → 租户专属用户 → tenant。
func (s *TenantService) DeleteTenant(id uuid.UUID, operatorUserID uuid.UUID, adminPassword string) error {
	// 1. 验证操作者密码
	operator, err := s.userRepo.FindByID(operatorUserID)
	if err != nil {
		return newServiceError(errcode.ErrResourceNotFound, "操作用户不存在")
	}
	if !hash.CheckPassword(adminPassword, operator.PasswordHash) {
		return newServiceError(errcode.ErrWrongPassword, "管理员密码错误")
	}

	// 2. 确认租户存在
	tenant, err := s.tenantRepo.FindByID(id)
	if err != nil {
		return newServiceError(errcode.ErrResourceNotFound, "租户不存在")
	}

	// 3. 在事务中执行级联删除
	return s.db.Transaction(func(tx *gorm.DB) error {
		tenantID := tenant.ID

		// 3a. 删除 org_member_roles（通过 org_members 子查询）
		if err := tx.Exec(`
			DELETE FROM org_member_roles
			WHERE org_member_id IN (
				SELECT id FROM org_members WHERE tenant_id = ?
			)`, tenantID).Error; err != nil {
			return newServiceError(errcode.ErrDatabase, "删除成员角色关联失败")
		}

		// 3b. 删除 org_members
		if err := tx.Exec("DELETE FROM org_members WHERE tenant_id = ?", tenantID).Error; err != nil {
			return newServiceError(errcode.ErrDatabase, "删除组织成员失败")
		}

		// 3c. 删除 org_roles
		if err := tx.Exec("DELETE FROM org_roles WHERE tenant_id = ?", tenantID).Error; err != nil {
			return newServiceError(errcode.ErrDatabase, "删除组织角色失败")
		}

		// 3d. 删除 departments
		if err := tx.Exec("DELETE FROM departments WHERE tenant_id = ?", tenantID).Error; err != nil {
			return newServiceError(errcode.ErrDatabase, "删除部门失败")
		}

		// 3e. 收集该租户下所有用户 ID（通过 user_role_assignments）
		var userIDs []uuid.UUID
		if err := tx.Raw(`
			SELECT DISTINCT user_id FROM user_role_assignments WHERE tenant_id = ?
		`, tenantID).Scan(&userIDs).Error; err != nil {
			return newServiceError(errcode.ErrDatabase, "查询租户用户失败")
		}

		// 3f. 删除 user_role_assignments
		if err := tx.Exec("DELETE FROM user_role_assignments WHERE tenant_id = ?", tenantID).Error; err != nil {
			return newServiceError(errcode.ErrDatabase, "删除用户角色分配失败")
		}

		// 3g. 清理 login_history 中的 tenant_id 引用
		if err := tx.Exec("UPDATE login_history SET tenant_id = NULL WHERE tenant_id = ?", tenantID).Error; err != nil {
			return newServiceError(errcode.ErrDatabase, "清理登录历史失败")
		}

		// 3h. 清除租户的 admin_user_id 引用（避免外键阻塞）
		if err := tx.Exec("UPDATE tenants SET admin_user_id = NULL WHERE id = ?", tenantID).Error; err != nil {
			return newServiceError(errcode.ErrDatabase, "清除租户管理员引用失败")
		}

		// 3i. 删除仅属于该租户的用户（在其他租户无角色分配的用户）
		if len(userIDs) > 0 {
			// 找出在其他租户仍有角色的用户，排除它们
			if err := tx.Exec(`
				DELETE FROM login_history
				WHERE user_id IN (?)
				AND user_id NOT IN (
					SELECT DISTINCT user_id FROM user_role_assignments
				)
			`, userIDs).Error; err != nil {
				return newServiceError(errcode.ErrDatabase, "删除用户登录历史失败")
			}

			if err := tx.Exec(`
				DELETE FROM users
				WHERE id IN (?)
				AND id NOT IN (
					SELECT DISTINCT user_id FROM user_role_assignments
				)
			`, userIDs).Error; err != nil {
				return newServiceError(errcode.ErrDatabase, "删除租户用户失败")
			}
		}

		// 3j. 删除租户本身
		if err := tx.Exec("DELETE FROM tenants WHERE id = ?", tenantID).Error; err != nil {
			return newServiceError(errcode.ErrDatabase, "删除租户失败")
		}

		return nil
	})
}

// GetTenantStats 返回租户的成员、部门和角色计数。
func (s *TenantService) GetTenantStats(id uuid.UUID) (*dto.TenantStatsResponse, error) {
	_, err := s.tenantRepo.FindByID(id)
	if err != nil {
		return nil, newServiceError(errcode.ErrResourceNotFound, "租户不存在")
	}

	memberCount, deptCount, roleCount, err := s.tenantRepo.GetStats(id)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	return &dto.TenantStatsResponse{
		TenantID:        id.String(),
		MemberCount:     memberCount,
		DepartmentCount: deptCount,
		RoleCount:       roleCount,
	}, nil
}

// ListTenantMembers 返回指定租户的所有成员（系统管理员视角）。
func (s *TenantService) ListTenantMembers(tenantID uuid.UUID) ([]dto.TenantMemberItem, error) {
	_, err := s.tenantRepo.FindByID(tenantID)
	if err != nil {
		return nil, newServiceError(errcode.ErrResourceNotFound, "租户不存在")
	}

	var members []model.OrgMember
	if err := s.db.Where("tenant_id = ?", tenantID).
		Preload("User").
		Preload("Department").
		Preload("Roles").
		Order("created_at ASC").
		Find(&members).Error; err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	result := make([]dto.TenantMemberItem, len(members))
	for i, m := range members {
		roleNames := make([]string, len(m.Roles))
		for j, r := range m.Roles {
			roleNames[j] = r.Name
		}
		result[i] = dto.TenantMemberItem{
			ID:             m.ID.String(),
			Username:       m.User.Username,
			DisplayName:    m.User.DisplayName,
			Email:          m.User.Email,
			Phone:          m.User.Phone,
			DepartmentName: m.Department.Name,
			RoleNames:      roleNames,
			Position:       m.Position,
			Status:         m.Status,
			CreatedAt:      m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	return result, nil
}

// toTenantResponse 将 model.Tenant 转换为 dto.TenantResponse。
func toTenantResponse(t *model.Tenant) dto.TenantResponse {
	oaConnID := ""
	if t.OADBConnectionID != nil {
		oaConnID = t.OADBConnectionID.String()
	}
	primaryModelID := ""
	if t.PrimaryModelID != nil {
		primaryModelID = t.PrimaryModelID.String()
	}
	fallbackModelID := ""
	if t.FallbackModelID != nil {
		fallbackModelID = t.FallbackModelID.String()
	}
	adminUserID := ""
	if t.AdminUserID != nil {
		adminUserID = t.AdminUserID.String()
	}

	temp := t.Temperature

	return dto.TenantResponse{
		ID:                  t.ID.String(),
		Name:                t.Name,
		Code:                t.Code,
		Description:         t.Description,
		Status:              t.Status,
		OADBConnectionID:    oaConnID,
		TokenQuota:          t.TokenQuota,
		TokenUsed:           t.TokenUsed,
		MaxConcurrency:      t.MaxConcurrency,
		PrimaryModelID:      primaryModelID,
		FallbackModelID:     fallbackModelID,
		MaxTokensPerRequest: t.MaxTokensPerRequest,
		Temperature:         temp,
		TimeoutSeconds:      t.TimeoutSeconds,
		RetryCount:          t.RetryCount,
		SSOEnabled:          t.SSOEnabled,
		SSOEndpoint:         t.SSOEndpoint,
		LogRetentionDays:    t.LogRetentionDays,
		DataRetentionDays:   t.DataRetentionDays,
		ContactName:         t.ContactName,
		ContactEmail:        t.ContactEmail,
		ContactPhone:        t.ContactPhone,
		AdminUserID:         adminUserID,
		CreatedAt:           t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:           t.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
