package repository

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

// OrgRepo provides data access methods for departments, org roles, and org members.
type OrgRepo struct {
	*BaseRepo
}

// NewOrgRepo creates a new OrgRepo instance.
func NewOrgRepo(db *gorm.DB) *OrgRepo {
	return &OrgRepo{BaseRepo: NewBaseRepo(db)}
}

// --- Department methods ---

// ListDepartments returns all departments scoped to the current tenant.
func (r *OrgRepo) ListDepartments(c *gin.Context) ([]model.Department, error) {
	var departments []model.Department
	if err := r.WithTenant(c).Order("sort_order ASC").Find(&departments).Error; err != nil {
		return nil, err
	}
	return departments, nil
}

// CreateDepartment creates a new department record.
func (r *OrgRepo) CreateDepartment(dept *model.Department) error {
	return r.DB.Create(dept).Error
}

// UpdateDepartment updates an existing department record.
func (r *OrgRepo) UpdateDepartment(dept *model.Department) error {
	return r.DB.Save(dept).Error
}

// DeleteDepartment deletes a department by ID.
func (r *OrgRepo) DeleteDepartment(id uuid.UUID) error {
	return r.DB.Where("id = ?", id).Delete(&model.Department{}).Error
}

// CountMembersByDept returns the number of org members in a given department.
func (r *OrgRepo) CountMembersByDept(deptID uuid.UUID) (int64, error) {
	var count int64
	if err := r.DB.Model(&model.OrgMember{}).Where("department_id = ?", deptID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// FindDepartmentByID finds a department by ID scoped to the current tenant.
func (r *OrgRepo) FindDepartmentByID(c *gin.Context, id uuid.UUID) (*model.Department, error) {
	var dept model.Department
	if err := r.WithTenant(c).Where("id = ?", id).First(&dept).Error; err != nil {
		return nil, err
	}
	return &dept, nil
}

// --- OrgRole methods ---

// ListRoles returns all org roles scoped to the current tenant.
func (r *OrgRepo) ListRoles(c *gin.Context) ([]model.OrgRole, error) {
	var roles []model.OrgRole
	if err := r.WithTenant(c).Order("created_at ASC").Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// CreateRole creates a new org role record.
func (r *OrgRepo) CreateRole(role *model.OrgRole) error {
	return r.DB.Create(role).Error
}

// UpdateRole updates an existing org role record.
func (r *OrgRepo) UpdateRole(role *model.OrgRole) error {
	return r.DB.Save(role).Error
}

// DeleteRole deletes an org role by ID.
func (r *OrgRepo) DeleteRole(id uuid.UUID) error {
	return r.DB.Where("id = ?", id).Delete(&model.OrgRole{}).Error
}

// FindRoleByID finds an org role by ID scoped to the current tenant.
func (r *OrgRepo) FindRoleByID(c *gin.Context, id uuid.UUID) (*model.OrgRole, error) {
	var role model.OrgRole
	if err := r.WithTenant(c).Where("id = ?", id).First(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

// FindRolesByIDs finds multiple org roles by their IDs.
func (r *OrgRepo) FindRolesByIDs(ids []uuid.UUID) ([]model.OrgRole, error) {
	var roles []model.OrgRole
	if err := r.DB.Where("id IN ?", ids).Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// --- Member methods ---

// ListMembers returns all org members scoped to the current tenant, with User, Department, and Roles preloaded.
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

// CreateMember creates a new org member record.
func (r *OrgRepo) CreateMember(member *model.OrgMember) error {
	return r.DB.Create(member).Error
}

// UpdateMember updates an existing org member record.
func (r *OrgRepo) UpdateMember(member *model.OrgMember) error {
	return r.DB.Save(member).Error
}

// DeleteMember deletes an org member by ID.
func (r *OrgRepo) DeleteMember(id uuid.UUID) error {
	return r.DB.Where("id = ?", id).Delete(&model.OrgMember{}).Error
}

// FindMemberByID finds an org member by ID scoped to the current tenant, with associations preloaded.
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

// FindByUserAndTenant finds an org member by user ID and tenant ID.
func (r *OrgRepo) FindByUserAndTenant(userID, tenantID uuid.UUID) (*model.OrgMember, error) {
	var member model.OrgMember
	if err := r.DB.Where("user_id = ? AND tenant_id = ?", userID, tenantID).First(&member).Error; err != nil {
		return nil, err
	}
	return &member, nil
}
