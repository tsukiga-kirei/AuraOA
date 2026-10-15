package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/datatypes"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/repository"
)

func validStrictness(value string) bool {
	return value == "strict" || value == "standard" || value == "loose"
}

// effectiveAuditAIConfig 在管理员允许时，将个人尺度转换为完整提示词配置。
// 只改 audit_strictness 字段不会改变模型行为，因此四段提示词必须作为一个整体切换。
func effectiveAuditAIConfig(
	base datatypes.JSON,
	override string,
	allowed bool,
	templates *repository.SystemPromptTemplateRepo,
) (datatypes.JSON, error) {
	if !allowed || override == "" {
		return base, nil
	}
	if !validStrictness(override) {
		return nil, fmt.Errorf("个人审核尺度无效")
	}
	var current model.AIConfigData
	if err := json.Unmarshal(base, &current); err != nil {
		return nil, fmt.Errorf("解析租户审核提示词失败: %w", err)
	}
	if current.AuditStrictness == override {
		return base, nil
	}
	items, err := templates.GetByStrictnessAuditWorkbench(override)
	if err != nil || len(items) == 0 {
		return nil, fmt.Errorf("读取个人审核尺度提示词失败")
	}
	return auditAIConfigFromTemplates(override, items, current)
}

func auditAIConfigFromTemplates(strictness string, items []model.SystemPromptTemplate, base ...model.AIConfigData) (datatypes.JSON, error) {
	effective := model.AIConfigData{AuditStrictness: strictness}
	// 个人尺度只切换提示词，空值传输策略仍沿用租户配置。
	if len(base) > 0 {
		effective.SkipEmptyFields = base[0].SkipEmptyFields
		effective.SkipEmptyFieldKeys = base[0].SkipEmptyFieldKeys
	}
	for _, item := range items {
		switch {
		case item.PromptType == "system" && item.Phase == "reasoning":
			effective.SystemReasoningPrompt = item.Content
		case item.PromptType == "system" && item.Phase == "extraction":
			effective.SystemExtractionPrompt = item.Content
		case item.PromptType == "user" && item.Phase == "reasoning":
			effective.UserReasoningPrompt = item.Content
		case item.PromptType == "user" && item.Phase == "extraction":
			effective.UserExtractionPrompt = item.Content
		}
	}
	raw, err := json.Marshal(effective)
	return datatypes.JSON(raw), err
}

// effectiveArchiveAIConfig 与审核工作台保持相同的个人尺度语义，但只读取归档专用模板。
func effectiveArchiveAIConfig(
	base datatypes.JSON,
	override string,
	allowed bool,
	templates *repository.SystemPromptTemplateRepo,
) (datatypes.JSON, error) {
	if !allowed || override == "" {
		return base, nil
	}
	if !validStrictness(override) {
		return nil, fmt.Errorf("个人复核尺度无效")
	}
	var current model.ArchiveAIConfigData
	if err := json.Unmarshal(base, &current); err != nil {
		return nil, fmt.Errorf("解析租户归档复盘提示词失败: %w", err)
	}
	if current.AuditStrictness == override {
		return base, nil
	}
	items, err := templates.ListAll()
	if err != nil {
		return nil, fmt.Errorf("读取个人复核尺度提示词失败")
	}
	return archiveAIConfigFromTemplates(override, items, current)
}

func archiveAIConfigFromTemplates(strictness string, items []model.SystemPromptTemplate, base ...model.ArchiveAIConfigData) (datatypes.JSON, error) {
	effective := model.ArchiveAIConfigData{AuditStrictness: strictness}
	// 与审核侧对称，个人尺度不能丢失租户空值传输策略。
	if len(base) > 0 {
		effective.SkipEmptyFields = base[0].SkipEmptyFields
		effective.SkipEmptyFieldKeys = base[0].SkipEmptyFieldKeys
	}
	matched := 0
	for _, item := range items {
		if item.Strictness == nil || *item.Strictness != strictness || !strings.HasPrefix(item.PromptKey, "archive_") {
			continue
		}
		matched++
		switch {
		case item.PromptType == "system" && item.Phase == "reasoning":
			effective.SystemReasoningPrompt = item.Content
		case item.PromptType == "system" && item.Phase == "extraction":
			effective.SystemExtractionPrompt = item.Content
		case item.PromptType == "user" && item.Phase == "reasoning":
			effective.UserReasoningPrompt = item.Content
		case item.PromptType == "user" && item.Phase == "extraction":
			effective.UserExtractionPrompt = item.Content
		}
	}
	if matched == 0 {
		return nil, fmt.Errorf("个人复核尺度没有可用提示词")
	}
	raw, err := json.Marshal(effective)
	return datatypes.JSON(raw), err
}
