package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

// TenantRepo provides data access methods for tenant management (system_admin scope).
// Unlike OrgRepo, TenantRepo does NOT use WithTenant since it manages tenants themselves.
type TenantRepo struct {
	*BaseRepo
}

// NewTenantRepo creates a new TenantRepo instance.
func NewTenantRepo(db *gorm.DB) *TenantRepo {
	return &TenantRepo{BaseRepo: NewBaseRepo(db)}
}

// List returns all tenants ordered by creation time.
func (r *TenantRepo) List() ([]model.Tenant, error) {
	var tenants []model.Tenant
	if err := r.DB.Order("created_at ASC").Find(&tenants).Error; err != nil {
		return nil, err
	}
	return tenants, nil
}

// Create creates a new tenant record.
func (r *TenantRepo) Create(tenant *model.Tenant) error {
	return r.DB.Create(tenant).Error
}

// Update updates an existing tenant record.
func (r *TenantRepo) Update(tenant *model.Tenant) error {
	return r.DB.Save(tenant).Error
}

// Delete deletes a tenant by ID.
func (r *TenantRepo) Delete(id uuid.UUID) error {
	return r.DB.Where("id = ?", id).Delete(&model.Tenant{}).Error
}

// FindByID finds a tenant by its UUID.
func (r *TenantRepo) FindByID(id uuid.UUID) (*model.Tenant, error) {
	var tenant model.Tenant
	if err := r.DB.Where("id = ?", id).First(&tenant).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

// FindByCode finds a tenant by its unique code.
func (r *TenantRepo) FindByCode(code string) (*model.Tenant, error) {
	var tenant model.Tenant
	if err := r.DB.Where("code = ?", code).First(&tenant).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

// GetStats returns member count, department count, and role count for a given tenant.
func (r *TenantRepo) GetStats(tenantID uuid.UUID) (memberCount, deptCount, roleCount int64, err error) {
	if err = r.DB.Model(&model.OrgMember{}).Where("tenant_id = ?", tenantID).Count(&memberCount).Error; err != nil {
		return
	}
	if err = r.DB.Model(&model.Department{}).Where("tenant_id = ?", tenantID).Count(&deptCount).Error; err != nil {
		return
	}
	if err = r.DB.Model(&model.OrgRole{}).Where("tenant_id = ?", tenantID).Count(&roleCount).Error; err != nil {
		return
	}
	return
}
