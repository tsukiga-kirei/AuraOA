package service

import (
	"encoding/json"
	"strings"
	"testing"

	"auraoa/go-service/internal/dto"
	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/oa"
)

func TestConditionalRulesReachAllPromptStages(t *testing.T) {
	rules := formatRuleLineForPrompt(1, "mandatory", "必须上传采购审批文件", "单一来源采购")
	data := &oa.ProcessData{}
	audit := &model.AIConfigData{SystemReasoningPrompt: "租户推理要求", SystemExtractionPrompt: "租户提取要求"}
	archive := &model.ArchiveAIConfigData{SystemReasoningPrompt: "租户推理要求", SystemExtractionPrompt: "租户提取要求"}
	a := BuildReasoningPrompt(audit, "采购", data, rules, "", nil, nil, "")
	b := BuildExtractionPrompt(audit, "推理结果", rules)
	c := BuildArchiveReasoningPrompt(archive, "采购", data, rules, "", nil, nil, "")
	d := BuildArchiveExtractionPrompt(archive, "推理结果", rules)
	requests := []struct{ system, user string }{
		{a.SystemPrompt, a.UserPrompt}, {b.SystemPrompt, b.UserPrompt},
		{c.SystemPrompt, c.UserPrompt}, {d.SystemPrompt, d.UserPrompt},
	}
	for i, req := range requests {
		if !strings.HasPrefix(req.system, "租户") || !strings.Contains(req.system, "passed=true") || !strings.Contains(req.system, "不能因为前提为空就判定通过") {
			t.Fatalf("阶段 %d 未保留租户提示词或统一前提语义: %s", i, req.system)
		}
		if !strings.Contains(req.user, rules) {
			t.Fatalf("阶段 %d 的自定义模板没有规则占位符时丢失规则", i)
		}
	}
	if got := withRuleApplicabilityInstructions(a.SystemPrompt); got != a.SystemPrompt {
		t.Fatal("默认模板已有统一说明时不能重复追加")
	}
}

func TestRuleConditionFormattingPreservesLegacyRules(t *testing.T) {
	if got := formatRuleLineForPrompt(1, "mandatory", "检查金额", "  "); got != "1. 检查金额" {
		t.Fatalf("空前提改变旧规则格式: %q", got)
	}
	if got := formatRuleLineForPrompt(1, "mandatory", "检查金额", "采购金额超过100万元"); !strings.Contains(got, "适用前提：采购金额超过100万元") {
		t.Fatalf("有前提的规则未成组展示: %q", got)
	}
}

func TestRuleImportKeepsSameContentWithDifferentConditions(t *testing.T) {
	rules, err := normalizeImportedDrafts([]dto.RuleImportDraft{
		{RuleContent: "上传审批文件", ApplicabilityCondition: " 单一来源采购 "},
		{RuleContent: "上传审批文件", ApplicabilityCondition: "单一来源采购"},
		{RuleContent: "上传审批文件", ApplicabilityCondition: "竞争性采购"},
		{RuleContent: "上传审批文件"},
	})
	if err != nil || len(rules) != 3 || rules[0].ApplicabilityCondition != "单一来源采购" {
		t.Fatalf("前提被丢失或不同场景被误去重: %+v, %v", rules, err)
	}
}

func TestRuleConditionSourceFingerprintCompatibility(t *testing.T) {
	for _, module := range []string{"audit", "archive"} {
		t.Run(module, func(t *testing.T) {
			var before, after []ruleSource
			if module == "audit" {
				rule := model.AuditRule{RuleContent: "上传审批文件"}
				before = normalizedAuditRuleSources([]model.AuditRule{rule})
				rule.ApplicabilityCondition = "单一来源采购"
				after = normalizedAuditRuleSources([]model.AuditRule{rule})
			} else {
				rule := model.ArchiveRule{RuleContent: "上传审批文件"}
				before = normalizedArchiveRuleSources([]model.ArchiveRule{rule})
				rule.ApplicabilityCondition = "单一来源采购"
				after = normalizedArchiveRuleSources([]model.ArchiveRule{rule})
			}
			raw, _ := json.Marshal(before)
			if strings.Contains(string(raw), "applicability_condition") {
				t.Fatal("空前提不应改变旧配置指纹结构")
			}
			if stableJSONFingerprint(before) == stableJSONFingerprint(after) {
				t.Fatal("仅前提改变必须形成新的配置指纹")
			}
			encoded, _ := json.Marshal(after)
			var restored []ruleSource
			if err := json.Unmarshal(encoded, &restored); err != nil || restored[0].ApplicabilityCondition != "单一来源采购" {
				t.Fatalf("前提未能在版本快照中往返: %s, %v", encoded, err)
			}
		})
	}
}
