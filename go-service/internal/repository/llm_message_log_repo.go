package repository

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

// LLMMessageLogRepo 提供租户大模型消息记录的数据访问方法。
type LLMMessageLogRepo struct {
	*BaseRepo
}

// NewLLMMessageLogRepo 创建一个新的 LLMMessageLogRepo 实例。
func NewLLMMessageLogRepo(db *gorm.DB) *LLMMessageLogRepo {
	return &LLMMessageLogRepo{BaseRepo: NewBaseRepo(db)}
}

// Create 写入一条大模型消息记录。
func (r *LLMMessageLogRepo) Create(log *model.TenantLLMMessageLog) error {
	return r.DB.Create(log).Error
}

// TokenUsageSummary Token 消耗统计汇总结构。
type TokenUsageSummary struct {
	TenantID      uuid.UUID `json:"tenant_id"`
	ModelConfigID uuid.UUID `json:"model_config_id"`
	TotalInput    int64     `json:"total_input"`
	TotalOutput   int64     `json:"total_output"`
	TotalTokens   int64     `json:"total_tokens"`
	CallCount     int64     `json:"call_count"`
}

// QueryByTimeRange 按时间范围和可选模型筛选查询 Token 消耗统计。
func (r *LLMMessageLogRepo) QueryByTimeRange(c *gin.Context, startTime, endTime time.Time, modelConfigID *uuid.UUID) ([]TokenUsageSummary, error) {
	query := r.WithTenant(c).Model(&model.TenantLLMMessageLog{}).
		Select("tenant_id, model_config_id, SUM(input_tokens) as total_input, SUM(output_tokens) as total_output, SUM(total_tokens) as total_tokens, COUNT(*) as call_count").
		Where("created_at >= ? AND created_at <= ?", startTime, endTime)

	if modelConfigID != nil {
		query = query.Where("model_config_id = ?", *modelConfigID)
	}

	query = query.Group("tenant_id, model_config_id")

	var summaries []TokenUsageSummary
	if err := query.Find(&summaries).Error; err != nil {
		return nil, err
	}
	return summaries, nil
}

// QueryAllTenantsTokenUsage 查询所有租户的 Token 消耗统计（system_admin 用）。
func (r *LLMMessageLogRepo) QueryAllTenantsTokenUsage(startTime, endTime time.Time) ([]TokenUsageSummary, error) {
	var summaries []TokenUsageSummary
	err := r.DB.Model(&model.TenantLLMMessageLog{}).
		Select("tenant_id, model_config_id, SUM(input_tokens) as total_input, SUM(output_tokens) as total_output, SUM(total_tokens) as total_tokens, COUNT(*) as call_count").
		Where("created_at >= ? AND created_at <= ?", startTime, endTime).
		Group("tenant_id, model_config_id").
		Find(&summaries).Error
	if err != nil {
		return nil, err
	}
	return summaries, nil
}
