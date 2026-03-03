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

//更新更新现有租户记录。
func (r *TenantRepo) Update(tenant *model.Tenant) error {
	return r.DB.Save(tenant).Error
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
