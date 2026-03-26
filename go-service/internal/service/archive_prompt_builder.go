package service

import (
	"strings"

	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/ai"
	"oa-smart-audit/go-service/internal/pkg/oa"
)

// BuildArchiveReasoningPrompt 组装归档复盘推理阶段请求。
func BuildArchiveReasoningPrompt(
	aiConfig *model.ArchiveAIConfigData,
	processType string,
	processData *oa.ProcessData,
	rules string,
	currentNode string,
	fieldSet SelectedFieldSet,
	flowSnapshot *oa.ProcessFlowSnapshot,
) *ai.ChatRequest {
	mainDataStr := formatMainData(filterFields(processData.MainData, fieldSet["main"]))
	detailDataStr := formatGroupedDetailData(processData.DetailTables, fieldSet)

	flowHistory := "（暂未提供审批流历史）"
	flowGraph := "（暂未提供审批流图）"
	if flowSnapshot != nil {
		if strings.TrimSpace(flowSnapshot.HistoryText) != "" {
			flowHistory = flowSnapshot.HistoryText
		}
		if strings.TrimSpace(flowSnapshot.GraphText) != "" {
			flowGraph = flowSnapshot.GraphText
		}
	}

	userPrompt := aiConfig.UserReasoningPrompt
	userPrompt = strings.ReplaceAll(userPrompt, "{{process_type}}", processType)
	userPrompt = strings.ReplaceAll(userPrompt, "{{main_table}}", mainDataStr)
	userPrompt = strings.ReplaceAll(userPrompt, "{{fields}}", mainDataStr)
	userPrompt = strings.ReplaceAll(userPrompt, "{{detail_tables}}", detailDataStr)
	userPrompt = strings.ReplaceAll(userPrompt, "{{rules}}", rules)
	userPrompt = strings.ReplaceAll(userPrompt, "{{current_node}}", currentNode)
	userPrompt = strings.ReplaceAll(userPrompt, "{{flow_history}}", flowHistory)
	userPrompt = strings.ReplaceAll(userPrompt, "{{flow_graph}}", flowGraph)

	return &ai.ChatRequest{
		SystemPrompt: aiConfig.SystemReasoningPrompt,
		UserPrompt:   userPrompt,
		RequestType:  "archive",
	}
}

// BuildArchiveExtractionPrompt 组装归档复盘提取阶段请求。
func BuildArchiveExtractionPrompt(aiConfig *model.ArchiveAIConfigData, reasoningResult string, rules string) *ai.ChatRequest {
	userPrompt := aiConfig.UserExtractionPrompt
	userPrompt = strings.ReplaceAll(userPrompt, "{{reasoning_result}}", reasoningResult)
	userPrompt = strings.ReplaceAll(userPrompt, "{{rules}}", rules)

	return &ai.ChatRequest{
		SystemPrompt: aiConfig.SystemExtractionPrompt,
		UserPrompt:   userPrompt,
		RequestType:  "archive",
	}
}
