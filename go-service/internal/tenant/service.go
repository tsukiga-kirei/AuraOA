package tenant

import (
	"context"
	"time"
)

type KBMode string

const (
	KBModeRulesOnly KBMode = "rules_only"
	KBModeRAGOnly   KBMode = "rag_only"
	KBModeHybrid    KBMode = "hybrid"
)

type TenantInput struct {
	Name           string `json:"name" binding:"required"`
	OAType         string `json:"oa_type"`
	OAJDBCConfig   string `json:"oa_jdbc_config"`
	TokenQuota     int    `json:"token_quota"`
	MaxConcurrency int    `json:"max_concurrency"`
}

type Tenant struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	TokenQuota     int       `json:"token_quota"`
	TokenUsed      int       `json:"token_used"`
	MaxConcurrency int       `json:"max_concurrency"`
	OAType         string    `json:"oa_type"`
	CreatedAt      time.Time `json:"created_at"`
}

type QuotaConfig struct {
	TokenQuota     int `json:"token_quota"`
	MaxConcurrency int `json:"max_concurrency"`
}

type TenantConfig struct {
	Tenant
	KBModes map[string]KBMode `json:"kb_modes"` // processType -> KBMode
}

// TenantService manages tenant CRUD, quota and KB mode configuration.
type TenantService interface {
	CreateTenant(ctx context.Context, input TenantInput) (Tenant, error)
	UpdateTenantQuota(ctx context.Context, tenantID string, quota QuotaConfig) error
	GetTenantConfig(ctx context.Context, tenantID string) (TenantConfig, error)
	SetKBMode(ctx context.Context, tenantID string, processType string, mode KBMode) error
}
