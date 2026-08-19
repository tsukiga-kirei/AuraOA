package service

import (
	"testing"

	"auraoa/go-service/internal/model"
)

func TestApplyAuditExtractionPromptTemplates(t *testing.T) {
	data := model.AIConfigData{
		SystemReasoningPrompt:  "保留系统推理",
		UserReasoningPrompt:    "保留用户推理",
		SystemExtractionPrompt: "客户端系统提取",
		UserExtractionPrompt:   "客户端用户提取",
	}
	templates := []model.SystemPromptTemplate{
		{PromptType: "system", Phase: "extraction", Content: "模板系统提取"},
		{PromptType: "user", Phase: "extraction", Content: "模板用户提取"},
	}

	applyAuditExtractionPromptTemplates(&data, templates)

	if data.SystemExtractionPrompt != "模板系统提取" || data.UserExtractionPrompt != "模板用户提取" {
		t.Fatalf("结构提取提示词未全部锁定到模板: %+v", data)
	}
	if data.SystemReasoningPrompt != "保留系统推理" || data.UserReasoningPrompt != "保留用户推理" {
		t.Fatalf("推理阶段提示词不应被结构提取锁定逻辑修改: %+v", data)
	}
}

func TestApplyArchiveExtractionPromptTemplates(t *testing.T) {
	standard := "standard"
	strict := "strict"
	data := model.ArchiveAIConfigData{
		SystemExtractionPrompt: "客户端系统提取",
		UserExtractionPrompt:   "客户端用户提取",
	}
	templates := []model.SystemPromptTemplate{
		{PromptKey: "archive_system_extraction_standard", PromptType: "system", Phase: "extraction", Strictness: &standard, Content: "归档模板系统提取"},
		{PromptKey: "archive_user_extraction_standard", PromptType: "user", Phase: "extraction", Strictness: &standard, Content: "归档模板用户提取"},
		{PromptKey: "archive_user_extraction_strict", PromptType: "user", Phase: "extraction", Strictness: &strict, Content: "错误尺度"},
		{PromptKey: "audit_user_extraction_standard", PromptType: "user", Phase: "extraction", Strictness: &standard, Content: "错误模块"},
	}

	applyArchiveExtractionPromptTemplates(&data, templates, standard)

	if data.SystemExtractionPrompt != "归档模板系统提取" || data.UserExtractionPrompt != "归档模板用户提取" {
		t.Fatalf("归档结构提取提示词未按模块和尺度锁定: %+v", data)
	}
}
