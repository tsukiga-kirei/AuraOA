package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"go.uber.org/zap"

	"oa-smart-audit/go-service/internal/cache"
	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/crypto"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	jwtpkg "oa-smart-audit/go-service/internal/pkg/jwt"
	pkglogger "oa-smart-audit/go-service/internal/pkg/logger"
	"oa-smart-audit/go-service/internal/pkg/oa"
	"oa-smart-audit/go-service/internal/repository"
)

const (
	batchAuditMaxLimit = 10

	// auditErrStaleMessage 超时任务写入 error_message 的固定文案
	auditErrStaleMessage = "审核任务超时（请重新发起）"
	// auditJobMaxAge 非终态（pending/reasoning/extracting）超过此时长则标记为 failed（排队过久或执行卡住）
	auditJobMaxAge = 30 * time.Minute
	// auditProcessTimeout 单条异步任务 AI+OA 链路 context 上限，须小于 auditJobMaxAge，避免与对账任务竞态
	auditProcessTimeout = 25 * time.Minute
)

// AuditExecuteService 审核执行业务逻辑：串联 OA 数据 → 提示词构建 → AI 调用 → 结果解析 → 写入日志。
type AuditExecuteService struct {
	auditLogRepo      *repository.AuditLogRepo
	auditSnapshotRepo *repository.AuditProcessSnapshotRepo
	configRepo        *repository.ProcessAuditConfigRepo
	ruleRepo          *repository.AuditRuleRepo
	userConfigRepo    *repository.UserPersonalConfigRepo
	tenantRepo        *repository.TenantRepo
	oaConnRepo        *repository.OAConnectionRepo
	aiModelRepo       *repository.AIModelRepo
	aiCaller          *AIModelCallerService
	db                *gorm.DB
	rdb               *redis.Client
	notifSvc          *UserNotificationService
	cancelMap         sync.Map
	cache             *cache.CacheManager
	invalidator       *cache.InvalidationManager
}

// NewAuditExecuteService 创建 AuditExecuteService，注入所有依赖仓储和服务。
func NewAuditExecuteService(
	auditLogRepo *repository.AuditLogRepo,
	auditSnapshotRepo *repository.AuditProcessSnapshotRepo,
	configRepo *repository.ProcessAuditConfigRepo,
	ruleRepo *repository.AuditRuleRepo,
	userConfigRepo *repository.UserPersonalConfigRepo,
	tenantRepo *repository.TenantRepo,
	oaConnRepo *repository.OAConnectionRepo,
	aiModelRepo *repository.AIModelRepo,
	aiCaller *AIModelCallerService,
	db *gorm.DB,
	rdb *redis.Client,
	notifSvc *UserNotificationService,
	cacheManager *cache.CacheManager,
	invalidationManager *cache.InvalidationManager,
) *AuditExecuteService {
	return &AuditExecuteService{
		auditLogRepo:      auditLogRepo,
		auditSnapshotRepo: auditSnapshotRepo,
		configRepo:        configRepo,
		ruleRepo:          ruleRepo,
		userConfigRepo:    userConfigRepo,
		tenantRepo:        tenantRepo,
		oaConnRepo:        oaConnRepo,
		aiModelRepo:       aiModelRepo,
		aiCaller:          aiCaller,
		db:                db,
		rdb:               rdb,
		notifSvc:          notifSvc,
		cache:             cacheManager,
		invalidator:       invalidationManager,
	}
}

func (s *AuditExecuteService) BatchRdb() *redis.Client {
	return s.rdb
}

// AuditExecuteRequest 审核执行请求
type AuditExecuteRequest struct {
	ProcessID   string `json:"process_id" binding:"required"`
	ProcessType string `json:"process_type" binding:"required"`
	Title       string `json:"title"`
}

// AuditExecuteResponse 审核执行响应
type AuditExecuteResponse struct {
	Status         string                 `json:"status,omitempty"` // pending：已入队异步处理；completed：同步完成（保留兼容）
	ID             string                 `json:"id"`
	TraceID        string                 `json:"trace_id"`
	ProcessID      string                 `json:"process_id"`
	Recommendation string                 `json:"recommendation,omitempty"`
	OverallScore   int                    `json:"overall_score,omitempty"`
	RuleResults    []model.RuleResultJSON `json:"rule_results,omitempty"`
	RiskPoints     []string               `json:"risk_points,omitempty"`
	Suggestions    []string               `json:"suggestions,omitempty"`
	Confidence     int                    `json:"confidence,omitempty"`
	AIReasoning    string                 `json:"ai_reasoning,omitempty"`
	DurationMs     int                    `json:"duration_ms,omitempty"`
	CreatedAt      string                 `json:"created_at"`
	ParseError     string                 `json:"parse_error,omitempty"`
	RawContent     string                 `json:"raw_content,omitempty"`
}

// createPendingAuditLog 校验配置并写入 pending 记录（供单条异步与批量同步共用）。
func (s *AuditExecuteService) createPendingAuditLog(c *gin.Context, req *AuditExecuteRequest) (logID uuid.UUID, tenantID uuid.UUID, userID uuid.UUID, err error) {
	tenantID, userID, err = s.extractIDs(c)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, err
	}

	tenant, err := s.tenantRepo.FindByID(tenantID)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, newServiceError(errcode.ErrDatabase, "获取租户信息失败")
	}
	if tenant.PrimaryModelID == nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, newServiceError(errcode.ErrNoAIModelConfig, "租户未配置主用 AI 模型")
	}
	if _, err := s.aiModelRepo.FindByID(*tenant.PrimaryModelID); err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, newServiceError(errcode.ErrNoAIModelConfig, "AI 模型配置不存在")
	}

	if _, err := s.configRepo.GetByProcessType(c, req.ProcessType); err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, newServiceError(errcode.ErrNoProcessConfig, fmt.Sprintf("流程 '%s' 的审核配置不存在", req.ProcessType))
	}

	logID = uuid.New()
	now := time.Now()
	logEntry := &model.AuditLog{
		ID:             logID,
		TenantID:       tenantID,
		UserID:         userID,
		ProcessID:      req.ProcessID,
		Title:          req.Title,
		ProcessType:    req.ProcessType,
		Status:         model.JobStatusPending,
		Recommendation: "",
		Score:          0,
		AuditResult:    datatypes.JSON([]byte("{}")),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.auditLogRepo.Create(logEntry); err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, newServiceError(errcode.ErrDatabase, "审核日志写入失败")
	}
	return logID, tenantID, userID, nil
}

// Execute 异步提交审核：写入 pending 记录并入 Redis Stream，立即返回 job id。
func (s *AuditExecuteService) Execute(c *gin.Context, req *AuditExecuteRequest) (*AuditExecuteResponse, error) {
	if s.rdb == nil {
		return nil, newServiceError(errcode.ErrInternalServer, "异步队列未初始化（Redis 不可用）")
	}

	logID, tenantID, userID, err := s.createPendingAuditLog(c, req)
	if err != nil {
		return nil, err
	}

	log, _ := s.auditLogRepo.GetByID(c, logID)
	createdAt := log.CreatedAt

	if _, err := EnqueueAuditJob(c.Request.Context(), s.rdb, logID, tenantID, userID); err != nil {
		_ = s.auditLogRepo.UpdateFields(c, logID, map[string]interface{}{
			"status":        model.JobStatusFailed,
			"error_message": "任务入队失败: " + err.Error(),
			"updated_at":    time.Now(),
		})
		pkglogger.Global().Warn("审核任务入队失败",
			zap.String("logID", logID.String()),
			zap.Error(err),
		)
		return nil, newServiceError(errcode.ErrRedisConn, "审核任务入队失败: "+err.Error())
	}

	pkglogger.Global().Info("审核任务已入队",
		zap.String("logID", logID.String()),
		zap.String("processID", req.ProcessID),
		zap.String("tenantID", tenantID.String()),
	)

	return &AuditExecuteResponse{
		Status:    model.JobStatusPending,
		ID:        logID.String(),
		TraceID:   fmt.Sprintf("TR-%s", logID.String()[:8]),
		ProcessID: req.ProcessID,
		CreatedAt: createdAt.Format(time.RFC3339),
	}, nil
}

func (s *AuditExecuteService) workerGinContext(ctx context.Context, tenantID, userID uuid.UUID) *gin.Context {
	rec := httptest.NewRecorder()
	gc, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPost, "/", nil).WithContext(ctx)
	gc.Request = req
	gc.Set("tenant_id", tenantID.String())
	gc.Set("jwt_claims", &jwtpkg.JWTClaims{Sub: userID.String(), Username: ""})
	return gc
}

func (s *AuditExecuteService) markAuditFailed(c *gin.Context, id uuid.UUID, err error) {
	msg := err.Error()
	var se *ServiceError
	if errors.As(err, &se) {
		msg = se.Message
	}
	_ = s.auditLogRepo.UpdateFields(c, id, map[string]interface{}{
		"status":        model.JobStatusFailed,
		"error_message": msg,
		"updated_at":    time.Now(),
	})
}

// markAuditFailedDB 不依赖 Gin Context（避免 context 已取消时无法落库），用于收尾保存失败或超时标记。
func (s *AuditExecuteService) markAuditFailedDB(tenantID, id uuid.UUID, message string) error {
	return s.db.Model(&model.AuditLog{}).
		Where("id = ? AND tenant_id = ?", id, tenantID).
		Updates(map[string]interface{}{
			"status":        model.JobStatusFailed,
			"error_message": message,
			"updated_at":    time.Now(),
		}).Error
}

// markAuditFailedOrTimeout 失败落库；若因 context 超时（AI/OA 过久），用 DB 直写避免 Gin Context 已取消导致无法更新。
func (s *AuditExecuteService) markAuditFailedOrTimeout(c *gin.Context, tenantID, id uuid.UUID, err error) {
	if err == nil {
		return
	}
	if errors.Is(err, context.DeadlineExceeded) {
		_ = s.markAuditFailedDB(tenantID, id, "审核任务执行超时（请重新发起）")
		return
	}
	s.markAuditFailed(c, id, err)
}

// applyStaleAuditTimeout 轮询时若任务过久未结束，标记为 failed 并返回最新行。
func (s *AuditExecuteService) applyStaleAuditTimeout(c *gin.Context, log *model.AuditLog) (*model.AuditLog, error) {
	if log == nil {
		return nil, nil
	}
	switch log.Status {
	case model.JobStatusCompleted, model.JobStatusFailed:
		return log, nil
	}
	if time.Since(log.CreatedAt) <= auditJobMaxAge {
		return log, nil
	}
	if err := s.auditLogRepo.UpdateFields(c, log.ID, map[string]interface{}{
		"status":        model.JobStatusFailed,
		"error_message": auditErrStaleMessage,
		"updated_at":    time.Now(),
	}); err != nil {
		return log, nil
	}
	return s.auditLogRepo.GetByID(c, log.ID)
}

// FailStaleAuditJobs 将长时间未结束的非终态任务标记为失败（全租户后台对账）。
func (s *AuditExecuteService) FailStaleAuditJobs(ctx context.Context) (int64, error) {
	cutoff := time.Now().Add(-auditJobMaxAge)
	res := s.db.WithContext(ctx).Model(&model.AuditLog{}).
		Where("status IN ? AND created_at < ?", []string{
			model.JobStatusPending,
			model.JobStatusAssembling,
			model.JobStatusReasoning,
			model.JobStatusExtracting,
		}, cutoff).
		Updates(map[string]interface{}{
			"status":        model.JobStatusFailed,
			"error_message": auditErrStaleMessage,
			"updated_at":    time.Now(),
		})
	if res.RowsAffected > 0 {
		pkglogger.Global().Warn("超时审核任务已标记为失败",
			zap.Int64("count", res.RowsAffected),
		)
	}
	return res.RowsAffected, res.Error
}

// updateAuditLogIfNotCancelled 用户已中止（failed）时不再被后续阶段覆盖，避免 Cancel 后任务仍写完成为 completed。
func (s *AuditExecuteService) updateAuditLogIfNotCancelled(tenantID, auditLogID uuid.UUID, updates map[string]interface{}) (int64, error) {
	res := s.db.Model(&model.AuditLog{}).
		Where("id = ? AND tenant_id = ? AND status NOT IN ?", auditLogID, tenantID, []string{model.JobStatusFailed}).
		Updates(updates)
	return res.RowsAffected, res.Error
}

// processAuditJob 由 Redis Stream Worker 调用，执行完整审核链路。
func (s *AuditExecuteService) processAuditJob(ctx context.Context, auditLogID, tenantID, userID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, auditProcessTimeout)
	s.cancelMap.Store(auditLogID.String(), cancel)
	defer func() {
		cancel()
		s.cancelMap.Delete(auditLogID.String())
	}()

	c := s.workerGinContext(ctx, tenantID, userID)
	log, err := s.auditLogRepo.GetByID(c, auditLogID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if log.Status != model.JobStatusPending {
		return nil
	}
	// 队列积压过久：不再执行，直接标记失败（与 FailStaleAuditJobs 一致）
	if time.Since(log.CreatedAt) > auditJobMaxAge {
		_ = s.markAuditFailedDB(tenantID, auditLogID, auditErrStaleMessage)
		return nil
	}

	pkglogger.Global().Info("开始执行审核任务",
		zap.String("auditLogID", auditLogID.String()),
		zap.String("processType", log.ProcessType),
	)

	startTime := time.Now()
	tenant, err := s.tenantRepo.FindByID(tenantID)
	if err != nil {
		s.markAuditFailedOrTimeout(c, tenantID, auditLogID, newServiceError(errcode.ErrDatabase, "获取租户信息失败"))
		pkglogger.Global().Warn("审核任务执行失败",
			zap.String("auditLogID", auditLogID.String()),
			zap.Error(err),
		)
		return err
	}

	// 获取租户专属 logger，后续审核日志同时写入租户文件和全局文件
	tlog := pkglogger.GetTenantLogger(tenant.Code)

	if tenant.PrimaryModelID == nil {
		se := newServiceError(errcode.ErrNoAIModelConfig, "租户未配置主用 AI 模型")
		s.markAuditFailedOrTimeout(c, tenantID, auditLogID, se)
		tlog.Warn("审核任务执行失败",
			zap.String("auditLogID", auditLogID.String()),
			zap.Error(se),
		)
		return se
	}
	modelCfg, err := s.aiModelRepo.FindByID(*tenant.PrimaryModelID)
	if err != nil {
		se := newServiceError(errcode.ErrNoAIModelConfig, "AI 模型配置不存在")
		s.markAuditFailedOrTimeout(c, tenantID, auditLogID, se)
		tlog.Warn("审核任务执行失败",
			zap.String("auditLogID", auditLogID.String()),
			zap.Error(se),
		)
		return se
	}

	// 解密 API Key，确保云端模型能够携带真实秘钥发起 HTTP 调用
	if modelCfg.APIKey != "" {
		decrypted, err := crypto.Decrypt(modelCfg.APIKey)
		if err != nil {
			se := newServiceError(errcode.ErrInternalServer, "API Key 解密失败")
			s.markAuditFailedOrTimeout(c, tenantID, auditLogID, se)
			tlog.Warn("审核任务执行失败",
				zap.String("auditLogID", auditLogID.String()),
				zap.Error(se),
			)
			return se
		}
		modelCfg.APIKey = decrypted
	}

	req := &AuditExecuteRequest{ProcessID: log.ProcessID, ProcessType: log.ProcessType, Title: log.Title}

	config, rules, err := s.getProcessConfigCached(c, tenantID, req.ProcessType)
	if err != nil {
		s.markAuditFailedOrTimeout(c, tenantID, auditLogID, err)
		tlog.Warn("审核任务执行失败",
			zap.String("auditLogID", auditLogID.String()),
			zap.Error(err),
		)
		return err
	}

	var aiConfig model.AIConfigData
	if err := json.Unmarshal(config.AIConfig, &aiConfig); err != nil {
		se := newServiceError(errcode.ErrInternalServer, "AI 配置解析失败")
		s.markAuditFailedOrTimeout(c, tenantID, auditLogID, se)
		tlog.Warn("审核任务执行失败",
			zap.String("auditLogID", auditLogID.String()),
			zap.Error(se),
		)
		return se
	}

	fieldSet, mergedRulesText := s.resolveUserConfig(c, userID, config, rules, req.ProcessType)

	n, err := s.updateAuditLogIfNotCancelled(tenantID, auditLogID, map[string]interface{}{
		"status":     model.JobStatusAssembling,
		"updated_at": time.Now(),
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return nil
	}

	processData, err := s.fetchOAData(c, tenant, req.ProcessID)
	if err != nil {
		s.markAuditFailedOrTimeout(c, tenantID, auditLogID, err)
		tlog.Warn("审核任务执行失败",
			zap.String("auditLogID", auditLogID.String()),
			zap.Error(err),
		)
		return err
	}

	currentNode := "当前节点"

	reasoningReq := BuildReasoningPrompt(&aiConfig, req.ProcessType, processData, mergedRulesText, currentNode, fieldSet)
	reasoningReq.Temperature = float64(tenant.Temperature)
	reasoningReq.MaxTokens = tenant.MaxTokensPerRequest
	reasoningReq.ModelConfig = modelCfg

	// 注入流式回调，将增量写入 Redis 并通过 PubSub 广播
	reasoningReq.StreamChunkFunc = func(chunk string) {
		key := "audit:reasoning:" + auditLogID.String()
		s.rdb.Append(context.Background(), key, chunk)
		s.rdb.Expire(context.Background(), key, 24*time.Hour)
		s.rdb.Publish(context.Background(), "audit:stream:"+auditLogID.String(), chunk)
	}

	n, err = s.updateAuditLogIfNotCancelled(tenantID, auditLogID, map[string]interface{}{
		"status":     model.JobStatusReasoning,
		"updated_at": time.Now(),
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return nil
	}

	reasoningResp, err := s.aiCaller.Chat(c, tenantID, userID, modelCfg, reasoningReq)
	if err != nil {
		s.markAuditFailedOrTimeout(c, tenantID, auditLogID, err)
		tlog.Warn("审核任务执行失败",
			zap.String("auditLogID", auditLogID.String()),
			zap.Error(err),
		)
		return err
	}
	aiReasoning := reasoningResp.Content

	n, err = s.updateAuditLogIfNotCancelled(tenantID, auditLogID, map[string]interface{}{
		"status":       model.JobStatusExtracting,
		"ai_reasoning": aiReasoning,
		"updated_at":   time.Now(),
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return nil
	}

	extractionReq := BuildExtractionPrompt(&aiConfig, aiReasoning, mergedRulesText)
	extractionReq.Temperature = 0.1
	extractionReq.MaxTokens = tenant.MaxTokensPerRequest
	extractionReq.ModelConfig = modelCfg
	extractionReq.SkipQuotaCheck = true

	extractionResp, err := s.aiCaller.Chat(c, tenantID, userID, modelCfg, extractionReq)
	if err != nil {
		s.markAuditFailedOrTimeout(c, tenantID, auditLogID, err)
		tlog.Warn("审核任务执行失败",
			zap.String("auditLogID", auditLogID.String()),
			zap.Error(err),
		)
		return err
	}

	totalDuration := int(time.Since(startTime).Milliseconds())
	parsed, parseErr := ParseAuditResult(extractionResp.Content)

	tlog.Info("AI 推理完成",
		zap.String("auditLogID", auditLogID.String()),
		zap.Int("durationMs", totalDuration),
	)

	updates := map[string]interface{}{
		"duration_ms":  totalDuration,
		"raw_content":  extractionResp.Content,
		"ai_reasoning": aiReasoning,
		"updated_at":   time.Now(),
	}

	if parseErr != nil {
		updates["status"] = model.JobStatusFailed
		updates["recommendation"] = ""
		updates["score"] = 0
		updates["confidence"] = 0
		updates["parse_error"] = parseErr.Error()
		updates["audit_result"] = datatypes.JSON([]byte("{}"))
		tlog.Warn("审核结果解析失败",
			zap.String("auditLogID", auditLogID.String()),
			zap.Error(parseErr),
		)
	} else {
		resultJSON, _ := json.Marshal(parsed)
		updates["status"] = model.JobStatusCompleted
		updates["recommendation"] = parsed.Recommendation
		updates["score"] = parsed.OverallScore
		updates["confidence"] = parsed.Confidence
		updates["audit_result"] = datatypes.JSON(resultJSON)
		tlog.Info("审核任务执行完成",
			zap.String("auditLogID", auditLogID.String()),
			zap.String("recommendation", parsed.Recommendation),
			zap.Int("score", parsed.OverallScore),
		)
	}

	finalRows, err := s.updateAuditLogIfNotCancelled(tenantID, auditLogID, updates)
	if err != nil {
		_ = s.markAuditFailedDB(tenantID, auditLogID, "保存审核结果失败: "+err.Error())
		return err
	}
	if finalRows == 0 {
		return nil
	}
	if parseErr == nil && parsed != nil {
		if err := s.auditSnapshotRepo.UpsertAppendValid(c, tenantID, log.ProcessID, auditLogID, log.Title, log.ProcessType, parsed.Recommendation, parsed.OverallScore, parsed.Confidence); err != nil {
			return err
		}
		// 审核完成通知
		if s.notifSvc != nil {
			s.notifSvc.CreateByTenant(userID, tenantID, "audit",
				fmt.Sprintf("审核完成：%s", log.Title),
				fmt.Sprintf("评分 %d，建议：%s", parsed.OverallScore, parsed.Recommendation),
				fmt.Sprintf("/workbench?processId=%s", log.ProcessID),
			)
		}

		// 审核完成后清除相关缓存（使用 Background context 避免请求取消影响缓存失效）
		if s.invalidator != nil {
			bgCtx := context.Background()
			_ = s.invalidator.InvalidateUserTodoCache(bgCtx, tenantID, userID)
			_ = s.invalidator.InvalidateSnapshotCache(bgCtx, tenantID, "audit")
			_ = s.invalidator.InvalidateStatsCache(bgCtx, tenantID, "audit")
		}
	}
	return nil
}

// CancelJob 主动中止审核任务，设置状态为 failed 并取消执行上下文
func (s *AuditExecuteService) CancelJob(c *gin.Context, id uuid.UUID) error {
	tenantID, _, err := s.extractIDs(c)
	if err != nil {
		return err
	}
	err = s.markAuditFailedDB(tenantID, id, "已主动中止")
	if cancelFunc, ok := s.cancelMap.Load(id.String()); ok {
		cancelFunc.(context.CancelFunc)()
	}
	if err == nil {
		pkglogger.Global().Info("审核任务已取消",
			zap.String("jobID", id.String()),
			zap.String("tenantID", tenantID.String()),
		)
	}
	return err
}

// BatchExecute 批量审核（上限 10 条）：同步逐条执行，不经过 Redis Stream，避免与 Worker 重复消费。
func (s *AuditExecuteService) BatchExecute(c *gin.Context, items []AuditExecuteRequest) (*BatchAuditResult, error) {
	if len(items) > batchAuditMaxLimit {
		return nil, newServiceError(errcode.ErrBatchLimitExceeded,
			fmt.Sprintf("批量审核上限 %d 条，当前 %d 条", batchAuditMaxLimit, len(items)))
	}

	result := &BatchAuditResult{
		Total: len(items),
	}

	for _, item := range items {
		logID, tenantID, userID, err := s.createPendingAuditLog(c, &item)
		if err != nil {
			result.Failed++
			result.Results = append(result.Results, AuditExecuteResponse{
				ProcessID:  item.ProcessID,
				ParseError: err.Error(),
			})
			continue
		}
		if err := s.processAuditJob(c.Request.Context(), logID, tenantID, userID); err != nil {
			result.Failed++
			result.Results = append(result.Results, AuditExecuteResponse{
				ID:         logID.String(),
				ProcessID:  item.ProcessID,
				ParseError: err.Error(),
			})
			continue
		}
		final, err := s.auditLogRepo.GetByID(c, logID)
		if err != nil {
			result.Failed++
			result.Results = append(result.Results, AuditExecuteResponse{
				ProcessID:  item.ProcessID,
				ParseError: "读取审核结果失败",
			})
			continue
		}
		resp := auditExecuteResponseFromLog(final)
		result.Success++
		result.Results = append(result.Results, *resp)
	}

	return result, nil
}

func auditExecuteResponseFromLog(log *model.AuditLog) *AuditExecuteResponse {
	traceID := fmt.Sprintf("TR-%s-%s", time.Now().Format("20060102150405"), log.ID.String()[:8])
	resp := &AuditExecuteResponse{
		ID:          log.ID.String(),
		TraceID:     traceID,
		ProcessID:   log.ProcessID,
		AIReasoning: log.AIReasoning,
		DurationMs:  log.DurationMs,
		CreatedAt:   log.CreatedAt.Format(time.RFC3339),
		Status:      log.Status,
	}
	if log.Status == model.JobStatusFailed {
		resp.Recommendation = ""
		resp.ParseError = log.ErrorMessage
		resp.RuleResults = []model.RuleResultJSON{}
		resp.RiskPoints = []string{}
		resp.Suggestions = []string{}
		return resp
	}
	if log.ParseError != "" {
		resp.Recommendation = ""
		resp.OverallScore = 0
		resp.Confidence = 0
		resp.ParseError = log.ParseError
		resp.RawContent = log.RawContent
		resp.RuleResults = []model.RuleResultJSON{}
		resp.RiskPoints = []string{}
		resp.Suggestions = []string{}
		return resp
	}
	var parsed model.AuditResultJSON
	if err := json.Unmarshal(log.AuditResult, &parsed); err == nil {
		resp.Recommendation = parsed.Recommendation
		resp.OverallScore = parsed.OverallScore
		resp.RuleResults = parsed.RuleResults
		resp.RiskPoints = parsed.RiskPoints
		resp.Suggestions = parsed.Suggestions
		resp.Confidence = parsed.Confidence
	}
	return resp
}

type BatchAuditResult struct {
	Results []AuditExecuteResponse `json:"results"`
	Total   int                    `json:"total"`
	Success int                    `json:"success"`
	Failed  int                    `json:"failed"`
}

// GetAuditChain 获取审核链：仅包含有效解析成功的记录，顺序与快照中一致。
func (s *AuditExecuteService) GetAuditChain(c *gin.Context, processID string) ([]repository.AuditLogWithUser, error) {
	snap, err := s.auditSnapshotRepo.GetByProcessID(c, processID)
	if err != nil {
		return nil, err
	}
	if snap == nil {
		return []repository.AuditLogWithUser{}, nil
	}
	ids := parseSnapshotValidLogIDs(snap.ValidLogIDs)
	return s.auditLogRepo.ListByIDsWithUserOrdered(c, ids)
}

// SubscribeJobStream 获取特定流程的 SSE 流和控制句柄
func (s *AuditExecuteService) SubscribeJobStream(ctx context.Context, id uuid.UUID) (<-chan string, func(), error) {
	pubsub := s.rdb.Subscribe(ctx, "audit:stream:"+id.String())
	ch := make(chan string)

	history, _ := s.rdb.Get(ctx, "audit:reasoning:"+id.String()).Result()

	go func() {
		defer close(ch)
		// 如果已有累计，则首先把累计发给前端铺底
		if history != "" {
			ch <- history
		}
		for msg := range pubsub.Channel() {
			ch <- msg.Payload
		}
	}()

	return ch, func() { pubsub.Close() }, nil
}

// GetAuditJobStatus 轮询异步审核任务状态（含进度阶段说明）。
func (s *AuditExecuteService) GetAuditJobStatus(c *gin.Context, id uuid.UUID) (map[string]interface{}, error) {
	log, err := s.auditLogRepo.GetByID(c, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, newServiceError(errcode.ErrResourceNotFound, "审核任务不存在")
		}
		return nil, newServiceError(errcode.ErrDatabase, "查询审核任务失败")
	}
	log, err = s.applyStaleAuditTimeout(c, log)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "查询审核任务失败")
	}
	out := buildAuditResultFromLog(log)
	out["updated_at"] = log.UpdatedAt.Format(time.RFC3339)
	out["progress_steps"] = auditProgressSteps(log.Status)
	return out, nil
}

func auditProgressSteps(status string) []map[string]interface{} {
	defs := []struct {
		key   string
		label string
	}{
		{model.JobStatusPending, "排队中"},
		{model.JobStatusAssembling, "组装提示词"},
		{model.JobStatusReasoning, "推理分析"},
		{model.JobStatusExtracting, "结构化提取"},
	}
	phaseIdx := map[string]int{
		model.JobStatusPending:    0,
		model.JobStatusAssembling: 1,
		model.JobStatusReasoning:  2,
		model.JobStatusExtracting: 3,
	}
	cur, ok := phaseIdx[status]
	if !ok {
		if status == model.JobStatusCompleted {
			cur = 3
		} else if status == model.JobStatusFailed {
			cur = 2
		} else {
			cur = 0
		}
	}
	var steps []map[string]interface{}
	for i, d := range defs {
		m := map[string]interface{}{"key": d.key, "label": d.label}
		switch {
		case status == model.JobStatusFailed && i == cur:
			m["failed"] = true
		case i < cur:
			m["done"] = true
		case i == cur && cur < 4 && status != model.JobStatusFailed:
			m["current"] = true
		}
		steps = append(steps, m)
	}
	if status == model.JobStatusCompleted {
		steps = append(steps, map[string]interface{}{"key": "done", "label": "已完成", "done": true})
	}
	return steps
}

// ListAuditLogs 数据管理页：分页查询当前租户审核日志。
func (s *AuditExecuteService) ListAuditLogs(c *gin.Context, filter repository.AuditLogFilter, page, pageSize int) ([]repository.AuditLogWithUser2, int64, error) {
	items, total, err := s.auditLogRepo.ListPagedWithUser(c, filter, page, pageSize)
	if err != nil {
		return nil, 0, newServiceError(errcode.ErrDatabase, "查询审核日志失败")
	}
	return items, total, nil
}

// GetAuditLogStats 数据管理页：获取当前租户审核日志统计。
func (s *AuditExecuteService) GetAuditLogStats(c *gin.Context) (*repository.AuditLogStats, error) {
	stats, err := s.auditLogRepo.CountStats(c, nil)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "统计查询失败")
	}
	return stats, nil
}

// ListPendingForBatch 为调度器提供：按当前上下文中的 OA 用户拉取待审批流程（已按租户配置过滤），
// 供 cron audit_batch 任务批量调用（任务归属用户即 OA 待办所属用户）。
func (s *AuditExecuteService) ListPendingForBatch(c *gin.Context, workflowIds []string, dateRangeDays int, limit int) ([]AuditExecuteRequest, error) {
	tenantID, _, err := s.extractIDs(c)
	if err != nil {
		return nil, err
	}
	username := s.extractUsername(c)
	if username == "" {
		return nil, newServiceError(errcode.ErrParamValidation, "无法解析 OA 登录用户名，请检查任务归属用户账号")
	}
	adapter, err := s.getOAAdapter(tenantID)
	if err != nil {
		return nil, err
	}

	filter := oa.TodoListFilter{}
	if dateRangeDays > 0 {
		start := time.Now().AddDate(0, 0, -dateRangeDays)
		filter.SubmitDateStart = &start
	}

	items, err := adapter.FetchTodoList(c.Request.Context(), username, filter)
	if err != nil {
		return nil, newServiceError(errcode.ErrOAQueryFailed, "获取 OA 用户待办失败: "+err.Error())
	}

	// 4. 获取当前待办项的快照状态，用于排除已处理项（已复核流程有快照记录）
	todoPIDs := make([]string, len(items))
	for i, it := range items {
		todoPIDs[i] = it.ProcessID
	}
	snapshotMap, _ := s.getSnapshotMapCached(c, tenantID, todoPIDs)

	// 按租户已配置的主表名和流程类型过滤（AND 关系）
	allowedTables := s.getAllowedMainTables(c)
	allowedTypes := make(map[string]bool)
	for _, pt := range s.getAllowedProcessTypes(c) {
		allowedTypes[strings.ToLower(pt)] = true
	}
	wfMap := make(map[string]bool)
	for _, id := range workflowIds {
		wfMap[id] = true
	}

	var result []AuditExecuteRequest
	for _, item := range items {
		// 1. 权限与配置过滤（主表名 AND 流程类型必须同时匹配）
		if !allowedTables[strings.ToLower(item.MainTableName)] || !allowedTypes[strings.ToLower(item.ProcessType)] {
			continue
		}
		// 2. 指定流程过滤（若有）
		if len(wfMap) > 0 && !wfMap[item.ProcessID] && !wfMap[item.ProcessType] {
			continue
		}

		// 3. 排除已处理（待 AI 审核逻辑：若 snapshotMap 中不存在有效记录，则为待处理）
		if _, exists := snapshotMap[item.ProcessID]; exists {
			continue
		}

		result = append(result, AuditExecuteRequest{
			ProcessID:   item.ProcessID,
			ProcessType: item.ProcessType,
			Title:       item.Title,
		})
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result, nil
}

func todoListFilterFromAuditParams(p dto.AuditListParams) oa.TodoListFilter {
	return oa.TodoListFilter{
		SubmitDateStart:        p.SubmitDateStart,
		SubmitDateEndExclusive: p.SubmitDateEndExclusive,
	}
}

// collectTodoProcessIDsForExclusion 当前用户仍在 OA 待办中的流程 id（不按 workflow_requestbase.createdate 过滤）。
// 「全部已完成」与统计 completed_count 排除待办时必须用全量待办；若与列表同一日期条件，会把「提交日不在范围内但仍在待办」的流程误判为已完成。
func (s *AuditExecuteService) collectTodoProcessIDsForExclusion(c *gin.Context, username string, adapter oa.OAAdapter) ([]string, error) {
	todoItems, err := adapter.FetchTodoList(c.Request.Context(), username, oa.TodoListFilter{})
	if err != nil {
		return nil, err
	}
	allowedTables := s.getAllowedMainTables(c)
	allowedTypes := make(map[string]bool)
	for _, pt := range s.getAllowedProcessTypes(c) {
		allowedTypes[strings.ToLower(pt)] = true
	}
	var ids []string
	for _, item := range todoItems {
		if allowedTables[strings.ToLower(item.MainTableName)] && allowedTypes[strings.ToLower(item.ProcessType)] {
			ids = append(ids, item.ProcessID)
		}
	}
	return ids, nil
}

func normalizeAuditPage(page, pageSize int) (p int, ps int, start int, end int) {
	p = page
	if p < 1 {
		p = 1
	}
	ps = pageSize
	if ps < 1 || ps > 100 {
		ps = 20
	}
	start = (p - 1) * ps
	end = start + ps
	return p, ps, start, end
}

func applyAuditListFilters(items []map[string]interface{}, params dto.AuditListParams) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		if kw := strings.TrimSpace(params.Keyword); kw != "" {
			title, _ := item["title"].(string)
			if !strings.Contains(strings.ToLower(title), strings.ToLower(kw)) {
				continue
			}
		}
		if ap := strings.TrimSpace(params.Applicant); ap != "" {
			app, _ := item["applicant"].(string)
			if !strings.Contains(strings.ToLower(app), strings.ToLower(ap)) {
				continue
			}
		}
		if strings.TrimSpace(params.ProcessType) != "" {
			parts := strings.Split(params.ProcessType, ",")
			pt, _ := item["process_type"].(string)
			ok := false
			for _, x := range parts {
				x = strings.TrimSpace(x)
				if x != "" && strings.EqualFold(x, pt) {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}
		if dept := strings.TrimSpace(params.Department); dept != "" {
			d, _ := item["department"].(string)
			if d != dept {
				continue
			}
		}
		if st := strings.TrimSpace(params.AuditStatus); st != "" {
			res, _ := item["audit_result"].(map[string]interface{})
			if res == nil {
				continue
			}
			rec, _ := res["recommendation"].(string)
			if rec != st {
				continue
			}
		}
		out = append(out, item)
	}
	return out
}

// GetStats 获取审核工作台统计（无日期筛选，概览等场景使用）。
func (s *AuditExecuteService) GetStats(c *gin.Context) (map[string]int, error) {
	return s.GetStatsWithParams(c, dto.AuditListParams{})
}

// GetStatsWithParams 与列表共用提交/审核时间范围（start_date、end_date）。
// 将 keyword/applicant/department/mainTableNames 下推到 OA SQL，避免全量拉取。
func (s *AuditExecuteService) GetStatsWithParams(c *gin.Context, params dto.AuditListParams) (map[string]int, error) {
	tenantID, userID, err := s.extractIDs(c)
	if err != nil {
		return nil, err
	}

	// 构建缓存键：audit:stats:{tenant_id}:{user_id}:{date_range_hash}
	// 由于 AuditListParams 的日期字段标记为 json:"-"，需要构建包含日期的哈希输入
	if s.cache != nil && s.cache.IsEnabled() {
		dateRangeHash := cache.ComputeFilterHash(map[string]interface{}{
			"submit_date_start":         params.SubmitDateStart,
			"submit_date_end_exclusive": params.SubmitDateEndExclusive,
		})
		keyBuilder := cache.NewKeyBuilder("audit", tenantID)
		cacheKey := keyBuilder.Stats(userID, dateRangeHash)

		var cached cache.CachedStats
		if hit, _ := s.cache.Get(c.Request.Context(), cacheKey, &cached); hit {
			if statsMap, ok := cached.Stats.(map[string]interface{}); ok {
				result := make(map[string]int, len(statsMap))
				for k, v := range statsMap {
					switch val := v.(type) {
					case float64:
						result[k] = int(val)
					case int:
						result[k] = val
					}
				}
				return result, nil
			}
		}
	}

	_, _ = s.FailStaleAuditJobs(context.Background())

	username := s.extractUsername(c)
	if username == "" {
		return nil, newServiceError(errcode.ErrNoAuthToken, "用户信息缺失")
	}

	adapter, err := s.getOAAdapter(tenantID)
	if err != nil {
		return nil, err
	}

	// 使用筛选后的 OA 数据（keyword/applicant/department/mainTableNames/processTypes 已下推到 SQL）
	allowedTables := s.getAllowedMainTables(c)
	processTypes := s.getAllowedProcessTypes(c)
	todoItems, err := s.fetchTodoListFiltered(c, adapter, username, params, allowedTables, processTypes)
	if err != nil {
		return nil, err
	}

	processIDs := make([]string, len(todoItems))
	for i, item := range todoItems {
		processIDs[i] = item.ProcessID
	}
	snapshotMap, err := s.getSnapshotMapCached(c, tenantID, processIDs)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "查询审核快照失败")
	}

	pendingAI, aiDone := 0, 0
	for _, item := range todoItems {
		if snapshotMap[item.ProcessID] != nil {
			aiDone++
		} else {
			pendingAI++
		}
	}

	todoExcludeIDs, err := s.collectTodoProcessIDsForExclusion(c, username, adapter)
	if err != nil {
		return nil, newServiceError(errcode.ErrOAQueryFailed, "获取 OA 待办失败: "+err.Error())
	}

	var completedCount int64
	q := s.db.Model(&model.AuditProcessSnapshot{}).Where("tenant_id = ?", tenantID)
	if len(todoExcludeIDs) > 0 {
		q = q.Where("process_id NOT IN ?", todoExcludeIDs)
	}
	configuredTypes := s.getAllowedProcessTypes(c)
	if len(configuredTypes) > 0 {
		q = q.Where("process_type IN ?", configuredTypes)
	}
	if params.SubmitDateStart != nil {
		q = q.Where("updated_at >= ?", params.SubmitDateStart)
	}
	if params.SubmitDateEndExclusive != nil {
		q = q.Where("updated_at < ?", params.SubmitDateEndExclusive)
	}
	q.Count(&completedCount)

	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var todayCompleted int64
	s.db.Model(&model.AuditProcessSnapshot{}).
		Where("tenant_id = ? AND updated_at >= ?", tenantID, startOfDay).
		Count(&todayCompleted)

	result := map[string]int{
		"pending_ai_count":      pendingAI,
		"ai_done_count":         aiDone,
		"completed_count":       int(completedCount),
		"today_completed_count": int(todayCompleted),
	}

	// 写入缓存
	if s.cache != nil && s.cache.IsEnabled() {
		dateRangeHash := cache.ComputeFilterHash(map[string]interface{}{
			"submit_date_start":         params.SubmitDateStart,
			"submit_date_end_exclusive": params.SubmitDateEndExclusive,
		})
		keyBuilder := cache.NewKeyBuilder("audit", tenantID)
		cacheKey := keyBuilder.Stats(userID, dateRangeHash)
		toCache := cache.CachedStats{
			Stats:    result,
			CachedAt: time.Now(),
		}
		_ = s.cache.Set(c.Request.Context(), cacheKey, toCache, cache.DefaultTTLStats)
	}

	return result, nil
}

// ListProcessesPaged 分页查询审核工作台（待 AI / AI 已审核：OA 待办可按 createdate；全部已完成：排除「当前全量待办」后按快照 updated_at）。
func (s *AuditExecuteService) ListProcessesPaged(c *gin.Context, params dto.AuditListParams) (*dto.AuditProcessListResponse, error) {
	tenantID, userID, err := s.extractIDs(c)
	if err != nil {
		return nil, err
	}
	_, _ = s.FailStaleAuditJobs(context.Background())

	username := s.extractUsername(c)
	if username == "" {
		return nil, newServiceError(errcode.ErrNoAuthToken, "用户信息缺失")
	}

	tab := strings.TrimSpace(params.Tab)
	if tab == "" {
		tab = "pending_ai"
	}

	adapter, err := s.getOAAdapter(tenantID)
	if err != nil {
		return nil, err
	}

	if tab == "completed" {
		return s.listCompletedProcessesPaged(c, tenantID, username, adapter, params)
	}

	// 构建缓存键
	keyBuilder := cache.NewKeyBuilder("audit", tenantID)
	filterHash := cache.ComputeFilterHash(params)
	cacheKey := keyBuilder.TodoList(userID, filterHash)

	// 尝试从缓存获取
	if s.cache != nil && s.cache.IsEnabled() {
		var cached cache.CachedTodoList
		if hit, _ := s.cache.Get(c.Request.Context(), cacheKey, &cached); hit {
			return &dto.AuditProcessListResponse{
				Items:    cached.Items,
				Total:    cached.Total,
				Page:     params.Page,
				PageSize: params.PageSize,
			}, nil
		}
	}

	// pending_ai / ai_done：将 keyword/applicant/department/mainTableNames/processTypes 下推到 OA SQL，
	// 减少从 OA 拉取的数据量。tab 分组（是否有 snapshot）仍需在内存中完成。
	allowedTables := s.getAllowedMainTables(c)
	processTypes := s.getAllowedProcessTypes(c)

	// 获取筛选后的全量数据（不分页），用于 tab 分组
	todoResult, err := s.fetchTodoListFiltered(c, adapter, username, params, allowedTables, processTypes)
	if err != nil {
		return nil, err
	}

	processIDs := make([]string, len(todoResult))
	for i, item := range todoResult {
		processIDs[i] = item.ProcessID
	}
	auditMap, err := s.auditLogRepo.GetLatestResultMap(c, processIDs)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "查询审核记录失败")
	}
	snapshotMap, err := s.getSnapshotMapCached(c, tenantID, processIDs)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "查询审核快照失败")
	}

	var results []map[string]interface{}
	for _, item := range todoResult {
		record := map[string]interface{}{
			"process_id":         item.ProcessID,
			"title":              item.Title,
			"applicant":          item.Applicant,
			"department":         item.Department,
			"process_type":       item.ProcessType,
			"process_type_label": item.ProcessTypeLabel,
			"current_node":       item.CurrentNode,
			"submit_time":        item.SubmitTime,
			"urgency":            item.Urgency,
			"has_audit":          false,
			"audit_result":       nil,
			"in_todo":            true,
		}

		snap := snapshotMap[item.ProcessID]
		hasValid := snap != nil
		auditLog, hasLatest := auditMap[item.ProcessID]

		if hasValid {
			validLog, err := s.auditLogRepo.GetByID(c, snap.LatestValidLogID)
			if err == nil && validLog != nil {
				record["has_audit"] = true
				record["audit_result"] = buildAuditResultFromLog(validLog)
				record["audit_status"] = model.JobStatusCompleted
			}
		}
		if hasLatest {
			st := auditLog.Status
			switch st {
			case model.JobStatusPending, model.JobStatusAssembling, model.JobStatusReasoning, model.JobStatusExtracting:
				record["audit_status"] = st
				record["audit_result"] = buildAuditResultFromLog(auditLog)
			case model.JobStatusFailed:
				if !hasValid {
					record["audit_status"] = nil
					record["audit_result"] = nil
					record["has_audit"] = false
				}
			case model.JobStatusCompleted:
				if !hasValid {
					record["audit_status"] = nil
					record["audit_result"] = nil
					record["has_audit"] = false
				}
			}
		}

		switch tab {
		case "pending_ai":
			if !hasValid {
				results = append(results, record)
			}
		case "ai_done":
			if hasValid {
				results = append(results, record)
			}
		}
	}

	// audit_status 筛选（approve/return/review）仍在内存中完成
	filtered := applyAuditStatusFilter(results, params)
	total := len(filtered)
	page, ps, start, end := normalizeAuditPage(params.Page, params.PageSize)
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	items := filtered[start:end]

	response := &dto.AuditProcessListResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: ps,
	}

	// 写入缓存
	if s.cache != nil && s.cache.IsEnabled() {
		toCache := cache.CachedTodoList{
			Items:      items,
			Total:      total,
			CachedAt:   time.Now(),
			FilterHash: filterHash,
		}
		_ = s.cache.Set(c.Request.Context(), cacheKey, toCache, cache.DefaultTTLAuditTodo)
	}

	return response, nil
}

// fetchTodoListFiltered 使用 FetchTodoListPaged 将 keyword/applicant/department/mainTableNames 下推到 OA SQL，
// 但不做 OA 层分页（因为 tab 分组需要全量筛选后数据）。分批拉取全量筛选结果。
func (s *AuditExecuteService) fetchTodoListFiltered(c *gin.Context, adapter oa.OAAdapter, username string, params dto.AuditListParams, allowedTables map[string]bool, processTypes []string) ([]oa.TodoItem, error) {
	mainTableNames := make([]string, 0, len(allowedTables))
	for t := range allowedTables {
		mainTableNames = append(mainTableNames, t)
	}

	const batchSize = 500
	pagedFilter := oa.TodoListPagedFilter{
		TodoListFilter: todoListFilterFromAuditParams(params),
		Keyword:        params.Keyword,
		Applicant:      params.Applicant,
		Department:     params.Department,
		MainTableNames: mainTableNames,
		ProcessTypes:   processTypes,
		Page:           1,
		PageSize:       batchSize,
	}

	result, err := adapter.FetchTodoListPaged(c.Request.Context(), username, pagedFilter)
	if err != nil {
		return nil, newServiceError(errcode.ErrOAQueryFailed, "获取 OA 待办失败: "+err.Error())
	}

	items := result.Items
	// 分批拉取剩余数据
	for len(items) < result.Total {
		pagedFilter.Page++
		batch, err := adapter.FetchTodoListPaged(c.Request.Context(), username, pagedFilter)
		if err != nil || len(batch.Items) == 0 {
			break
		}
		items = append(items, batch.Items...)
	}

	return items, nil
}

// applyAuditStatusFilter 仅过滤 audit_status（approve/return/review），
// keyword/applicant/department/processType 已在 OA SQL 中过滤。
func applyAuditStatusFilter(items []map[string]interface{}, params dto.AuditListParams) []map[string]interface{} {
	st := strings.TrimSpace(params.AuditStatus)
	if st == "" {
		return items
	}
	out := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		res, _ := item["audit_result"].(map[string]interface{})
		if res == nil {
			continue
		}
		rec, _ := res["recommendation"].(string)
		if rec != st {
			continue
		}
		out = append(out, item)
	}
	return out
}

func (s *AuditExecuteService) listCompletedProcessesPaged(c *gin.Context, tenantID uuid.UUID, username string, adapter oa.OAAdapter, params dto.AuditListParams) (*dto.AuditProcessListResponse, error) {
	todoProcessIDs, err := s.collectTodoProcessIDsForExclusion(c, username, adapter)
	if err != nil {
		return nil, newServiceError(errcode.ErrOAQueryFailed, "获取 OA 待办失败: "+err.Error())
	}

	configuredTypes := s.getAllowedProcessTypes(c)

	// 先查总数（用于分页）
	countQ := s.db.Model(&model.AuditProcessSnapshot{}).Where("tenant_id = ?", tenantID)
	if len(todoProcessIDs) > 0 {
		countQ = countQ.Where("process_id NOT IN ?", todoProcessIDs)
	}
	if len(configuredTypes) > 0 {
		countQ = countQ.Where("process_type IN ?", configuredTypes)
	}
	if params.SubmitDateStart != nil {
		countQ = countQ.Where("updated_at >= ?", params.SubmitDateStart)
	}
	if params.SubmitDateEndExclusive != nil {
		countQ = countQ.Where("updated_at < ?", params.SubmitDateEndExclusive)
	}
	// keyword 筛选 title
	if kw := strings.TrimSpace(params.Keyword); kw != "" {
		countQ = countQ.Where("LOWER(title) LIKE ?", "%"+strings.ToLower(kw)+"%")
	}
	// processType 筛选
	if pt := strings.TrimSpace(params.ProcessType); pt != "" {
		parts := strings.Split(pt, ",")
		trimmed := make([]string, 0, len(parts))
		for _, p := range parts {
			if t := strings.TrimSpace(p); t != "" {
				trimmed = append(trimmed, t)
			}
		}
		if len(trimmed) > 0 {
			countQ = countQ.Where("process_type IN ?", trimmed)
		}
	}

	var total int64
	countQ.Count(&total)

	page, ps, _, _ := normalizeAuditPage(params.Page, params.PageSize)
	if total == 0 {
		return &dto.AuditProcessListResponse{
			Items: []map[string]interface{}{}, Total: 0, Page: page, PageSize: ps,
		}, nil
	}

	// 分页查询 snapshots（真分页，LIMIT/OFFSET 在 DB 层）
	offset := (page - 1) * ps
	var snaps []model.AuditProcessSnapshot
	dataQ := s.db.Where("tenant_id = ?", tenantID).Order("updated_at DESC")
	if len(todoProcessIDs) > 0 {
		dataQ = dataQ.Where("process_id NOT IN ?", todoProcessIDs)
	}
	if len(configuredTypes) > 0 {
		dataQ = dataQ.Where("process_type IN ?", configuredTypes)
	}
	if params.SubmitDateStart != nil {
		dataQ = dataQ.Where("updated_at >= ?", params.SubmitDateStart)
	}
	if params.SubmitDateEndExclusive != nil {
		dataQ = dataQ.Where("updated_at < ?", params.SubmitDateEndExclusive)
	}
	if kw := strings.TrimSpace(params.Keyword); kw != "" {
		dataQ = dataQ.Where("LOWER(title) LIKE ?", "%"+strings.ToLower(kw)+"%")
	}
	if pt := strings.TrimSpace(params.ProcessType); pt != "" {
		parts := strings.Split(pt, ",")
		trimmed := make([]string, 0, len(parts))
		for _, p := range parts {
			if t := strings.TrimSpace(p); t != "" {
				trimmed = append(trimmed, t)
			}
		}
		if len(trimmed) > 0 {
			dataQ = dataQ.Where("process_type IN ?", trimmed)
		}
	}
	if err := dataQ.Offset(offset).Limit(ps).Find(&snaps).Error; err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "查询已完成审核记录失败")
	}

	// 批量查询 valid log（替代逐条 GetByID）
	logIDs := make([]uuid.UUID, 0, len(snaps))
	seen := make(map[string]bool)
	uniqueSnaps := make([]model.AuditProcessSnapshot, 0, len(snaps))
	for _, snap := range snaps {
		if seen[snap.ProcessID] {
			continue
		}
		seen[snap.ProcessID] = true
		logIDs = append(logIDs, snap.LatestValidLogID)
		uniqueSnaps = append(uniqueSnaps, snap)
	}

	logMap, err := s.auditLogRepo.GetByIDs(c, logIDs)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "批量查询审核日志失败")
	}

	var results []map[string]interface{}
	for _, snap := range uniqueSnaps {
		validLog := logMap[snap.LatestValidLogID]
		if validLog == nil {
			continue
		}
		results = append(results, map[string]interface{}{
			"process_id":         snap.ProcessID,
			"title":              snap.Title,
			"applicant":          "",
			"department":         "",
			"process_type":       snap.ProcessType,
			"process_type_label": "",
			"current_node":       "已完成",
			"submit_time":        validLog.CreatedAt.Format("2006-01-02 15:04"),
			"urgency":            "low",
			"has_audit":          true,
			"audit_result":       buildAuditResultFromLog(validLog),
			"in_todo":            false,
		})
	}

	// audit_status 筛选（approve/return/review）
	filtered := applyAuditStatusFilter(results, params)

	return &dto.AuditProcessListResponse{
		Items:    filtered,
		Total:    int(total),
		Page:     page,
		PageSize: ps,
	}, nil
}

func buildAuditResultFromLog(log *model.AuditLog) map[string]interface{} {
	switch log.Status {
	case model.JobStatusPending, model.JobStatusAssembling, model.JobStatusReasoning, model.JobStatusExtracting:
		out := map[string]interface{}{
			"id":           log.ID.String(),
			"trace_id":     fmt.Sprintf("TR-%s", log.ID.String()[:8]),
			"process_id":   log.ProcessID,
			"status":       log.Status,
			"ai_reasoning": log.AIReasoning,
			"created_at":   log.CreatedAt.Format(time.RFC3339),
		}
		if log.ErrorMessage != "" {
			out["error_message"] = log.ErrorMessage
		}
		return out
	case model.JobStatusFailed:
		return map[string]interface{}{
			"id":             log.ID.String(),
			"trace_id":       fmt.Sprintf("TR-%s", log.ID.String()[:8]),
			"process_id":     log.ProcessID,
			"status":         log.Status,
			"error_message":  log.ErrorMessage,
			"ai_reasoning":   log.AIReasoning,
			"created_at":     log.CreatedAt.Format(time.RFC3339),
			"recommendation": "review",
			"overall_score":  0,
			"confidence":     0,
			"rule_results":   []interface{}{},
			"risk_points":    []string{},
			"suggestions":    []string{},
		}
	}

	result := map[string]interface{}{
		"id":             log.ID.String(),
		"trace_id":       fmt.Sprintf("TR-%s", log.ID.String()[:8]),
		"process_id":     log.ProcessID,
		"status":         log.Status,
		"recommendation": log.Recommendation,
		"overall_score":  log.Score,
		"confidence":     log.Confidence,
		"ai_reasoning":   log.AIReasoning,
		"duration_ms":    log.DurationMs,
		"created_at":     log.CreatedAt.Format(time.RFC3339),
	}
	if log.ErrorMessage != "" {
		result["error_message"] = log.ErrorMessage
	}

	if log.ParseError != "" {
		result["parse_error"] = log.ParseError
		result["raw_content"] = log.RawContent
		result["rule_results"] = []interface{}{}
		result["risk_points"] = []string{}
		result["suggestions"] = []string{}
	} else {
		var parsed model.AuditResultJSON
		if err := json.Unmarshal(log.AuditResult, &parsed); err == nil {
			result["rule_results"] = parsed.RuleResults
			result["risk_points"] = parsed.RiskPoints
			result["suggestions"] = parsed.Suggestions
		} else {
			result["rule_results"] = []interface{}{}
			result["risk_points"] = []string{}
			result["suggestions"] = []string{}
		}
	}
	return result
}

// getAllowedMainTables 获取当前租户所有启用的流程审核配置的主表名集合（小写），用于过滤 OA 待办。
func (s *AuditExecuteService) getAllowedMainTables(c *gin.Context) map[string]bool {
	configs, err := s.configRepo.ListByTenant(c)
	if err != nil {
		return map[string]bool{}
	}
	m := make(map[string]bool, len(configs))
	for _, cfg := range configs {
		if cfg.Status == "active" && cfg.MainTableName != "" {
			m[strings.ToLower(cfg.MainTableName)] = true
		}
	}
	return m
}

// getAllowedProcessTypes 获取当前租户所有启用的流程类型名称列表。
func (s *AuditExecuteService) getAllowedProcessTypes(c *gin.Context) []string {
	configs, err := s.configRepo.ListByTenant(c)
	if err != nil {
		return nil
	}
	var types []string
	for _, cfg := range configs {
		if cfg.Status == "active" {
			types = append(types, cfg.ProcessType)
		}
	}
	return types
}

func (s *AuditExecuteService) decryptOAConn(conn *model.OADatabaseConnection) error {
	password, err := crypto.Decrypt(conn.Password)
	if err != nil {
		return newServiceError(errcode.ErrInternalServer, "OA 数据库密码解密失败")
	}
	conn.Password = password
	return nil
}

func (s *AuditExecuteService) getOAAdapter(tenantID uuid.UUID) (oa.OAAdapter, error) {
	tenant, err := s.tenantRepo.FindByID(tenantID)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "获取租户失败")
	}
	if tenant.OADBConnectionID == nil {
		return nil, newServiceError(errcode.ErrOAConnectionFailed, "租户未配置 OA 数据库连接")
	}
	conn, err := s.oaConnRepo.FindByID(*tenant.OADBConnectionID)
	if err != nil {
		return nil, newServiceError(errcode.ErrOAConnectionFailed, "OA 数据库连接配置不存在")
	}
	if err := s.decryptOAConn(conn); err != nil {
		return nil, err
	}
	adapter, err := oa.NewOAAdapter(conn.OAType, conn)
	if err != nil {
		return nil, newServiceError(errcode.ErrOAConnectionFailed, "创建 OA 适配器失败: "+err.Error())
	}
	return adapter, nil
}

func (s *AuditExecuteService) fetchOAData(c *gin.Context, tenant *model.Tenant, processID string) (*oa.ProcessData, error) {
	if tenant.OADBConnectionID == nil {
		return nil, newServiceError(errcode.ErrOAConnectionFailed, "租户未配置 OA 数据库连接")
	}
	conn, err := s.oaConnRepo.FindByID(*tenant.OADBConnectionID)
	if err != nil {
		return nil, newServiceError(errcode.ErrOAConnectionFailed, "OA 数据库连接配置不存在")
	}
	if err := s.decryptOAConn(conn); err != nil {
		return nil, err
	}
	adapter, err := oa.NewOAAdapter(conn.OAType, conn)
	if err != nil {
		return nil, newServiceError(errcode.ErrOAConnectionFailed, "创建 OA 适配器失败: "+err.Error())
	}
	data, err := adapter.FetchProcessData(c.Request.Context(), processID)
	if err != nil {
		return nil, newServiceError(errcode.ErrOAQueryFailed, "拉取 OA 流程数据失败: "+err.Error())
	}
	return data, nil
}

func (s *AuditExecuteService) extractIDs(c *gin.Context) (uuid.UUID, uuid.UUID, error) {
	tidVal, exists := c.Get("tenant_id")
	if !exists {
		return uuid.Nil, uuid.Nil, newServiceError(errcode.ErrNoAuthToken, "租户ID缺失")
	}
	tenantID, err := uuid.Parse(fmt.Sprintf("%v", tidVal))
	if err != nil {
		return uuid.Nil, uuid.Nil, newServiceError(errcode.ErrNoAuthToken, "租户ID格式无效")
	}

	claimsVal, _ := c.Get("jwt_claims")
	claims, ok := claimsVal.(*jwtpkg.JWTClaims)
	if !ok {
		return uuid.Nil, uuid.Nil, newServiceError(errcode.ErrNoAuthToken, "用户认证信息缺失")
	}
	userID, err := uuid.Parse(claims.Sub)
	if err != nil {
		return uuid.Nil, uuid.Nil, newServiceError(errcode.ErrNoAuthToken, "用户ID格式无效")
	}

	return tenantID, userID, nil
}

func (s *AuditExecuteService) extractUsername(c *gin.Context) string {
	claimsVal, _ := c.Get("jwt_claims")
	if claims, ok := claimsVal.(*jwtpkg.JWTClaims); ok {
		return claims.Username
	}
	return ""
}

func isRuleEnabled(r *model.AuditRule) bool {
	return r.Enabled == nil || *r.Enabled
}

func formatRules(rules []model.AuditRule) string {
	if len(rules) == 0 {
		return "（无审核规则）"
	}
	var sb strings.Builder
	for i, r := range rules {
		if !isRuleEnabled(&r) {
			continue
		}
		sb.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, r.RuleScope, r.RuleContent))
	}
	if sb.Len() == 0 {
		return "（无启用的审核规则）"
	}
	return sb.String()
}

// resolveUserConfig 解析租户流程配置 + 用户个人配置，返回最终字段集和规则文本。
//
// 核心原则：以租户配置为权威来源，用户个人配置是"锦上添花"。
//   - 租户已删除的字段/规则 → 自动忽略用户中的对应残留（读时过滤）
//   - 租户关闭 AllowCustomFields → 用户 field_overrides 不生效
//   - 租户关闭 AllowCustomRules → 用户 custom_rules 不生效
//   - mandatory 规则 → 始终强制启用，无论租户 Enabled 或用户 override
func (s *AuditExecuteService) resolveUserConfig(
	c *gin.Context, userID uuid.UUID,
	config *model.ProcessAuditConfig,
	tenantRules []model.AuditRule,
	processType string,
) (SelectedFieldSet, string) {
	// 解析租户权限配置
	var perms model.UserPermissionsData
	if err := json.Unmarshal(config.UserPermissions, &perms); err != nil {
		perms = model.UserPermissionsData{
			AllowCustomFields: true, AllowCustomRules: true, AllowModifyStrictness: true,
		}
	}

	// 获取用户个人配置
	var userDetail *model.AuditDetailItem
	userCfg, _ := s.userConfigRepo.GetByUserID(c, userID)
	if userCfg != nil {
		var items []model.AuditDetailItem
		_ = json.Unmarshal(userCfg.AuditDetails, &items)
		for i := range items {
			if items[i].ProcessType == processType || items[i].ConfigID == config.ID {
				userDetail = &items[i]
				break
			}
		}
	}

	// ── 字段解析 ──
	fieldSet := s.resolveFieldSet(config, userDetail, perms)

	// ── 规则解析 ──
	rulesText := s.resolveRulesText(tenantRules, userDetail, perms)

	return fieldSet, rulesText
}

func (s *AuditExecuteService) resolveFieldSet(
	config *model.ProcessAuditConfig,
	userDetail *model.AuditDetailItem,
	perms model.UserPermissionsData,
) SelectedFieldSet {
	if config.FieldMode == "all" {
		return nil
	}

	type fieldItem struct {
		FieldKey string `json:"field_key"`
		Selected bool   `json:"selected"`
	}
	type detailTableItem struct {
		TableName string      `json:"table_name"`
		Fields    []fieldItem `json:"fields"`
	}

	var mainFields []fieldItem
	var detailTables []detailTableItem
	_ = json.Unmarshal(config.MainFields, &mainFields)
	_ = json.Unmarshal(config.DetailTables, &detailTables)

	// 构建租户字段索引：只有租户当前字段列表中存在的 key 才有资格被选中
	tenantFieldIndex := make(map[string]map[string]bool) // table -> fieldKey -> exists
	tenantFieldIndex["main"] = make(map[string]bool)
	for _, f := range mainFields {
		tenantFieldIndex["main"][f.FieldKey] = true
	}
	for _, dt := range detailTables {
		tenantFieldIndex[dt.TableName] = make(map[string]bool)
		for _, f := range dt.Fields {
			tenantFieldIndex[dt.TableName][f.FieldKey] = true
		}
	}

	// 用户额外字段：仅在权限允许且字段仍存在于租户列表时生效
	userAddedMap := make(map[string]map[string]bool)
	if userDetail != nil && perms.AllowCustomFields {
		for _, key := range userDetail.FieldConfig.FieldOverrides {
			parts := strings.SplitN(key, ":", 2)
			if len(parts) != 2 {
				continue
			}
			table, field := parts[0], parts[1]
			if tenantFieldIndex[table] == nil || !tenantFieldIndex[table][field] {
				continue
			}
			if userAddedMap[table] == nil {
				userAddedMap[table] = make(map[string]bool)
			}
			userAddedMap[table][field] = true
		}
	}

	fieldSet := make(SelectedFieldSet)

	mainSet := make(map[string]bool)
	for _, f := range mainFields {
		if f.Selected || (userAddedMap["main"] != nil && userAddedMap["main"][f.FieldKey]) {
			mainSet[strings.ToLower(f.FieldKey)] = true
		}
	}
	if len(mainSet) > 0 {
		fieldSet["main"] = mainSet
	}

	for _, dt := range detailTables {
		dtSet := make(map[string]bool)
		for _, f := range dt.Fields {
			if f.Selected || (userAddedMap[dt.TableName] != nil && userAddedMap[dt.TableName][f.FieldKey]) {
				dtSet[strings.ToLower(f.FieldKey)] = true
			}
		}
		if len(dtSet) > 0 {
			fieldSet[dt.TableName] = dtSet
		}
	}

	return fieldSet
}

func (s *AuditExecuteService) resolveRulesText(
	tenantRules []model.AuditRule,
	userDetail *model.AuditDetailItem,
	perms model.UserPermissionsData,
) string {
	// 构建用户规则开关覆盖 map（仅引用仍存在于租户规则中的 ID）
	tenantRuleIDs := make(map[string]bool, len(tenantRules))
	for _, r := range tenantRules {
		tenantRuleIDs[r.ID.String()] = true
	}

	toggleMap := make(map[string]bool)
	var customRules []model.CustomRule
	if userDetail != nil {
		for _, t := range userDetail.RuleConfig.RuleToggleOverrides {
			if !tenantRuleIDs[t.RuleID] {
				continue
			}
			toggleMap[t.RuleID] = t.Enabled
		}
		customRules = userDetail.RuleConfig.CustomRules
	}

	var sb strings.Builder
	idx := 1

	for _, r := range tenantRules {
		// mandatory 规则始终强制启用
		if r.RuleScope == "mandatory" {
			sb.WriteString(fmt.Sprintf("%d. [%s] %s\n", idx, r.RuleScope, r.RuleContent))
			idx++
			continue
		}

		// 非 mandatory：先取租户默认 Enabled，再看用户覆盖
		enabled := isRuleEnabled(&r)
		if override, ok := toggleMap[r.ID.String()]; ok {
			enabled = override
		}
		if !enabled {
			continue
		}
		sb.WriteString(fmt.Sprintf("%d. [%s] %s\n", idx, r.RuleScope, r.RuleContent))
		idx++
	}

	// 用户自定义规则：仅在权限允许时追加
	if perms.AllowCustomRules {
		for _, cr := range customRules {
			if !cr.Enabled {
				continue
			}
			sb.WriteString(fmt.Sprintf("%d. [用户自定义] %s\n", idx, cr.Content))
			idx++
		}
	}

	if sb.Len() == 0 {
		return "（无启用的审核规则）"
	}
	return sb.String()
}

func parseSnapshotValidLogIDs(raw datatypes.JSON) []uuid.UUID {
	var s []string
	_ = json.Unmarshal(raw, &s)
	out := make([]uuid.UUID, 0, len(s))
	for _, x := range s {
		id, err := uuid.Parse(strings.TrimSpace(x))
		if err == nil {
			out = append(out, id)
		}
	}
	return out
}

// cachedAuditConfig 缓存的审核配置（config + rules 一起缓存）
type cachedAuditConfig struct {
	Config model.ProcessAuditConfig `json:"config"`
	Rules  []model.AuditRule        `json:"rules"`
}

// getProcessConfigCached 获取流程审核配置和规则，优先从缓存读取。
// 缓存键格式: audit:config:{tenant_id}:{process_type}，TTL 10 分钟。
func (s *AuditExecuteService) getProcessConfigCached(c *gin.Context, tenantID uuid.UUID, processType string) (*model.ProcessAuditConfig, []model.AuditRule, error) {
	ctx := c.Request.Context()

	// 尝试从缓存获取
	if s.cache != nil && s.cache.IsEnabled() {
		keyBuilder := cache.NewKeyBuilder("audit", tenantID)
		cacheKey := keyBuilder.ProcessConfig(processType)

		var cached cachedAuditConfig
		if hit, _ := s.cache.Get(ctx, cacheKey, &cached); hit {
			return &cached.Config, cached.Rules, nil
		}
	}

	// 缓存未命中，从数据库查询
	config, err := s.configRepo.GetByProcessType(c, processType)
	if err != nil {
		return nil, nil, newServiceError(errcode.ErrNoProcessConfig, fmt.Sprintf("流程 '%s' 的审核配置不存在", processType))
	}

	rules, err := s.ruleRepo.ListByConfigID(c, config.ID)
	if err != nil {
		return nil, nil, newServiceError(errcode.ErrDatabase, "获取审核规则失败")
	}

	// 写入缓存
	if s.cache != nil && s.cache.IsEnabled() {
		keyBuilder := cache.NewKeyBuilder("audit", tenantID)
		cacheKey := keyBuilder.ProcessConfig(processType)
		toCache := cachedAuditConfig{
			Config: *config,
			Rules:  rules,
		}
		_ = s.cache.Set(ctx, cacheKey, toCache, cache.DefaultTTLProcessConfig)
	}

	return config, rules, nil
}

// getSnapshotMapCached 获取审核快照映射，优先从缓存读取。
// 缓存键格式: audit:snapshot:{tenant_id}:{process_ids_hash}，TTL 5 分钟。
// 支持批量查询优化：将 processIDs 列表哈希为缓存键，避免逐条查询。
func (s *AuditExecuteService) getSnapshotMapCached(c *gin.Context, tenantID uuid.UUID, processIDs []string) (map[string]*model.AuditProcessSnapshot, error) {
	if len(processIDs) == 0 {
		return map[string]*model.AuditProcessSnapshot{}, nil
	}

	ctx := c.Request.Context()

	// 尝试从缓存获取
	if s.cache != nil && s.cache.IsEnabled() {
		processIDsHash := cache.ComputeFilterHash(processIDs)
		keyBuilder := cache.NewKeyBuilder("audit", tenantID)
		cacheKey := keyBuilder.Snapshot(processIDsHash)

		var cached cache.CachedSnapshot
		if hit, _ := s.cache.Get(ctx, cacheKey, &cached); hit {
			// 将 map[string]interface{} 转换回 map[string]*model.AuditProcessSnapshot
			result := make(map[string]*model.AuditProcessSnapshot, len(cached.Snapshots))
			for pid, raw := range cached.Snapshots {
				data, err := json.Marshal(raw)
				if err != nil {
					continue
				}
				var snap model.AuditProcessSnapshot
				if err := json.Unmarshal(data, &snap); err != nil {
					continue
				}
				result[pid] = &snap
			}
			return result, nil
		}
	}

	// 缓存未命中，从数据库查询
	snapshotMap, err := s.auditSnapshotRepo.GetMapByProcessIDs(c, processIDs)
	if err != nil {
		return nil, err
	}

	// 写入缓存
	if s.cache != nil && s.cache.IsEnabled() {
		processIDsHash := cache.ComputeFilterHash(processIDs)
		keyBuilder := cache.NewKeyBuilder("audit", tenantID)
		cacheKey := keyBuilder.Snapshot(processIDsHash)

		snapshots := make(map[string]interface{}, len(snapshotMap))
		for pid, snap := range snapshotMap {
			snapshots[pid] = snap
		}
		toCache := cache.CachedSnapshot{
			Snapshots: snapshots,
			CachedAt:  time.Now(),
		}
		_ = s.cache.Set(ctx, cacheKey, toCache, cache.DefaultTTLSnapshot)
	}

	return snapshotMap, nil
}
