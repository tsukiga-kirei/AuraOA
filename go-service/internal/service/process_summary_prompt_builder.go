package service

import (
	"fmt"
	"strings"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/ai"
	"auraoa/go-service/internal/pkg/oa"
)

const fixedSummarySystemPrompt = `你是企业 OA 审批流程的总结助手。你的任务是基于给定流程字段、明细、附件识别内容和审批流信息，生成给审批人快速阅读的中文总结。

必须遵守：
1. 只根据输入内容总结，不要编造不存在的事实。
2. 涉及金额、日期、人员、供应商、项目、附件结论等关键信息时尽量保留原值。
3. 若字段为空或附件解析失败，需要明确写“未提供”或“附件解析失败”，不要猜测。
4. 输出必须是 JSON 对象，不要输出 Markdown 代码块、不要添加 JSON 外的解释。
5. JSON 格式固定为：{"content":"一段可直接展示的总结","points":["要点1","要点2"]}。`

// BuildSummaryBlockPrompt 组装单个总结块的 AI 请求。
func BuildSummaryBlockPrompt(
	processType string,
	processData *oa.ProcessData,
	flowSnapshot *oa.ProcessFlowSnapshot,
	block model.SummaryBlockConfig,
	fieldSet SelectedFieldSet,
	processSummary *oa.ProcessRequestSummary,
) *ai.ChatRequest {
	if processData == nil {
		processData = &oa.ProcessData{
			MainData:     map[string]interface{}{},
			DetailTables: map[string][]map[string]interface{}{},
			FieldLabels:  map[string]map[string]string{},
			Attachments:  []oa.AttachmentInfo{},
		}
	}
	mainDataStr := formatMainData(filterFields(processData.MainData, selectedKeysForTable(fieldSet, "main")), processData.FieldLabels["main"])
	detailDataStr := formatGroupedDetailData(processData.DetailTables, fieldSet, processData.FieldLabels)
	attachmentsStr := formatAttachments(filterAttachmentsForFieldSet(processData.Attachments, fieldSet), 10000)

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

	meta := "（未提供流程摘要）"
	if processSummary != nil {
		meta = fmt.Sprintf(
			"流程编号：%s\n流程标题：%s\n申请人：%s\n部门：%s\n流程类型：%s\n当前节点：%s\n提交时间：%s",
			processSummary.ProcessID,
			processSummary.Title,
			processSummary.Applicant,
			processSummary.Department,
			firstNonEmpty(processSummary.ProcessTypeLabel, processSummary.ProcessType, processType),
			processSummary.CurrentNode,
			processSummary.SubmitTime,
		)
	}
	if block.IncludeMeta != nil && !*block.IncludeMeta {
		meta = "（当前总结块未传入流程基础信息）"
	}

	userRequirements := substituteSummaryBlockUserPrompt(
		strings.TrimSpace(block.UserPrompt),
		meta,
		mainDataStr,
		detailDataStr,
		attachmentsStr,
		flowHistory,
		flowGraph,
	)

	userPrompt := fmt.Sprintf(`请生成「%s」总结块。

流程基础信息：
%s

主表字段：
%s

明细表字段：
%s

附件识别内容：
%s

审批流历史：
%s

审批流图：
%s

本总结块的用户要求：
%s

请严格返回 JSON 对象：{"content":"...","points":["..."]}`,
		block.Title,
		meta,
		mainDataStr,
		detailDataStr,
		attachmentsStr,
		flowHistory,
		flowGraph,
		userRequirements,
	)

	return &ai.ChatRequest{
		SystemPrompt: fixedSummarySystemPrompt,
		UserPrompt:   userPrompt,
		RequestType:  "summary",
		CallType:     "structured",
	}
}

func substituteSummaryBlockUserPrompt(
	userPrompt string,
	processMeta string,
	mainDataStr string,
	detailDataStr string,
	attachmentsStr string,
	flowHistory string,
	flowGraph string,
) string {
	userPrompt = strings.ReplaceAll(userPrompt, "{{process_meta}}", processMeta)
	userPrompt = strings.ReplaceAll(userPrompt, "{{main_table}}", mainDataStr)
	userPrompt = strings.ReplaceAll(userPrompt, "{{detail_tables}}", detailDataStr)
	userPrompt = strings.ReplaceAll(userPrompt, "{{attachments}}", attachmentsStr)
	userPrompt = strings.ReplaceAll(userPrompt, "{{flow_history}}", flowHistory)
	userPrompt = strings.ReplaceAll(userPrompt, "{{flow_graph}}", flowGraph)
	return replaceSystemPromptVariables(userPrompt)
}

func selectedKeysForTable(fieldSet SelectedFieldSet, table string) map[string]bool {
	if fieldSet == nil {
		return nil
	}
	return fieldSet[table]
}

func filterAttachmentsForFieldSet(attachments []oa.AttachmentInfo, fieldSet SelectedFieldSet) []oa.AttachmentInfo {
	if fieldSet == nil {
		return attachments
	}
	allowed := map[string]bool{}
	for _, keys := range fieldSet {
		if keys == nil {
			return attachments
		}
		for key := range keys {
			allowed[strings.ToLower(key)] = true
		}
	}
	if len(allowed) == 0 {
		return nil
	}
	out := make([]oa.AttachmentInfo, 0, len(attachments))
	for _, item := range attachments {
		if allowed[strings.ToLower(item.FieldKey)] {
			out = append(out, item)
		}
	}
	return out
}
