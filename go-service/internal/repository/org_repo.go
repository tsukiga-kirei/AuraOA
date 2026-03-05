package repository

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

//OrgRepo 为部门、组织角色和组织成员提供数据访问方法。
type OrgRepo struct {
	*BaseRepo
}

//NewOrgRepo 创建一个新的 OrgRepo 实例。
func NewOrgRepo(db *gorm.DB) *OrgRepo {
	return &OrgRepo{BaseRepo: NewBaseRepo(db)}
}

//--- 部门方法 ---

//ListDepartments 返回当前租户范围内的所有部门。
func (r *OrgRepo) ListDepartments(c *gin.Context) ([]model.Department, error) {
	var departments []model.Department
	if err := r.WithTenant(c).Order("sort_order ASC").Find(&departments).Error; err != nil {
		return nil, err
	}
	return departments, nil
}

//CreateDepartment 创建一个新的部门记录。
func (r *OrgRepo) CreateDepartment(dept *model.Department) error {
	return r.DB.Create(dept).Error
}

//UpdateDepartment 更新现有部门记录。
func (r *OrgRepo) UpdateDepartment(dept *model.Department) error {
	return r.DB.Save(dept).Error
}

//DeleteDepartment 按 ID 删除部门。
func (r *OrgRepo) DeleteDepartment(id uuid.UUID) error {
	return r.DB.Where("id = ?", id).Delete(&model.Department{}).Error
}

//CountMembersByDept 返回给定部门中的组织成员数量。
func (r *OrgRepo) CountMembersByDept(deptID uuid.UUID) (int64, error) {
	var count int64
	if err := r.DB.Model(&model.OrgMember{}).Where("department_id = ?", deptID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

//FindDepartmentByID 按 ID 查找当前租户范围内的部门。
func (r *OrgRepo) FindDepartmentByID(c *gin.Context, id uuid.UUID) (*model.Department, error) {
	var dept model.Department
	if err := r.WithTenant(c).Where("id = ?", id).First(&dept).Error; err != nil {
		return nil, err
	}
	return &dept, nil
}

//--- OrgRole 方法 ---

//ListRoles 返回当前租户范围内的所有组织角色。
func (r *OrgRepo) ListRoles(c *gin.Context) ([]model.OrgRole, error) {
	var roles []model.OrgRole
	if err := r.WithTenant(c).Order("created_at ASC").Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

//CreateRole 创建新的组织角色记录。
func (r *OrgRepo) CreateRole(role *model.OrgRole) error {
	return r.DB.Create(role).Error
}

//UpdateRole 更新现有组织角色记录。
func (r *OrgRepo) UpdateRole(role *model.OrgRole) error {
	return r.DB.Save(role).Error
}

//DeleteRole 按 ID 删除组织角色。
func (r *OrgRepo) DeleteRole(id uuid.UUID) error {
	return r.DB.Where("id = ?", id).Delete(&model.OrgRole{}).Error
}

//FindRoleByID 按 ID 查找当前租户范围内的组织角色。
func (r *OrgRepo) FindRoleByID(c *gin.Context, id uuid.UUID) (*model.OrgRole, error) {
	var role model.OrgRole
	if err := r.WithTenant(c).Where("id = ?", id).First(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

//FindRolesByIDs 按 ID 查找多个组织角色。
func (r *OrgRepo) FindRolesByIDs(ids []uuid.UUID) ([]model.OrgRole, error) {
	var roles []model.OrgRole
	if err := r.DB.Where("id IN ?", ids).Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

//--- 成员方法 ---

//ListMembers 返回当前租户范围内的所有组织成员，并预加载用户、部门和角色。
func (r *OrgRepo) ListMembers(c *gin.Context) ([]model.OrgMember, error) {
	var members []model.OrgMember
	if err := r.WithTenant(c).
		Preload("User").
		Preload("Department").
		Preload("Roles").
		Order("created_at ASC").
		Find(&members).Error; err != nil {
		return nil, err
	}
	return members, nil
}

//CreateMember 创建新的组织成员记录。
func (r *OrgRepo) CreateMember(member *model.OrgMember) error {
	return r.DB.Create(member).Error
}

//UpdateMember 更新现有的组织成员记录。
func (r *OrgRepo) UpdateMember(member *model.OrgMember) error {
	return r.DB.Model(member).Select("department_id", "position", "status", "updated_at").Updates(member).Error
}

//DeleteMember 按 ID 删除组织成员。
func (r *OrgRepo) DeleteMember(id uuid.UUID) error {
	return r.DB.Where("id = ?", id).Delete(&model.OrgMember{}).Error
}

//FindMemberByID 按 ID 查找当前租户范围内的组织成员，并预加载关联。
func (r *OrgRepo) FindMemberByID(c *gin.Context, id uuid.UUID) (*model.OrgMember, error) {
	var member model.OrgMember
	if err := r.WithTenant(c).
		Preload("User").
		Preload("Department").
		Preload("Roles").
		Where("id = ?", id).
		First(&member).Error; err != nil {
		return nil, err
	}
	return &member, nil
}

//FindByUserAndTenant 按用户 ID 和租户 ID 查找组织成员。
func (r *OrgRepo) FindByUserAndTenant(userID, tenantID uuid.UUID) (*model.OrgMember, error) {
	var member model.OrgMember
	if err := r.DB.Where("user_id = ? AND tenant_id = ?", userID, tenantID).First(&member).Error; err != nil {
		return nil, err
	}
	return &member, nil
}
