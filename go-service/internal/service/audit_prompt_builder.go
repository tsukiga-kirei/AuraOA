package service

import (
	"strings"

	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/ai"
	"oa-smart-audit/go-service/internal/pkg/oa"
)

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
		RequestType:  "audit",
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
		RequestType:  "audit",
	}
}
