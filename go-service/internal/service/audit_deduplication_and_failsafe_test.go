package service

import (
	"testing"

	"auraoa/go-service/internal/model"
)

func TestParseAuditResult_DeductionsAndFailSafe(t *testing.T) {
	// 场景 1：模型返回 85 分，带 deductions，但 risk_points 和 suggestions 为空
	raw := `{
		"recommendation": "review",
		"overall_score": 85,
		"confidence": 90,
		"deductions": [
			{
				"item": "未附发票",
				"points": 15,
				"reason": "单笔超过5000元未上传增值税发票"
			}
		],
		"rule_results": [
			{
				"rule_content": "单笔超5000需发票",
				"passed": false,
				"reason": "未上传发票附件"
			}
		],
		"risk_points": [],
		"suggestions": []
	}`

	parsed, err := ParseAuditResult(raw)
	if err != nil {
		t.Fatalf("ParseAuditResult 失败: %v", err)
	}
	if parsed.OverallScore != 85 {
		t.Errorf("期望得分 85，实际得到 %d", parsed.OverallScore)
	}
	if len(parsed.RiskPoints) == 0 {
		t.Errorf("触发了自检兜底，RiskPoints 不应为空")
	}
	if parsed.RiskPoints[0] != "未附发票：单笔超过5000元未上传增值税发票" {
		t.Errorf("RiskPoints 补充内容不符合预期: %v", parsed.RiskPoints)
	}

	// 场景 2：模型返回 85 分，没有 deductions 字段，且 risk_points 为空，自动从未通过规则补充
	raw2 := `{
		"recommendation": "review",
		"overall_score": 85,
		"confidence": 90,
		"rule_results": [
			{
				"rule_content": "单笔超5000需发票",
				"passed": false,
				"reason": "未检测到有效发票附件"
			}
		],
		"risk_points": [],
		"suggestions": []
	}`

	parsed2, err := ParseAuditResult(raw2)
	if err != nil {
		t.Fatalf("ParseAuditResult 失败: %v", err)
	}
	if len(parsed2.RiskPoints) == 0 {
		t.Errorf("触发了未通过规则兜底，RiskPoints 不应为空")
	}

	// 场景 3：满分 100 分，允许两项皆为空
	raw100 := `{
		"recommendation": "approve",
		"overall_score": 100,
		"confidence": 95,
		"rule_results": [
			{
				"rule_content": "规则1",
				"passed": true,
				"reason": "合规"
			}
		],
		"risk_points": [],
		"suggestions": []
	}`
	parsed100, err := ParseAuditResult(raw100)
	if err != nil {
		t.Fatalf("ParseAuditResult 失败: %v", err)
	}
	if len(parsed100.RiskPoints) != 0 {
		t.Errorf("满分 100 分时 RiskPoints 应保持为空，实际为: %v", parsed100.RiskPoints)
	}
}

func TestParseArchiveReviewResult_FailSafe(t *testing.T) {
	raw := `{
		"overall_compliance": "partially_compliant",
		"overall_score": 75,
		"confidence": 85,
		"deductions": [
			{
				"item": "缺少验收单",
				"points": 25,
				"reason": "合同完工未见最终验收签署单"
			}
		],
		"flow_audit": {
			"is_complete": true
		},
		"rule_audit": [
			{
				"rule_name": "完工验收单检查",
				"passed": false,
				"reasoning": "未见验收签署单"
			}
		],
		"risk_points": [],
		"suggestions": []
	}`

	parsed, err := ParseArchiveReviewResult(raw)
	if err != nil {
		t.Fatalf("ParseArchiveReviewResult 失败: %v", err)
	}
	if parsed.OverallScore != 75 {
		t.Errorf("期望得分 75，实际得到 %d", parsed.OverallScore)
	}
	if len(parsed.RiskPoints) == 0 {
		t.Errorf("归档解析自检兜底未生效，RiskPoints 不应为空")
	}
	if parsed.RiskPoints[0] != "缺少验收单：合同完工未见最终验收签署单" {
		t.Errorf("RiskPoints 补充内容不符合预期: %v", parsed.RiskPoints)
	}
}

func TestResolveAuditRulesExternalContext_Deduplication(t *testing.T) {
	svc := &AuditExecuteService{}
	// 如果 externalCtx 为 nil 或规则为空，返回空字符串
	if got := svc.resolveAuditRulesExternalContext(nil, nil, "p1", nil, nil); got != "" {
		t.Errorf("期望空字符串，实际得到: %s", got)
	}

	rules := []model.AuditRule{
		{
			RuleContent:    "规则1：核查合同额",
			ContextEnabled: true,
			ContextMounts:  []byte(`[{"type":"workflow","name":"合同流程","source_field":"main:contract_id","enabled":true}]`),
		},
		{
			RuleContent:    "规则2：核查合同付款条款",
			ContextEnabled: true,
			ContextMounts:  []byte(`[{"type":"workflow","name":"合同流程","source_field":"main:contract_id","enabled":true}]`),
		},
	}

	// 测试结构初始化无崩溃
	if got := svc.resolveAuditRulesExternalContext(nil, nil, "p1", nil, rules); got != "" {
		t.Errorf("externalCtx 为空时应返回空，实际得到: %s", got)
	}
}
