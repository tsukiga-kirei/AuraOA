package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

// TenantRepo 提供租户管理的数据访问方法，供 system_admin 使用。
// 与 OrgRepo 不同，TenantRepo 不使用 WithTenant，因为它本身就是管理租户的入口。
type TenantRepo struct {
	*BaseRepo
}

// NewTenantRepo 创建 TenantRepo 实例。
func NewTenantRepo(db *gorm.DB) *TenantRepo {
	return &TenantRepo{BaseRepo: NewBaseRepo(db)}
}

// List 查询所有租户，按创建时间升序排列。
func (r *TenantRepo) List() ([]model.Tenant, error) {
	var tenants []model.Tenant
	if err := r.DB.Order("created_at ASC").Find(&tenants).Error; err != nil {
		return nil, err
	}
	return tenants, nil
}

// Create 创建新的租户记录。
func (r *TenantRepo) Create(tenant *model.Tenant) error {
	return r.DB.Create(tenant).Error
}

// Update 更新租户记录，使用 Model+Updates 模式避免零值字段被覆盖。
func (r *TenantRepo) Update(tenant *model.Tenant) error {
	return r.DB.Model(tenant).Where("id = ?", tenant.ID).Updates(tenant).Error
}

// UpdateFields 通过字段 map 更新租户指定字段，支持零值更新。
func (r *TenantRepo) UpdateFields(id uuid.UUID, fields map[string]interface{}) error {
	return r.DB.Model(&model.Tenant{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 按 ID 删除租户记录。
func (r *TenantRepo) Delete(id uuid.UUID) error {
	return r.DB.Where("id = ?", id).Delete(&model.Tenant{}).Error
}

// FindByID 按 UUID 查询单个租户，不存在时返回 gorm.ErrRecordNotFound。
func (r *TenantRepo) FindByID(id uuid.UUID) (*model.Tenant, error) {
	var tenant model.Tenant
	if err := r.DB.Where("id = ?", id).First(&tenant).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

// FindByCode 按租户唯一标识码查询租户。
func (r *TenantRepo) FindByCode(code string) (*model.Tenant, error) {
	var tenant model.Tenant
	if err := r.DB.Where("code = ?", code).First(&tenant).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

// ListActive 查询所有状态为 active 的租户，用于登录页面展示可选租户列表。
func (r *TenantRepo) ListActive() ([]model.Tenant, error) {
	var tenants []model.Tenant
	if err := r.DB.Where("status = ?", "active").Order("created_at ASC").Find(&tenants).Error; err != nil {
		return nil, err
	}
	return tenants, nil
}

// DashboardPlatformTenantCounts 统计全平台租户总数与活跃租户数，用于系统管理员仪表盘。
func (r *TenantRepo) DashboardPlatformTenantCounts() (total, active int64, err error) {
	if err = r.DB.Model(&model.Tenant{}).Count(&total).Error; err != nil {
		return 0, 0, err
	}
	if err = r.DB.Model(&model.Tenant{}).Where("status = ?", "active").Count(&active).Error; err != nil {
		return 0, 0, err
	}
	return total, active, nil
}

// DashboardPlatformTokenSum 汇总全平台所有租户的 token_used 和 token_quota，用于平台 Token 用量概览。
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

// GetStats 查询指定租户的成员数、部门数和角色数，用于租户详情页统计展示。
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

// TenantWithUserCount 租户列表行，含活跃成员数量。
type TenantWithUserCount struct {
	TenantID   uuid.UUID `gorm:"column:tenant_id"`
	TenantName string    `gorm:"column:tenant_name"`
	TenantCode string    `gorm:"column:tenant_code"`
	UserCount  int64     `gorm:"column:user_count"`
}

// DashboardTenantListWithUserCount 查询所有租户及其活跃成员数量，用于系统管理员仪表盘租户列表。
func (r *TenantRepo) DashboardTenantListWithUserCount() ([]TenantWithUserCount, error) {
	sql := `
SELECT t.id AS tenant_id,
       t.name AS tenant_name,
       t.code AS tenant_code,
       COALESCE(mc.cnt, 0)::bigint AS user_count
FROM tenants t
LEFT JOIN (
  SELECT tenant_id, COUNT(*)::bigint AS cnt
  FROM org_members
  WHERE status = 'active'
  GROUP BY tenant_id
) mc ON mc.tenant_id = t.id
ORDER BY t.created_at ASC`

	var rows []TenantWithUserCount
	err := r.DB.Raw(sql).Scan(&rows).Error
	return rows, err
}

// DashboardActiveTenantIDs 查询近 30 天内有审核或归档快照记录的租户 ID 集合，用于判断活跃租户。
func (r *TenantRepo) DashboardActiveTenantIDs() (map[string]bool, error) {
	sql := `
SELECT DISTINCT tenant_id::text AS tid FROM (
  SELECT tenant_id FROM audit_process_snapshots
  WHERE updated_at >= NOW() - INTERVAL '30 days'
  UNION
  SELECT tenant_id FROM archive_process_snapshots
  WHERE updated_at >= NOW() - INTERVAL '30 days'
) sub`

	type row struct {
		Tid string `gorm:"column:tid"`
	}
	var rows []row
	if err := r.DB.Raw(sql).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(rows))
	for _, rw := range rows {
		out[rw.Tid] = true
	}
	return out, nil
}

// TenantTokenRow 按租户分列的 Token 用量行。
type TenantTokenRow struct {
	TenantID   uuid.UUID `gorm:"column:tenant_id"`
	TenantName string    `gorm:"column:tenant_name"`
	TenantCode string    `gorm:"column:tenant_code"`
	TokenUsed  int64     `gorm:"column:token_used"`
	TokenQuota int64     `gorm:"column:token_quota"`
}

// DashboardTenantTokenList 查询各租户的 token_used 和 token_quota，按用量降序排列，用于 Token 消耗排行。
func (r *TenantRepo) DashboardTenantTokenList() ([]TenantTokenRow, error) {
	sql := `
SELECT id AS tenant_id,
       name AS tenant_name,
       code AS tenant_code,
       COALESCE(token_used, 0)::bigint AS token_used,
       COALESCE(token_quota, 0)::bigint AS token_quota
FROM tenants
ORDER BY token_used DESC`

	var rows []TenantTokenRow
	err := r.DB.Raw(sql).Scan(&rows).Error
	return rows, err
}
