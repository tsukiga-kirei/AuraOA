package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/crypto"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	jwtpkg "oa-smart-audit/go-service/internal/pkg/jwt"
	"oa-smart-audit/go-service/internal/pkg/oa"
	"oa-smart-audit/go-service/internal/repository"
)

const batchAuditMaxLimit = 10

// AuditExecuteService 审核执行业务逻辑：串联 OA 数据 → 提示词构建 → AI 调用 → 结果解析 → 写入日志。
type AuditExecuteService struct {
	auditLogRepo   *repository.AuditLogRepo
	configRepo     *repository.ProcessAuditConfigRepo
	ruleRepo       *repository.AuditRuleRepo
	userConfigRepo *repository.UserPersonalConfigRepo
	tenantRepo     *repository.TenantRepo
	oaConnRepo     *repository.OAConnectionRepo
	aiModelRepo    *repository.AIModelRepo
	aiCaller       *AIModelCallerService
	db             *gorm.DB
	rdb            *redis.Client
}

func NewAuditExecuteService(
	auditLogRepo *repository.AuditLogRepo,
	configRepo *repository.ProcessAuditConfigRepo,
	ruleRepo *repository.AuditRuleRepo,
	userConfigRepo *repository.UserPersonalConfigRepo,
	tenantRepo *repository.TenantRepo,
	oaConnRepo *repository.OAConnectionRepo,
	aiModelRepo *repository.AIModelRepo,
	aiCaller *AIModelCallerService,
	db *gorm.DB,
	rdb *redis.Client,
) *AuditExecuteService {
	return &AuditExecuteService{
		auditLogRepo:   auditLogRepo,
		configRepo:     configRepo,
		ruleRepo:       ruleRepo,
		userConfigRepo: userConfigRepo,
		tenantRepo:     tenantRepo,
		oaConnRepo:     oaConnRepo,
		aiModelRepo:    aiModelRepo,
		aiCaller:       aiCaller,
		db:             db,
		rdb:            rdb,
	}
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
		Status:         model.AuditStatusPending,
		Recommendation: "review",
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
			"status":        model.AuditStatusFailed,
			"error_message": "任务入队失败: " + err.Error(),
			"updated_at":    time.Now(),
		})
		return nil, newServiceError(errcode.ErrRedisConn, "审核任务入队失败: "+err.Error())
	}

	return &AuditExecuteResponse{
		Status:    model.AuditStatusPending,
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
		"status":        model.AuditStatusFailed,
		"error_message": msg,
		"updated_at":    time.Now(),
	})
}

// processAuditJob 由 Redis Stream Worker 调用，执行完整审核链路。
func (s *AuditExecuteService) processAuditJob(ctx context.Context, auditLogID, tenantID, userID uuid.UUID) error {
	c := s.workerGinContext(ctx, tenantID, userID)
	log, err := s.auditLogRepo.GetByID(c, auditLogID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if log.Status != model.AuditStatusPending {
		return nil
	}

	startTime := time.Now()
	tenant, err := s.tenantRepo.FindByID(tenantID)
	if err != nil {
		s.markAuditFailed(c, auditLogID, newServiceError(errcode.ErrDatabase, "获取租户信息失败"))
		return err
	}
	if tenant.PrimaryModelID == nil {
		se := newServiceError(errcode.ErrNoAIModelConfig, "租户未配置主用 AI 模型")
		s.markAuditFailed(c, auditLogID, se)
		return se
	}
	modelCfg, err := s.aiModelRepo.FindByID(*tenant.PrimaryModelID)
	if err != nil {
		se := newServiceError(errcode.ErrNoAIModelConfig, "AI 模型配置不存在")
		s.markAuditFailed(c, auditLogID, se)
		return se
	}

	req := &AuditExecuteRequest{ProcessID: log.ProcessID, ProcessType: log.ProcessType, Title: log.Title}

	config, err := s.configRepo.GetByProcessType(c, req.ProcessType)
	if err != nil {
		se := newServiceError(errcode.ErrNoProcessConfig, fmt.Sprintf("流程 '%s' 的审核配置不存在", req.ProcessType))
		s.markAuditFailed(c, auditLogID, se)
		return se
	}

	var aiConfig model.AIConfigData
	if err := json.Unmarshal(config.AIConfig, &aiConfig); err != nil {
		se := newServiceError(errcode.ErrInternalServer, "AI 配置解析失败")
		s.markAuditFailed(c, auditLogID, se)
		return se
	}

	rules, err := s.ruleRepo.ListByConfigID(c, config.ID)
	if err != nil {
		se := newServiceError(errcode.ErrDatabase, "获取审核规则失败")
		s.markAuditFailed(c, auditLogID, se)
		return se
	}

	fieldSet, mergedRulesText := s.resolveUserConfig(c, userID, config, rules, req.ProcessType)

	_ = s.auditLogRepo.UpdateFields(c, auditLogID, map[string]interface{}{
		"status":     model.AuditStatusReasoning,
		"updated_at": time.Now(),
	})

	processData, err := s.fetchOAData(c, tenant, req.ProcessID)
	if err != nil {
		s.markAuditFailed(c, auditLogID, err)
		return err
	}

	currentNode := "当前节点"

	reasoningReq := BuildReasoningPrompt(&aiConfig, req.ProcessType, processData, mergedRulesText, currentNode, fieldSet)
	reasoningReq.Temperature = float64(tenant.Temperature)
	reasoningReq.MaxTokens = tenant.MaxTokensPerRequest
	reasoningReq.ModelConfig = modelCfg

	reasoningResp, err := s.aiCaller.Chat(c, tenantID, userID, modelCfg, reasoningReq)
	if err != nil {
		s.markAuditFailed(c, auditLogID, err)
		return err
	}
	aiReasoning := reasoningResp.Content

	_ = s.auditLogRepo.UpdateFields(c, auditLogID, map[string]interface{}{
		"status":       model.AuditStatusExtracting,
		"ai_reasoning": aiReasoning,
		"updated_at":   time.Now(),
	})

	extractionReq := BuildExtractionPrompt(&aiConfig, aiReasoning, mergedRulesText)
	extractionReq.Temperature = 0.1
	extractionReq.MaxTokens = tenant.MaxTokensPerRequest
	extractionReq.ModelConfig = modelCfg

	extractionResp, err := s.aiCaller.Chat(c, tenantID, userID, modelCfg, extractionReq)
	if err != nil {
		s.markAuditFailed(c, auditLogID, err)
		return err
	}

	totalDuration := int(time.Since(startTime).Milliseconds())
	parsed, parseErr := ParseAuditResult(extractionResp.Content)

	updates := map[string]interface{}{
		"status":       model.AuditStatusCompleted,
		"duration_ms":  totalDuration,
		"raw_content":  extractionResp.Content,
		"ai_reasoning": aiReasoning,
		"updated_at":   time.Now(),
	}

	if parseErr != nil {
		updates["recommendation"] = "review"
		updates["score"] = 0
		updates["confidence"] = 0
		updates["parse_error"] = parseErr.Error()
		updates["audit_result"] = datatypes.JSON([]byte("{}"))
	} else {
		resultJSON, _ := json.Marshal(parsed)
		updates["recommendation"] = parsed.Recommendation
		updates["score"] = parsed.OverallScore
		updates["confidence"] = parsed.Confidence
		updates["audit_result"] = datatypes.JSON(resultJSON)
	}

	return s.auditLogRepo.UpdateFields(c, auditLogID, updates)
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
	if log.Status == model.AuditStatusFailed {
		resp.Recommendation = "review"
		resp.ParseError = log.ErrorMessage
		resp.RuleResults = []model.RuleResultJSON{}
		resp.RiskPoints = []string{}
		resp.Suggestions = []string{}
		return resp
	}
	if log.ParseError != "" {
		resp.Recommendation = "review"
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

// GetAuditChain 获取审核链：仅展示已完成的 AI 审核记录（租户内所有用户）。
func (s *AuditExecuteService) GetAuditChain(c *gin.Context, processID string) ([]model.AuditLog, error) {
	return s.auditLogRepo.ListCompletedByProcessID(c, processID)
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
		{model.AuditStatusPending, "排队中"},
		{model.AuditStatusReasoning, "推理分析"},
		{model.AuditStatusExtracting, "结构化提取"},
	}
	phaseIdx := map[string]int{
		model.AuditStatusPending:    0,
		model.AuditStatusReasoning:  1,
		model.AuditStatusExtracting: 2,
	}
	cur, ok := phaseIdx[status]
	if !ok {
		if status == model.AuditStatusCompleted {
			cur = 3
		} else if status == model.AuditStatusFailed {
			cur = 2
		} else {
			cur = 0
		}
	}
	var steps []map[string]interface{}
	for i, d := range defs {
		m := map[string]interface{}{"key": d.key, "label": d.label}
		switch {
		case status == model.AuditStatusFailed && i == cur:
			m["failed"] = true
		case i < cur:
			m["done"] = true
		case i == cur && cur < 3 && status != model.AuditStatusFailed:
			m["current"] = true
		}
		steps = append(steps, m)
	}
	if status == model.AuditStatusCompleted {
		steps = append(steps, map[string]interface{}{"key": "done", "label": "已完成", "done": true})
	}
	return steps
}

// GetStats 获取审核工作台统计（结合 OA 待办 + 租户配置 + 审核记录）。
func (s *AuditExecuteService) GetStats(c *gin.Context) (map[string]int, error) {
	tenantID, _, err := s.extractIDs(c)
	if err != nil {
		return nil, err
	}
	username := s.extractUsername(c)
	if username == "" {
		return nil, newServiceError(errcode.ErrNoAuthToken, "用户信息缺失")
	}

	adapter, err := s.getOAAdapter(tenantID)
	if err != nil {
		return nil, err
	}

	todoItems, err := adapter.FetchTodoList(c.Request.Context(), username)
	if err != nil {
		return nil, newServiceError(errcode.ErrOAQueryFailed, "获取 OA 待办失败: "+err.Error())
	}

	// 按租户配置的主表名过滤
	allowedTables := s.getAllowedMainTables(c)
	var filtered []oa.TodoItem
	for _, item := range todoItems {
		if allowedTables[strings.ToLower(item.MainTableName)] {
			filtered = append(filtered, item)
		}
	}

	processIDs := make([]string, len(filtered))
	for i, item := range filtered {
		processIDs[i] = item.ProcessID
	}
	auditMap, _ := s.auditLogRepo.GetLatestResultMap(c, processIDs)

	pendingAI, aiDone := 0, 0
	for _, item := range filtered {
		latest := auditMap[item.ProcessID]
		hasCompleted := latest != nil && latest.Status == model.AuditStatusCompleted
		if hasCompleted {
			aiDone++
		} else {
			pendingAI++
		}
	}

	// completed：有审核记录但不在当前待办中的流程数
	var completedCount int64
	q := s.db.Model(&model.AuditLog{}).Where("tenant_id = ? AND status = ?", tenantID, model.AuditStatusCompleted)
	if len(processIDs) > 0 {
		q = q.Where("process_id NOT IN ?", processIDs)
	}
	configuredTypes := s.getAllowedProcessTypes(c)
	if len(configuredTypes) > 0 {
		q = q.Where("process_type IN ?", configuredTypes)
	}
	q.Select("COUNT(DISTINCT process_id)").Scan(&completedCount)

	// 今日审核成功条数（completed 且当日 updated_at）
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var todayCompleted int64
	s.db.Model(&model.AuditLog{}).
		Where("tenant_id = ? AND status = ? AND updated_at >= ?", tenantID, model.AuditStatusCompleted, startOfDay).
		Count(&todayCompleted)

	return map[string]int{
		"pending_ai_count":      pendingAI,
		"ai_done_count":         aiDone,
		"completed_count":       int(completedCount),
		"today_completed_count": int(todayCompleted),
	}, nil
}

// ListProcesses 获取审核工作台流程列表（结合 OA 待办 + 租户配置 + AI 审核状态）。
func (s *AuditExecuteService) ListProcesses(c *gin.Context, tab string, username string) ([]map[string]interface{}, error) {
	tenantID, _, err := s.extractIDs(c)
	if err != nil {
		return nil, err
	}

	// 获取 OA 适配器
	adapter, err := s.getOAAdapter(tenantID)
	if err != nil {
		return nil, err
	}

	// 从 OA 拉取用户待办
	todoItems, err := adapter.FetchTodoList(c.Request.Context(), username)
	if err != nil {
		return nil, newServiceError(errcode.ErrOAQueryFailed, "获取 OA 待办失败: "+err.Error())
	}

	// 按租户配置的主表名过滤：只保留租户已配置审核的流程
	allowedTables := s.getAllowedMainTables(c)
	var filteredTodo []oa.TodoItem
	for _, item := range todoItems {
		if allowedTables[strings.ToLower(item.MainTableName)] {
			filteredTodo = append(filteredTodo, item)
		}
	}

	// 获取已有审核记录
	processIDs := make([]string, len(filteredTodo))
	for i, item := range filteredTodo {
		processIDs[i] = item.ProcessID
	}
	auditMap, err := s.auditLogRepo.GetLatestResultMap(c, processIDs)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "查询审核记录失败")
	}

	var results []map[string]interface{}
	for _, item := range filteredTodo {
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

		auditLog, hasLatest := auditMap[item.ProcessID]
		hasCompleted := hasLatest && auditLog.Status == model.AuditStatusCompleted
		if hasLatest {
			record["audit_status"] = auditLog.Status
			record["audit_result"] = buildAuditResultFromLog(auditLog)
		}
		if hasCompleted {
			record["has_audit"] = true
		}

		switch tab {
		case "pending_ai":
			if !hasCompleted {
				results = append(results, record)
			}
		case "ai_done":
			if hasCompleted {
				results = append(results, record)
			}
		case "completed":
			// completed 在下面单独查询
		}
	}

	// 对 completed tab：查询已有审核记录但不在当前待办的流程
	if tab == "completed" {
		results, err = s.listCompletedProcesses(c, tenantID, username, adapter, processIDs)
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

func (s *AuditExecuteService) listCompletedProcesses(c *gin.Context, tenantID uuid.UUID, username string, adapter oa.OAAdapter, todoProcessIDs []string) ([]map[string]interface{}, error) {
	todoSet := make(map[string]bool)
	for _, id := range todoProcessIDs {
		todoSet[id] = true
	}

	// 查询该租户所有审核日志中不在当前待办里的流程，且属于租户配置的流程类型
	configuredTypes := s.getAllowedProcessTypes(c)
	var logs []model.AuditLog
	query := s.db.Where("tenant_id = ? AND status = ?", tenantID, model.AuditStatusCompleted).Order("created_at DESC")
	if len(todoProcessIDs) > 0 {
		query = query.Where("process_id NOT IN ?", todoProcessIDs)
	}
	if len(configuredTypes) > 0 {
		query = query.Where("process_type IN ?", configuredTypes)
	}
	if err := query.Find(&logs).Error; err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "查询已完成审核记录失败")
	}

	seen := make(map[string]bool)
	var results []map[string]interface{}
	for _, log := range logs {
		if seen[log.ProcessID] {
			continue
		}
		seen[log.ProcessID] = true
		results = append(results, map[string]interface{}{
			"process_id":         log.ProcessID,
			"title":              log.Title,
			"applicant":          "",
			"department":         "",
			"process_type":       log.ProcessType,
			"process_type_label": "",
			"current_node":       "已完成",
			"submit_time":        log.CreatedAt.Format("2006-01-02 15:04"),
			"urgency":            "low",
			"has_audit":          true,
			"audit_result":       buildAuditResultFromLog(&log),
			"in_todo":            false,
		})
	}
	return results, nil
}

func buildAuditResultFromLog(log *model.AuditLog) map[string]interface{} {
	switch log.Status {
	case model.AuditStatusPending, model.AuditStatusReasoning, model.AuditStatusExtracting:
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
	case model.AuditStatusFailed:
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
