package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

const (
	ExecutionConfigModuleAudit   = "audit"
	ExecutionConfigModuleSummary = "summary"
	ExecutionConfigModuleArchive = "archive"
)

// ExecutionConfigVersion 保存一次实际执行所需的不可变最终生效配置。
type ExecutionConfigVersion struct {
	ID                  uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID            uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	Module              string         `gorm:"size:20;not null" json:"module"`
	SourceConfigID      uuid.UUID      `gorm:"type:uuid;not null" json:"source_config_id"`
	BaseConfigVersionID *uuid.UUID     `gorm:"type:uuid" json:"base_config_version_id,omitempty"`
	VersionNo           int            `gorm:"not null" json:"version_no"`
	Fingerprint         string         `gorm:"size:80;not null" json:"fingerprint"`
	ConfigSnapshot      datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"config_snapshot"`
	CreatedBy           *uuid.UUID     `gorm:"type:uuid" json:"created_by,omitempty"`
	CreatedAt           time.Time      `json:"created_at"`
}

func (ExecutionConfigVersion) TableName() string { return "execution_config_versions" }

// TenantConfigVersion 保存管理员租户配置的不可变基础版本，不包含任何用户个人覆盖。
type TenantConfigVersion struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID       uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	Module         string         `gorm:"size:20;not null" json:"module"`
	SourceConfigID uuid.UUID      `gorm:"type:uuid;not null" json:"source_config_id"`
	VersionNo      int            `gorm:"not null" json:"version_no"`
	Fingerprint    string         `gorm:"size:80;not null" json:"fingerprint"`
	ConfigSnapshot datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"config_snapshot"`
	IsActive       bool           `gorm:"not null;default:false" json:"is_active"`
	CreatedBy      *uuid.UUID     `gorm:"type:uuid" json:"created_by,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

func (TenantConfigVersion) TableName() string { return "tenant_config_versions" }

// ProcessExecutionConfigBinding 固定流程实例后续执行所沿用的配置版本。
type ProcessExecutionConfigBinding struct {
	Scope           string     `gorm:"size:64;not null;default:''" json:"scope"`
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID        uuid.UUID  `gorm:"type:uuid;not null" json:"tenant_id"`
	Module          string     `gorm:"size:20;not null" json:"module"`
	ProcessID       string     `gorm:"size:100;not null" json:"process_id"`
	ProcessType     string     `gorm:"size:200;not null;default:''" json:"process_type"`
	ConfigVersionID uuid.UUID  `gorm:"type:uuid;not null" json:"config_version_id"`
	BoundBy         *uuid.UUID `gorm:"type:uuid" json:"bound_by,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (ProcessExecutionConfigBinding) TableName() string {
	return "process_execution_config_bindings"
}
