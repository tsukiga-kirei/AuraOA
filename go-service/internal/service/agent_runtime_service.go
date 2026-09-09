package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/datatypes"

	"auraoa/go-service/internal/cache"
	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/agenttools"
	"auraoa/go-service/internal/pkg/ai"
	"auraoa/go-service/internal/pkg/apptime"
	pkglogger "auraoa/go-service/internal/pkg/logger"
	"auraoa/go-service/internal/repository"
)

const maxAgentLoopSteps = 8

// AgentRuntimeService 负责智能体单轮对话的编排循环（包括工具多步调用、状态流式推送与落库）
type AgentRuntimeService struct {
	chatRepo     *repository.ChatRepo
	agentRepo    *repository.AgentRepo
	tenantRepo   *repository.TenantRepo
	aiModelRepo  *repository.AIModelRepo
	aiCaller     *AIModelCallerService
	permService  *AgentPermissionService
	skillService *SkillService
	mcpService   *MCPService
	toolExecutor agenttools.ToolExecutor
	invalidator  *cache.InvalidationManager
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

// SetInvalidator 设置缓存失效管理器
func (s *AgentRuntimeService) SetInvalidator(invalidator *cache.InvalidationManager) {
	s.invalidator = invalidator
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
	Thought    string      `json:"thought,omitempty"`
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
	startTime := apptime.Now()
	ctx := c.Request.Context()
	tenantForLog, _ := s.tenantRepo.FindByID(tenantID)
	logger := pkglogger.Global()
	if tenantForLog != nil {
		logger = pkglogger.GetTenantLogger(tenantForLog.Code)
	}

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

	// 平台模板在租户创建覆盖后，历史会话也应使用该租户最新定义。
	agent, err = s.agentRepo.GetAgentByCode(tenantID, agent.AgentCode)
	if err != nil {
		return err
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
		if b.ToolType == "skill" && effectiveTools[b.ToolCode] {
			skillCodes = append(skillCodes, strings.TrimPrefix(b.ToolCode, "skill:"))
		}
	}
	skills, _ := s.skillService.ResolveAgentSkillsOverview(ctx, tenantID, skillCodes)
	skillsPrompt := s.skillService.BuildSkillsPromptSection(skills)
	skillContents := make(map[string]string)
	for _, skill := range skills {
		key := "skill:" + skill.Code
		skillContents[key] = skill.Content
		toolDefinitions = append(toolDefinitions, ai.ToolDefinition{Type: "function", Function: ai.FunctionSpec{Name: key, Description: "读取技能指南：" + skill.Name + "。" + skill.Description, Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}}})
	}
	// 模型函数名只使用 ASCII；权限键与 UI 仍保留可读的 mcp:/skill: 编码。
	toolAliases := make(map[string]string)
	for i := range toolDefinitions {
		key := toolDefinitions[i].Function.Name
		alias := key
		if strings.Contains(key, ":") {
			sum := sha256.Sum256([]byte(key))
			alias = "tool_" + hex.EncodeToString(sum[:16])
		}
		toolAliases[alias] = key
		toolDefinitions[i].Function.Name = alias
	}

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

	// 5.1 若会话为首轮（标题为空或默认“新对话”），立即提前推送初始标题，使前端头部与列表即时响应
	if session.Title == "新对话" || session.Title == "" {
		initTitle := userPrompt
		if len([]rune(initTitle)) > 15 {
			initTitle = string([]rune(initTitle)[:15]) + "..."
		}
		_ = s.chatRepo.UpdateSession(tenantID, sessionID, map[string]interface{}{
			"title": initTitle,
		})
		_ = sink("session", map[string]interface{}{
			"session_id": sessionID.String(),
			"title":      initTitle,
		})
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
	if len(historyMsgs) > 40 {
		historyMsgs = historyMsgs[len(historyMsgs)-40:]
	}
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

	// 7.1 若会话为首轮，更新临时标题并立即通知前端，同时异步调用 AI 提炼更精准的 4-10 字主题标题
	if session.Title == "新对话" || session.Title == "" {
		initialTitle := strings.TrimSpace(userPrompt)
		if len([]rune(initialTitle)) > 15 {
			initialTitle = string([]rune(initialTitle)[:15]) + "..."
		}
		if initialTitle != "" {
			_ = s.chatRepo.UpdateSession(tenantID, sessionID, map[string]interface{}{
				"title": initialTitle,
			})
			session.Title = initialTitle
		}
		cCopy := c.Copy()
		go s.asyncSummarizeTitle(cCopy, tenantID, userID, sessionID, modelCfg, userPrompt, sink)
	}

	var fallbackCfg *model.AIModelConfig
	fallbackID := tenant.ChatFallbackModelID
	if fallbackID == nil {
		fallbackID = tenant.FallbackModelID
	}
	if fallbackID != nil {
		fallbackCfg, _ = s.aiModelRepo.FindByID(*fallbackID)
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
			s.saveAssistantMessage(sessionID, tenantID, finalContent.String(), finalReasoning.String(), "interrupted", accumulatedRecords, totalTokenUsage, apptime.Now().Sub(startTime).Milliseconds())
			return nil
		default:
		}

		_ = sink("status", map[string]interface{}{"status": "thinking", "step": step})

		var turnDelta strings.Builder
		var turnReasoning strings.Builder

		expectedTitle := session.Title
		if expectedTitle == "新对话" || expectedTitle == "" {
			trimmed := strings.TrimSpace(userPrompt)
			if len([]rune(trimmed)) > 20 {
				expectedTitle = string([]rune(trimmed)[:20]) + "..."
			} else if trimmed != "" {
				expectedTitle = trimmed
			}
		}

		req := &ai.ChatRequest{
			Messages:       conversation,
			Tools:          toolDefinitions,
			ModelConfig:    modelCfg,
			Temperature:    0.3,
			MaxTokens:      modelCfg.MaxTokens,
			EnableThinking: modelCfg.SupportsThinking,
			RequestType:    "chat",
			BusinessLogID:  &sessionID,
			ProcessID:      sessionID.String(),
			ProcessTitle:   expectedTitle,
			CallType:       "reasoning",
			StreamResetFunc: func() {
				turnDelta.Reset()
				turnReasoning.Reset()
				_ = sink("reset", map[string]interface{}{"content": finalContent.String(), "reasoning_content": finalReasoning.String()})
			},
			StreamChunkFunc: func(chunk string) {
				turnDelta.WriteString(chunk)
				_ = sink("delta", map[string]interface{}{"content": chunk})
			},
			StreamReasoningChunkFunc: func(chunk string) {
				turnReasoning.WriteString(chunk)
				_ = sink("reasoning", map[string]interface{}{"content": chunk})
			},
		}

		if step == maxAgentLoopSteps {
			req.Tools = nil
		}
		resp, err := s.aiCaller.ChatWithFallback(c, tenantID, userID, modelCfg, fallbackCfg, req)
		if err != nil {
			logger.Error("AI 调用失败", zap.Error(err), zap.Int("step", step))
			_ = sink("error", map[string]interface{}{"message": "AI 模型处理异常: " + err.Error()})
			s.saveAssistantMessage(sessionID, tenantID, finalContent.String(), finalReasoning.String(), "error", accumulatedRecords, totalTokenUsage, apptime.Now().Sub(startTime).Milliseconds())
			return err
		}

		if turnDelta.Len() == 0 && resp.Content != "" {
			turnDelta.WriteString(resp.Content)
			_ = sink("delta", map[string]interface{}{"content": resp.Content})
		}

		totalTokenUsage.InputTokens += resp.TokenUsage.InputTokens
		totalTokenUsage.OutputTokens += resp.TokenUsage.OutputTokens
		totalTokenUsage.TotalTokens += resp.TokenUsage.TotalTokens

		// 检查是否有 Tool Calls
		if len(resp.ToolCalls) == 0 {
			// 没有工具调用，轮次结束！本轮 turnDelta 才是最终答复正文
			if turnReasoning.Len() > 0 {
				finalReasoning.WriteString(turnReasoning.String())
			}
			if turnDelta.Len() > 0 {
				finalContent.WriteString(turnDelta.String())
			}
			break
		}

		// 有工具调用：当前轮次产生的 turnDelta 属于调用工具前的思考与意图说明，不属于最终正文！
		var turnThought strings.Builder
		if turnReasoning.Len() > 0 {
			turnThought.WriteString(turnReasoning.String())
		}
		if turnDelta.Len() > 0 {
			if turnThought.Len() > 0 {
				turnThought.WriteString("\n\n")
			}
			turnThought.WriteString(turnDelta.String())
		}

		// 将本轮前置思考追加到全局累计思考中
		if turnThought.Len() > 0 {
			if finalReasoning.Len() > 0 {
				finalReasoning.WriteString("\n\n")
			}
			finalReasoning.WriteString(turnThought.String())
		}

		// 发送 reset 事件清空临时流入正文的调用工具前说明，恢复纯净正文，并同步最新思考内容
		_ = sink("reset", map[string]interface{}{
			"content":           finalContent.String(),
			"reasoning_content": finalReasoning.String(),
		})

		stepThought := turnThought.String()

		if step == maxAgentLoopSteps {
			if err := s.saveAssistantMessage(sessionID, tenantID, finalContent.String(), finalReasoning.String(), "error", accumulatedRecords, totalTokenUsage, apptime.Now().Sub(startTime).Milliseconds()); err != nil {
				return err
			}
			return fmt.Errorf("工具调用已达到本轮上限，请缩小问题范围后继续")
		}

		// 将助手的 Tool Calls 消息追加到多轮上下文
		conversation = append(conversation, ai.ChatMessage{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		// 逐一执行 Tool Call
		for _, tc := range resp.ToolCalls {
			toolName := toolAliases[tc.Function.Name]
			if toolName == "" {
				toolName = tc.Function.Name
			}
			toolArgs := tc.Function.Arguments

			// 获取 UIKind
			uiKind := "mcp_generic"
			if strings.HasPrefix(toolName, "skill:") {
				uiKind = "skill"
			}
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
					"thought":      stepThought,
				})
				_ = sink("tool_result", map[string]interface{}{
					"tool_code":    toolName,
					"tool_call_id": tc.ID,
					"ui_kind":      uiKind,
					"status":       "error",
					"payload":      map[string]interface{}{"error": errText},
					"thought":      stepThought,
				})
				accumulatedRecords = append(accumulatedRecords, ToolExecutionRecord{
					ToolCode:   toolName,
					ToolCallID: tc.ID,
					UIKind:     uiKind,
					Status:     "error",
					Arguments:  toolArgs,
					Payload:    map[string]interface{}{"error": errText},
					Thought:    stepThought,
				})
				conversation = append(conversation, ai.ChatMessage{
					Role:       "tool",
					Name:       tc.Function.Name,
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
				"thought":      stepThought,
			})

			var payload interface{}
			var execErr error

			if strings.HasPrefix(toolName, "skill:") {
				uiKind = "skill"
				payload = map[string]interface{}{"content": skillContents[toolName]}
			} else if strings.HasPrefix(toolName, "mcp:") {
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
				"thought":      stepThought,
			})

			accumulatedRecords = append(accumulatedRecords, ToolExecutionRecord{
				ToolCode:   toolName,
				ToolCallID: tc.ID,
				UIKind:     uiKind,
				Status:     toolStatus,
				Arguments:  toolArgs,
				Payload:    payload,
				Thought:    stepThought,
			})

			// 回填 tool 结果消息到上下文
			conversation = append(conversation, ai.ChatMessage{
				Role:       "tool",
				Name:       tc.Function.Name,
				ToolCallID: tc.ID,
				Content:    toolResultContent,
			})
		}
	}

	durationMs := apptime.Now().Sub(startTime).Milliseconds()

	// 9. 保存最终 Assistant 消息入库（包含执行总耗时）
	if err := s.saveAssistantMessage(sessionID, tenantID, finalContent.String(), finalReasoning.String(), "success", accumulatedRecords, totalTokenUsage, durationMs); err != nil {
		return fmt.Errorf("保存回复失败: %w", err)
	}

	_ = sink("done", map[string]interface{}{
		"status":      "completed",
		"token_usage": totalTokenUsage,
		"duration_ms": durationMs,
	})

	return nil
}

func (s *AgentRuntimeService) saveAssistantMessage(
	sessionID, tenantID uuid.UUID,
	content, reasoning, status string,
	toolRecords []ToolExecutionRecord,
	usage ai.TokenUsage,
	durationMs int64,
) error {
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
		DurationMs:       durationMs,
	}
	if err := s.chatRepo.CreateMessage(&msg); err != nil {
		return err
	}
	if s.invalidator != nil {
		_ = s.invalidator.InvalidateDashboardCache(context.Background(), tenantID)
	}
	return nil
}

// asyncSummarizeTitle 异步调用 AI 将用户首轮问题提炼为 4-10 字简练精准的主题标题
func (s *AgentRuntimeService) asyncSummarizeTitle(
	c *gin.Context,
	tenantID, userID, sessionID uuid.UUID,
	modelCfg *model.AIModelConfig,
	userPrompt string,
	sink StreamEventSink,
) {
	if modelCfg == nil || strings.TrimSpace(userPrompt) == "" {
		return
	}

	sysPrompt := "你是一个专业的对话标题提炼助手。请根据用户的提问，总结出一段精炼、准确的主题标题（4至10个汉字以内），直接输出标题文本，切勿包含标点符号、书名号、引号或多余解释。"
	req := &ai.ChatRequest{
		RequestType: "chat",
		CallType:    "structured",
		Messages: []ai.ChatMessage{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.2,
		MaxTokens:   30,
	}

	resp, err := s.aiCaller.Chat(c, tenantID, userID, modelCfg, req)
	if err != nil || resp == nil || strings.TrimSpace(resp.Content) == "" {
		return
	}

	cleanTitle := strings.TrimSpace(resp.Content)
	cleanTitle = strings.Trim(cleanTitle, "\"'`《》【】。，！？ ")
	if len([]rune(cleanTitle)) > 20 {
		cleanTitle = string([]rune(cleanTitle)[:20])
	}
	if cleanTitle != "" {
		_ = s.chatRepo.UpdateSession(tenantID, sessionID, map[string]interface{}{
			"title": cleanTitle,
		})
		_ = sink("session", map[string]interface{}{
			"session_id": sessionID.String(),
			"title":      cleanTitle,
		})
	}
}
