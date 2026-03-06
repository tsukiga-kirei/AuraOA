package service

import (
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/repository"
)

// TenantService 处理 system_admin 的租户 CRUD 和统计信息。
type TenantService struct {
	tenantRepo       *repository.TenantRepo
	systemConfigRepo *repository.SystemConfigRepo
	db               *gorm.DB
}

// NewTenantService 创建一个新的 TenantService 实例。
func NewTenantService(tenantRepo *repository.TenantRepo, systemConfigRepo *repository.SystemConfigRepo, db *gorm.DB) *TenantService {
	return &TenantService{
		tenantRepo:       tenantRepo,
		systemConfigRepo: systemConfigRepo,
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
// 它使用事务还创建三个默认系统角色；如果任何步骤失败，则整个操作将回滚。
func (s *TenantService) CreateTenant(req *dto.CreateTenantRequest) (*dto.TenantResponse, error) {
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
		ContactName:         req.ContactName,
		ContactEmail:        req.ContactEmail,
		ContactPhone:        req.ContactPhone,
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

	// 启动事务：自动创建租户+默认角色
	tx := s.db.Begin()
	defer tx.Rollback() // 提交后无操作

	if err := tx.Create(tenant).Error; err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	// 为每个默认角色构建 page_permissions JSON
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
