package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/ai"
	"auraoa/go-service/internal/pkg/errcode"
	pkglogger "auraoa/go-service/internal/pkg/logger"
	"auraoa/go-service/internal/pkg/sanitize"
	"auraoa/go-service/internal/pkg/systemflags"
	"auraoa/go-service/internal/repository"
)

// AIModelCallerService 负责 AI 模型调用的完整生命周期管理：
// Token 配额预扣与结算、调用执行、异步日志写入。
type AIModelCallerService struct {
	tenantRepo *repository.TenantRepo
	logRepo    *repository.LLMMessageLogRepo
	db         *gorm.DB
	sysFlags   *systemflags.Resolver
}

// NewAIModelCallerService 初始化 AI 调用服务，注入租户仓储、日志仓储和数据库连接。
// sysFlags 可为 nil；非 nil 时根据 system.enable_data_encryption 对提示词做脱敏。
func NewAIModelCallerService(
	tenantRepo *repository.TenantRepo,
	logRepo *repository.LLMMessageLogRepo,
	db *gorm.DB,
	sysFlags *systemflags.Resolver,
) *AIModelCallerService {
	return &AIModelCallerService{
		tenantRepo: tenantRepo,
		logRepo:    logRepo,
		db:         db,
		sysFlags:   sysFlags,
	}
}

// Chat 执行单次 AI 对话调用，完整流程为：
// 1. 预扣 Token 配额（防止并发超额）
// 2. 创建对应部署类型的调用器并发起请求
// 3. 调用失败时回滚预扣额度
// 4. 调用成功后结算实际消耗，并异步写入调用日志
func (s *AIModelCallerService) Chat(c *gin.Context, tenantID, userID uuid.UUID, modelCfg *model.AIModelConfig, req *ai.ChatRequest) (*ai.ChatResponse, error) {
	requestCopy := *req
	requestCopy.Messages = append([]ai.ChatMessage(nil), req.Messages...)
	requestCopy.EnableThinking = req.EnableThinking && modelCfg.SupportsThinking
	requestCopy.ModelConfig = modelCfg
	req = &requestCopy
	if s.sysFlags != nil && s.sysFlags.DataEncryptionEnabled() {
		req.UserPrompt = sanitize.SanitizeText(req.UserPrompt)
		req.SystemPrompt = sanitize.SanitizeText(req.SystemPrompt)
		for i := range req.Messages {
			req.Messages[i].Content = sanitize.SanitizeText(req.Messages[i].Content)
		}
	}

	// 检查 Token 配额（预扣 max_tokens 防止并发超额）
	reserved := 0
	if !req.SkipQuotaCheck {
		reserved = req.MaxTokens
		if reserved == 0 {
			reserved = modelCfg.MaxTokens
		}
		if err := s.reserveTokenQuota(tenantID, reserved); err != nil {
			return nil, err
		}
	}

	// 创建 AI 调用器
	caller, err := ai.NewAIModelCaller(modelCfg)
	if err != nil {
		// 预扣失败回滚
		if !req.SkipQuotaCheck {
			_ = s.releaseTokenQuota(tenantID, reserved)
		}
		return nil, newServiceError(errcode.ErrAIDeployTypeUnsupported, err.Error())
	}

	// 执行调用
	startTime := time.Now()
	resp, err := caller.Chat(c.Request.Context(), req)
	if err != nil {
		// 调用失败回滚预扣
		if !req.SkipQuotaCheck {
			_ = s.releaseTokenQuota(tenantID, reserved)
		}
		return nil, newServiceError(errcode.ErrAICallFailed, "AI模型调用失败: "+err.Error())
	}

	// 补充调用耗时
	if resp.DurationMs == 0 {
		resp.DurationMs = time.Since(startTime).Milliseconds()
	}

	// 结算：用实际消耗替换预扣额度（释放预扣，加上实际值）
	_ = s.settleTokenUsage(tenantID, reserved, resp.TokenUsage.TotalTokens)

	// 异步写入日志（带重试）
	logUserPrompt := req.UserPrompt
	if len(req.Messages) > 0 {
		if raw, err := json.Marshal(map[string]interface{}{"messages": req.Messages, "tools": req.Tools}); err == nil {
			logUserPrompt = sanitize.SanitizeText(string(raw))
		}
	}
	s.asyncWriteLog(tenantID, userID, modelCfg.ID, req, resp, req.SystemPrompt, logUserPrompt)

	return resp, nil
}

// getRetryCount 从 system_configs 读取 AI 调用重试次数（key: system.ai_retry_count），默认 3。
func (s *AIModelCallerService) getRetryCount() int {
	var value string
	err := s.db.Raw("SELECT value FROM system_configs WHERE key = ?", "system.ai_retry_count").Scan(&value).Error
	if err != nil || value == "" {
		return 3
	}
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return 3
	}
	return n
}

// ChatWithFallback 带重试和备用模型切换的 AI 调用。
// 流程：主模型重试 N 次 → 若全部失败且有备用模型 → 备用模型重试 N 次 → 全部失败则返回错误。
// 重试次数 N 读取自 system_configs（key: system.ai_retry_count，默认 3）。
func (s *AIModelCallerService) ChatWithFallback(
	c *gin.Context,
	tenantID, userID uuid.UUID,
	primaryCfg *model.AIModelConfig,
	fallbackCfg *model.AIModelConfig,
	req *ai.ChatRequest,
) (*ai.ChatResponse, error) {
	retryCount := s.getRetryCount()
	if retryCount <= 0 {
		retryCount = 1
	}

	// ── 尝试主模型 ──
	var lastErr error
	for i := 0; i < retryCount; i++ {
		if i > 0 && req.StreamResetFunc != nil {
			req.StreamResetFunc()
		}
		resp, err := s.Chat(c, tenantID, userID, primaryCfg, req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		adjusted, adjustedMaxTokens := adjustMaxTokensForContextError(req, err)
		retryable := adjusted || isRetryableAIError(err)
		pkglogger.Global().Warn("主模型调用失败",
			zap.Int("attempt", i+1),
			zap.Int("maxRetries", retryCount),
			zap.String("model", primaryCfg.ModelName),
			zap.String("processID", req.ProcessID),
			zap.String("businessLogID", optionalUUIDString(req.BusinessLogID)),
			zap.Bool("retryable", retryable),
			zap.Bool("maxTokensAdjusted", adjusted),
			zap.Int("adjustedMaxTokens", adjustedMaxTokens),
			zap.Error(err),
		)
		if !retryable {
			break
		}
		if i < retryCount-1 {
			// 指数退避: 1s, 2s, 4s ...
			time.Sleep(time.Duration(1<<i) * time.Second)
		}
	}

	// ── 主模型全部失败，尝试备用模型 ──
	if fallbackCfg == nil || sameAIModelRoute(primaryCfg, fallbackCfg) {
		pkglogger.Global().Error("主模型调用全部失败且无备用模型",
			zap.String("model", primaryCfg.ModelName),
			zap.Int("retries", retryCount),
			zap.String("processID", req.ProcessID),
			zap.Bool("fallbackSameAsPrimary", fallbackCfg != nil),
			zap.Error(lastErr),
		)
		return nil, lastErr
	}

	pkglogger.Global().Warn("主模型调用全部失败，切换到备用模型",
		zap.String("primaryModel", primaryCfg.ModelName),
		zap.String("fallbackModel", fallbackCfg.ModelName),
		zap.Int("retries", retryCount),
	)

	for i := 0; i < retryCount; i++ {
		if req.StreamResetFunc != nil {
			req.StreamResetFunc()
		}
		resp, err := s.Chat(c, tenantID, userID, fallbackCfg, req)
		if err == nil {
			pkglogger.Global().Info("备用模型调用成功",
				zap.String("fallbackModel", fallbackCfg.ModelName),
				zap.Int("attempt", i+1),
			)
			return resp, nil
		}
		lastErr = err
		adjusted, adjustedMaxTokens := adjustMaxTokensForContextError(req, err)
		retryable := adjusted || isRetryableAIError(err)
		pkglogger.Global().Warn("备用模型调用失败",
			zap.Int("attempt", i+1),
			zap.Int("maxRetries", retryCount),
			zap.String("model", fallbackCfg.ModelName),
			zap.String("processID", req.ProcessID),
			zap.String("businessLogID", optionalUUIDString(req.BusinessLogID)),
			zap.Bool("retryable", retryable),
			zap.Bool("maxTokensAdjusted", adjusted),
			zap.Int("adjustedMaxTokens", adjustedMaxTokens),
			zap.Error(err),
		)
		if !retryable {
			break
		}
		if i < retryCount-1 {
			time.Sleep(time.Duration(1<<i) * time.Second)
		}
	}

	pkglogger.Global().Error("主模型和备用模型均调用失败",
		zap.String("primaryModel", primaryCfg.ModelName),
		zap.String("fallbackModel", fallbackCfg.ModelName),
		zap.Int("retriesPerModel", retryCount),
		zap.Error(lastErr),
	)
	return nil, newServiceError(
		errcode.ErrAICallFailed,
		fmt.Sprintf("主模型和备用模型均调用失败：%s", truncate(lastErr.Error(), 1200)),
	)
}

var contextLengthErrorPattern = regexp.MustCompile(
	`(?i)maximum context length is ([0-9]+) tokens.*requested ([0-9]+) output tokens.*at least ([0-9]+) input tokens`,
)

func adjustMaxTokensForContextError(req *ai.ChatRequest, err error) (bool, int) {
	if req == nil || err == nil {
		return false, 0
	}
	matches := contextLengthErrorPattern.FindStringSubmatch(err.Error())
	if len(matches) != 4 {
		return false, 0
	}
	contextWindow, parseContextErr := strconv.Atoi(matches[1])
	requestedOutput, parseOutputErr := strconv.Atoi(matches[2])
	inputTokens, parseInputErr := strconv.Atoi(matches[3])
	if parseContextErr != nil || parseOutputErr != nil || parseInputErr != nil {
		return false, 0
	}
	// 保留少量安全余量，避免服务端“至少 N 个输入 token”的估值再次越界。
	available := contextWindow - inputTokens - 512
	if available < 512 {
		return false, 0
	}
	current := req.MaxTokens
	if current <= 0 {
		current = requestedOutput
	}
	if available >= current {
		return false, 0
	}
	req.MaxTokens = available
	return true, available
}

func isRetryableAIError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	nonRetryableMarkers := []string{
		"状态码 400",
		"status code 400",
		"maximum context length",
		"context_length_exceeded",
		"input_tokens",
		"badrequesterror",
		"api key 无效",
		"unauthorized",
	}
	for _, marker := range nonRetryableMarkers {
		if strings.Contains(message, strings.ToLower(marker)) {
			return false
		}
	}
	return true
}

func sameAIModelRoute(primary, fallback *model.AIModelConfig) bool {
	if primary == nil || fallback == nil {
		return false
	}
	if primary.ID == fallback.ID {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(primary.Provider), strings.TrimSpace(fallback.Provider)) &&
		strings.EqualFold(strings.TrimSpace(primary.ModelName), strings.TrimSpace(fallback.ModelName)) &&
		strings.TrimRight(strings.TrimSpace(primary.Endpoint), "/") ==
			strings.TrimRight(strings.TrimSpace(fallback.Endpoint), "/") &&
		primary.APIKey == fallback.APIKey
}

func optionalUUIDString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

// pythonAIRequest 发往 Python AI 服务的请求体，包含提示词和完整模型配置。
type pythonAIRequest struct {
	SystemPrompt string                 `json:"system_prompt"`
	UserPrompt   string                 `json:"user_prompt"`
	ModelConfig  map[string]interface{} `json:"model_config"`
	AuditContext map[string]interface{} `json:"audit_context"`
}

// pythonAIResponse Python AI 服务返回的响应体，包含生成内容和 Token 统计。
type pythonAIResponse struct {
	Content    string        `json:"content"`
	TokenUsage ai.TokenUsage `json:"token_usage"`
	ModelID    string        `json:"model_id"`
	DurationMs int64         `json:"duration_ms"`
}

// ChatViaPython 通过 HTTP 转发至 Python AI 服务执行审核推理。
// 调用前对用户提示词执行数据脱敏，调用后结算 Token 并异步写入日志。
// 适用于需要 Python 侧特殊处理（如 RAG 检索、复杂上下文注入）的场景。
func (s *AIModelCallerService) ChatViaPython(c *gin.Context, tenantID, userID uuid.UUID, modelCfg *model.AIModelConfig, req *ai.ChatRequest, auditContext map[string]interface{}) (*ai.ChatResponse, error) {
	// 预扣 Token 配额
	reserved := 0
	if !req.SkipQuotaCheck {
		reserved = req.MaxTokens
		if reserved == 0 {
			reserved = modelCfg.MaxTokens
		}
		if err := s.reserveTokenQuota(tenantID, reserved); err != nil {
			return nil, err
		}
	}

	// 数据脱敏：与 system.enable_data_encryption 对齐；未注入 sysFlags 时保持原默认（始终脱敏）
	userPrompt := req.UserPrompt
	systemPrompt := req.SystemPrompt
	if s.sysFlags == nil || s.sysFlags.DataEncryptionEnabled() {
		userPrompt = sanitize.SanitizeText(userPrompt)
		systemPrompt = sanitize.SanitizeText(systemPrompt)
	}

	// 构建请求体（包含完整模型配置供 Python 端使用）
	pyReq := pythonAIRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		ModelConfig: map[string]interface{}{
			"model_id":    modelCfg.ID.String(),
			"provider":    modelCfg.Provider,
			"deploy_type": modelCfg.DeployType,
			"model_name":  modelCfg.ModelName,
			"endpoint":    modelCfg.Endpoint,
			"api_key":     modelCfg.APIKey,
			"max_tokens":  modelCfg.MaxTokens,
			"temperature": req.Temperature,
		},
		AuditContext: auditContext,
	}

	bodyBytes, err := json.Marshal(pyReq)
	if err != nil {
		return nil, newServiceError(errcode.ErrInternalServer, "请求序列化失败")
	}

	// 获取 Python AI 服务地址
	aiServiceURL := os.Getenv("AI_SERVICE_URL")
	if aiServiceURL == "" {
		aiServiceURL = "http://ai-service:8000"
	}

	// 发送 HTTP 请求到 Python AI 服务
	startTime := time.Now()
	httpResp, err := http.Post(
		fmt.Sprintf("%s/api/v1/chat/completions", aiServiceURL),
		"application/json",
		bytes.NewReader(bodyBytes),
	)
	if err != nil {
		if !req.SkipQuotaCheck {
			_ = s.releaseTokenQuota(tenantID, reserved)
		}
		return nil, newServiceError(errcode.ErrAICallFailed, "Python AI服务调用失败: "+err.Error())
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		if !req.SkipQuotaCheck {
			_ = s.releaseTokenQuota(tenantID, reserved)
		}
		respBody, _ := io.ReadAll(httpResp.Body)
		return nil, newServiceError(errcode.ErrAICallFailed, fmt.Sprintf("Python AI服务返回错误(%d): %s", httpResp.StatusCode, string(respBody)))
	}

	// 解析响应
	var pyResp pythonAIResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&pyResp); err != nil {
		if !req.SkipQuotaCheck {
			_ = s.releaseTokenQuota(tenantID, reserved)
		}
		return nil, newServiceError(errcode.ErrAICallFailed, "Python AI服务响应解析失败")
	}

	resp := &ai.ChatResponse{
		Content:    pyResp.Content,
		TokenUsage: pyResp.TokenUsage,
		ModelID:    pyResp.ModelID,
		DurationMs: pyResp.DurationMs,
	}

	// 补充调用耗时
	if resp.DurationMs == 0 {
		resp.DurationMs = time.Since(startTime).Milliseconds()
	}

	// 结算：用实际消耗替换预扣额度
	_ = s.settleTokenUsage(tenantID, reserved, resp.TokenUsage.TotalTokens)

	// 异步写入日志（带重试）
	s.asyncWriteLog(tenantID, userID, modelCfg.ID, req, resp, systemPrompt, userPrompt)

	return resp, nil
}

// ── Token 配额原子操作 ─────────────────────────────────────

// reserveTokenQuota 原子预扣 Token 配额，防止并发场景下超额消耗。
// 使用条件更新 token_used + amount <= token_quota，只有满足条件的请求才能成功预扣。
func (s *AIModelCallerService) reserveTokenQuota(tenantID uuid.UUID, amount int) error {
	result := s.db.Model(&model.Tenant{}).
		Where("id = ? AND token_used + ? <= token_quota", tenantID, amount).
		Update("token_used", gorm.Expr("token_used + ?", amount))

	if result.Error != nil {
		return newServiceError(errcode.ErrDatabase, "Token配额预扣失败")
	}
	if result.RowsAffected == 0 {
		return newServiceError(errcode.ErrTokenQuotaExceeded, "租户Token配额不足")
	}
	return nil
}

// releaseTokenQuota 回滚预扣的 Token 配额，在调用失败时恢复租户可用额度。
// 使用 GREATEST 防止因并发操作导致 token_used 出现负值。
func (s *AIModelCallerService) releaseTokenQuota(tenantID uuid.UUID, amount int) error {
	return s.db.Model(&model.Tenant{}).
		Where("id = ?", tenantID).
		Update("token_used", gorm.Expr("GREATEST(token_used - ?, 0)", amount)).Error
}

// settleTokenUsage 结算实际 Token 消耗：释放预扣额度，加上实际消耗量。
// 等价于 token_used = token_used - reserved + actual，diff 可为负（实际 < 预扣时退还差额）。
func (s *AIModelCallerService) settleTokenUsage(tenantID uuid.UUID, reserved, actual int) error {
	diff := actual - reserved // 可能为负（实际消耗 < 预扣）
	if diff == 0 {
		return nil
	}
	return s.db.Model(&model.Tenant{}).
		Where("id = ?", tenantID).
		Update("token_used", gorm.Expr("GREATEST(token_used + ?, 0)", diff)).Error
}

// ── 异步日志写入（带重试） ─────────────────────────────────

const logMaxRetries = 3

// asyncWriteLog 在独立 goroutine 中异步写入 LLM 调用日志，失败时按指数退避重试最多 3 次。
// 重试耗尽后降级为标准日志输出，不影响主流程返回。
func (s *AIModelCallerService) asyncWriteLog(
	tenantID, userID uuid.UUID,
	modelConfigID uuid.UUID,
	req *ai.ChatRequest,
	resp *ai.ChatResponse,
	systemPrompt, userPrompt string,
) {
	go func() {
		requestType := strings.TrimSpace(req.RequestType)
		if requestType == "" {
			requestType = "audit"
		}
		callType := strings.TrimSpace(req.CallType)
		if callType == "" {
			callType = "reasoning"
		}
		now := time.Now()
		entry := &model.TenantLLMMessageLog{
			ID:            uuid.New(),
			TenantID:      tenantID,
			UserID:        &userID,
			ModelConfigID: &modelConfigID,
			RequestType:   strings.ToValidUTF8(requestType, "\uFFFD"),
			CallType:      strings.ToValidUTF8(callType, "\uFFFD"),
			ProcessID:     strings.ToValidUTF8(strings.TrimSpace(req.ProcessID), "\uFFFD"),
			ProcessTitle:  strings.ToValidUTF8(strings.TrimSpace(req.ProcessTitle), "\uFFFD"),
			BusinessLogID: req.BusinessLogID,
			InputTokens:   resp.TokenUsage.InputTokens,
			OutputTokens:  resp.TokenUsage.OutputTokens,
			TotalTokens:   resp.TokenUsage.TotalTokens,
			DurationMs:    int(resp.DurationMs),
			CreatedAt:     now,
		}
		payload := &model.TenantLLMMessagePayload{
			ID:               uuid.New(),
			LLMMessageLogID:  entry.ID,
			TenantID:         tenantID,
			SystemPrompt:     strings.ToValidUTF8(systemPrompt, "\uFFFD"),
			UserPrompt:       strings.ToValidUTF8(userPrompt, "\uFFFD"),
			ReasoningContent: strings.ToValidUTF8(resp.ReasoningContent, "\uFFFD"),
			ResponseContent:  strings.ToValidUTF8(resp.Content, "\uFFFD"),
			CreatedAt:        now,
		}

		var err error
		for attempt := 0; attempt < logMaxRetries; attempt++ {
			if err = s.logRepo.CreateWithPayload(entry, payload); err == nil {
				return
			}
			// 指数退避: 1s, 2s, 4s
			time.Sleep(time.Duration(1<<attempt) * time.Second)
		}
		// 重试耗尽，降级写入结构化日志（运维可通过日志采集发现）
		pkglogger.Global().Warn("LLM调用日志写入失败（重试耗尽）",
			zap.String("tenantID", tenantID.String()),
			zap.String("modelConfigID", modelConfigID.String()),
			zap.Int("totalTokens", resp.TokenUsage.TotalTokens),
			zap.Error(err),
		)
	}()
}
