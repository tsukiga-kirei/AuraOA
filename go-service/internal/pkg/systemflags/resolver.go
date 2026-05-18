// Package systemflags 提供对 system_configs 中安全相关开关的带缓存读取，避免每条业务请求反复查库。
package systemflags

import (
	"strings"
	"sync"
	"time"

	"auraoa/go-service/internal/repository"
)

const cacheTTL = 5 * time.Second

// Resolver 从 system_configs 读取布尔开关，内存缓存数秒以降低数据库压力。
type Resolver struct {
	repo *repository.SystemConfigRepo
	mu   sync.RWMutex
	// cached
	expiresAt      time.Time
	auditTrail     bool
	dataEncryption bool
}

// NewResolver 创建开关解析器；repo 不可为 nil。
func NewResolver(repo *repository.SystemConfigRepo) *Resolver {
	return &Resolver{repo: repo}
}

func (r *Resolver) readBool(key string, defaultVal bool) bool {
	if r.repo == nil {
		return defaultVal
	}
	val, err := r.repo.FindByKey(key)
	if err != nil || strings.TrimSpace(val) == "" {
		return defaultVal
	}
	v := strings.TrimSpace(strings.ToLower(val))
	return v == "true" || v == "1" || v == "yes"
}

func (r *Resolver) refresh() {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	if !r.expiresAt.IsZero() && now.Before(r.expiresAt) {
		return
	}
	r.auditTrail = r.readBool("system.enable_audit_trail", true)
	r.dataEncryption = r.readBool("system.enable_data_encryption", false)
	r.expiresAt = now.Add(cacheTTL)
}

// AuditTrailEnabled 是否记录用户操作审计（默认 true，与迁移种子一致）。
func (r *Resolver) AuditTrailEnabled() bool {
	r.refresh()
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.auditTrail
}

// DataEncryptionEnabled 是否对 OA 等业务敏感文本做脱敏/落库前处理（默认 false）。
func (r *Resolver) DataEncryptionEnabled() bool {
	r.refresh()
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.dataEncryption
}
