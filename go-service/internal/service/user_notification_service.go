package service

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/repository"
)

const (
	userNotificationMaxLimit     = 100
	userNotificationDefaultLimit = 20
)

// UserNotificationService 用户通知业务逻辑。
type UserNotificationService struct {
	repo     *repository.UserNotificationRepo
	userRepo *repository.UserRepo
}

// NewUserNotificationService 创建服务。
func NewUserNotificationService(repo *repository.UserNotificationRepo, userRepo *repository.UserRepo) *UserNotificationService {
	return &UserNotificationService{repo: repo, userRepo: userRepo}
}

// Create 校验角色分配属于该用户后写入通知（内部调用，无 HTTP）。
func (s *UserNotificationService) Create(userID, roleAssignmentID uuid.UUID, category, title, body, linkPath string) error {
	a, err := s.userRepo.FindRoleAssignmentByID(roleAssignmentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return newServiceError(errcode.ErrResourceNotFound, "角色分配不存在")
		}
		return newServiceError(errcode.ErrDatabase, "查询角色分配失败")
	}
	if a.UserID != userID {
		return newServiceError(errcode.ErrResourceNotFound, "角色分配不存在")
	}
	if category == "" {
		category = "general"
	}
	n := &model.UserNotification{
		UserID:           userID,
		RoleAssignmentID: roleAssignmentID,
		Category:         category,
		Title:            title,
		Body:             body,
		LinkPath:         linkPath,
	}
	if err := s.repo.Create(n); err != nil {
		return newServiceError(errcode.ErrDatabase, "写入通知失败")
	}
	return nil
}

// CreateByTenant 根据 userID + tenantID 查找 business 角色分配后写入通知。
// 找不到角色分配时静默跳过（不影响主流程）。
func (s *UserNotificationService) CreateByTenant(userID, tenantID uuid.UUID, category, title, body, linkPath string) {
	a, err := s.userRepo.FindBusinessRoleAssignment(userID, tenantID)
	if err != nil {
		return
	}
	_ = s.Create(userID, a.ID, category, title, body, linkPath)
}

// List 当前 JWT 角色分配下的通知列表。
func (s *UserNotificationService) List(userID, roleAssignmentID uuid.UUID, limit, offset int, unreadOnly bool) (*dto.UserNotificationListResponse, error) {
	if limit <= 0 {
		limit = userNotificationDefaultLimit
	}
	if limit > userNotificationMaxLimit {
		limit = userNotificationMaxLimit
	}
	if offset < 0 {
		offset = 0
	}
	rows, total, err := s.repo.ListForAssignment(userID, roleAssignmentID, limit, offset, unreadOnly)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "查询通知失败")
	}
	items := make([]dto.UserNotificationItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, toNotificationItem(&row))
	}
	return &dto.UserNotificationListResponse{Items: items, Total: total}, nil
}

// UnreadCount 未读数。
func (s *UserNotificationService) UnreadCount(userID, roleAssignmentID uuid.UUID) (*dto.UserNotificationUnreadResponse, error) {
	n, err := s.repo.UnreadCount(userID, roleAssignmentID)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "查询未读数失败")
	}
	return &dto.UserNotificationUnreadResponse{Count: n}, nil
}

// MarkRead 单条已读。
func (s *UserNotificationService) MarkRead(userID, roleAssignmentID, notificationID uuid.UUID) error {
	n, err := s.repo.MarkRead(notificationID, userID, roleAssignmentID)
	if err != nil {
		return newServiceError(errcode.ErrDatabase, "更新通知失败")
	}
	if n == 0 {
		return newServiceError(errcode.ErrResourceNotFound, "通知不存在或无权访问")
	}
	return nil
}

// MarkAllRead 全部已读。
func (s *UserNotificationService) MarkAllRead(userID, roleAssignmentID uuid.UUID) error {
	if err := s.repo.MarkAllRead(userID, roleAssignmentID); err != nil {
		return newServiceError(errcode.ErrDatabase, "更新通知失败")
	}
	return nil
}

func toNotificationItem(row *model.UserNotification) dto.UserNotificationItem {
	read := row.ReadAt != nil
	return dto.UserNotificationItem{
		ID:        row.ID.String(),
		Category:  row.Category,
		Title:     row.Title,
		Body:      row.Body,
		LinkPath:  row.LinkPath,
		Read:      read,
		CreatedAt: row.CreatedAt,
	}
}
