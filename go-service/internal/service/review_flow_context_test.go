package service

import (
	"testing"

	"auraoa/go-service/internal/model"
)

func TestNeedsAuditFlowContext(t *testing.T) {
	rules := []model.AuditRule{{RuleContent: "规则", Enabled: boolPointer(true)}}
	if needsAuditFlowContext(&model.AIConfigData{UserReasoningPrompt: "只审核金额"}, rules) {
		t.Fatal("提示词不引用审批流且规则未关联审批流时不应拉取")
	}
	if !needsAuditFlowContext(&model.AIConfigData{UserReasoningPrompt: "审批节点：{{current_node}}"}, rules) {
		t.Fatal("提示词引用当前节点时应拉取")
	}
	rules[0].RelatedFlow = true
	if !needsAuditFlowContext(&model.AIConfigData{UserReasoningPrompt: "只审核金额"}, rules) {
		t.Fatal("启用规则关联审批流时应拉取")
	}
	rules[0].Enabled = boolPointer(false)
	if needsAuditFlowContext(&model.AIConfigData{UserReasoningPrompt: "只审核金额"}, rules) {
		t.Fatal("未启用规则关联审批流时不应拉取")
	}
}

func TestNeedsArchiveFlowContext(t *testing.T) {
	rules := []model.ArchiveRule{{RuleContent: "规则", Enabled: boolPointer(true)}}
	if needsArchiveFlowContext(&model.ArchiveAIConfigData{UserReasoningPrompt: "只复盘金额"}, rules) {
		t.Fatal("提示词不引用审批流且规则未关联审批流时不应拉取")
	}
	rules[0].RelatedFlow = true
	if !needsArchiveFlowContext(&model.ArchiveAIConfigData{UserReasoningPrompt: "只复盘金额"}, rules) {
		t.Fatal("启用规则关联审批流时应拉取")
	}
}
