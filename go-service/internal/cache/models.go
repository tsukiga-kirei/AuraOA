// Package cache 提供统一的缓存管理能力
package cache

import (
	"time"
)

// CachedTodoList 缓存的待办列表
// 用于存储审核工作台的待办流程列表数据
type CachedTodoList struct {
	Items      []map[string]interface{} `json:"items"`       // 待办项列表
	Total      int                      `json:"total"`       // 总数
	CachedAt   time.Time                `json:"cached_at"`   // 缓存时间戳
	FilterHash string                   `json:"filter_hash"` // 筛选条件哈希
}

// CachedArchiveList 缓存的归档列表
// 用于存储归档复盘模块的已归档流程列表数据
type CachedArchiveList struct {
	Items      []map[string]interface{} `json:"items"`       // 归档项列表
	Total      int                      `json:"total"`       // 总数
	Page       int                      `json:"page"`        // 当前页码
	PageSize   int                      `json:"page_size"`   // 每页大小
	CachedAt   time.Time                `json:"cached_at"`   // 缓存时间戳
	FilterHash string                   `json:"filter_hash"` // 筛选条件哈希
}

// CachedProcessConfig 缓存的流程配置
// 用于存储审核或归档的流程配置和规则数据
type CachedProcessConfig struct {
	Config   interface{} `json:"config"`    // 流程配置
	Rules    interface{} `json:"rules"`     // 审核/归档规则
	CachedAt time.Time   `json:"cached_at"` // 缓存时间戳
}

// CachedSnapshot 缓存的快照映射
// 用于存储流程快照数据，记录审核或归档复盘的有效结论
type CachedSnapshot struct {
	Snapshots map[string]interface{} `json:"snapshots"` // 快照映射，key 为流程 ID
	CachedAt  time.Time              `json:"cached_at"` // 缓存时间戳
}

// CachedStats 缓存的统计数据
// 用于存储审核工作台或归档复盘的统计信息
type CachedStats struct {
	Stats    interface{} `json:"stats"`     // 统计数据
	CachedAt time.Time   `json:"cached_at"` // 缓存时间戳
}

// Note: CacheStatsSnapshot is defined in stats.go to avoid circular dependencies
// and keep cache statistics logic together with the CacheStats struct.
