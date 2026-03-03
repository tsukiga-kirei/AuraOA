package service

import (
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/repository"
)

//TenantService 处理 system_admin 的租户 CRUD 和统计信息。
type TenantService struct {
	tenantRepo *repository.TenantRepo
	db         *gorm.DB
}

//NewTenantService 创建一个新的 TenantService 实例。
func NewTenantService(tenantRepo *repository.TenantRepo, db *gorm.DB) *TenantService {
	return &TenantService{
		tenantRepo: tenantRepo,
		db:         db,
	}
}

//ListTenants 返回所有租户。
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

//CreateTenant 在检查代码唯一性后创建一个新租户。
//它使用事务还创建三个默认系统角色；如果任何步骤失败，则整个操作将回滚。
func (s *TenantService) CreateTenant(req *dto.CreateTenantRequest) (*dto.TenantResponse, error) {
	//检查代码唯一性
	existing, _ := s.tenantRepo.FindByCode(req.Code)
	if existing != nil {
		return nil, newServiceError(errcode.ErrResourceConflict, "租户编码已存在")
	}

	//构建 AIConfig JSON
	aiConfigJSON, _ := json.Marshal(req.AIConfig)
	if req.AIConfig == nil {
		aiConfigJSON = []byte("{}")
	}

	tenant := &model.Tenant{
		Name:           req.Name,
		Code:           req.Code,
		Description:    req.Description,
		OAType:         req.OAType,
		TokenQuota:     req.TokenQuota,
		MaxConcurrency: req.MaxConcurrency,
		AIConfig:       aiConfigJSON,
		ContactName:    req.ContactName,
		ContactEmail:   req.ContactEmail,
		ContactPhone:   req.ContactPhone,
	}

	//如果未提供，则应用默认值
	if tenant.OAType == "" {
		tenant.OAType = "weaver_e9"
	}
	if tenant.TokenQuota == 0 {
		tenant.TokenQuota = 10000
	}
	if tenant.MaxConcurrency == 0 {
		tenant.MaxConcurrency = 10
	}

	//启动事务：自动创建租户+默认角色
	tx := s.db.Begin()
	defer tx.Rollback() //提交后无操作

	if err := tx.Create(tenant).Error; err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	//为每个默认角色构建 page_permissions JSON
	businessPerms, _ := json.Marshal([]string{"/overview", "/dashboard", "/settings"})
	auditPerms, _ := json.Marshal([]string{"/overview", "/dashboard", "/cron", "/archive", "/settings"})
	adminPerms, _ := json.Marshal([]string{
		"/overview", "/dashboard", "/cron", "/archive", "/settings",
		"/admin/tenant/rules", "/admin/tenant/org", "/admin/tenant/data", "/admin/tenant/user-configs",
	})

	defaultRoles := []model.OrgRole{
		{
			TenantID:        tenant.ID,
			Name:            "业务用户",
			Description:     "普通业务人员，可使用审核工作台等前台功能。仪表盘为所有角色默认拥有。",
			PagePermissions: businessPerms,
			IsSystem:        true,
		},
		{
			TenantID:        tenant.ID,
			Name:            "审计管理员",
			Description:     "在业务用户基础上，额外拥有归档复盘权限，可进行合规复核。",
			PagePermissions: auditPerms,
			IsSystem:        true,
		},
		{
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

//UpdateTenant 更新现有租户的字段。
func (s *TenantService) UpdateTenant(id uuid.UUID, req *dto.UpdateTenantRequest) (*dto.TenantResponse, error) {
	tenant, err := s.tenantRepo.FindByID(id)
	if err != nil {
		return nil, newServiceError(errcode.ErrResourceNotFound, "租户不存在")
	}

	//更新字段（如果提供）
	if req.Name != "" {
		tenant.Name = req.Name
	}
	if req.Status != "" {
		tenant.Status = req.Status
	}
	if req.Description != "" {
		tenant.Description = req.Description
	}
	if req.OAType != "" {
		tenant.OAType = req.OAType
	}
	if req.TokenQuota != 0 {
		tenant.TokenQuota = req.TokenQuota
	}
	if req.MaxConcurrency != 0 {
		tenant.MaxConcurrency = req.MaxConcurrency
	}
	if req.AIConfig != nil {
		aiConfigJSON, _ := json.Marshal(req.AIConfig)
		tenant.AIConfig = aiConfigJSON
	}
	if req.SSOEnabled != nil {
		tenant.SSOEnabled = *req.SSOEnabled
	}
	if req.SSOEndpoint != "" {
		tenant.SSOEndpoint = req.SSOEndpoint
	}
	if req.LogRetentionDays != 0 {
		tenant.LogRetentionDays = req.LogRetentionDays
	}
	if req.DataRetentionDays != 0 {
		tenant.DataRetentionDays = req.DataRetentionDays
	}
	if req.AllowCustomModel != nil {
		tenant.AllowCustomModel = *req.AllowCustomModel
	}
	if req.ContactName != "" {
		tenant.ContactName = req.ContactName
	}
	if req.ContactEmail != "" {
		tenant.ContactEmail = req.ContactEmail
	}
	if req.ContactPhone != "" {
		tenant.ContactPhone = req.ContactPhone
	}

	if err := s.tenantRepo.Update(tenant); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	resp := toTenantResponse(tenant)
	return &resp, nil
}

//DeleteTenant 通过 ID 删除租户。
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

//GetTenantStats 返回租户的成员、部门和角色计数。
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

//toTenantResponse 将 model.Tenant 转换为 dto.TenantResponse。
func toTenantResponse(t *model.Tenant) dto.TenantResponse {
	var aiConfig interface{}
	_ = json.Unmarshal(t.AIConfig, &aiConfig)

	return dto.TenantResponse{
		ID:                t.ID.String(),
		Name:              t.Name,
		Code:              t.Code,
		Description:       t.Description,
		Status:            t.Status,
		OAType:            t.OAType,
		TokenQuota:        t.TokenQuota,
		TokenUsed:         t.TokenUsed,
		MaxConcurrency:    t.MaxConcurrency,
		AIConfig:          aiConfig,
		SSOEnabled:        t.SSOEnabled,
		SSOEndpoint:       t.SSOEndpoint,
		LogRetentionDays:  t.LogRetentionDays,
		DataRetentionDays: t.DataRetentionDays,
		AllowCustomModel:  t.AllowCustomModel,
		ContactName:       t.ContactName,
		ContactEmail:      t.ContactEmail,
		ContactPhone:      t.ContactPhone,
		CreatedAt:         t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:         t.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
