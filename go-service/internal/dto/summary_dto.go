package dto

import "gorm.io/datatypes"

// CreateProcessSummaryConfigRequest 创建流程总结配置请求。
type CreateProcessSummaryConfigRequest struct {
	ProcessType      string         `json:"process_type" binding:"required"`
	ProcessTypeLabel string         `json:"process_type_label"`
	MainTableName    string         `json:"main_table_name"`
	MainFields       datatypes.JSON `json:"main_fields"`
	DetailTables     datatypes.JSON `json:"detail_tables"`
	SummaryBlocks    datatypes.JSON `json:"summary_blocks"`
	EmbedEnabled     *bool          `json:"embed_enabled"`
	EmbedConfig      datatypes.JSON `json:"embed_config"`
	Status           string         `json:"status"`
}

// UpdateProcessSummaryConfigRequest 更新流程总结配置请求。
type UpdateProcessSummaryConfigRequest struct {
	ProcessType      string         `json:"process_type"`
	ProcessTypeLabel string         `json:"process_type_label"`
	MainTableName    string         `json:"main_table_name"`
	MainFields       datatypes.JSON `json:"main_fields"`
	DetailTables     datatypes.JSON `json:"detail_tables"`
	SummaryBlocks    datatypes.JSON `json:"summary_blocks"`
	EmbedEnabled     *bool          `json:"embed_enabled"`
	EmbedConfig      datatypes.JSON `json:"embed_config"`
	Status           string         `json:"status"`
}
