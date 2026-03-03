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

// TenantService handles tenant CRUD and statistics for system_admin.
type TenantService struct {
	tenantRepo *repository.TenantRepo
	db         *gorm.DB
}

// NewTenantService creates a new TenantService instance.
func NewTenantService(tenantRepo *repository.TenantRepo, db *gorm.DB) *TenantService {
	return &TenantService{
		tenantRepo: tenantRepo,
		db:         db,
	}
}

// ListTenants returns all tenants.
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

// CreateTenant creates a new tenant after checking code uniqueness.
func (s *TenantService) CreateTenant(req *dto.CreateTenantRequest) (*dto.TenantResponse, error) {
	// Check code uniqueness
	existing, _ := s.tenantRepo.FindByCode(req.Code)
	if existing != nil {
		return nil, newServiceError(errcode.ErrResourceConflict, "租户编码已存在")
	}

	// Build AIConfig JSON
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

	// Apply defaults if not provided
	if tenant.OAType == "" {
		tenant.OAType = "weaver_e9"
	}
	if tenant.TokenQuota == 0 {
		tenant.TokenQuota = 10000
	}
	if tenant.MaxConcurrency == 0 {
		tenant.MaxConcurrency = 10
	}

	if err := s.tenantRepo.Create(tenant); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	resp := toTenantResponse(tenant)
	return &resp, nil
}

// UpdateTenant updates an existing tenant's fields.
func (s *TenantService) UpdateTenant(id uuid.UUID, req *dto.UpdateTenantRequest) (*dto.TenantResponse, error) {
	tenant, err := s.tenantRepo.FindByID(id)
	if err != nil {
		return nil, newServiceError(errcode.ErrResourceNotFound, "租户不存在")
	}

	// Update fields if provided
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

// DeleteTenant deletes a tenant by ID.
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

// GetTenantStats returns member, department, and role counts for a tenant.
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

// toTenantResponse converts a model.Tenant to dto.TenantResponse.
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
