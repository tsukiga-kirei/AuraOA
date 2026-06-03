package service

import (
	"encoding/json"
	"testing"

	"auraoa/go-service/internal/model"
)

func TestFormatRuleLineForPrompt(t *testing.T) {
	got := formatRuleLineForPrompt(1, "default_on", "发票是否为国机财务公司")
	want := "1. 发票是否为国机财务公司"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestStripRuleScopePrefix_legacyBracket(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"[default_on] 发票是否为国机财务公司", "发票是否为国机财务公司"},
		{"[mandatory] 必须校验金额", "必须校验金额"},
		{"[custom] 自定义规则", "自定义规则"},
		{"[用户自定义] 自定义规则", "自定义规则"},
		{"1. [default_on] 带序号前缀", "带序号前缀"},
		{"（默认开启）发票校验", "发票校验"},
		{"发票是否为国机财务公司", "发票是否为国机财务公司"},
	}
	for _, tc := range cases {
		if got := stripRuleScopePrefix(tc.in); got != tc.want {
			t.Fatalf("stripRuleScopePrefix(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizeAuditRuleResults(t *testing.T) {
	in := []model.RuleResultJSON{
		{RuleContent: "[default_on] 规则A", Passed: false, Reason: "x"},
	}
	out := normalizeAuditRuleResults(in)
	if out[0].RuleContent != "规则A" {
		t.Fatalf("got %q", out[0].RuleContent)
	}
}

func TestNormalizeArchiveLogStoredResult(t *testing.T) {
	raw := []byte(`{"overall_compliance":"compliant","overall_score":90,"rule_audit":[{"rule_id":"[default_on] 规则A","rule_name":"[default_on] 规则A","passed":false,"reasoning":"x"}]}`)
	out := normalizeArchiveLogStoredResult(raw)
	var parsed model.ArchiveResultJSON
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.RuleAudit[0].RuleName != "规则A" {
		t.Fatalf("rule_name=%q", parsed.RuleAudit[0].RuleName)
	}
}

func TestNormalizeAuditLogStoredResult(t *testing.T) {
	raw := []byte(`{"recommendation":"review","overall_score":80,"rule_results":[{"rule_content":"[default_on] 规则A","passed":false,"reason":"x"}]}`)
	out := normalizeAuditLogStoredResult(raw)
	var parsed model.AuditResultJSON
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.RuleResults[0].RuleContent != "规则A" {
		t.Fatalf("rule_content=%q", parsed.RuleResults[0].RuleContent)
	}
}
