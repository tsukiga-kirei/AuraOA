package repository

import (
	"context"

	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

// OperationAuditLogRepo 用户操作审计日志写入。
type OperationAuditLogRepo struct {
	*BaseRepo
}

// NewOperationAuditLogRepo 创建仓储。
func NewOperationAuditLogRepo(db *gorm.DB) *OperationAuditLogRepo {
	return &OperationAuditLogRepo{BaseRepo: NewBaseRepo(db)}
}

// Create 插入一条审计记录。
func (r *OperationAuditLogRepo) Create(ctx context.Context, log *model.OperationAuditLog) error {
	return r.DB.WithContext(ctx).Create(log).Error
}
