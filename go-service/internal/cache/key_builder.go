// Package cache 提供统一的缓存管理能力
package cache

import (
	"fmt"

	"github.com/google/uuid"
)

// CacheKeyBuilder 缓存键构建器
// 用于生成符合 {module}:{tenant_id}:{resource}:{identifier} 规范的缓存键
type CacheKeyBuilder struct {
	module   string
	tenantID string
}

// NewKeyBuilder 创建键构建器
// module: 模块名称，如 "audit", "archive", "dashboard"
// tenantID: 租户 UUID
func NewKeyBuilder(module string, tenantID uuid.UUID) *CacheKeyBuilder {
	return &CacheKeyBuilder{
		module:   module,
		tenantID: tenantID.String(),
	}
}

// TodoList 生成待办列表缓存键
// 格式: audit:todo:{tenant_id}:{user_id}:{filter_hash}
func (b *CacheKeyBuilder) TodoList(userID uuid.UUID, filterHash string) string {
	return fmt.Sprintf("audit:todo:%s:%s:%s", b.tenantID, userID.String(), filterHash)
}

// ArchiveList 生成归档列表缓存键
// 格式: archive:list:{tenant_id}:{user_id}:{filter_hash}
func (b *CacheKeyBuilder) ArchiveList(userID uuid.UUID, filterHash string) string {
	return fmt.Sprintf("archive:list:%s:%s:%s", b.tenantID, userID.String(), filterHash)
}

// ProcessConfig 生成流程配置缓存键
// 格式: {module}:config:{tenant_id}:{process_type}
// 用于审核配置 (audit:config:...) 或归档配置 (archive:config:...)
func (b *CacheKeyBuilder) ProcessConfig(processType string) string {
	return fmt.Sprintf("%s:config:%s:%s", b.module, b.tenantID, processType)
}

// Snapshot 生成快照缓存键
// 格式: {module}:snapshot:{tenant_id}:{process_ids_hash}
// 用于审核快照 (audit:snapshot:...) 或归档快照 (archive:snapshot:...)
func (b *CacheKeyBuilder) Snapshot(processIDsHash string) string {
	return fmt.Sprintf("%s:snapshot:%s:%s", b.module, b.tenantID, processIDsHash)
}

// Stats 生成统计数据缓存键
// 格式: {module}:stats:{tenant_id}:{user_id}:{date_range_hash}
// 用于审核统计 (audit:stats:...) 或归档统计 (archive:stats:...)
func (b *CacheKeyBuilder) Stats(userID uuid.UUID, dateRangeHash string) string {
	return fmt.Sprintf("%s:stats:%s:%s:%s", b.module, b.tenantID, userID.String(), dateRangeHash)
}

// Dashboard 生成仪表盘缓存键
// 格式: dashboard:{tenant_id}:{user_id}:{role}
// 注意：此方法忽略 module 字段，始终使用 "dashboard" 前缀
func (b *CacheKeyBuilder) Dashboard(userID uuid.UUID, role string) string {
	return fmt.Sprintf("dashboard:%s:%s:%s", b.tenantID, userID.String(), role)
}

// TodoListPrefix 生成待办列表缓存键前缀（用于批量删除）
// 格式: audit:todo:{tenant_id}:{user_id}:
func (b *CacheKeyBuilder) TodoListPrefix(userID uuid.UUID) string {
	return fmt.Sprintf("audit:todo:%s:%s:", b.tenantID, userID.String())
}

// ArchiveListPrefix 生成归档列表缓存键前缀（用于批量删除）
// 格式: archive:list:{tenant_id}:{user_id}:
func (b *CacheKeyBuilder) ArchiveListPrefix(userID uuid.UUID) string {
	return fmt.Sprintf("archive:list:%s:%s:", b.tenantID, userID.String())
}

// ConfigPrefix 生成配置缓存键前缀（用于批量删除）
// 格式: {module}:config:{tenant_id}:
func (b *CacheKeyBuilder) ConfigPrefix() string {
	return fmt.Sprintf("%s:config:%s:", b.module, b.tenantID)
}

// SnapshotPrefix 生成快照缓存键前缀（用于批量删除）
// 格式: {module}:snapshot:{tenant_id}:
func (b *CacheKeyBuilder) SnapshotPrefix() string {
	return fmt.Sprintf("%s:snapshot:%s:", b.module, b.tenantID)
}

// StatsPrefix 生成统计缓存键前缀（用于批量删除）
// 格式: {module}:stats:{tenant_id}:
func (b *CacheKeyBuilder) StatsPrefix() string {
	return fmt.Sprintf("%s:stats:%s:", b.module, b.tenantID)
}

// DashboardPrefix 生成仪表盘缓存键前缀（用于批量删除）
// 格式: dashboard:{tenant_id}:
func (b *CacheKeyBuilder) DashboardPrefix() string {
	return fmt.Sprintf("dashboard:%s:", b.tenantID)
}

// TenantPrefix 生成租户级别缓存键前缀（用于清除租户全部缓存）
// 返回多个前缀，需要分别删除
func (b *CacheKeyBuilder) TenantPrefixes() []string {
	return []string{
		fmt.Sprintf("audit:todo:%s:", b.tenantID),
		fmt.Sprintf("audit:config:%s:", b.tenantID),
		fmt.Sprintf("audit:snapshot:%s:", b.tenantID),
		fmt.Sprintf("audit:stats:%s:", b.tenantID),
		fmt.Sprintf("archive:list:%s:", b.tenantID),
		fmt.Sprintf("archive:config:%s:", b.tenantID),
		fmt.Sprintf("archive:snapshot:%s:", b.tenantID),
		fmt.Sprintf("archive:stats:%s:", b.tenantID),
		fmt.Sprintf("dashboard:%s:", b.tenantID),
	}
}

// ModulePrefix 生成模块级别缓存键前缀
// 格式: {module}:
func ModulePrefix(module string) string {
	return fmt.Sprintf("%s:", module)
}
