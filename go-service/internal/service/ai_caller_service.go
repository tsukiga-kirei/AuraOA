package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/ai"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/pkg/sanitize"
	"oa-smart-audit/go-service/internal/repository"
)

// AIModelCallerService 处理 AI 模型调用的业务逻辑，包括 Token 统计和日志记录。
type AIModelCallerService struct {
	tenantRepo *repository.TenantRepo
	logRepo    *repository.LLMMessageLogRepo
	db         *gorm.DB
}

// NewAIModelCallerService 创建一个新的 AIModelCallerService 实例。
func NewAIModelCallerService(
	tenantRepo *repository.TenantRepo,
	logRepo *repository.LLMMessageLogRepo,
	db *gorm.DB,
) *AIModelCallerService {
	return &AIModelCallerService{
		tenantRepo: tenantRepo,
		logRepo:    logRepo,
		db:         db,
	}
}

// Chat 执行 AI 模型调用，包含 Token 配额检查、调用执行、Token 累加和异步日志写入。
func (s *AIModelCallerService) Chat(c *gin.Context, tenantID, userID uuid.UUID, modelCfg *model.AIModelConfig, req *ai.ChatRequest) (*ai.ChatResponse, error) {
	// 检查 Token 配额
	tenant, err := s.tenantRepo.FindByID(tenantID)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "租户不存在")
	}

	if tenant.TokenUsed >= tenant.TokenQuota {
		return nil, newServiceError(errcode.ErrTokenQuotaExceeded, "租户Token配额已用尽")
	}

	// 创建 AI 调用器
	caller, err := ai.NewAIModelCaller(modelCfg)
	if err != nil {
		return nil, newServiceError(errcode.ErrAIDeployTypeUnsupported, err.Error())
	}

	// 执行调用
	startTime := time.Now()
	resp, err := caller.Chat(c.Request.Context(), req)
	if err != nil {
		return nil, newServiceError(errcode.ErrAICallFailed, "AI模型调用失败: "+err.Error())
	}

	// 补充调用耗时
	if resp.DurationMs == 0 {
		resp.DurationMs = time.Since(startTime).Milliseconds()
	}

	// 累加 Token 用量
	if err := s.tenantRepo.UpdateFields(tenantID, map[string]interface{}{
		"token_used": gorm.Expr("token_used + ?", resp.TokenUsage.TotalTokens),
	}); err != nil {
		// Token 累加失败不阻塞主流程，仅记录
		_ = err
	}

	// 异步写入日志
	modelConfigID := modelCfg.ID
	go func() {
		log := &model.TenantLLMMessageLog{
			ID:            uuid.New(),
			TenantID:      tenantID,
			UserID:        &userID,
			ModelConfigID: &modelConfigID,
			RequestType:   "audit",
			InputTokens:   resp.TokenUsage.InputTokens,
			OutputTokens:  resp.TokenUsage.OutputTokens,
			TotalTokens:   resp.TokenUsage.TotalTokens,
			DurationMs:    int(resp.DurationMs),
			CreatedAt:     time.Now(),
		}
		_ = s.logRepo.Create(log)
	}()

	return resp, nil
}

// pythonAIRequest Go → Python AI 服务的请求体格式。
type pythonAIRequest struct {
	SystemPrompt string                 `json:"system_prompt"`
	UserPrompt   string                 `json:"user_prompt"`
	ModelConfig  map[string]interface{} `json:"model_config"`
	AuditContext map[string]interface{} `json:"audit_context"`
}

// pythonAIResponse Python → Go AI 服务的响应体格式。
type pythonAIResponse struct {
	Content    string        `json:"content"`
	TokenUsage ai.TokenUsage `json:"token_usage"`
	ModelID    string        `json:"model_id"`
	DurationMs int64         `json:"duration_ms"`
}

// ChatViaPython 通过 HTTP 调用 Python AI 服务执行审核。
// 调用前对用户提示词执行数据脱敏，调用后累加 Token 并异步写入日志。
func (s *AIModelCallerService) ChatViaPython(c *gin.Context, tenantID, userID uuid.UUID, modelCfg *model.AIModelConfig, req *ai.ChatRequest, auditContext map[string]interface{}) (*ai.ChatResponse, error) {
	// 检查 Token 配额
	tenant, err := s.tenantRepo.FindByID(tenantID)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "租户不存在")
	}
	if tenant.TokenUsed >= tenant.TokenQuota {
		return nil, newServiceError(errcode.ErrTokenQuotaExceeded, "租户Token配额已用尽")
	}

	// 数据脱敏：对用户提示词中的敏感信息进行脱敏
	sanitizedUserPrompt := sanitize.SanitizeText(req.UserPrompt)

	// 构建请求体
	pyReq := pythonAIRequest{
		SystemPrompt: req.SystemPrompt,
		UserPrompt:   sanitizedUserPrompt,
		ModelConfig: map[string]interface{}{
			"model_id":    modelCfg.ID.String(),
			"provider":    modelCfg.Provider,
			"model_name":  modelCfg.ModelName,
			"endpoint":    modelCfg.Endpoint,
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
		return nil, newServiceError(errcode.ErrAICallFailed, "Python AI服务调用失败: "+err.Error())
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(httpResp.Body)
		return nil, newServiceError(errcode.ErrAICallFailed, fmt.Sprintf("Python AI服务返回错误(%d): %s", httpResp.StatusCode, string(respBody)))
	}

	// 解析响应
	var pyResp pythonAIResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&pyResp); err != nil {
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

	// 累加 Token 用量
	_ = s.tenantRepo.UpdateFields(tenantID, map[string]interface{}{
		"token_used": gorm.Expr("token_used + ?", resp.TokenUsage.TotalTokens),
	})

	// 异步写入日志
	modelConfigID := modelCfg.ID
	go func() {
		log := &model.TenantLLMMessageLog{
			ID:            uuid.New(),
			TenantID:      tenantID,
			UserID:        &userID,
			ModelConfigID: &modelConfigID,
			RequestType:   "audit",
			InputTokens:   resp.TokenUsage.InputTokens,
			OutputTokens:  resp.TokenUsage.OutputTokens,
			TotalTokens:   resp.TokenUsage.TotalTokens,
			DurationMs:    int(resp.DurationMs),
			CreatedAt:     time.Now(),
		}
		_ = s.logRepo.Create(log)
	}()

	return resp, nil
}
