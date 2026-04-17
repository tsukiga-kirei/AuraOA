// Package cache 提供统一的缓存管理能力
package cache

import (
	"sync/atomic"
)

// CacheStats 缓存统计信息（线程安全）
// 使用 atomic 操作确保并发安全
type CacheStats struct {
	hits   int64
	misses int64
	errors int64
}

// NewCacheStats 创建新的缓存统计实例
func NewCacheStats() *CacheStats {
	return &CacheStats{}
}

// IncrHit 增加命中计数（线程安全）
func (s *CacheStats) IncrHit() {
	atomic.AddInt64(&s.hits, 1)
}

// IncrMiss 增加未命中计数（线程安全）
func (s *CacheStats) IncrMiss() {
	atomic.AddInt64(&s.misses, 1)
}

// IncrError 增加错误计数（线程安全）
func (s *CacheStats) IncrError() {
	atomic.AddInt64(&s.errors, 1)
}

// GetSnapshot 获取统计快照
// 返回当前统计数据的快照，包含命中率计算
func (s *CacheStats) GetSnapshot() CacheStatsSnapshot {
	hits := atomic.LoadInt64(&s.hits)
	misses := atomic.LoadInt64(&s.misses)
	errs := atomic.LoadInt64(&s.errors)

	var hitRate float64
	total := hits + misses
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}

	return CacheStatsSnapshot{
		Hits:    hits,
		Misses:  misses,
		Errors:  errs,
		HitRate: hitRate,
	}
}

// GetHits 获取命中次数
func (s *CacheStats) GetHits() int64 {
	return atomic.LoadInt64(&s.hits)
}

// GetMisses 获取未命中次数
func (s *CacheStats) GetMisses() int64 {
	return atomic.LoadInt64(&s.misses)
}

// GetErrors 获取错误次数
func (s *CacheStats) GetErrors() int64 {
	return atomic.LoadInt64(&s.errors)
}

// Reset 重置所有统计计数器
func (s *CacheStats) Reset() {
	atomic.StoreInt64(&s.hits, 0)
	atomic.StoreInt64(&s.misses, 0)
	atomic.StoreInt64(&s.errors, 0)
}

// CacheStatsSnapshot 缓存统计快照
// 用于返回某一时刻的统计数据副本
type CacheStatsSnapshot struct {
	Hits     int64   `json:"hits"`      // 命中次数
	Misses   int64   `json:"misses"`    // 未命中次数
	Errors   int64   `json:"errors"`    // 错误次数
	HitRate  float64 `json:"hit_rate"`  // 命中率 (0.0 - 1.0)
	KeyCount int64   `json:"key_count"` // 缓存键数量（由 CacheManager 填充）
}

// GetTotal 获取总请求次数（命中 + 未命中）
func (s CacheStatsSnapshot) GetTotal() int64 {
	return s.Hits + s.Misses
}
