package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"oa-smart-audit/go-service/internal/model"
)

// extractionPayload 审核提取阶段的宽松解析结构，核心字段为 recommendation（审核结论）。
// 分数字段使用 float64，避免模型输出 85.0 时整型反序列化失败。
type extractionPayload struct {
	Recommendation string                 `json:"recommendation"`
	OverallScore   float64                `json:"overall_score"`
	Score          float64                `json:"score"`
	RuleResults    []model.RuleResultJSON `json:"rule_results"`
	RiskPoints     []string               `json:"risk_points"`
	Suggestions    []string               `json:"suggestions"`
	Confidence     float64                `json:"confidence"`
}

// ParseAuditResult 解析 AI 提取阶段返回的 JSON，转换为结构化审核结论。
// recommendation 字段缺失或无法归一化为 approve/return/review 时返回错误。
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
		return nil, fmt.Errorf("缺少有效结论：请提供 recommendation（approve/return/review）")
	}
	if rec != "approve" && rec != "return" && rec != "review" {
		return nil, fmt.Errorf("审核结论无法归一化: recommendation=%q", p.Recommendation)
	}
	out.Recommendation = rec

	return out, nil
}

// coalesceRuleResults 将 nil 规则结果切片转为空切片，确保 JSON 序列化输出 [] 而非 null。
func coalesceRuleResults(r []model.RuleResultJSON) []model.RuleResultJSON {
	if r == nil {
		return []model.RuleResultJSON{}
	}
	return r
}

// normalizeAuditRecommendation 将模型输出的审核结论别名统一归一化为三种标准值：
// approve（通过）、return（退回）、review（人工复核）。
// 无法识别的值原样返回小写，由调用方决定是否报错。
func normalizeAuditRecommendation(s string) string {
	if s == "" {
		return ""
	}
	lower := strings.ToLower(strings.TrimSpace(s))
	switch lower {
	case "approve", "approved", "pass", "通过", "同意", "批准":
		return "approve"
	case "return", "returned", "reject", "rejected", "退回", "拒绝":
		return "return"
	case "review", "pending_review", "manual", "复核", "待复核", "人工":
		return "review"
	default:
		return lower
	}
}
