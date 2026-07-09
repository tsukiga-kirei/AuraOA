package service

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"auraoa/go-service/internal/pkg/errcode"
	"auraoa/go-service/internal/repository"
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

// ListProcesses 分页查询租户 AI 调用流程列表。
func (s *LLMMessageLogService) ListProcesses(c *gin.Context, filter repository.LLMLogFilter, page, pageSize int) ([]repository.LLMProcessListRow, int64, error) {
	items, total, err := s.logRepo.ListProcessesPaged(c, filter, page, pageSize)
	if err != nil {
		return nil, 0, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	return items, total, nil
}

// GetProcessChain 获取指定流程的全部 AI 调用记录（时间倒序）。
func (s *LLMMessageLogService) GetProcessChain(c *gin.Context, processID string) ([]repository.LLMLogDetailWithPayload, error) {
	if strings.TrimSpace(processID) == "" {
		return nil, newServiceError(errcode.ErrParamValidation, "流程ID不能为空")
	}
	items, err := s.logRepo.ListCallsByProcessID(c, processID)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	return items, nil
}

// GetLogStats 获取租户 AI 调用记录统计。
func (s *LLMMessageLogService) GetLogStats(c *gin.Context) (*repository.LLMLogStats, error) {
	stats, err := s.logRepo.CountStats(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	return stats, nil
}

// GetLogDetail 获取单条 AI 调用记录详情（含提示词）。
func (s *LLMMessageLogService) GetLogDetail(c *gin.Context, id uuid.UUID) (*repository.LLMLogDetailWithPayload, error) {
	detail, err := s.logRepo.GetByIDWithPayload(c, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, newServiceError(errcode.ErrResourceNotFound, "AI 调用记录不存在")
		}
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	return detail, nil
}
