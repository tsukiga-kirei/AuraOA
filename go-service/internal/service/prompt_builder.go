package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/ai"
	"oa-smart-audit/go-service/internal/pkg/oa"
)

// BuildReasoningPrompt 组装推理阶段的 AI 审核请求。
// system_reasoning_prompt → 系统角色消息，user_reasoning_prompt 渲染后 → 用户角色消息。
func BuildReasoningPrompt(aiConfig *model.AIConfigData, processType string, processData *oa.ProcessData, rules string, currentNode string) *ai.ChatRequest {
	mainDataStr := formatMainData(processData.MainData)
	detailDataStr := formatDetailData(processData.DetailData)

	userPrompt := aiConfig.UserReasoningPrompt
	userPrompt = strings.ReplaceAll(userPrompt, "{{process_type}}", processType)
	userPrompt = strings.ReplaceAll(userPrompt, "{{main_table}}", mainDataStr)
	userPrompt = strings.ReplaceAll(userPrompt, "{{fields}}", mainDataStr)
	userPrompt = strings.ReplaceAll(userPrompt, "{{detail_tables}}", detailDataStr)
	userPrompt = strings.ReplaceAll(userPrompt, "{{rules}}", rules)
	userPrompt = strings.ReplaceAll(userPrompt, "{{current_node}}", currentNode)
	// TODO: {{flow_history}} 和 {{flow_graph}} 暂置空，待审批流功能实现后补充
	userPrompt = strings.ReplaceAll(userPrompt, "{{flow_history}}", "（暂未提供）")
	userPrompt = strings.ReplaceAll(userPrompt, "{{flow_graph}}", "（暂未提供）")

	return &ai.ChatRequest{
		SystemPrompt: aiConfig.SystemReasoningPrompt,
		UserPrompt:   userPrompt,
	}
}

// BuildExtractionPrompt 组装提取阶段的 AI 审核请求。
// 将推理阶段的输出作为 {{reasoning_result}} 注入提取阶段的用户提示词。
func BuildExtractionPrompt(aiConfig *model.AIConfigData, reasoningResult string, rules string) *ai.ChatRequest {
	userPrompt := aiConfig.UserExtractionPrompt
	userPrompt = strings.ReplaceAll(userPrompt, "{{reasoning_result}}", reasoningResult)
	userPrompt = strings.ReplaceAll(userPrompt, "{{rules}}", rules)

	return &ai.ChatRequest{
		SystemPrompt: aiConfig.SystemExtractionPrompt,
		UserPrompt:   userPrompt,
	}
}

// BuildPrompt 保留向后兼容，等同于 BuildReasoningPrompt 的简化版。
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
// 包含宽松解析：自动提取 markdown 代码块中的 JSON，容忍前后多余文字。
func ParseAuditResult(raw string) (*model.AuditResultJSON, error) {
	cleaned := cleanJSONResponse(raw)
	var result model.AuditResultJSON
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return nil, fmt.Errorf("JSON 解析失败: %w, 原始内容: %s", err, truncate(raw, 500))
	}
	if result.Recommendation == "" {
		return nil, fmt.Errorf("缺少 recommendation 字段")
	}
	if result.Recommendation != "approve" && result.Recommendation != "return" && result.Recommendation != "review" {
		return nil, fmt.Errorf("recommendation 值无效: %s", result.Recommendation)
	}
	return &result, nil
}

func cleanJSONResponse(raw string) string {
	s := strings.TrimSpace(raw)
	// 提取 markdown 代码块中的 JSON
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
	// 找第一个 { 和最后一个 }
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		s = s[start : end+1]
	}
	return s
}

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

func formatDetailData(data []map[string]interface{}) string {
	if len(data) == 0 {
		return "（无明细表数据）"
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", data)
	}
	return string(b)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
