package service

import (
	"strings"
	"testing"

	"auraoa/go-service/internal/dto"
	"auraoa/go-service/internal/pkg/errcode"
)

func TestNormalizeImportedDrafts(t *testing.T) {
	rules, err := normalizeImportedDrafts([]dto.RuleImportDraft{
		{RuleContent: "  合同金额不得超过预算  ", RuleScope: "mandatory", Confidence: 1.2},
		{RuleContent: "合同金额不得超过预算", RuleScope: "mandatory", Confidence: .8},
		{RuleContent: "需要查询供应商黑名单", RuleScope: "unknown", ContextEnabled: true, Confidence: -.2},
		{RuleContent: "   "},
	})
	if err != nil {
		t.Fatalf("normalizeImportedDrafts() error = %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("expected 2 unique rules, got %d", len(rules))
	}
	if rules[0].Confidence != 1 {
		t.Fatalf("expected confidence clamped to 1, got %v", rules[0].Confidence)
	}
	if rules[1].RuleScope != "default_on" || rules[1].Confidence != 0 {
		t.Fatalf("unexpected normalized fallback: %+v", rules[1])
	}
	if !rules[1].ContextRecommended || rules[1].ContextEnabled {
		t.Fatalf("expected legacy context flag converted to recommendation: %+v", rules[1])
	}
}

func TestHasEnabledContextMounts(t *testing.T) {
	if hasEnabledContextMounts([]byte(`[]`)) {
		t.Fatal("empty mounts must not enable external context")
	}
	if hasEnabledContextMounts([]byte(`[ {"type":"workflow","enabled":false} ]`)) {
		t.Fatal("disabled mounts must not enable external context")
	}
	if !hasEnabledContextMounts([]byte(`[ {"type":"workflow","enabled":true} ]`)) {
		t.Fatal("enabled mount should enable external context")
	}
}

func TestConfirmRejectsUnsupportedImportSource(t *testing.T) {
	svc := &RuleImportService{}
	_, err := svc.Confirm(nil, ruleImportAudit, &dto.ConfirmRuleImportRequest{Source: "unknown"})
	serviceErr, ok := err.(*ServiceError)
	if !ok || serviceErr.Code != errcode.ErrParamValidation {
		t.Fatalf("expected invalid source error, got %v", err)
	}
}

func TestSplitRuleImportTextKeepsUnicode(t *testing.T) {
	input := strings.Repeat("审批规则", 9)
	chunks := splitRuleImportText(input, 10)
	if len(chunks) != 4 {
		t.Fatalf("expected 4 chunks, got %d", len(chunks))
	}
	if strings.Join(chunks, "") != input {
		t.Fatal("chunks did not preserve unicode input")
	}
}
