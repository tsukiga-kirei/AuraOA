package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"oa-smart-audit/go-service/internal/model"
)

// extractionPayload 提取阶段宽松解析：主字段为 recommendation；若模型误用 overall_compliance 则映射为 recommendation。
// 分数类字段用 float64，避免部分模型输出 85.0 导致整型反序列化失败。
type extractionPayload struct {
	Recommendation    string                 `json:"recommendation"`
	OverallCompliance string                 `json:"overall_compliance"`
	OverallScore      float64                `json:"overall_score"`
	Score             float64                `json:"score"`
	RuleResults       []model.RuleResultJSON `json:"rule_results"`
	RiskPoints        []string               `json:"risk_points"`
	Suggestions       []string               `json:"suggestions"`
	Confidence        float64                `json:"confidence"`
}

// ParseAuditResult 解析 AI 提取阶段返回的 JSON 为结构化结果。
// 兼容：1) recommendation（approve/return/review）；2) 仅 overall_compliance（compliant 族）时映射为 recommendation；3) overall_score 与 score 互为补充。
func ParseAuditResult(raw string) (*model.AuditResultJSON, error) {
	cleaned := cleanJSONResponse(raw)
	var p extractionPayload
	if err := json.Unmarshal([]byte(cleaned), &p); err != nil {
		return nil, fmt.Errorf("JSON 解析失败: %w, 原始内容: %s", err, truncate(raw, 500))
	}

	out := &model.AuditResultJSON{
		RuleResults:  coalesceRuleResults(p.RuleResults),
		RiskPoints:   coalesceStringSlice(p.RiskPoints),
		Suggestions:  coalesceStringSlice(p.Suggestions),
		OverallScore: pickOverallScoreInt(p.OverallScore, p.Score),
		Confidence:   clampPercentInt(p.Confidence),
	}

	rec := normalizeAuditRecommendation(strings.TrimSpace(p.Recommendation))
	if rec == "" {
		rec = mapComplianceAliasToRecommendation(p.OverallCompliance)
	}
	if rec == "" {
		return nil, fmt.Errorf("缺少有效结论：请提供 recommendation（approve/return/review）或 overall_compliance（如 compliant/non_compliant/partially_compliant）")
	}
	if rec != "approve" && rec != "return" && rec != "review" {
		return nil, fmt.Errorf("审核结论无法归一化: recommendation=%q overall_compliance=%q", p.Recommendation, p.OverallCompliance)
	}
	out.Recommendation = rec

	return out, nil
}

func coalesceRuleResults(r []model.RuleResultJSON) []model.RuleResultJSON {
	if r == nil {
		return []model.RuleResultJSON{}
	}
	return r
}
