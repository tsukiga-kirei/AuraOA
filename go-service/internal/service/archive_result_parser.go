package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"oa-smart-audit/go-service/internal/model"
)

type archiveExtractionPayload struct {
	Recommendation    string                     `json:"recommendation"`
	OverallCompliance string                     `json:"overall_compliance"`
	OverallScore      float64                    `json:"overall_score"`
	Score             float64                    `json:"score"`
	Confidence        float64                    `json:"confidence"`
	FlowAudit         archiveFlowAuditPayload    `json:"flow_audit"`
	FieldAudit        []archiveFieldAuditPayload `json:"field_audit"`
	RuleAudit         []archiveRuleAuditPayload  `json:"rule_audit"`
	RuleResults       []archiveRuleResultPayload `json:"rule_results"` // dashboard 格式兼容
	RiskPoints        []string                   `json:"risk_points"`
	Suggestions       []string                   `json:"suggestions"`
	AISummary         string                     `json:"ai_summary"`
	Summary           string                     `json:"summary"`
}

type archiveFlowAuditPayload struct {
	IsComplete   bool                           `json:"is_complete"`
	MissingNodes []string                       `json:"missing_nodes"`
	NodeResults  []archiveFlowNodeResultPayload `json:"node_results"`
}

type archiveFlowNodeResultPayload struct {
	NodeID    string `json:"node_id"`
	NodeName  string `json:"node_name"`
	Compliant bool   `json:"compliant"`
	Reasoning string `json:"reasoning"`
}

type archiveFieldAuditPayload struct {
	FieldKey  string `json:"field_key"`
	FieldName string `json:"field_name"`
	Passed    bool   `json:"passed"`
	Reasoning string `json:"reasoning"`
}

type archiveRuleAuditPayload struct {
	RuleID    string `json:"rule_id"`
	RuleName  string `json:"rule_name"`
	Passed    bool   `json:"passed"`
	Reasoning string `json:"reasoning"`
}

// archiveRuleResultPayload 兼容 dashboard 风格的 rule_results 格式
type archiveRuleResultPayload struct {
	RuleContent string `json:"rule_content"`
	Passed      bool   `json:"passed"`
	Reason      string `json:"reason"`
}

// ParseArchiveReviewResult 解析归档复盘提取结果。
func ParseArchiveReviewResult(raw string) (*model.ArchiveResultJSON, error) {
	cleaned := cleanJSONResponse(raw)
	var payload archiveExtractionPayload
	if err := json.Unmarshal([]byte(cleaned), &payload); err != nil {
		return nil, fmt.Errorf("JSON 解析失败: %w, 原始内容: %s", err, truncate(raw, 500))
	}

	compliance := normalizeArchiveCompliance(payload.OverallCompliance, payload.Recommendation)
	if compliance == "" {
		return nil, fmt.Errorf("缺少有效结论：请提供 overall_compliance（compliant/non_compliant/partially_compliant）")
	}

	result := &model.ArchiveResultJSON{
		OverallCompliance: compliance,
		OverallScore:      pickOverallScoreInt(payload.OverallScore, payload.Score),
		Confidence:        clampPercentInt(payload.Confidence),
		FlowAudit: model.ArchiveFlowAuditJSON{
			IsComplete:   payload.FlowAudit.IsComplete,
			MissingNodes: coalesceStringSlice(payload.FlowAudit.MissingNodes),
			NodeResults:  make([]model.ArchiveFlowNodeResultJSON, 0, len(payload.FlowAudit.NodeResults)),
		},
		FieldAudit:  make([]model.ArchiveFieldAuditJSON, 0, len(payload.FieldAudit)),
		RuleAudit:   make([]model.ArchiveRuleAuditJSON, 0, len(payload.RuleAudit)),
		RiskPoints:  coalesceStringSlice(payload.RiskPoints),
		Suggestions: coalesceStringSlice(payload.Suggestions),
		AISummary:   strings.TrimSpace(firstNonEmpty(payload.AISummary, payload.Summary)),
	}

	for _, item := range payload.FlowAudit.NodeResults {
		result.FlowAudit.NodeResults = append(result.FlowAudit.NodeResults, model.ArchiveFlowNodeResultJSON{
			NodeID:    firstNonEmpty(item.NodeID, item.NodeName),
			NodeName:  item.NodeName,
			Compliant: item.Compliant,
			Reasoning: item.Reasoning,
		})
	}
	for _, item := range payload.FieldAudit {
		result.FieldAudit = append(result.FieldAudit, model.ArchiveFieldAuditJSON{
			FieldKey:  firstNonEmpty(item.FieldKey, item.FieldName),
			FieldName: item.FieldName,
			Passed:    item.Passed,
			Reasoning: item.Reasoning,
		})
	}
	for _, item := range payload.RuleAudit {
		result.RuleAudit = append(result.RuleAudit, model.ArchiveRuleAuditJSON{
			RuleID:    firstNonEmpty(item.RuleID, item.RuleName),
			RuleName:  item.RuleName,
			Passed:    item.Passed,
			Reasoning: item.Reasoning,
		})
	}
	// 兼容 dashboard 风格 rule_results：当 rule_audit 为空且 rule_results 非空时回退
	if len(result.RuleAudit) == 0 && len(payload.RuleResults) > 0 {
		for _, item := range payload.RuleResults {
			result.RuleAudit = append(result.RuleAudit, model.ArchiveRuleAuditJSON{
				RuleID:    item.RuleContent,
				RuleName:  item.RuleContent,
				Passed:    item.Passed,
				Reasoning: item.Reason,
			})
		}
	}

	return result, nil
}

func normalizeArchiveCompliance(compliance string, recommendation string) string {
	switch strings.ToLower(strings.TrimSpace(compliance)) {
	case "compliant":
		return "compliant"
	case "non_compliant", "noncompliant", "not_compliant", "incompliant":
		return "non_compliant"
	case "partially_compliant", "partial_compliant", "partial", "partial_compliance":
		return "partially_compliant"
	}

	switch normalizeAuditRecommendation(recommendation) {
	case "approve":
		return "compliant"
	case "return":
		return "non_compliant"
	case "review":
		return "partially_compliant"
	default:
		return ""
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
