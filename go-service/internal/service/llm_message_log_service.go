package service

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/repository"
)

// LLMMessageLogService 处理大模型消息记录的查询业务逻辑。
type LLMMessageLogService struct {
	logRepo *repository.LLMMessageLogRepo
}

// NewLLMMessageLogService 创建一个新的 LLMMessageLogService 实例。
func NewLLMMessageLogService(logRepo *repository.LLMMessageLogRepo) *LLMMessageLogService {
	return &LLMMessageLogService{logRepo: logRepo}
}

// QueryTokenUsage 按租户查询 Token 消耗统计。
func (s *LLMMessageLogService) QueryTokenUsage(c *gin.Context, startTime, endTime time.Time, modelConfigID *uuid.UUID) ([]repository.TokenUsageSummary, error) {
	summaries, err := s.logRepo.QueryByTimeRange(c, startTime, endTime, modelConfigID)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	return summaries, nil
}

// QueryAllTenantsTokenUsage 查询所有租户的 Token 消耗统计（system_admin 用）。
func (s *LLMMessageLogService) QueryAllTenantsTokenUsage(startTime, endTime time.Time) ([]repository.TokenUsageSummary, error) {
	summaries, err := s.logRepo.QueryAllTenantsTokenUsage(startTime, endTime)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	return summaries, nil
}
