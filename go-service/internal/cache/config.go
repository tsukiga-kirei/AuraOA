// Package cache 提供统一的缓存管理能力
package cache

import "time"

// 默认 TTL 常量
const (
	// DefaultTTLAuditTodo 待办列表默认 TTL (3分钟)
	DefaultTTLAuditTodo = 3 * time.Minute

	// DefaultTTLArchiveList 归档列表默认 TTL (5分钟)
	DefaultTTLArchiveList = 5 * time.Minute

	// DefaultTTLProcessConfig 流程配置默认 TTL (10分钟)
	DefaultTTLProcessConfig = 10 * time.Minute

	// DefaultTTLSnapshot 快照数据默认 TTL (5分钟)
	DefaultTTLSnapshot = 5 * time.Minute

	// DefaultTTLStats 统计数据默认 TTL (5分钟)
	DefaultTTLStats = 5 * time.Minute

	// DefaultTTLDashboard 仪表盘默认 TTL (2分钟)
	DefaultTTLDashboard = 2 * time.Minute

	// DefaultTTL 默认 TTL (5分钟)
	DefaultTTL = 5 * time.Minute

	// DefaultHitRateThreshold 默认命中率告警阈值
	DefaultHitRateThreshold = 0.5
)

// Config 缓存配置
type Config struct {
	Enabled          bool          `yaml:"enabled" json:"enabled"`                       // 是否启用缓存
	DefaultTTL       time.Duration `yaml:"default_ttl" json:"default_ttl"`               // 默认 TTL
	HitRateThreshold float64       `yaml:"hit_rate_threshold" json:"hit_rate_threshold"` // 命中率告警阈值
	TTL              TTLConfig     `yaml:"ttl" json:"ttl"`                               // 各模块 TTL 配置
}

// TTLConfig 各模块的 TTL 配置
type TTLConfig struct {
	AuditTodo     time.Duration `yaml:"audit_todo" json:"audit_todo"`         // 待办列表 TTL
	ArchiveList   time.Duration `yaml:"archive_list" json:"archive_list"`     // 归档列表 TTL
	ProcessConfig time.Duration `yaml:"process_config" json:"process_config"` // 流程配置 TTL
	Snapshot      time.Duration `yaml:"snapshot" json:"snapshot"`             // 快照数据 TTL
	Stats         time.Duration `yaml:"stats" json:"stats"`                   // 统计数据 TTL
	Dashboard     time.Duration `yaml:"dashboard" json:"dashboard"`           // 仪表盘 TTL
}

// NewDefaultConfig 创建默认缓存配置
func NewDefaultConfig() Config {
	return Config{
		Enabled:          true,
		DefaultTTL:       DefaultTTL,
		HitRateThreshold: DefaultHitRateThreshold,
		TTL:              NewDefaultTTLConfig(),
	}
}

// NewDefaultTTLConfig 创建默认 TTL 配置
func NewDefaultTTLConfig() TTLConfig {
	return TTLConfig{
		AuditTodo:     DefaultTTLAuditTodo,
		ArchiveList:   DefaultTTLArchiveList,
		ProcessConfig: DefaultTTLProcessConfig,
		Snapshot:      DefaultTTLSnapshot,
		Stats:         DefaultTTLStats,
		Dashboard:     DefaultTTLDashboard,
	}
}

// GetAuditTodoTTL 获取待办列表 TTL，如果未配置则返回默认值
func (c *TTLConfig) GetAuditTodoTTL() time.Duration {
	if c.AuditTodo > 0 {
		return c.AuditTodo
	}
	return DefaultTTLAuditTodo
}

// GetArchiveListTTL 获取归档列表 TTL，如果未配置则返回默认值
func (c *TTLConfig) GetArchiveListTTL() time.Duration {
	if c.ArchiveList > 0 {
		return c.ArchiveList
	}
	return DefaultTTLArchiveList
}

// GetProcessConfigTTL 获取流程配置 TTL，如果未配置则返回默认值
func (c *TTLConfig) GetProcessConfigTTL() time.Duration {
	if c.ProcessConfig > 0 {
		return c.ProcessConfig
	}
	return DefaultTTLProcessConfig
}

// GetSnapshotTTL 获取快照数据 TTL，如果未配置则返回默认值
func (c *TTLConfig) GetSnapshotTTL() time.Duration {
	if c.Snapshot > 0 {
		return c.Snapshot
	}
	return DefaultTTLSnapshot
}

// GetStatsTTL 获取统计数据 TTL，如果未配置则返回默认值
func (c *TTLConfig) GetStatsTTL() time.Duration {
	if c.Stats > 0 {
		return c.Stats
	}
	return DefaultTTLStats
}

// GetDashboardTTL 获取仪表盘 TTL，如果未配置则返回默认值
func (c *TTLConfig) GetDashboardTTL() time.Duration {
	if c.Dashboard > 0 {
		return c.Dashboard
	}
	return DefaultTTLDashboard
}

// Validate 验证配置有效性
func (c *Config) Validate() error {
	if c.DefaultTTL < 0 {
		c.DefaultTTL = DefaultTTL
	}
	if c.HitRateThreshold < 0 || c.HitRateThreshold > 1 {
		c.HitRateThreshold = DefaultHitRateThreshold
	}
	return nil
}

// ApplyDefaults 应用默认值到未配置的字段
func (c *Config) ApplyDefaults() {
	if c.DefaultTTL == 0 {
		c.DefaultTTL = DefaultTTL
	}
	if c.HitRateThreshold == 0 {
		c.HitRateThreshold = DefaultHitRateThreshold
	}
	c.TTL.ApplyDefaults()
}

// ApplyDefaults 应用默认值到未配置的 TTL 字段
func (t *TTLConfig) ApplyDefaults() {
	if t.AuditTodo == 0 {
		t.AuditTodo = DefaultTTLAuditTodo
	}
	if t.ArchiveList == 0 {
		t.ArchiveList = DefaultTTLArchiveList
	}
	if t.ProcessConfig == 0 {
		t.ProcessConfig = DefaultTTLProcessConfig
	}
	if t.Snapshot == 0 {
		t.Snapshot = DefaultTTLSnapshot
	}
	if t.Stats == 0 {
		t.Stats = DefaultTTLStats
	}
	if t.Dashboard == 0 {
		t.Dashboard = DefaultTTLDashboard
	}
}
