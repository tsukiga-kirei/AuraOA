package service

import (
	"encoding/json"
	"testing"

	"auraoa/go-service/internal/dto"
	"auraoa/go-service/internal/model"
)

func TestAuditAIConfigFromTemplatesAppliesPersonalStrictness(t *testing.T) {
	loose := "loose"
	raw, err := auditAIConfigFromTemplates(loose, []model.SystemPromptTemplate{
		{PromptType: "system", Phase: "reasoning", Content: "宽松系统推理"},
		{PromptType: "user", Phase: "reasoning", Content: "宽松用户推理"},
		{PromptType: "system", Phase: "extraction", Content: "宽松系统提取"},
		{PromptType: "user", Phase: "extraction", Content: "宽松用户提取"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var got model.AIConfigData
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.AuditStrictness != loose || got.SystemReasoningPrompt != "宽松系统推理" || got.UserExtractionPrompt != "宽松用户提取" {
		t.Fatalf("个人尺度没有形成完整提示词配置: %+v", got)
	}
}

func TestArchiveAIConfigFromTemplatesOnlyUsesMatchingModuleAndStrictness(t *testing.T) {
	loose := "loose"
	standard := "standard"
	raw, err := archiveAIConfigFromTemplates(loose, []model.SystemPromptTemplate{
		{PromptKey: "archive_system_reasoning_loose", Strictness: &loose, PromptType: "system", Phase: "reasoning", Content: "宽松归档推理"},
		{PromptKey: "archive_system_reasoning_standard", Strictness: &standard, PromptType: "system", Phase: "reasoning", Content: "标准归档推理"},
		{PromptKey: "audit_user_reasoning_loose", Strictness: &loose, PromptType: "user", Phase: "reasoning", Content: "错误模块"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var got model.ArchiveAIConfigData
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.AuditStrictness != loose || got.SystemReasoningPrompt != "宽松归档推理" || got.UserReasoningPrompt != "" {
		t.Fatalf("归档个人尺度模板过滤不正确: %+v", got)
	}
}

func TestVersionedCustomRulesPreservesAddedVersion(t *testing.T) {
	existing := []model.CustomRule{{ID: "r1", BaseConfigVersion: 2, AddedInPersonalVersion: 3}}
	result := versionedCustomRules([]dto.CustomRuleDTO{{ID: "r1"}, {ID: "r2"}}, existing, 4, 5)
	if result[0].BaseConfigVersion != 2 || result[0].AddedInPersonalVersion != 3 {
		t.Fatalf("已有规则的加入版本不应被重写: %+v", result[0])
	}
	if result[1].BaseConfigVersion != 4 || result[1].AddedInPersonalVersion != 5 {
		t.Fatalf("新规则应记录本次基础和个人版本: %+v", result[1])
	}
}

func TestValidatePersonalVersionRejectsStaleBaseAndPersonalVersion(t *testing.T) {
	if err := validatePersonalVersion(3, 2, 4, 4); err == nil {
		t.Fatal("租户基础版本过期时必须拒绝保存")
	}
	if err := validatePersonalVersion(3, 3, 4, 3); err == nil {
		t.Fatal("个人版本过期时必须拒绝保存")
	}
	if err := validatePersonalVersion(3, 3, 4, 4); err != nil {
		t.Fatalf("版本一致时应允许保存: %v", err)
	}
}
