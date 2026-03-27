package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

//TenantRepo 提供用于租户管理（system_admin 范围）的数据访问方法。
//与 OrgRepo 不同，TenantRepo 不使用 WithTenant，因为它自己管理租户。
type TenantRepo struct {
	*BaseRepo
}

//NewTenantRepo 创建一个新的 TenantRepo 实例。
func NewTenantRepo(db *gorm.DB) *TenantRepo {
	return &TenantRepo{BaseRepo: NewBaseRepo(db)}
}

//列表返回按创建时间排序的所有租户。
func (r *TenantRepo) List() ([]model.Tenant, error) {
	var tenants []model.Tenant
	if err := r.DB.Order("created_at ASC").Find(&tenants).Error; err != nil {
		return nil, err
	}
	return tenants, nil
}

//创建创建新的租户记录。
func (r *TenantRepo) Create(tenant *model.Tenant) error {
	return r.DB.Create(tenant).Error
}

//Update 更新现有租户记录。使用 Model+Select 模式避免零值覆盖。
func (r *TenantRepo) Update(tenant *model.Tenant) error {
	return r.DB.Model(tenant).Where("id = ?", tenant.ID).Updates(tenant).Error
}

//UpdateFields 通过 map 更新指定字段，支持零值更新。
func (r *TenantRepo) UpdateFields(id uuid.UUID, fields map[string]interface{}) error {
	return r.DB.Model(&model.Tenant{}).Where("id = ?", id).Updates(fields).Error
}

//删除通过 ID 删除租户。
func (r *TenantRepo) Delete(id uuid.UUID) error {
	return r.DB.Where("id = ?", id).Delete(&model.Tenant{}).Error
}

//FindByID 通过 UUID 查找租户。
func (r *TenantRepo) FindByID(id uuid.UUID) (*model.Tenant, error) {
	var tenant model.Tenant
	if err := r.DB.Where("id = ?", id).First(&tenant).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

//FindByCode 通过其唯一代码查找租户。
func (r *TenantRepo) FindByCode(code string) (*model.Tenant, error) {
	var tenant model.Tenant
	if err := r.DB.Where("code = ?", code).First(&tenant).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

//ListActive 返回仅活跃状态的租户（用于公共登录页面）。
func (r *TenantRepo) ListActive() ([]model.Tenant, error) {
	var tenants []model.Tenant
	if err := r.DB.Where("status = ?", "active").Order("created_at ASC").Find(&tenants).Error; err != nil {
		return nil, err
	}
	return tenants, nil
}

// DashboardPlatformTenantCounts 全平台租户总数与活跃租户数。
func (r *TenantRepo) DashboardPlatformTenantCounts() (total, active int64, err error) {
	if err = r.DB.Model(&model.Tenant{}).Count(&total).Error; err != nil {
		return 0, 0, err
	}
	if err = r.DB.Model(&model.Tenant{}).Where("status = ?", "active").Count(&active).Error; err != nil {
		return 0, 0, err
	}
	return total, active, nil
}

// DashboardPlatformTokenSum 全平台各租户 token_used / token_quota 求和。
func (r *TenantRepo) DashboardPlatformTokenSum() (used, quota int64, err error) {
	type sumRow struct {
		SUsed  int64 `gorm:"column:s_used"`
		SQuota int64 `gorm:"column:s_quota"`
	}
	var row sumRow
	err = r.DB.Model(&model.Tenant{}).
		Select("COALESCE(SUM(token_used), 0)::bigint AS s_used, COALESCE(SUM(token_quota), 0)::bigint AS s_quota").
		Scan(&row).Error
	return row.SUsed, row.SQuota, err
}

//GetStats 返回给定租户的成员计数、部门计数和角色计数。
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
