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

// AIModelCallerService 负责 AI 模型调用的完整生命周期管理：
// Token 配额预扣与结算、调用执行、异步日志写入。
type AIModelCallerService struct {
	tenantRepo *repository.TenantRepo
	logRepo    *repository.LLMMessageLogRepo
	db         *gorm.DB
}

// NewAIModelCallerService 初始化 AI 调用服务，注入租户仓储、日志仓储和数据库连接。
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

// Chat 执行单次 AI 对话调用，完整流程为：
// 1. 预扣 Token 配额（防止并发超额）
// 2. 创建对应部署类型的调用器并发起请求
// 3. 调用失败时回滚预扣额度
// 4. 调用成功后结算实际消耗，并异步写入调用日志
func (s *AIModelCallerService) Chat(c *gin.Context, tenantID, userID uuid.UUID, modelCfg *model.AIModelConfig, req *ai.ChatRequest) (*ai.ChatResponse, error) {
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
	s.asyncWriteLog(tenantID, userID, modelCfg.ID, req.RequestType, resp)

	return resp, nil
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
	s.asyncWriteLog(tenantID, userID, modelCfg.ID, req.RequestType, resp)

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
func (s *AIModelCallerService) asyncWriteLog(tenantID, userID uuid.UUID, modelConfigID uuid.UUID, requestType string, resp *ai.ChatResponse) {
	go func() {
		entry := &model.TenantLLMMessageLog{
			ID:            uuid.New(),
			TenantID:      tenantID,
			UserID:        &userID,
			ModelConfigID: &modelConfigID,
			RequestType:   requestType,
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
