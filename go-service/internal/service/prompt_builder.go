package service

import (
	"strings"

	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/ai"
)

// BuildPrompt 组装完整的 AI 审核请求（推理阶段）。
// system_reasoning_prompt → 系统角色消息，user_reasoning_prompt 渲染后 → 用户角色消息。
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
