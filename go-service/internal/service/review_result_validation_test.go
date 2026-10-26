package service

import (
	"encoding/json"
	"strings"
	"testing"

	"auraoa/go-service/internal/model"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

const validReviewAudit = `{"recommendation":"approve","overall_score":95,"confidence":90,"rule_results":[{"rule_content":"检查金额","passed":true,"reason":"一致"}],"risk_points":[],"suggestions":[]}`
const validReviewArchive = `{"overall_compliance":"compliant","overall_score":95,"confidence":90,"flow_audit":{"is_complete":true,"missing_nodes":[],"node_results":[{"node_id":"finance","node_name":"财务","compliant":true,"reasoning":"已审批"}]},"field_audit":[{"field_key":"amount","field_name":"金额","passed":true,"reasoning":"一致"}],"rule_audit":[{"rule_id":"r1","rule_name":"检查金额","passed":true,"reasoning":"一致"}],"risk_points":[],"suggestions":[],"ai_summary":"核验完成"}`

func TestReviewParsersRejectMissingResults(t *testing.T) {
	for _, raw := range []string{"", "{}", "null", "[]", `{"overall_score":95}`} {
		if result, err := ParseAuditResult(raw); err == nil || result != nil {
			t.Fatalf("审核空结果不能形成业务结论：%q, %+v, %v", raw, result, err)
		}
		if result, err := ParseArchiveReviewResult(raw); err == nil || result != nil {
			t.Fatalf("归档空结果不能形成业务结论：%q, %+v, %v", raw, result, err)
		}
	}
	for _, module := range []string{"audit", "archive"} {
		initial := validReviewAudit
		if module == "archive" {
			initial = validReviewArchive
		}
		var base map[string]json.RawMessage
		_ = json.Unmarshal([]byte(initial), &base)
		for _, key := range []string{"overall_score", "confidence", "risk_points", "suggestions"} {
			for _, value := range []string{"", "null", `"wrong"`} {
				fields := map[string]json.RawMessage{}
				for k, v := range base {
					fields[k] = v
				}
				if value == "" {
					delete(fields, key)
				} else {
					fields[key] = json.RawMessage(value)
				}
				raw, _ := json.Marshal(fields)
				var err error
				if module == "archive" {
					_, err = ParseArchiveReviewResult(string(raw))
				} else {
					_, err = ParseAuditResult(string(raw))
				}
				if err == nil {
					t.Errorf("%s 接受无效字段 %s=%s", module, key, value)
				}
			}
		}
	}
}

func TestReviewParsersKeepValidZeroAndLegacyScoreAlias(t *testing.T) {
	for _, raw := range []string{
		strings.ReplaceAll(strings.Replace(validReviewAudit, "95", "0", 1), `"approve"`, `"return"`),
		strings.ReplaceAll(strings.Replace(validReviewAudit, "95", "0.0", 1), `"approve"`, `"return"`),
	} {
		result, err := ParseAuditResult(raw)
		if err != nil || result.OverallScore != 0 || result.Recommendation != "return" {
			t.Fatalf("合法零分被拒绝：%+v %v", result, err)
		}
	}
	aliased := strings.Replace(validReviewAudit, `"overall_score":95`, `"score":85.0`, 1)
	if result, err := ParseAuditResult(aliased); err != nil || result.OverallScore != 85 {
		t.Fatalf("旧分数字段不兼容：%+v %v", result, err)
	}
	zero := strings.Replace(validReviewAudit, `"overall_score":95`, `"overall_score":0,"score":95`, 1)
	if result, err := ParseAuditResult(zero); err != nil || result.OverallScore != 0 {
		t.Fatalf("显式零分不能回退到别名：%+v %v", result, err)
	}
}

func TestReviewRuleCoverageRejectsOmission(t *testing.T) {
	audit, _ := ParseAuditResult(validReviewAudit)
	archive, _ := ParseArchiveReviewResult(validReviewArchive)
	auditRules := []model.AuditRule{{RuleContent: "检查金额"}, {RuleContent: "检查审批"}}
	archiveRules := []model.ArchiveRule{{RuleContent: "检查金额"}, {RuleContent: "检查审批"}}
	if validateAuditRuleCoverage(audit, auditRules) == nil || validateArchiveRuleCoverage(archive, archiveRules) == nil {
		t.Fatal("不能遗漏本次启用规则")
	}
	disabled := false
	auditRules[1].Enabled, archiveRules[1].Enabled = &disabled, &disabled
	if err := validateAuditRuleCoverage(audit, auditRules); err != nil {
		t.Fatal(err)
	}
	if err := validateArchiveRuleCoverage(archive, archiveRules); err != nil {
		t.Fatal(err)
	}
	// 个人规则也在提示词中，不能将合法追加规则误判为结构错误。
	audit.RuleResults = append(audit.RuleResults, model.RuleResultJSON{RuleContent: "个人规则", Passed: true})
	if err := validateAuditRuleCoverage(audit, auditRules); err != nil {
		t.Fatal(err)
	}
	archiveRules[0].ID = uuid.New()
	archive.RuleAudit[0].RuleID = archiveRules[0].ID.String()
	archive.RuleAudit[0].RuleName = "金额核验"
	if err := validateArchiveRuleCoverage(archive, archiveRules); err != nil {
		t.Fatalf("有明确规则 ID 时应允许展示名简写：%v", err)
	}
}

func TestArchiveRecheckPreservesEvaluationObjects(t *testing.T) {
	for _, pair := range [][2]string{
		{`"flow_audit":{"is_complete":true,"missing_nodes":[],"node_results":[{"node_id":"finance","node_name":"财务","compliant":true,"reasoning":"已审批"}]}`, `"flow_audit":null`},
		{`"field_audit":[{"field_key":"amount","field_name":"金额","passed":true,"reasoning":"一致"}]`, `"field_audit":null`},
		{`"field_audit":[{"field_key":"amount","field_name":"金额","passed":true,"reasoning":"一致"}]`, `"field_audit":[]`},
		{`"node_results":[{"node_id":"finance","node_name":"财务","compliant":true,"reasoning":"已审批"}]`, `"node_results":[]`},
		{`"node_id":"finance"`, `"node_id":"other"`},
		{`"field_key":"amount"`, `"field_key":"other"`},
	} {
		final := strings.Replace(validReviewArchive, pair[0], pair[1], 1)
		if final == validReviewArchive {
			t.Fatal("测试样例未替换")
		}
		if err := validateAIRecheckResult(json.RawMessage(final), json.RawMessage(validReviewArchive), "archive"); err == nil {
			t.Errorf("复核不能丢失核验对象：%s", pair[1])
		}
	}
	final := strings.ReplaceAll(strings.ReplaceAll(validReviewArchive, `"passed":true`, `"passed":false`), "一致", "存在差异")
	if err := validateAIRecheckResult(json.RawMessage(final), json.RawMessage(validReviewArchive), "archive"); err != nil {
		t.Fatalf("复核应允许改变判断与说明：%v", err)
	}
}

func TestFailedReviewResponseHasNoBusinessConclusion(t *testing.T) {
	id := uuid.New()
	audit := &model.AuditLog{ID: id, Status: model.JobStatusFailed, ParseError: "AI 未返回有效结果", AuditResult: datatypes.JSON(`{}`)}
	archive := &model.ArchiveLog{ID: id, Status: model.JobStatusFailed, ParseError: audit.ParseError, ArchiveResult: datatypes.JSON(`{}`)}
	for _, result := range []map[string]interface{}{buildAuditResultFromLog(audit), buildArchiveResultFromLog(archive)} {
		for _, key := range []string{"overall_score", "recommendation", "overall_compliance", "confidence"} {
			if _, ok := result[key]; ok {
				t.Errorf("失败响应不应生成业务字段 %s：%v", key, result)
			}
		}
		if result["error_message"] != audit.ParseError {
			t.Errorf("失败原因未提供给页面：%v", result)
		}
	}
}
