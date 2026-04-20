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

// CachedOAArchivedData 缓存的 OA 归档全量数据
// 按日期范围缓存 OA 跨库查询的全量归档流程列表，
// 供翻页、页签切换、搜索等操作在内存中复用，避免重复查询 OA。
type CachedOAArchivedData struct {
	Items    []CachedArchivedItem `json:"items"`     // 全量归档流程列表
	Total    int                  `json:"total"`     // OA 返回的总数
	CachedAt time.Time            `json:"cached_at"` // 缓存时间戳
}

// CachedArchivedItem 缓存的归档流程条目（与 oa.ArchivedItem 字段一致）
type CachedArchivedItem struct {
	ProcessID        string `json:"process_id"`
	Title            string `json:"title"`
	Applicant        string `json:"applicant"`
	Department       string `json:"department"`
	ProcessType      string `json:"process_type"`
	ProcessTypeLabel string `json:"process_type_label"`
	CurrentNode      string `json:"current_node"`
	SubmitTime       string `json:"submit_time"`
	ArchiveTime      string `json:"archive_time"`
	MainTableName    string `json:"main_table_name"`
}

// CachedOATodoData 缓存的 OA 待办全量数据
// 按日期范围缓存 OA 跨库查询的全量待办流程列表。
type CachedOATodoData struct {
	Items    []CachedTodoItem `json:"items"`     // 全量待办流程列表
	Total    int              `json:"total"`     // OA 返回的总数
	CachedAt time.Time        `json:"cached_at"` // 缓存时间戳
}

// CachedTodoItem 缓存的待办流程条目（与 oa.TodoItem 字段一致）
type CachedTodoItem struct {
	ProcessID        string `json:"process_id"`
	Title            string `json:"title"`
	Applicant        string `json:"applicant"`
	Department       string `json:"department"`
	ProcessType      string `json:"process_type"`
	ProcessTypeLabel string `json:"process_type_label"`
	CurrentNode      string `json:"current_node"`
	SubmitTime       string `json:"submit_time"`
	Urgency          string `json:"urgency"`
	MainTableName    string `json:"main_table_name"`
}

// Note: CacheStatsSnapshot is defined in stats.go to avoid circular dependencies
// and keep cache statistics logic together with the CacheStats struct.
