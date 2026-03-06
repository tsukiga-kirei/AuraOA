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

// CreateTenant 在检查代码唯一性后创建一个新租户。
// 它使用事务创建租户、默认角色、默认部门和租户管理员账号；如果任何步骤失败，则整个操作将回滚。
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

// UpdateTenant 更新现有租户的字段。
// 使用 map 构建更新字段，避免零值覆盖已有数据。
func (s *TenantService) UpdateTenant(id uuid.UUID, req *dto.UpdateTenantRequest) (*dto.TenantResponse, error) {
	tenant, err := s.tenantRepo.FindByID(id)
	if err != nil {
		return nil, newServiceError(errcode.ErrResourceNotFound, "租户不存在")
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

	// 重新查询最新数据返回
	tenant, err = s.tenantRepo.FindByID(id)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	resp := toTenantResponse(tenant)
	return &resp, nil
}

// DeleteTenant 通过 ID 删除租户。
func (s *TenantService) DeleteTenant(id uuid.UUID) error {
	_, err := s.tenantRepo.FindByID(id)
	if err != nil {
		return newServiceError(errcode.ErrResourceNotFound, "租户不存在")
	}
	if err := s.tenantRepo.Delete(id); err != nil {
		return newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	return nil
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
		CreatedAt:           t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:           t.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
