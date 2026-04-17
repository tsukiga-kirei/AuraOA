// Package cache 提供统一的缓存管理能力
package cache

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// InvalidationManager 缓存失效管理器
// 负责在数据变更时清除相关缓存，确保数据一致性
type InvalidationManager struct {
	cache  *CacheManager
	logger *zap.Logger
}

// NewInvalidationManager 创建失效管理器
func NewInvalidationManager(cache *CacheManager, logger *zap.Logger) *InvalidationManager {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &InvalidationManager{
		cache:  cache,
		logger: logger,
	}
}

// InvalidateUserTodoCache 清除用户待办列表缓存
// 当用户执行审核操作后调用，清除该用户的所有待办列表缓存
// Validates: Requirements 2.5, 4.4, 6.2
func (m *InvalidationManager) InvalidateUserTodoCache(ctx context.Context, tenantID, userID uuid.UUID) error {
	keyBuilder := NewKeyBuilder("audit", tenantID)
	prefix := keyBuilder.TodoListPrefix(userID)

	m.logger.Info("清除用户待办列表缓存",
		zap.String("tenantID", tenantID.String()),
		zap.String("userID", userID.String()),
		zap.String("prefix", prefix),
	)

	if err := m.cache.DeleteByPrefix(ctx, prefix); err != nil {
		m.logger.Error("清除用户待办列表缓存失败",
			zap.String("tenantID", tenantID.String()),
			zap.String("userID", userID.String()),
			zap.Error(err),
		)
		return err
	}

	return nil
}

// InvalidateUserArchiveCache 清除用户归档列表缓存
// 当用户执行复盘操作后调用，清除该用户的所有归档列表缓存
// Validates: Requirements 3.5, 4.5, 6.3
func (m *InvalidationManager) InvalidateUserArchiveCache(ctx context.Context, tenantID, userID uuid.UUID) error {
	keyBuilder := NewKeyBuilder("archive", tenantID)
	prefix := keyBuilder.ArchiveListPrefix(userID)

	m.logger.Info("清除用户归档列表缓存",
		zap.String("tenantID", tenantID.String()),
		zap.String("userID", userID.String()),
		zap.String("prefix", prefix),
	)

	if err := m.cache.DeleteByPrefix(ctx, prefix); err != nil {
		m.logger.Error("清除用户归档列表缓存失败",
			zap.String("tenantID", tenantID.String()),
			zap.String("userID", userID.String()),
			zap.Error(err),
		)
		return err
	}

	return nil
}

// InvalidateSnapshotCache 清除快照缓存
// 当审核或复盘任务完成并更新快照时调用
// module: "audit" 或 "archive"
// Validates: Requirements 4.4, 4.5
func (m *InvalidationManager) InvalidateSnapshotCache(ctx context.Context, tenantID uuid.UUID, module string) error {
	keyBuilder := NewKeyBuilder(module, tenantID)
	prefix := keyBuilder.SnapshotPrefix()

	m.logger.Info("清除快照缓存",
		zap.String("tenantID", tenantID.String()),
		zap.String("module", module),
		zap.String("prefix", prefix),
	)

	if err := m.cache.DeleteByPrefix(ctx, prefix); err != nil {
		m.logger.Error("清除快照缓存失败",
			zap.String("tenantID", tenantID.String()),
			zap.String("module", module),
			zap.Error(err),
		)
		return err
	}

	return nil
}

// InvalidateConfigCache 清除配置缓存
// 当审核规则或归档规则变更时调用
// module: "audit" 或 "archive"
// Validates: Requirements 6.1, 6.2, 6.3
func (m *InvalidationManager) InvalidateConfigCache(ctx context.Context, tenantID uuid.UUID, module string) error {
	keyBuilder := NewKeyBuilder(module, tenantID)
	prefix := keyBuilder.ConfigPrefix()

	m.logger.Info("清除配置缓存",
		zap.String("tenantID", tenantID.String()),
		zap.String("module", module),
		zap.String("prefix", prefix),
	)

	if err := m.cache.DeleteByPrefix(ctx, prefix); err != nil {
		m.logger.Error("清除配置缓存失败",
			zap.String("tenantID", tenantID.String()),
			zap.String("module", module),
			zap.Error(err),
		)
		return err
	}

	return nil
}

// InvalidateStatsCache 清除统计缓存
// 当审核或复盘任务状态变更时调用
// module: "audit" 或 "archive"
// Validates: Requirements 5.4
func (m *InvalidationManager) InvalidateStatsCache(ctx context.Context, tenantID uuid.UUID, module string) error {
	keyBuilder := NewKeyBuilder(module, tenantID)
	prefix := keyBuilder.StatsPrefix()

	m.logger.Info("清除统计缓存",
		zap.String("tenantID", tenantID.String()),
		zap.String("module", module),
		zap.String("prefix", prefix),
	)

	if err := m.cache.DeleteByPrefix(ctx, prefix); err != nil {
		m.logger.Error("清除统计缓存失败",
			zap.String("tenantID", tenantID.String()),
			zap.String("module", module),
			zap.Error(err),
		)
		return err
	}

	return nil
}

// InvalidateTenantCache 清除租户全部缓存
// 当租户配置变更或 OA 连接配置变更时调用
// 清除该租户的所有配置相关缓存和 OA 数据相关缓存
// Validates: Requirements 6.1, 6.4, 6.6
func (m *InvalidationManager) InvalidateTenantCache(ctx context.Context, tenantID uuid.UUID) error {
	keyBuilder := NewKeyBuilder("", tenantID)
	prefixes := keyBuilder.TenantPrefixes()

	m.logger.Info("清除租户全部缓存",
		zap.String("tenantID", tenantID.String()),
		zap.Int("prefixCount", len(prefixes)),
	)

	var lastErr error
	for _, prefix := range prefixes {
		if err := m.cache.DeleteByPrefix(ctx, prefix); err != nil {
			m.logger.Error("清除租户缓存前缀失败",
				zap.String("tenantID", tenantID.String()),
				zap.String("prefix", prefix),
				zap.Error(err),
			)
			lastErr = err
			// 继续删除其他前缀，不中断
		}
	}

	if lastErr != nil {
		return lastErr
	}

	m.logger.Info("租户全部缓存清除完成",
		zap.String("tenantID", tenantID.String()),
	)
	return nil
}

// InvalidateModuleCache 清除指定模块全部缓存
// 用于管理接口手动清除指定模块的缓存
// module: "audit", "archive", 或 "dashboard"
// Validates: Requirements 6.5
func (m *InvalidationManager) InvalidateModuleCache(ctx context.Context, module string) error {
	prefix := ModulePrefix(module)

	m.logger.Info("清除模块全部缓存",
		zap.String("module", module),
		zap.String("prefix", prefix),
	)

	if err := m.cache.DeleteByPrefix(ctx, prefix); err != nil {
		m.logger.Error("清除模块缓存失败",
			zap.String("module", module),
			zap.Error(err),
		)
		return err
	}

	m.logger.Info("模块全部缓存清除完成",
		zap.String("module", module),
	)
	return nil
}

// InvalidateDashboardCache 清除仪表盘缓存
// 当统计数据变更时调用
func (m *InvalidationManager) InvalidateDashboardCache(ctx context.Context, tenantID uuid.UUID) error {
	keyBuilder := NewKeyBuilder("dashboard", tenantID)
	prefix := keyBuilder.DashboardPrefix()

	m.logger.Info("清除仪表盘缓存",
		zap.String("tenantID", tenantID.String()),
		zap.String("prefix", prefix),
	)

	if err := m.cache.DeleteByPrefix(ctx, prefix); err != nil {
		m.logger.Error("清除仪表盘缓存失败",
			zap.String("tenantID", tenantID.String()),
			zap.Error(err),
		)
		return err
	}

	return nil
}

// InvalidateAllUserCaches 清除用户的所有缓存（待办+归档+仪表盘）
// 便捷方法，用于需要清除用户所有相关缓存的场景
func (m *InvalidationManager) InvalidateAllUserCaches(ctx context.Context, tenantID, userID uuid.UUID) error {
	var lastErr error

	if err := m.InvalidateUserTodoCache(ctx, tenantID, userID); err != nil {
		lastErr = err
	}

	if err := m.InvalidateUserArchiveCache(ctx, tenantID, userID); err != nil {
		lastErr = err
	}

	if err := m.InvalidateDashboardCache(ctx, tenantID); err != nil {
		lastErr = err
	}

	return lastErr
}

// InvalidateAuditRelatedCaches 清除审核相关的所有缓存
// 便捷方法，在审核操作完成后调用
// 清除：用户待办列表、审核快照、审核统计、仪表盘
func (m *InvalidationManager) InvalidateAuditRelatedCaches(ctx context.Context, tenantID, userID uuid.UUID) error {
	var lastErr error

	if err := m.InvalidateUserTodoCache(ctx, tenantID, userID); err != nil {
		lastErr = err
	}

	if err := m.InvalidateSnapshotCache(ctx, tenantID, "audit"); err != nil {
		lastErr = err
	}

	if err := m.InvalidateStatsCache(ctx, tenantID, "audit"); err != nil {
		lastErr = err
	}

	if err := m.InvalidateDashboardCache(ctx, tenantID); err != nil {
		lastErr = err
	}

	return lastErr
}

// InvalidateArchiveRelatedCaches 清除归档相关的所有缓存
// 便捷方法，在复盘操作完成后调用
// 清除：用户归档列表、归档快照、归档统计、仪表盘
func (m *InvalidationManager) InvalidateArchiveRelatedCaches(ctx context.Context, tenantID, userID uuid.UUID) error {
	var lastErr error

	if err := m.InvalidateUserArchiveCache(ctx, tenantID, userID); err != nil {
		lastErr = err
	}

	if err := m.InvalidateSnapshotCache(ctx, tenantID, "archive"); err != nil {
		lastErr = err
	}

	if err := m.InvalidateStatsCache(ctx, tenantID, "archive"); err != nil {
		lastErr = err
	}

	if err := m.InvalidateDashboardCache(ctx, tenantID); err != nil {
		lastErr = err
	}

	return lastErr
}
