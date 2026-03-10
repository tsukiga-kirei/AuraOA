package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
	// 检查 Token 配额（预扣 max_tokens 防止并发超额）
	reserved := req.MaxTokens
	if reserved == 0 {
		reserved = modelCfg.MaxTokens
	}
	if err := s.reserveTokenQuota(tenantID, reserved); err != nil {
		return nil, err
	}

	// 创建 AI 调用器
	caller, err := ai.NewAIModelCaller(modelCfg)
	if err != nil {
		// 预扣失败回滚
		_ = s.releaseTokenQuota(tenantID, reserved)
		return nil, newServiceError(errcode.ErrAIDeployTypeUnsupported, err.Error())
	}

	// 执行调用
	startTime := time.Now()
	resp, err := caller.Chat(c.Request.Context(), req)
	if err != nil {
		// 调用失败回滚预扣
		_ = s.releaseTokenQuota(tenantID, reserved)
		return nil, newServiceError(errcode.ErrAICallFailed, "AI模型调用失败: "+err.Error())
	}

	// 补充调用耗时
	if resp.DurationMs == 0 {
		resp.DurationMs = time.Since(startTime).Milliseconds()
	}

	// 结算：用实际消耗替换预扣额度（释放预扣，加上实际值）
	_ = s.settleTokenUsage(tenantID, reserved, resp.TokenUsage.TotalTokens)

	// 异步写入日志（带重试）
	s.asyncWriteLog(tenantID, userID, modelCfg.ID, resp)

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
// 调用前对用户提示词执行数据脱敏，调用后结算 Token 并异步写入日志。
func (s *AIModelCallerService) ChatViaPython(c *gin.Context, tenantID, userID uuid.UUID, modelCfg *model.AIModelConfig, req *ai.ChatRequest, auditContext map[string]interface{}) (*ai.ChatResponse, error) {
	// 预扣 Token 配额
	reserved := req.MaxTokens
	if reserved == 0 {
		reserved = modelCfg.MaxTokens
	}
	if err := s.reserveTokenQuota(tenantID, reserved); err != nil {
		return nil, err
	}

	// 数据脱敏：对用户提示词中的敏感信息进行脱敏
	sanitizedUserPrompt := sanitize.SanitizeText(req.UserPrompt)

	// 构建请求体（包含完整模型配置供 Python 端使用）
	pyReq := pythonAIRequest{
		SystemPrompt: req.SystemPrompt,
		UserPrompt:   sanitizedUserPrompt,
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
		_ = s.releaseTokenQuota(tenantID, reserved)
		return nil, newServiceError(errcode.ErrAICallFailed, "Python AI服务调用失败: "+err.Error())
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		_ = s.releaseTokenQuota(tenantID, reserved)
		respBody, _ := io.ReadAll(httpResp.Body)
		return nil, newServiceError(errcode.ErrAICallFailed, fmt.Sprintf("Python AI服务返回错误(%d): %s", httpResp.StatusCode, string(respBody)))
	}

	// 解析响应
	var pyResp pythonAIResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&pyResp); err != nil {
		_ = s.releaseTokenQuota(tenantID, reserved)
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
	s.asyncWriteLog(tenantID, userID, modelCfg.ID, resp)

	return resp, nil
}

// ── Token 配额原子操作 ─────────────────────────────────────

// reserveTokenQuota 原子预扣 Token 配额。
// 使用 UPDATE ... WHERE token_used + ? <= token_quota 保证不超额，
// 高并发下多个请求同时预扣时，只有总量不超配额的才能成功。
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

// releaseTokenQuota 回滚预扣的 Token 配额（调用失败时使用）。
func (s *AIModelCallerService) releaseTokenQuota(tenantID uuid.UUID, amount int) error {
	return s.db.Model(&model.Tenant{}).
		Where("id = ?", tenantID).
		Update("token_used", gorm.Expr("GREATEST(token_used - ?, 0)", amount)).Error
}

// settleTokenUsage 结算实际 Token 消耗：释放预扣额度，加上实际消耗。
// 等价于 token_used = token_used - reserved + actual
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

// asyncWriteLog 异步写入 LLM 调用日志，失败时指数退避重试。
func (s *AIModelCallerService) asyncWriteLog(tenantID, userID uuid.UUID, modelConfigID uuid.UUID, resp *ai.ChatResponse) {
	go func() {
		entry := &model.TenantLLMMessageLog{
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

		var err error
		for attempt := 0; attempt < logMaxRetries; attempt++ {
			if err = s.logRepo.Create(entry); err == nil {
				return
			}
			// 指数退避: 1s, 2s, 4s
			time.Sleep(time.Duration(1<<attempt) * time.Second)
		}
		// 重试耗尽，记录到标准日志（运维可通过日志采集发现）
		log.Printf("[WARN] LLM日志写入失败(tenant=%s, model=%s, tokens=%d): %v",
			tenantID, modelConfigID, resp.TokenUsage.TotalTokens, err)
	}()
}
