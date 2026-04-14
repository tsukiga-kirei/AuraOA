package model

import (
	"time"

	"github.com/google/uuid"
)

// DBDriverOption 数据库驱动选项。
type DBDriverOption struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Code        string    `gorm:"uniqueIndex;size:50;not null" json:"code"`
	Label       string    `gorm:"size:100;not null" json:"label"`
	DefaultPort int       `gorm:"not null;default:3306" json:"default_port"`
	SortOrder   int       `gorm:"not null;default:0" json:"sort_order"`
	Enabled     bool      `gorm:"not null;default:true" json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
}

func (DBDriverOption) TableName() string { return "db_driver_options" }
