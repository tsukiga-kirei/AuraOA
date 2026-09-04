package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/datatypes"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/agenttools"
	"auraoa/go-service/internal/pkg/ai"
	pkglogger "auraoa/go-service/internal/pkg/logger"
	"auraoa/go-service/internal/repository"
)

const maxAgentLoopSteps = 8

// AgentRuntimeService 负责智能体单轮对话的编排循环（包括工具多步调用、状态流式推送与落库）
type AgentRuntimeService struct {
	chatRepo        *repository.ChatRepo
	agentRepo       *repository.AgentRepo
	tenantRepo      *repository.TenantRepo
	aiModelRepo     *repository.AIModelRepo
	aiCaller        *AIModelCallerService
	permService     *AgentPermissionService
	skillService    *SkillService
	mcpService      *MCPService
	toolExecutor    agenttools.ToolExecutor
}

// NewAgentRuntimeService 初始化智能体运行时服务
func NewAgentRuntimeService(
	chatRepo *repository.ChatRepo,
	agentRepo *repository.AgentRepo,
	tenantRepo *repository.TenantRepo,
	aiModelRepo *repository.AIModelRepo,
	aiCaller *AIModelCallerService,
	permService *AgentPermissionService,
	skillService *SkillService,
	mcpService *MCPService,
	toolExecutor agenttools.ToolExecutor,
) *AgentRuntimeService {
	return &AgentRuntimeService{
		chatRepo:     chatRepo,
		agentRepo:    agentRepo,
		tenantRepo:   tenantRepo,
		aiModelRepo:  aiModelRepo,
		aiCaller:     aiCaller,
		permService:  permService,
		skillService: skillService,
		mcpService:   mcpService,
		toolExecutor: toolExecutor,
	}
}

// StreamEventSink 定义流式事件输出回调
type StreamEventSink func(event string, data interface{}) error

// ToolExecutionRecord 记录单轮内执行过的工具调用信息，用于落库和卡片展示
type ToolExecutionRecord struct {
	ToolCode   string      `json:"tool_code"`
	ToolCallID string      `json:"tool_call_id"`
	UIKind     string      `json:"ui_kind"`
	Status     string      `json:"status"` // running | success | error
	Arguments  string      `json:"arguments"`
	Payload    interface{} `json:"payload"`
}

// ExecuteMessageStream 执行会话消息的多轮编排流式循环
func (s *AgentRuntimeService) ExecuteMessageStream(
	c *gin.Context,
	tenantID, userID uuid.UUID,
	username string,
	sessionID uuid.UUID,
	userPrompt string,
	sink StreamEventSink,
) error {
	ctx := c.Request.Context()
	logger := pkglogger.Global()

	// 1. 获取会话与智能体
	session, err := s.chatRepo.GetSessionByID(tenantID, sessionID)
	if err != nil || session == nil {
		return fmt.Errorf("会话不存在")
	}
	if session.UserID != userID {
		return fmt.Errorf("无权访问该会话")
	}

	agent, err := s.agentRepo.GetAgentByID(session.AgentID)
	if err != nil || agent == nil {
		return fmt.Errorf("智能体定义不存在")
	}

	// 2. 权限收敛计算：计算有效工具集
	effectiveTools, err := s.permService.CalculateEffectiveToolsForAgent(ctx, tenantID, userID, agent)
	if err != nil {
		return fmt.Errorf("权限计算失败: %w", err)
	}

	// 3. 组装可用工具定义列表 (Tools Definition)
	var toolDefinitions []ai.ToolDefinition
	for _, spec := range agenttools.GetAllToolSpecs() {
		if effectiveTools[spec.Code] {
			toolDefinitions = append(toolDefinitions, spec.ToToolDefinition())
		}
	}

	// 加载 MCP 工具
	mcpServers, _ := s.agentRepo.ListMCPServers(tenantID)
	for _, mcpServer := range mcpServers {
		if !mcpServer.Enabled {
			continue
		}
		defs := ConvertMCPToolsToDefinitions(mcpServer.ServerCode, mcpServer.CachedTools)
		for _, d := range defs {
			if effectiveTools[d.Function.Name] {
				toolDefinitions = append(toolDefinitions, d)
			}
		}
	}

	// 4. 解析智能体 Skills 并拼接入系统提示词
	var skillCodes []string
	for _, b := range agent.ToolBindings {
		if b.ToolType == "skill" {
			skillCodes = append(skillCodes, b.ToolCode)
		}
	}
	skills, _ := s.skillService.ResolveAgentSkillsOverview(ctx, tenantID, skillCodes)
	skillsPrompt := s.skillService.BuildSkillsPromptSection(skills)

	systemPrompt := agent.SystemPrompt + skillsPrompt
	if session.ProcessID != nil && *session.ProcessID != "" {
		systemPrompt += fmt.Sprintf("\n\n当前上下文绑定 OA 流程 ID: %s。在用户询问时可优先围绕该流程进行分析或查询。", *session.ProcessID)
	}

	// 5. 保存用户消息到数据库
	userMsg := model.ChatMessage{
		SessionID: sessionID,
		TenantID:  tenantID,
		Role:      "user",
		Content:   userPrompt,
		Status:    "success",
		ToolCalls: datatypes.JSON([]byte(`[]`)),
	}
	if err := s.chatRepo.CreateMessage(&userMsg); err != nil {
		return fmt.Errorf("持久化用户消息失败: %w", err)
	}

	// 6. 构造历史消息上下文
	historyMsgs, err := s.chatRepo.ListMessagesBySession(tenantID, sessionID)
	if err != nil {
		return fmt.Errorf("读取历史消息失败: %w", err)
	}

	var conversation []ai.ChatMessage
	conversation = append(conversation, ai.ChatMessage{
		Role:    "system",
		Content: systemPrompt,
	})
	for _, m := range historyMsgs {
		conversation = append(conversation, ai.ChatMessage{
			Role:             m.Role,
			Content:          m.Content,
			ReasoningContent: m.ReasoningContent,
		})
	}

	// 7. 解析租户使用的 AI 模型（优先租户 chat 专用模型，其次租户主模型）
	tenant, _ := s.tenantRepo.FindByID(tenantID)
	var modelID *uuid.UUID
	if tenant != nil {
		if tenant.ChatPrimaryModelID != nil {
			modelID = tenant.ChatPrimaryModelID
		} else {
			modelID = tenant.PrimaryModelID
		}
	}
	if modelID == nil {
		return fmt.Errorf("当前租户未配置可用的 AI 模型")
	}
	modelCfg, err := s.aiModelRepo.FindByID(*modelID)
	if err != nil || modelCfg == nil {
		return fmt.Errorf("AI 模型配置不存在或已停用")
	}

	// 8. 进入智能体编排循环
	_ = sink("session", map[string]interface{}{
		"session_id": sessionID.String(),
		"title":      session.Title,
	})
	_ = sink("agent", map[string]interface{}{
		"agent_code": agent.AgentCode,
		"name":       agent.Name,
	})

	var accumulatedRecords []ToolExecutionRecord
	var finalContent strings.Builder
	var finalReasoning strings.Builder
	var totalTokenUsage ai.TokenUsage

	execCtx := &agenttools.ExecutionContext{
		Ctx:      ctx,
		GinCtx:   c,
		TenantID: tenantID,
		UserID:   userID,
		Username: username,
	}

	for step := 1; step <= maxAgentLoopSteps; step++ {
		select {
		case <-ctx.Done():
			// 客户端连接中断
			_ = sink("interrupted", map[string]interface{}{"message": "客户端已取消生成"})
			s.saveAssistantMessage(sessionID, tenantID, finalContent.String(), finalReasoning.String(), "interrupted", accumulatedRecords, totalTokenUsage)
			return nil
		default:
		}

		_ = sink("status", map[string]interface{}{"status": "thinking", "step": step})

		var turnDelta strings.Builder
		var turnReasoning strings.Builder

		req := &ai.ChatRequest{
			Messages:       conversation,
			Tools:          toolDefinitions,
			ModelConfig:    modelCfg,
			Temperature:    0.3,
			MaxTokens:      modelCfg.MaxTokens,
			EnableThinking: true,
			RequestType:    "chat",
			CallType:       "reasoning",
			StreamChunkFunc: func(chunk string) {
				turnDelta.WriteString(chunk)
				_ = sink("delta", map[string]interface{}{"content": chunk})
			},
			StreamReasoningChunkFunc: func(chunk string) {
				turnReasoning.WriteString(chunk)
				_ = sink("reasoning", map[string]interface{}{"content": chunk})
			},
		}

		resp, err := s.aiCaller.Chat(c, tenantID, userID, modelCfg, req)
		if err != nil {
			logger.Error("AI 调用失败", zap.Error(err), zap.Int("step", step))
			_ = sink("error", map[string]interface{}{"message": "AI 模型处理异常: " + err.Error()})
			s.saveAssistantMessage(sessionID, tenantID, finalContent.String(), finalReasoning.String(), "error", accumulatedRecords, totalTokenUsage)
			return err
		}

		totalTokenUsage.InputTokens += resp.TokenUsage.InputTokens
		totalTokenUsage.OutputTokens += resp.TokenUsage.OutputTokens
		totalTokenUsage.TotalTokens += resp.TokenUsage.TotalTokens

		if turnReasoning.Len() > 0 {
			finalReasoning.WriteString(turnReasoning.String())
		}
		if turnDelta.Len() > 0 {
			finalContent.WriteString(turnDelta.String())
		}

		// 检查是否有 Tool Calls
		if len(resp.ToolCalls) == 0 {
			// 没有工具调用，轮次结束！
			break
		}

		// 将助手的 Tool Calls 消息追加到多轮上下文
		conversation = append(conversation, ai.ChatMessage{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		// 逐一执行 Tool Call
		for _, tc := range resp.ToolCalls {
			toolName := tc.Function.Name
			toolArgs := tc.Function.Arguments

			// 获取 UIKind
			uiKind := "mcp_generic"
			if spec, ok := agenttools.BuiltinTools[toolName]; ok {
				uiKind = spec.UIKind
			}

			// 检查是否具备该工具的权限
			if !effectiveTools[toolName] {
				errText := fmt.Sprintf("您没有调用工具「%s」的权限", toolName)
				_ = sink("tool_start", map[string]interface{}{
					"tool_code":    toolName,
					"tool_call_id": tc.ID,
					"ui_kind":      uiKind,
					"status":       "running",
				})
				_ = sink("tool_result", map[string]interface{}{
					"tool_code":    toolName,
					"tool_call_id": tc.ID,
					"ui_kind":      uiKind,
					"status":       "error",
					"payload":      map[string]interface{}{"error": errText},
				})
				accumulatedRecords = append(accumulatedRecords, ToolExecutionRecord{
					ToolCode:   toolName,
					ToolCallID: tc.ID,
					UIKind:     uiKind,
					Status:     "error",
					Arguments:  toolArgs,
					Payload:    map[string]interface{}{"error": errText},
				})
				conversation = append(conversation, ai.ChatMessage{
					Role:       "tool",
					Name:       toolName,
					ToolCallID: tc.ID,
					Content:    errText,
				})
				continue
			}

			// 推送 tool_start 事件
			_ = sink("tool_start", map[string]interface{}{
				"tool_code":    toolName,
				"tool_call_id": tc.ID,
				"ui_kind":      uiKind,
				"status":       "running",
				"arguments":    toolArgs,
			})

			var payload interface{}
			var execErr error

			if strings.HasPrefix(toolName, "mcp:") {
				// MCP 工具调用: mcp:{server_code}:{tool_name}
				parts := strings.SplitN(strings.TrimPrefix(toolName, "mcp:"), ":", 2)
				if len(parts) == 2 {
					payload, execErr = s.mcpService.CallTool(ctx, tenantID, parts[0], parts[1], toolArgs)
				} else {
					execErr = fmt.Errorf("非法的 MCP 工具键格式")
				}
			} else {
				// 内置系统工具调用
				payload, uiKind, execErr = s.toolExecutor.Execute(toolName, toolArgs, execCtx)
			}

			toolStatus := "success"
			var toolResultContent string
			if execErr != nil {
				toolStatus = "error"
				payload = map[string]interface{}{"error": execErr.Error()}
				toolResultContent = fmt.Sprintf("工具调用失败: %s", execErr.Error())
			} else {
				payloadJSON, _ := json.Marshal(payload)
				toolResultContent = string(payloadJSON)
			}

			// 推送 tool_result 事件
			_ = sink("tool_result", map[string]interface{}{
				"tool_code":    toolName,
				"tool_call_id": tc.ID,
				"ui_kind":      uiKind,
				"status":       toolStatus,
				"payload":      payload,
			})

			accumulatedRecords = append(accumulatedRecords, ToolExecutionRecord{
				ToolCode:   toolName,
				ToolCallID: tc.ID,
				UIKind:     uiKind,
				Status:     toolStatus,
				Arguments:  toolArgs,
				Payload:    payload,
			})

			// 回填 tool 结果消息到上下文
			conversation = append(conversation, ai.ChatMessage{
				Role:       "tool",
				Name:       toolName,
				ToolCallID: tc.ID,
				Content:    toolResultContent,
			})
		}
	}

	// 9. 保存最终 Assistant 消息入库
	s.saveAssistantMessage(sessionID, tenantID, finalContent.String(), finalReasoning.String(), "success", accumulatedRecords, totalTokenUsage)

	// 10. 首轮自动生成短标题（如原标题为默认名）
	if session.Title == "新对话" || session.Title == "" {
		newTitle := userPrompt
		if len([]rune(newTitle)) > 20 {
			newTitle = string([]rune(newTitle)[:20]) + "..."
		}
		_ = s.chatRepo.UpdateSession(tenantID, sessionID, map[string]interface{}{
			"title": newTitle,
		})
		_ = sink("session", map[string]interface{}{
			"session_id": sessionID.String(),
			"title":      newTitle,
		})
	}

	_ = sink("done", map[string]interface{}{
		"status":      "completed",
		"token_usage": totalTokenUsage,
	})

	return nil
}

func (s *AgentRuntimeService) saveAssistantMessage(
	sessionID, tenantID uuid.UUID,
	content, reasoning, status string,
	toolRecords []ToolExecutionRecord,
	usage ai.TokenUsage,
) {
	toolJSON, _ := json.Marshal(toolRecords)
	usageJSON, _ := json.Marshal(usage)

	msg := model.ChatMessage{
		SessionID:        sessionID,
		TenantID:         tenantID,
		Role:             "assistant",
		Content:          content,
		ReasoningContent: reasoning,
		Status:           status,
		ToolCalls:        datatypes.JSON(toolJSON),
		TokenUsage:       datatypes.JSON(usageJSON),
	}
	_ = s.chatRepo.CreateMessage(&msg)
}
