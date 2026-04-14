package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"oa-smart-audit/go-service/internal/model"
)

// archiveExtractionPayload 归档复盘提取阶段的宽松解析结构。
// 分数字段使用 float64 避免模型输出 85.0 时整型反序列化失败。
// 同时兼容 rule_audit（标准格式）和 rule_results（dashboard 风格）两种规则结果格式。
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

// archiveRuleResultPayload 兼容 dashboard 风格的 rule_results 格式，
// 当模型未输出标准 rule_audit 时作为回退解析目标。
type archiveRuleResultPayload struct {
	RuleContent string `json:"rule_content"`
	Passed      bool   `json:"passed"`
	Reason      string `json:"reason"`
}

// ParseArchiveReviewResult 解析归档复盘提取阶段的 AI 输出，转换为结构化结果对象。
// 先清理原始文本（去除 markdown 包裹、省略号等），再反序列化并校验必填字段。
// overall_compliance 缺失或无法归一化时返回错误，要求模型重新输出。
func ParseArchiveReviewResult(raw string) (*model.ArchiveResultJSON, error) {
	cleaned := cleanJSONResponse(raw)
	var payload archiveExtractionPayload
	if err := json.Unmarshal([]byte(cleaned), &payload); err != nil {
		return nil, fmt.Errorf("JSON 解析失败: %w, 原始内容: %s", err, truncate(raw, 500))
	}

	compliance := normalizeArchiveCompliance(payload.OverallCompliance)
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

// normalizeArchiveCompliance 将模型输出的合规结论别名统一归一化为三种标准值：
// compliant（合规）、non_compliant（不合规）、partially_compliant（部分合规）。
// 无法识别的值返回空字符串，由调用方决定是否报错。
func normalizeArchiveCompliance(compliance string) string {
	if compliance == "" {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(compliance)) {
	case "compliant":
		return "compliant"
	case "non_compliant", "noncompliant", "not_compliant", "incompliant":
		return "non_compliant"
	case "partially_compliant", "partial_compliant", "partial", "partial_compliance":
		return "partially_compliant"
	default:
		return ""
	}
}
