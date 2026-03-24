package service

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/ai"
	"oa-smart-audit/go-service/internal/pkg/oa"
)

// extractionPayload 提取阶段宽松解析：兼容 recommendation 与 overall_compliance 两套口径（见 docs/todo/detail-todo.md）。
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

// SelectedFieldSet 描述用户最终生效的选中字段集合。
// key 为 "main" 或明细表名（如 "formtable_main_151_dt1"），value 为字段 key 的 set。
// 当 set 为 nil 时表示该表全选所有字段。
type SelectedFieldSet map[string]map[string]bool

// BuildReasoningPrompt 组装推理阶段的 AI 审核请求。
func BuildReasoningPrompt(aiConfig *model.AIConfigData, processType string, processData *oa.ProcessData, rules string, currentNode string, fieldSet SelectedFieldSet) *ai.ChatRequest {
	mainDataStr := formatMainData(filterFields(processData.MainData, fieldSet["main"]))
	detailDataStr := formatGroupedDetailData(processData.DetailTables, fieldSet)

	userPrompt := aiConfig.UserReasoningPrompt
	userPrompt = strings.ReplaceAll(userPrompt, "{{process_type}}", processType)
	userPrompt = strings.ReplaceAll(userPrompt, "{{main_table}}", mainDataStr)
	userPrompt = strings.ReplaceAll(userPrompt, "{{fields}}", mainDataStr)
	userPrompt = strings.ReplaceAll(userPrompt, "{{detail_tables}}", detailDataStr)
	userPrompt = strings.ReplaceAll(userPrompt, "{{rules}}", rules)
	userPrompt = strings.ReplaceAll(userPrompt, "{{current_node}}", currentNode)
	userPrompt = strings.ReplaceAll(userPrompt, "{{flow_history}}", "（暂未提供）")
	userPrompt = strings.ReplaceAll(userPrompt, "{{flow_graph}}", "（暂未提供）")

	return &ai.ChatRequest{
		SystemPrompt: aiConfig.SystemReasoningPrompt,
		UserPrompt:   userPrompt,
	}
}

// BuildExtractionPrompt 组装提取阶段的 AI 审核请求。
func BuildExtractionPrompt(aiConfig *model.AIConfigData, reasoningResult string, rules string) *ai.ChatRequest {
	userPrompt := aiConfig.UserExtractionPrompt
	userPrompt = strings.ReplaceAll(userPrompt, "{{reasoning_result}}", reasoningResult)
	userPrompt = strings.ReplaceAll(userPrompt, "{{rules}}", rules)

	return &ai.ChatRequest{
		SystemPrompt: aiConfig.SystemExtractionPrompt,
		UserPrompt:   userPrompt,
	}
}

// BuildPrompt 保留向后兼容。
func BuildPrompt(aiConfig *model.AIConfigData, processType string, fields string, rules string) *ai.ChatRequest {
	userPrompt := aiConfig.UserReasoningPrompt
	userPrompt = strings.ReplaceAll(userPrompt, "{{process_type}}", processType)
	userPrompt = strings.ReplaceAll(userPrompt, "{{fields}}", fields)
	userPrompt = strings.ReplaceAll(userPrompt, "{{main_table}}", fields)
	userPrompt = strings.ReplaceAll(userPrompt, "{{rules}}", rules)

	return &ai.ChatRequest{
		SystemPrompt: aiConfig.SystemReasoningPrompt,
		UserPrompt:   userPrompt,
	}
}

// ParseAuditResult 解析 AI 提取阶段返回的 JSON 为结构化结果。
// 兼容：1) recommendation（approve/return/review）；2) overall_compliance（compliant/non_compliant/partially_compliant 等，与归档口径一致）；3) overall_score 与 score 互为补充。
func ParseAuditResult(raw string) (*model.AuditResultJSON, error) {
	cleaned := cleanJSONResponse(raw)
	var p extractionPayload
	if err := json.Unmarshal([]byte(cleaned), &p); err != nil {
		return nil, fmt.Errorf("JSON 解析失败: %w, 原始内容: %s", err, truncate(raw, 500))
	}

	out := &model.AuditResultJSON{
		RuleResults:       coalesceRuleResults(p.RuleResults),
		RiskPoints:        coalesceStringSlice(p.RiskPoints),
		Suggestions:       coalesceStringSlice(p.Suggestions),
		OverallScore:      pickOverallScoreInt(p.OverallScore, p.Score),
		OverallCompliance: strings.TrimSpace(p.OverallCompliance),
		Confidence:        clampPercentInt(p.Confidence),
	}

	rec := normalizeAuditRecommendation(strings.TrimSpace(p.Recommendation))
	if rec == "" {
		rec = recommendationFromOverallCompliance(p.OverallCompliance)
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

func coalesceStringSlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func pickOverallScoreInt(overall, score float64) int {
	if overall != 0 {
		return clampPercentInt(overall)
	}
	return clampPercentInt(score)
}

func clampPercentInt(v float64) int {
	if v <= 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return int(v + 0.5)
}

// normalizeAuditRecommendation 将常见别名转为 approve/return/review。
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

// recommendationFromOverallCompliance 将归档/合规口径映射为审核台 recommendation（与 archive 模块 compliant 族一致）。
func recommendationFromOverallCompliance(s string) string {
	if s == "" {
		return ""
	}
	lower := strings.ToLower(strings.TrimSpace(s))
	switch lower {
	case "compliant":
		return "approve"
	case "non_compliant", "noncompliant", "not_compliant", "incompliant":
		return "return"
	case "partially_compliant", "partial_compliant", "partial", "partial_compliance":
		return "review"
	default:
		return ""
	}
}

// ── 字段过滤 ──

// filterFields 从 map 中只保留 allowedKeys 指定的字段。
// 当 allowedKeys 为 nil 时，返回原始 data（全选）。
func filterFields(data map[string]interface{}, allowedKeys map[string]bool) map[string]interface{} {
	if data == nil {
		return nil
	}
	if allowedKeys == nil {
		return data
	}
	filtered := make(map[string]interface{})
	for k, v := range data {
		normalKey := strings.ToLower(k)
		if allowedKeys[normalKey] || allowedKeys[k] {
			filtered[k] = v
		}
	}
	return filtered
}

// filterRowFields 对一组明细行批量做字段过滤。
func filterRowFields(rows []map[string]interface{}, allowedKeys map[string]bool) []map[string]interface{} {
	if allowedKeys == nil {
		return rows
	}
	result := make([]map[string]interface{}, len(rows))
	for i, row := range rows {
		result[i] = filterFields(row, allowedKeys)
	}
	return result
}

// ── 格式化 ──

func formatMainData(data map[string]interface{}) string {
	if len(data) == 0 {
		return "（无主表数据）"
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", data)
	}
	return string(b)
}

// formatGroupedDetailData 将按表分组的明细数据格式化为带表名标签的文本。
func formatGroupedDetailData(detailTables map[string][]map[string]interface{}, fieldSet SelectedFieldSet) string {
	if len(detailTables) == 0 {
		return "（无明细表数据）"
	}
	// 按表名排序保证输出稳定
	tableNames := make([]string, 0, len(detailTables))
	for name := range detailTables {
		tableNames = append(tableNames, name)
	}
	sort.Strings(tableNames)

	var sb strings.Builder
	for _, tableName := range tableNames {
		rows := detailTables[tableName]
		// 从表名提取友好标签（如 formtable_main_151_dt1 → 明细表1）
		label := tableName
		if idx := strings.LastIndex(tableName, "_dt"); idx != -1 && idx+3 < len(tableName) {
			label = "明细表" + tableName[idx+3:]
		}

		// 按用户选择的字段过滤
		var allowedKeys map[string]bool
		if fieldSet != nil {
			allowedKeys = fieldSet[tableName]
		}
		filteredRows := filterRowFields(rows, allowedKeys)

		sb.WriteString(fmt.Sprintf("### %s（%s）共 %d 行\n", label, tableName, len(filteredRows)))
		b, err := json.MarshalIndent(filteredRows, "", "  ")
		if err != nil {
			sb.WriteString(fmt.Sprintf("%v\n", filteredRows))
		} else {
			sb.Write(b)
			sb.WriteByte('\n')
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

// ── 辅助函数 ──

func cleanJSONResponse(raw string) string {
	s := strings.TrimSpace(raw)
	if idx := strings.Index(s, "```json"); idx >= 0 {
		s = s[idx+7:]
		if end := strings.Index(s, "```"); end >= 0 {
			s = s[:end]
		}
	} else if idx := strings.Index(s, "```"); idx >= 0 {
		s = s[idx+3:]
		if end := strings.Index(s, "```"); end >= 0 {
			s = s[:end]
		}
	}
	s = strings.TrimSpace(s)
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		s = s[start : end+1]
	}
	return s
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
