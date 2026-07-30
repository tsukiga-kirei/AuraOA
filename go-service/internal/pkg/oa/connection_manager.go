package oa

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"

	"auraoa/go-service/internal/model"
)

const (
	defaultOAPoolSize        = 10
	maxOAPoolSize            = 100
	defaultConnectionTimeout = 30 * time.Second
	minConnectionTimeout     = 5 * time.Second
	maxConnectionTimeout     = 300 * time.Second
	oaConnectionMaxIdleTime  = 5 * time.Minute
	oaConnectionMaxLifetime  = 30 * time.Minute
)

var errConnectionPoolInvalidated = errors.New("OA 连接配置已变更，请重试")

type managedConnectionPool struct {
	db          *gorm.DB
	sqlDB       *sql.DB
	fingerprint [sha256.Size]byte
	driver      string
	poolSize    int
}

type oaPoolOpener func(
	ctx context.Context,
	oaType string,
	conn *model.OADatabaseConnection,
	transient bool,
) (*gorm.DB, *sql.DB, error)

// ConnectionPoolStats 描述单个 OA 共享连接池的运行状态。
type ConnectionPoolStats struct {
	ConnectionID uuid.UUID
	Driver       string
	MaxOpen      int
	Open         int
	InUse        int
	Idle         int
	WaitCount    int64
	WaitDuration time.Duration
}

// ConnectionManager 按 OA 连接配置 ID 复用底层数据库连接池。
// 适配器本身保持轻量，每次获取时可注入不同的附件识别服务；MySQL、Oracle、达梦
// 共用同一套连接池创建、配置变更失效和进程退出关闭逻辑。
type ConnectionManager struct {
	mu          sync.RWMutex
	pools       map[uuid.UUID]*managedConnectionPool
	generations map[uuid.UUID]uint64
	open        oaPoolOpener
	group       singleflight.Group
	logger      *zap.Logger
	closed      bool
}

// NewConnectionManager 创建 OA 数据库共享连接池管理器。
func NewConnectionManager(logger *zap.Logger) *ConnectionManager {
	return newConnectionManager(logger, openOADatabase)
}

func newConnectionManager(logger *zap.Logger, opener oaPoolOpener) *ConnectionManager {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ConnectionManager{
		pools:       make(map[uuid.UUID]*managedConnectionPool),
		generations: make(map[uuid.UUID]uint64),
		open:        opener,
		logger:      logger,
	}
}

// GetAdapter 获取共享连接池上的轻量 OA 适配器。
// 当连接参数或连接池配置发生变化时，会按配置指纹自动替换旧连接池。
func (m *ConnectionManager) GetAdapter(
	ctx context.Context,
	oaType string,
	conn *model.OADatabaseConnection,
	attachmentSvc ...AttachmentRecognitionService,
) (OAAdapter, error) {
	if m == nil {
		return nil, fmt.Errorf("OA 连接池管理器未初始化")
	}
	if m.isClosed() {
		return nil, fmt.Errorf("OA 连接池管理器已关闭")
	}
	if err := validateOAAdapterConfig(oaType, conn); err != nil {
		return nil, err
	}
	if !conn.Enabled {
		m.Invalidate(conn.ID)
		return nil, fmt.Errorf("OA 数据库连接已停用")
	}
	if conn.ID == uuid.Nil {
		return nil, fmt.Errorf("共享 OA 连接缺少配置 ID")
	}

	pool, err := m.getOrOpenPool(ctx, oaType, conn)
	if err != nil {
		return nil, err
	}
	return newOAAdapterWithDB(oaType, conn, pool.db, attachmentSvc...)
}

// OpenTransientAdapter 创建不进入共享缓存的短生命周期适配器。
// 连接测试必须使用此方法，并在完成后调用返回的 closeFn。
func (m *ConnectionManager) OpenTransientAdapter(
	ctx context.Context,
	oaType string,
	conn *model.OADatabaseConnection,
	attachmentSvc ...AttachmentRecognitionService,
) (adapter OAAdapter, closeFn func() error, err error) {
	if m == nil {
		return nil, nil, fmt.Errorf("OA 连接池管理器未初始化")
	}
	if m.isClosed() {
		return nil, nil, fmt.Errorf("OA 连接池管理器已关闭")
	}
	if err := validateOAAdapterConfig(oaType, conn); err != nil {
		return nil, nil, err
	}

	db, sqlDB, err := m.open(ctx, oaType, conn, true)
	if err != nil {
		return nil, nil, err
	}
	adapter, err = newOAAdapterWithDB(oaType, conn, db, attachmentSvc...)
	if err != nil {
		_ = sqlDB.Close()
		return nil, nil, err
	}
	return adapter, sqlDB.Close, nil
}

func (m *ConnectionManager) getOrOpenPool(
	ctx context.Context,
	oaType string,
	conn *model.OADatabaseConnection,
) (*managedConnectionPool, error) {
	fingerprint := connectionFingerprint(oaType, conn)
	for {
		if m.isClosed() {
			return nil, fmt.Errorf("OA 连接池管理器已关闭")
		}
		if pool := m.findMatchingPool(conn.ID, fingerprint); pool != nil {
			return pool, nil
		}
		generation := m.currentGeneration(conn.ID)

		value, err, _ := m.group.Do(conn.ID.String(), func() (interface{}, error) {
			if pool := m.findMatchingPool(conn.ID, fingerprint); pool != nil {
				return pool, nil
			}

			db, sqlDB, err := m.open(ctx, oaType, conn, false)
			if err != nil {
				return nil, err
			}
			pool := &managedConnectionPool{
				db:          db,
				sqlDB:       sqlDB,
				fingerprint: fingerprint,
				driver:      conn.Driver,
				poolSize:    normalizePoolSize(conn.PoolSize),
			}

			m.mu.Lock()
			if m.closed {
				m.mu.Unlock()
				_ = sqlDB.Close()
				return nil, fmt.Errorf("OA 连接池管理器已关闭")
			}
			if m.generations[conn.ID] != generation {
				m.mu.Unlock()
				_ = sqlDB.Close()
				return nil, errConnectionPoolInvalidated
			}
			old := m.pools[conn.ID]
			m.pools[conn.ID] = pool
			m.mu.Unlock()

			if old != nil {
				m.closePool(conn.ID, old, "连接配置已变更")
			}
			m.logger.Info("OA 数据库共享连接池已建立",
				zap.String("connectionID", conn.ID.String()),
				zap.String("driver", conn.Driver),
				zap.Int("maxOpenConns", pool.poolSize),
			)
			return pool, nil
		})
		if err != nil {
			return nil, err
		}

		pool := value.(*managedConnectionPool)
		if pool.fingerprint == fingerprint {
			return pool, nil
		}
		// 配置更新与首次建池并发时，等待当前建池结束后按最新指纹重试。
		m.group.Forget(conn.ID.String())
	}
}

func (m *ConnectionManager) findMatchingPool(
	connectionID uuid.UUID,
	fingerprint [sha256.Size]byte,
) *managedConnectionPool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	pool := m.pools[connectionID]
	if m.closed || pool == nil || pool.fingerprint != fingerprint {
		return nil
	}
	return pool
}

func (m *ConnectionManager) isClosed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.closed
}

func (m *ConnectionManager) currentGeneration(connectionID uuid.UUID) uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.generations[connectionID]
}

// Invalidate 关闭并移除指定 OA 配置对应的共享连接池。
func (m *ConnectionManager) Invalidate(connectionID uuid.UUID) {
	if m == nil || connectionID == uuid.Nil {
		return
	}
	m.group.Forget(connectionID.String())
	m.mu.Lock()
	pool := m.pools[connectionID]
	delete(m.pools, connectionID)
	m.generations[connectionID]++
	m.mu.Unlock()
	if pool != nil {
		m.closePool(connectionID, pool, "连接配置已失效")
	}
}

// Stats 返回当前进程内全部 OA 共享连接池的只读统计快照。
func (m *ConnectionManager) Stats() []ConnectionPoolStats {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]ConnectionPoolStats, 0, len(m.pools))
	for id, pool := range m.pools {
		stats := pool.sqlDB.Stats()
		result = append(result, ConnectionPoolStats{
			ConnectionID: id,
			Driver:       pool.driver,
			MaxOpen:      stats.MaxOpenConnections,
			Open:         stats.OpenConnections,
			InUse:        stats.InUse,
			Idle:         stats.Idle,
			WaitCount:    stats.WaitCount,
			WaitDuration: stats.WaitDuration,
		})
	}
	return result
}

// Close 关闭当前进程内全部 OA 共享连接池。
func (m *ConnectionManager) Close() error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	pools := m.pools
	m.pools = make(map[uuid.UUID]*managedConnectionPool)
	m.closed = true
	m.mu.Unlock()

	var errs []error
	for id, pool := range pools {
		stats := pool.sqlDB.Stats()
		if err := pool.sqlDB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("关闭 OA 连接池 %s 失败: %w", id, err))
			continue
		}
		m.logger.Info("OA 数据库共享连接池已关闭",
			zap.String("connectionID", id.String()),
			zap.String("driver", pool.driver),
			zap.Int("openConnections", stats.OpenConnections),
			zap.Int("inUseConnections", stats.InUse),
			zap.Int("idleConnections", stats.Idle),
		)
	}
	return errors.Join(errs...)
}

func (m *ConnectionManager) closePool(
	connectionID uuid.UUID,
	pool *managedConnectionPool,
	reason string,
) {
	stats := pool.sqlDB.Stats()
	if err := pool.sqlDB.Close(); err != nil {
		m.logger.Warn("关闭 OA 数据库共享连接池失败",
			zap.String("connectionID", connectionID.String()),
			zap.String("driver", pool.driver),
			zap.String("reason", reason),
			zap.Error(err),
		)
		return
	}
	m.logger.Info("OA 数据库共享连接池已释放",
		zap.String("connectionID", connectionID.String()),
		zap.String("driver", pool.driver),
		zap.String("reason", reason),
		zap.Int("openConnections", stats.OpenConnections),
		zap.Int("inUseConnections", stats.InUse),
		zap.Int("idleConnections", stats.Idle),
	)
}

func connectionFingerprint(oaType string, conn *model.OADatabaseConnection) [sha256.Size]byte {
	parts := []string{
		oaType,
		conn.Driver,
		conn.Host,
		strconv.Itoa(conn.Port),
		conn.DatabaseName,
		conn.Username,
		conn.Password,
		strconv.Itoa(normalizePoolSize(conn.PoolSize)),
		normalizeConnectionTimeout(conn.ConnectionTimeout).String(),
	}
	return sha256.Sum256([]byte(strings.Join(parts, "\x00")))
}

func normalizePoolSize(poolSize int) int {
	if poolSize <= 0 {
		return defaultOAPoolSize
	}
	if poolSize > maxOAPoolSize {
		return maxOAPoolSize
	}
	return poolSize
}

func normalizeMaxIdleConns(poolSize int) int {
	maxIdle := poolSize / 2
	if maxIdle < 1 {
		return 1
	}
	return maxIdle
}

func normalizeConnectionTimeout(seconds int) time.Duration {
	if seconds <= 0 {
		return defaultConnectionTimeout
	}
	timeout := time.Duration(seconds) * time.Second
	if timeout < minConnectionTimeout {
		return minConnectionTimeout
	}
	if timeout > maxConnectionTimeout {
		return maxConnectionTimeout
	}
	return timeout
}
