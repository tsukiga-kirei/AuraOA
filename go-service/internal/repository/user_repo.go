package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

// UserRepo provides data access methods for users, login history, role assignments, and tenants.
type UserRepo struct {
	*BaseRepo
}

// NewUserRepo creates a new UserRepo instance.
func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{BaseRepo: NewBaseRepo(db)}
}

// FindByUsername finds a user by username.
func (r *UserRepo) FindByUsername(username string) (*model.User, error) {
	var user model.User
	if err := r.DB.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByID finds a user by UUID.
func (r *UserRepo) FindByID(id uuid.UUID) (*model.User, error) {
	var user model.User
	if err := r.DB.Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateLoginFail increments login_fail_count by 1.
// If the count reaches 5, it sets locked_until to now + 15 minutes.
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

// ResetLoginFail resets login_fail_count to 0 and clears locked_until.
func (r *UserRepo) ResetLoginFail(userID uuid.UUID) error {
	return r.DB.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"login_fail_count": 0,
		"locked_until":     nil,
	}).Error
}

// CreateLoginHistory creates a login history record.
func (r *UserRepo) CreateLoginHistory(history *model.LoginHistory) error {
	return r.DB.Create(history).Error
}

// FindRoleAssignments finds all role assignments for a user.
func (r *UserRepo) FindRoleAssignments(userID uuid.UUID) ([]model.UserRoleAssignment, error) {
	var assignments []model.UserRoleAssignment
	if err := r.DB.Where("user_id = ?", userID).Find(&assignments).Error; err != nil {
		return nil, err
	}
	return assignments, nil
}

// FindRoleAssignmentByID finds a specific role assignment by ID.
func (r *UserRepo) FindRoleAssignmentByID(id uuid.UUID) (*model.UserRoleAssignment, error) {
	var assignment model.UserRoleAssignment
	if err := r.DB.Where("id = ?", id).First(&assignment).Error; err != nil {
		return nil, err
	}
	return &assignment, nil
}

// FindTenantByID finds a tenant by ID.
func (r *UserRepo) FindTenantByID(tenantID uuid.UUID) (*model.Tenant, error) {
	var tenant model.Tenant
	if err := r.DB.Where("id = ?", tenantID).First(&tenant).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}
