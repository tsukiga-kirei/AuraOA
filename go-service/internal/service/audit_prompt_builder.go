package service

import (
	"strings"

	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/ai"
	"oa-smart-audit/go-service/internal/pkg/oa"
)

// BuildReasoningPrompt 组装审核推理阶段的 AI 请求。
// 将主表数据、明细表数据、规则文本、当前节点、审批流历史和流程图注入用户提示词模板的占位符中。
// flowSnapshot 为 nil 或内容为空时使用默认占位文本，不影响推理执行。
func BuildReasoningPrompt(aiConfig *model.AIConfigData, processType string, processData *oa.ProcessData, rules string, currentNode string, fieldSet SelectedFieldSet, flowSnapshot *oa.ProcessFlowSnapshot) *ai.ChatRequest {
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
		RequestType:  "audit",
	}
}

// BuildExtractionPrompt 组装审核提取阶段的 AI 请求。
// 将推理阶段输出和规则文本注入提取提示词模板，引导模型输出结构化 JSON 审核结论。
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
