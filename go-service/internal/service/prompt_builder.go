package service

import (
	"strings"

	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/ai"
)

// BuildPrompt 组装完整的 AI 审核请求。
// system_prompt → 系统角色消息，user_prompt_template 渲染后 → 用户角色消息。
func BuildPrompt(aiConfig *model.AIConfigData, processType string, fields string, rules string) *ai.ChatRequest {
	// 渲染用户提示词模板，替换变量
	userPrompt := aiConfig.UserPromptTemplate
	userPrompt = strings.ReplaceAll(userPrompt, "{{process_type}}", processType)
	userPrompt = strings.ReplaceAll(userPrompt, "{{fields}}", fields)
	userPrompt = strings.ReplaceAll(userPrompt, "{{rules}}", rules)

	return &ai.ChatRequest{
		SystemPrompt: aiConfig.SystemPrompt,
		UserPrompt:   userPrompt,
	}
}
