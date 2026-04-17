// Package cache 提供统一的缓存管理能力
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// CacheManager 缓存管理器，提供统一的缓存读写接口
type CacheManager struct {
	rdb              *redis.Client
	logger           *zap.Logger
	enabled          bool
	enabledMu        sync.RWMutex
	stats            *CacheStats
	defaultTTL       time.Duration
	ttlConfig        TTLConfig
	hitRateThreshold float64
}

// NewCacheManager 创建缓存管理器实例
func NewCacheManager(rdb *redis.Client, logger *zap.Logger, cfg Config) *CacheManager {
	if logger == nil {
		logger = zap.NewNop()
	}

	// 应用默认值
	cfg.ApplyDefaults()

	return &CacheManager{
		rdb:              rdb,
		logger:           logger,
		enabled:          cfg.Enabled,
		stats:            &CacheStats{},
		defaultTTL:       cfg.DefaultTTL,
		ttlConfig:        cfg.TTL,
		hitRateThreshold: cfg.HitRateThreshold,
	}
}

// Get 从缓存获取数据，返回是否命中
// dest 必须是指针类型，用于接收反序列化后的数据
func (m *CacheManager) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	if !m.IsEnabled() {
		return false, nil
	}

	if m.rdb == nil {
		return false, nil
	}

	data, err := m.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			m.stats.IncrMiss()
			m.logger.Debug("缓存未命中",
				zap.String("operation", "Get"),
				zap.String("key", key),
			)
			return false, nil
		}
		m.stats.IncrError()
		m.logger.Warn("缓存读取失败",
			zap.String("operation", "Get"),
			zap.String("key", key),
			zap.Error(err),
		)
		return false, err
	}

	if err := json.Unmarshal(data, dest); err != nil {
		m.stats.IncrError()
		m.logger.Error("缓存反序列化失败，删除损坏缓存",
			zap.String("operation", "Get"),
			zap.String("key", key),
			zap.Error(err),
		)
		// 删除损坏的缓存
		_ = m.rdb.Del(ctx, key)
		return false, err
	}

	m.stats.IncrHit()
	m.logger.Debug("缓存命中",
		zap.String("operation", "Get"),
		zap.String("key", key),
	)
	return true, nil
}

// Set 写入缓存，支持自定义 TTL
// 如果 ttl 为 0，则使用默认 TTL
func (m *CacheManager) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if !m.IsEnabled() {
		return nil
	}

	if m.rdb == nil {
		return nil
	}

	if ttl == 0 {
		ttl = m.defaultTTL
	}

	data, err := json.Marshal(value)
	if err != nil {
		m.stats.IncrError()
		m.logger.Error("缓存序列化失败",
			zap.String("operation", "Set"),
			zap.String("key", key),
			zap.Error(err),
		)
		return err
	}

	if err := m.rdb.Set(ctx, key, data, ttl).Err(); err != nil {
		m.stats.IncrError()
		m.logger.Warn("缓存写入失败",
			zap.String("operation", "Set"),
			zap.String("key", key),
			zap.Duration("ttl", ttl),
			zap.Error(err),
		)
		return err
	}

	m.logger.Debug("缓存写入成功",
		zap.String("operation", "Set"),
		zap.String("key", key),
		zap.Duration("ttl", ttl),
	)
	return nil
}

// Delete 删除指定缓存键
func (m *CacheManager) Delete(ctx context.Context, key string) error {
	if !m.IsEnabled() {
		return nil
	}

	if m.rdb == nil {
		return nil
	}

	if err := m.rdb.Del(ctx, key).Err(); err != nil {
		m.stats.IncrError()
		m.logger.Warn("缓存删除失败",
			zap.String("operation", "Delete"),
			zap.String("key", key),
			zap.Error(err),
		)
		return err
	}

	m.logger.Debug("缓存删除成功",
		zap.String("operation", "Delete"),
		zap.String("key", key),
	)
	return nil
}

// DeleteByPrefix 按前缀批量删除缓存键
// 使用 SCAN 命令避免阻塞 Redis
func (m *CacheManager) DeleteByPrefix(ctx context.Context, prefix string) error {
	if !m.IsEnabled() {
		return nil
	}

	if m.rdb == nil {
		return nil
	}

	pattern := prefix + "*"
	var cursor uint64
	var deletedCount int64

	for {
		keys, nextCursor, err := m.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			m.stats.IncrError()
			m.logger.Warn("缓存扫描失败",
				zap.String("operation", "DeleteByPrefix"),
				zap.String("prefix", prefix),
				zap.Error(err),
			)
			return err
		}

		if len(keys) > 0 {
			if err := m.rdb.Del(ctx, keys...).Err(); err != nil {
				m.stats.IncrError()
				m.logger.Warn("缓存批量删除失败",
					zap.String("operation", "DeleteByPrefix"),
					zap.String("prefix", prefix),
					zap.Int("keyCount", len(keys)),
					zap.Error(err),
				)
				return err
			}
			deletedCount += int64(len(keys))
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	m.logger.Debug("缓存按前缀删除完成",
		zap.String("operation", "DeleteByPrefix"),
		zap.String("prefix", prefix),
		zap.Int64("deletedCount", deletedCount),
	)
	return nil
}

// Exists 检查缓存键是否存在
func (m *CacheManager) Exists(ctx context.Context, key string) (bool, error) {
	if !m.IsEnabled() {
		return false, nil
	}

	if m.rdb == nil {
		return false, nil
	}

	count, err := m.rdb.Exists(ctx, key).Result()
	if err != nil {
		m.stats.IncrError()
		m.logger.Warn("缓存存在性检查失败",
			zap.String("operation", "Exists"),
			zap.String("key", key),
			zap.Error(err),
		)
		return false, err
	}

	return count > 0, nil
}

// GetStats 获取缓存统计信息
func (m *CacheManager) GetStats() CacheStatsSnapshot {
	snapshot := m.stats.GetSnapshot()

	// 尝试获取 Redis 中的键数量
	if m.IsEnabled() && m.rdb != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if dbSize, err := m.rdb.DBSize(ctx).Result(); err == nil {
			snapshot.KeyCount = dbSize
		}
	}

	// 命中率告警：当有足够的操作样本且命中率低于阈值时记录警告
	if total := snapshot.Hits + snapshot.Misses; total > 100 && snapshot.HitRate < m.hitRateThreshold {
		m.logger.Warn("缓存命中率低于阈值",
			zap.Float64("hitRate", snapshot.HitRate),
			zap.Float64("threshold", m.hitRateThreshold),
			zap.Int64("hits", snapshot.Hits),
			zap.Int64("misses", snapshot.Misses),
		)
	}

	return snapshot
}

// IsEnabled 检查缓存是否启用
func (m *CacheManager) IsEnabled() bool {
	m.enabledMu.RLock()
	defer m.enabledMu.RUnlock()
	return m.enabled
}

// SetEnabled 设置缓存启用状态
func (m *CacheManager) SetEnabled(enabled bool) {
	m.enabledMu.Lock()
	defer m.enabledMu.Unlock()
	m.enabled = enabled
	m.logger.Info("缓存状态变更",
		zap.Bool("enabled", enabled),
	)
}

// GetWithFallback 带降级的缓存获取
// 如果缓存命中，直接返回缓存数据
// 如果缓存未命中或禁用，执行 fallback 函数获取数据，并尝试写入缓存
// dest 必须是指针类型
func (m *CacheManager) GetWithFallback(ctx context.Context, key string, dest interface{}, fallback func() (interface{}, error)) error {
	return m.GetWithFallbackTTL(ctx, key, dest, m.defaultTTL, fallback)
}

// GetWithFallbackTTL 带降级和自定义 TTL 的缓存获取
func (m *CacheManager) GetWithFallbackTTL(ctx context.Context, key string, dest interface{}, ttl time.Duration, fallback func() (interface{}, error)) error {
	// 1. 尝试从缓存获取
	if m.IsEnabled() {
		hit, err := m.Get(ctx, key, dest)
		if err != nil {
			m.logger.Warn("缓存读取失败，降级为直接查询",
				zap.String("operation", "GetWithFallback"),
				zap.String("key", key),
				zap.Error(err),
			)
		} else if hit {
			return nil
		}
	}

	// 2. 缓存未命中或禁用，执行回源查询
	data, err := fallback()
	if err != nil {
		return err
	}

	// 3. 尝试写入缓存（失败不影响返回）
	if m.IsEnabled() {
		if setErr := m.Set(ctx, key, data, ttl); setErr != nil {
			m.logger.Warn("缓存写入失败",
				zap.String("operation", "GetWithFallback"),
				zap.String("key", key),
				zap.Error(setErr),
			)
		}
	}

	// 4. 将数据复制到目标
	return copyTo(data, dest)
}

// copyTo 将 src 数据复制到 dest 指针
// 通过 JSON 序列化/反序列化实现深拷贝
func copyTo(src interface{}, dest interface{}) error {
	if src == nil {
		return nil
	}

	// 检查 dest 是否为指针
	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Ptr {
		return errors.New("dest must be a pointer")
	}

	// 如果 src 和 dest 类型相同且 src 是指针，直接赋值
	srcVal := reflect.ValueOf(src)
	if srcVal.Type() == destVal.Type() {
		destVal.Elem().Set(srcVal.Elem())
		return nil
	}

	// 如果 src 是指针，获取其指向的值
	if srcVal.Kind() == reflect.Ptr {
		srcVal = srcVal.Elem()
	}

	// 如果类型兼容，直接赋值
	destElem := destVal.Elem()
	if srcVal.Type().AssignableTo(destElem.Type()) {
		destElem.Set(srcVal)
		return nil
	}

	// 否则通过 JSON 进行转换
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

// GetDefaultTTL 获取默认 TTL
func (m *CacheManager) GetDefaultTTL() time.Duration {
	return m.defaultTTL
}

// GetTTLConfig 获取 TTL 配置
func (m *CacheManager) GetTTLConfig() TTLConfig {
	return m.ttlConfig
}

// GetHitRateThreshold 获取命中率告警阈值
func (m *CacheManager) GetHitRateThreshold() float64 {
	return m.hitRateThreshold
}
