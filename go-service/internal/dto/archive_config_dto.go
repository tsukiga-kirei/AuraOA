package dto

import (
	"gorm.io/datatypes"
)

// ===================== 归档复盘配置 DTO =====================

// CreateProcessArchiveConfigRequest 创建归档复盘配置请求
type CreateProcessArchiveConfigRequest struct {
	ProcessType      string         `json:"process_type" binding:"required"`
	ProcessTypeLabel string         `json:"process_type_label"`
	MainTableName    string         `json:"main_table_name"`
	MainFields       datatypes.JSON `json:"main_fields"`
	DetailTables     datatypes.JSON `json:"detail_tables"`
	FieldMode        string         `json:"field_mode"`
	KBMode           string         `json:"kb_mode"`
	AIConfig         datatypes.JSON `json:"ai_config"`
	UserPermissions  datatypes.JSON `json:"user_permissions"`
	AccessControl    datatypes.JSON `json:"access_control"`
	Status           string         `json:"status"`
}

// UpdateProcessArchiveConfigRequest 更新归档复盘配置请求
type UpdateProcessArchiveConfigRequest struct {
	ProcessType      string         `json:"process_type"`
	ProcessTypeLabel string         `json:"process_type_label"`
	MainTableName    string         `json:"main_table_name"`
	MainFields       datatypes.JSON `json:"main_fields"`
	DetailTables     datatypes.JSON `json:"detail_tables"`
	FieldMode        string         `json:"field_mode"`
	KBMode           string         `json:"kb_mode"`
	AIConfig         datatypes.JSON `json:"ai_config"`
	UserPermissions  datatypes.JSON `json:"user_permissions"`
	AccessControl    datatypes.JSON `json:"access_control"`
	Status           string         `json:"status"`
}
