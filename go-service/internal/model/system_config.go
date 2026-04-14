package model

import (
	"time"

	"github.com/google/uuid"
)

// SystemConfig 全局键值配置项，存储系统级参数。
type SystemConfig struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Key       string    `gorm:"column:key;uniqueIndex;size:200;not null"`
	Value     string    `gorm:"column:value;type:text;not null;default:''"`
	Remark    string    `gorm:"column:remark;size:500"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// TableName 指定表名，避免 GORM 自动复数化。
func (SystemConfig) TableName() string {
	return "system_configs"
}
