package model

import (
	"time"

	"github.com/google/uuid"
)

// OADatabaseConnection 系统级 OA 数据库连接配置。
type OADatabaseConnection struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name              string    `gorm:"size:200;not null" json:"name"`
	OAType            string    `gorm:"size:50;not null" json:"oa_type"`
	OATypeLabel       string    `gorm:"size:100;default:''" json:"oa_type_label"`
	Driver            string    `gorm:"size:50;not null;default:mysql" json:"driver"`
	Host              string    `gorm:"size:255;not null;default:''" json:"host"`
	Port              int       `gorm:"not null;default:3306" json:"port"`
	DatabaseName      string    `gorm:"size:200;not null;default:''" json:"database_name"`
	Username          string    `gorm:"size:200;not null;default:''" json:"username"`
	Password          string    `gorm:"size:500;not null;default:''" json:"-"` // 不输出到JSON
	PoolSize          int       `gorm:"not null;default:10" json:"pool_size"`
	ConnectionTimeout int       `gorm:"not null;default:30" json:"connection_timeout"`
	TestOnBorrow      bool      `gorm:"not null;default:true" json:"test_on_borrow"`
	Status            string    `gorm:"size:20;not null;default:disconnected" json:"status"`
	SyncInterval      int       `gorm:"not null;default:30" json:"sync_interval"`
	Enabled           bool      `gorm:"not null;default:true" json:"enabled"`
	Description       string    `gorm:"type:text;default:''" json:"description"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (OADatabaseConnection) TableName() string { return "oa_database_connections" }
