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

// CountUsers 返回 users 表行数（用于首次部署初始化判断）。
func (r *UserRepo) CountUsers() (int64, error) {
	var count int64
	err := r.DB.Model(&model.User{}).Count(&count).Error
	return count, err
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

// FindBusinessRoleAssignment 查找用户在指定租户下的 business 角色分配（用于发送通知）。
func (r *UserRepo) FindBusinessRoleAssignment(userID, tenantID uuid.UUID) (*model.UserRoleAssignment, error) {
	var a model.UserRoleAssignment
	if err := r.DB.Where("user_id = ? AND tenant_id = ? AND role = 'business'", userID, tenantID).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// UpdatePasswordHash updates the user's password hash.
func (r *UserRepo) UpdatePasswordHash(userID uuid.UUID, hash string) error {
	return r.DB.Model(&model.User{}).Where("id = ?", userID).Update("password_hash", hash).Error
}

// UpdatePasswordHashAndTime updates the user's password hash and sets password_changed_at to now.
func (r *UserRepo) UpdatePasswordHashAndTime(userID uuid.UUID, hash string) error {
	return r.DB.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"password_hash":       hash,
		"password_changed_at": time.Now(),
	}).Error
}

// UpdateLocale updates the user's locale preference.
func (r *UserRepo) UpdateLocale(userID uuid.UUID, locale string) error {
	return r.DB.Model(&model.User{}).Where("id = ?", userID).Update("locale", locale).Error
}

// FindRecentLoginHistory returns the most recent login history entries for a user (up to limit).
func (r *UserRepo) FindRecentLoginHistory(userID uuid.UUID, limit int) ([]model.LoginHistory, error) {
	var histories []model.LoginHistory
	if err := r.DB.Where("user_id = ?", userID).Order("login_at DESC").Limit(limit).Find(&histories).Error; err != nil {
		return nil, err
	}
	return histories, nil
}

// UpdateProfile updates the user's display_name, email, and phone.
func (r *UserRepo) UpdateProfile(userID uuid.UUID, updates map[string]interface{}) error {
	return r.DB.Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error
}

