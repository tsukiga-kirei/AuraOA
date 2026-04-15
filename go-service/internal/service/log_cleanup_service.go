// Package service 包含应用程序的核心业务逻辑层。
package service

import (
	"context"
	"errors"
	"strconv"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/pkg/logger"
	"oa-smart-audit/go-service/internal/repository"
)

// LogCleanupService 日志清理服务，负责定期清理超过保留期限的日志备份文件。
// 全局保留天数优先从 system_configs 表读取，不存在时使用 config.yaml 中的兜底值。
// 各租户可通过自身的 LogRetentionDays 字段配置独立的保留天数。
type LogCleanupService struct {
	// systemConfigRepo 用于读取系统级配置项（如全局日志保留天数）
	systemConfigRepo *repository.SystemConfigRepo
	// tenantRepo 用于获取所有租户列表及其日志保留天数配置
	tenantRepo *repository.TenantRepo
	// fallbackRetentionDays 当 system_configs 中无对应配置时使用的兜底保留天数
	// 该值来自 config.yaml 的 log.global_retention_days 字段
	fallbackRetentionDays int
}

// NewLogCleanupService 创建并返回一个新的 LogCleanupService 实例。
// fallbackRetentionDays 为 config.yaml 中配置的兜底保留天数，
// 当数据库中未配置 system.global_log_retention_days 时生效。
func NewLogCleanupService(
	systemConfigRepo *repository.SystemConfigRepo,
	tenantRepo *repository.TenantRepo,
	fallbackRetentionDays int,
) *LogCleanupService {
	return &LogCleanupService{
		systemConfigRepo:      systemConfigRepo,
		tenantRepo:            tenantRepo,
		fallbackRetentionDays: fallbackRetentionDays,
	}
}

// RunCleanup 执行一次完整的日志清理流程：
//  1. 从 system_configs 读取 system.global_log_retention_days 作为全局保留天数
//  2. 若配置项不存在，则使用构造时传入的 fallbackRetentionDays
//  3. 调用 logger.CleanupGlobalLogs 清理全局日志备份文件
//  4. 获取所有租户列表，构建各租户的保留天数映射
//  5. 调用 logger.CleanupTenantLogs 清理各租户日志备份文件
//  6. 将清理结果（删除文件数、释放字节数）写入全局日志
func (s *LogCleanupService) RunCleanup(ctx context.Context) error {
	// 第一步：从 system_configs 读取全局日志保留天数
	retentionDays := s.fallbackRetentionDays
	valStr, err := s.systemConfigRepo.FindByKey("system.global_log_retention_days")
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 配置项不存在，使用 config.yaml 中的兜底值，不视为错误
			logger.Global().Info("system_configs 中未找到 system.global_log_retention_days，使用兜底值",
				zap.Int("fallbackRetentionDays", retentionDays),
			)
		} else {
			// 其他数据库错误，记录警告后继续使用兜底值
			logger.Global().Warn("读取 system.global_log_retention_days 失败，使用兜底值",
				zap.Int("fallbackRetentionDays", retentionDays),
				zap.Error(err),
			)
		}
	} else {
		// 配置项存在，将字符串值转换为整数
		parsed, parseErr := strconv.Atoi(valStr)
		if parseErr != nil {
			// 配置值格式非法，记录警告后继续使用兜底值
			logger.Global().Warn("system.global_log_retention_days 配置值格式非法，使用兜底值",
				zap.String("value", valStr),
				zap.Int("fallbackRetentionDays", retentionDays),
				zap.Error(parseErr),
			)
		} else {
			retentionDays = parsed
		}
	}

	// 第二步：清理全局日志备份文件（0 表示不保留备份，立即清理所有轮转文件）
	globalDeleted, globalFreed, globalErr := logger.CleanupGlobalLogs(retentionDays)
	if globalErr != nil {
		logger.Global().Warn("清理全局日志备份文件时发生错误",
			zap.Int("retentionDays", retentionDays),
			zap.Error(globalErr),
		)
	}

	// 第三步：获取所有租户列表
	tenants, err := s.tenantRepo.List()
	if err != nil {
		logger.Global().Error("获取租户列表失败，跳过租户日志清理",
			zap.Error(err),
		)
		// 全局日志已清理，记录结果后返回错误
		logger.Global().Info("日志清理任务完成（仅全局）",
			zap.Int("globalDeletedCount", globalDeleted),
			zap.Int64("globalFreedBytes", globalFreed),
		)
		return err
	}

	// 第四步：构建租户保留天数映射
	// 若租户的 LogRetentionDays 为 0，表示不保留备份，轮转后立即清理
	retentionMap := make(map[string]int, len(tenants))
	for _, t := range tenants {
		retentionMap[t.Code] = t.LogRetentionDays
	}

	// 第五步：清理各租户日志备份文件
	tenantDeleted, tenantFreed, tenantErr := logger.CleanupTenantLogs(retentionMap)
	if tenantErr != nil {
		logger.Global().Warn("清理租户日志备份文件时发生错误",
			zap.Error(tenantErr),
		)
	}

	// 第六步：记录本次清理的汇总结果
	totalDeleted := globalDeleted + tenantDeleted
	totalFreed := globalFreed + tenantFreed
	logger.Global().Info("日志清理任务完成",
		zap.Int("globalRetentionDays", retentionDays),
		zap.Int("globalDeletedCount", globalDeleted),
		zap.Int64("globalFreedBytes", globalFreed),
		zap.Int("tenantCount", len(tenants)),
		zap.Int("tenantDeletedCount", tenantDeleted),
		zap.Int64("tenantFreedBytes", tenantFreed),
		zap.Int("totalDeletedCount", totalDeleted),
		zap.Int64("totalFreedBytes", totalFreed),
	)

	return nil
}
