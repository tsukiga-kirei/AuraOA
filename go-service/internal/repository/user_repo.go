package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

//UserRepo 为用户、登录历史、角色分配和租户提供数据访问方法。
type UserRepo struct {
	*BaseRepo
}

//NewUserRepo 创建一个新的 UserRepo 实例。
func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{BaseRepo: NewBaseRepo(db)}
}

//FindByUsername 通过用户名查找用户。
func (r *UserRepo) FindByUsername(username string) (*model.User, error) {
	var user model.User
	if err := r.DB.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

//FindByID 通过 UUID 查找用户。
func (r *UserRepo) FindByID(id uuid.UUID) (*model.User, error) {
	var user model.User
	if err := r.DB.Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

//UpdateLoginFail 使 login_fail_count 增加 1。
//如果计数达到 5，它将locked_until 设置为 now + 15 分钟。
func (r *UserRepo) UpdateLoginFail(user *model.User) error {
	user.LoginFailCount++
	updates := map[string]interface{}{
		"login_fail_count": user.LoginFailCount,
	}
	if user.LoginFailCount >= 5 {
		lockedUntil := time.Now().Add(15 * time.Minute)
		user.LockedUntil = &lockedUntil
		updates["locked_until"] = lockedUntil
	}
	return r.DB.Model(user).Updates(updates).Error
}

//ResetLoginFail 将login_fail_count 重置为0 并清除locked_until。
func (r *UserRepo) ResetLoginFail(userID uuid.UUID) error {
	return r.DB.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"login_fail_count": 0,
		"locked_until":     nil,
	}).Error
}

//CreateLoginHistory 创建登录历史记录。
func (r *UserRepo) CreateLoginHistory(history *model.LoginHistory) error {
	return r.DB.Create(history).Error
}

//FindRoleAssignments 查找用户的所有角色分配。
func (r *UserRepo) FindRoleAssignments(userID uuid.UUID) ([]model.UserRoleAssignment, error) {
	var assignments []model.UserRoleAssignment
	if err := r.DB.Where("user_id = ?", userID).Find(&assignments).Error; err != nil {
		return nil, err
	}
	return assignments, nil
}

//FindRoleAssignmentByID 按 ID 查找特定角色分配。
func (r *UserRepo) FindRoleAssignmentByID(id uuid.UUID) (*model.UserRoleAssignment, error) {
	var assignment model.UserRoleAssignment
	if err := r.DB.Where("id = ?", id).First(&assignment).Error; err != nil {
		return nil, err
	}
	return &assignment, nil
}

//FindTenantByID 通过 ID 查找租户。
func (r *UserRepo) FindTenantByID(tenantID uuid.UUID) (*model.Tenant, error) {
	var tenant model.Tenant
	if err := r.DB.Where("id = ?", tenantID).First(&tenant).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

// UpdatePasswordHash updates the user's password hash.
func (r *UserRepo) UpdatePasswordHash(userID uuid.UUID, hash string) error {
	return r.DB.Model(&model.User{}).Where("id = ?", userID).Update("password_hash", hash).Error
}
