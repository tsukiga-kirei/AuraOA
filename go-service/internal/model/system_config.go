package model

import (
	"time"

	"github.com/google/uuid"
)

// SystemConfig represents a global key-value configuration entry.
type SystemConfig struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Key       string    `gorm:"column:key;uniqueIndex;size:200;not null"`
	Value     string    `gorm:"column:value;type:text;not null;default:''"`
	Remark    string    `gorm:"column:remark;size:500"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (SystemConfig) TableName() string {
	return "system_configs"
}
