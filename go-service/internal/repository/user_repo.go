package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

// UserRepo 提供用户、登录历史、角色分配等数据访问方法。
type UserRepo struct {
	*BaseRepo
}

// NewUserRepo 创建 UserRepo 实例。
func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{BaseRepo: NewBaseRepo(db)}
}

// CountUsers 统计 users 表总行数，用于首次部署时判断是否需要初始化管理员账号。
func (r *UserRepo) CountUsers() (int64, error) {
	var count int64
	err := r.DB.Model(&model.User{}).Count(&count).Error
	return count, err
}

// FindByUsername 按用户名查询用户，用于登录鉴权。
func (r *UserRepo) FindByUsername(username string) (*model.User, error) {
	var user model.User
	if err := r.DB.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByID 按 UUID 查询用户。
func (r *UserRepo) FindByID(id uuid.UUID) (*model.User, error) {
	var user model.User
	if err := r.DB.Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateLoginFail 登录失败时将 login_fail_count 加 1；
// 若累计失败次数达到 5 次，则将 locked_until 设置为当前时间 + 15 分钟。
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

// ResetLoginFail 登录成功后重置 login_fail_count 为 0 并清除 locked_until 锁定时间。
func (r *UserRepo) ResetLoginFail(userID uuid.UUID) error {
	return r.DB.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"login_fail_count": 0,
		"locked_until":     nil,
	}).Error
}

// CreateLoginHistory 写入一条登录历史记录。
func (r *UserRepo) CreateLoginHistory(history *model.LoginHistory) error {
	return r.DB.Create(history).Error
}

// FindRoleAssignments 查询指定用户的所有角色分配记录（含多租户、多角色）。
func (r *UserRepo) FindRoleAssignments(userID uuid.UUID) ([]model.UserRoleAssignment, error) {
	var assignments []model.UserRoleAssignment
	if err := r.DB.Where("user_id = ?", userID).Find(&assignments).Error; err != nil {
		return nil, err
	}
	return assignments, nil
}

// FindRoleAssignmentByID 按 ID 查询单条角色分配记录。
func (r *UserRepo) FindRoleAssignmentByID(id uuid.UUID) (*model.UserRoleAssignment, error) {
	var assignment model.UserRoleAssignment
	if err := r.DB.Where("id = ?", id).First(&assignment).Error; err != nil {
		return nil, err
	}
	return &assignment, nil
}

// FindTenantByID 按 ID 查询租户信息（供用户登录流程使用）。
func (r *UserRepo) FindTenantByID(tenantID uuid.UUID) (*model.Tenant, error) {
	var tenant model.Tenant
	if err := r.DB.Where("id = ?", tenantID).First(&tenant).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

// FindBusinessRoleAssignment 查询用户在指定租户下的 business 角色分配，用于发送业务通知。
func (r *UserRepo) FindBusinessRoleAssignment(userID, tenantID uuid.UUID) (*model.UserRoleAssignment, error) {
	var a model.UserRoleAssignment
	if err := r.DB.Where("user_id = ? AND tenant_id = ? AND role = 'business'", userID, tenantID).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// UpdatePasswordHash 更新用户的密码哈希值。
func (r *UserRepo) UpdatePasswordHash(userID uuid.UUID, hash string) error {
	return r.DB.Model(&model.User{}).Where("id = ?", userID).Update("password_hash", hash).Error
}

// UpdatePasswordHashAndTime 更新用户密码哈希值，同时将 password_changed_at 设置为当前时间。
func (r *UserRepo) UpdatePasswordHashAndTime(userID uuid.UUID, hash string) error {
	return r.DB.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"password_hash":       hash,
		"password_changed_at": time.Now(),
	}).Error
}

// UpdateLocale 更新用户的界面语言偏好设置。
func (r *UserRepo) UpdateLocale(userID uuid.UUID, locale string) error {
	return r.DB.Model(&model.User{}).Where("id = ?", userID).Update("locale", locale).Error
}

// FindRecentLoginHistory 查询指定用户最近 limit 条登录历史，按登录时间倒序排列。
func (r *UserRepo) FindRecentLoginHistory(userID uuid.UUID, limit int) ([]model.LoginHistory, error) {
	var histories []model.LoginHistory
	if err := r.DB.Where("user_id = ?", userID).Order("login_at DESC").Limit(limit).Find(&histories).Error; err != nil {
		return nil, err
	}
	return histories, nil
}

// UpdateProfile 更新用户的个人资料字段（display_name、email、phone 等）。
func (r *UserRepo) UpdateProfile(userID uuid.UUID, updates map[string]interface{}) error {
	return r.DB.Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error
}
