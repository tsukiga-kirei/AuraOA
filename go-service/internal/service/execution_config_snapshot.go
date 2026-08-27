package service

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"auraoa/go-service/internal/model"
)

// AuditExecutionConfigSnapshot 是审核工作台与 OA 嵌入审核实际使用的最终配置。
// 保存合并后的字段、规则与个人尺度，后续执行不再读取可能已经变化的当前配置。
type AuditExecutionConfigSnapshot struct {
	AIConfig       datatypes.JSON             `json:"ai_config"`
	FieldSet       map[string]map[string]bool `json:"field_set"`
	MergedRules    string                     `json:"merged_rules"`
	EffectiveRules []model.AuditRule          `json:"effective_rules"`
}

// ArchiveExecutionConfigSnapshot 是归档复盘实际使用的最终配置。
type ArchiveExecutionConfigSnapshot struct {
	AIConfig       datatypes.JSON             `json:"ai_config"`
	FieldSet       map[string]map[string]bool `json:"field_set"`
	MergedRules    string                     `json:"merged_rules"`
	EffectiveRules []model.ArchiveRule        `json:"effective_rules"`
}

// SummaryExecutionConfigSnapshot 是流程总结实际使用的启用块配置。
type SummaryExecutionConfigSnapshot struct {
	Blocks []model.SummaryBlockConfig `json:"blocks"`
}

func executionVersionNumber(version *model.ExecutionConfigVersion) *int {
	if version == nil {
		return nil
	}
	value := version.VersionNo
	return &value
}

func executionVersionID(version *model.ExecutionConfigVersion) *uuid.UUID {
	if version == nil {
		return nil
	}
	value := version.ID
	return &value
}

func decodeExecutionSnapshot[T any](version *model.ExecutionConfigVersion) (*T, error) {
	if version == nil {
		return nil, fmt.Errorf("执行配置版本不存在")
	}
	var snapshot T
	if err := json.Unmarshal(version.ConfigSnapshot, &snapshot); err != nil {
		return nil, fmt.Errorf("解析执行配置版本 v%d 失败: %w", version.VersionNo, err)
	}
	return &snapshot, nil
}
