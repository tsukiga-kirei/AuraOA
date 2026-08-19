package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/apptime"
	"auraoa/go-service/internal/pkg/crypto"
	"auraoa/go-service/internal/pkg/errcode"
	jwtpkg "auraoa/go-service/internal/pkg/jwt"
	pkglogger "auraoa/go-service/internal/pkg/logger"
	"auraoa/go-service/internal/pkg/oa"
	"auraoa/go-service/internal/pkg/sanitize"
	"auraoa/go-service/internal/pkg/systemflags"
	"auraoa/go-service/internal/repository"
)

const (
	summaryJobMaxAge        = 30 * time.Minute
	summaryProcessTimeout   = 25 * time.Minute
	summaryErrStaleMessage  = "总结任务超时（请重新发起）"
	summaryStreamKeyPrefix  = "summary:raw:"
	summaryPubSubKeyPrefix  = "summary:stream:"
	summaryDefaultBlockName = "流程摘要"
)

// ProcessSummaryService 执行 OA 嵌入流程总结。
type ProcessSummaryService struct {
	logRepo       *repository.ProcessSummaryLogRepo
	snapshotRepo  *repository.ProcessSummarySnapshotRepo
	configRepo    *repository.ProcessSummaryConfigRepo
	tenantRepo    *repository.TenantRepo
	oaConnRepo    *repository.OAConnectionRepo
	aiModelRepo   *repository.AIModelRepo
	aiCaller      *AIModelCallerService
	attachmentSvc *AttachmentRecognitionService
	db            *gorm.DB
	rdb           *redis.Client
	sysFlags      *systemflags.Resolver
	externalCtx   *ExternalContextService
	oaConnections *oa.ConnectionManager
}

func NewProcessSummaryService(
	logRepo *repository.ProcessSummaryLogRepo,
	snapshotRepo *repository.ProcessSummarySnapshotRepo,
	configRepo *repository.ProcessSummaryConfigRepo,
	tenantRepo *repository.TenantRepo,
	oaConnRepo *repository.OAConnectionRepo,
	aiModelRepo *repository.AIModelRepo,
	aiCaller *AIModelCallerService,
	attachmentSvc *AttachmentRecognitionService,
	db *gorm.DB,
	rdb *redis.Client,
	sysFlags *systemflags.Resolver,
	externalCtx *ExternalContextService,
	oaConnections *oa.ConnectionManager,
) *ProcessSummaryService {
	return &ProcessSummaryService{
		logRepo:       logRepo,
		snapshotRepo:  snapshotRepo,
		configRepo:    configRepo,
		tenantRepo:    tenantRepo,
		oaConnRepo:    oaConnRepo,
		aiModelRepo:   aiModelRepo,
		aiCaller:      aiCaller,
		attachmentSvc: attachmentSvc,
		db:            db,
		rdb:           rdb,
		sysFlags:      sysFlags,
		externalCtx:   externalCtx,
		oaConnections: oaConnections,
	}
}

type SummaryExecuteRequest struct {
	ProcessID        string     `json:"process_id" binding:"required"`
	ProcessType      string     `json:"process_type"`
	Title            string     `json:"title"`
	TriggerSource    string     `json:"trigger_source"`
	TriggerDetail    string     `json:"trigger_detail"`
	ScheduleConfigID *uuid.UUID `json:"-"`
}

type SummaryExecuteResponse struct {
	Status       string                            `json:"status,omitempty"`
	ID           string                            `json:"id"`
	TraceID      string                            `json:"trace_id"`
	ProcessID    string                            `json:"process_id"`
	Blocks       []model.ProcessSummaryBlockResult `json:"blocks,omitempty"`
	DurationMs   int                               `json:"duration_ms,omitempty"`
	CreatedAt    string                            `json:"created_at"`
	ParseError   string                            `json:"parse_error,omitempty"`
	RawContent   string                            `json:"raw_content,omitempty"`
	ErrorMessage string                            `json:"error_message,omitempty"`
}

type SummaryEmbedContextResponse struct {
	Supported          bool                      `json:"supported"`
	Reason             string                    `json:"reason,omitempty"`
	Message            string                    `json:"message,omitempty"`
	Process            *oa.ProcessRequestSummary `json:"process,omitempty"`
	EmbedEnabled       bool                      `json:"embed_enabled"`
	HasSummary         bool                      `json:"has_summary"`
	Stale              bool                      `json:"stale"`
	StaleBlockIDs      []string                  `json:"stale_block_ids,omitempty"`
	ShouldAutoSummary  bool                      `json:"should_auto_summary"`
	LastSummaryAt      string                    `json:"last_summary_at,omitempty"`
	RunningJobID       string                    `json:"running_job_id,omitempty"`
	SummaryResult      map[string]interface{}    `json:"summary_result,omitempty"`
	AutoRetryBlocked   bool                      `json:"auto_retry_blocked"`
	CurrentFingerprint string                    `json:"-"`
}

type summaryStreamChunk struct {
	BlockID string `json:"block_id"`
	Title   string `json:"title"`
	Chunk   string `json:"chunk"`
}

func (s *ProcessSummaryService) GetEmbedContext(c *gin.Context, processID string) (*SummaryEmbedContextResponse, error) {
	if strings.TrimSpace(processID) == "" {
		return nil, newServiceError(errcode.ErrParamValidation, "process_id 不能为空")
	}
	tenantID, _, err := s.extractIDs(c)
	if err != nil {
		return nil, err
	}
	adapter, err := s.getOAAdapter(c.Request.Context(), tenantID, false)
	if err != nil {
		return nil, err
	}
	summary, err := adapter.FetchProcessRequestSummary(c.Request.Context(), processID)
	if err != nil {
		return &SummaryEmbedContextResponse{
			Supported: false,
			Reason:    "not_found_in_oa",
			Message:   "未在 OA 中找到该流程，请确认 requestid 是否正确",
		}, nil
	}
	config, err := s.configRepo.GetByProcessType(c, summary.ProcessType)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &SummaryEmbedContextResponse{
				Supported: false,
				Reason:    "no_config",
				Message:   fmt.Sprintf("流程「%s」尚未配置 AI 总结", summary.ProcessType),
				Process:   summary,
			}, nil
		}
		return nil, newServiceError(errcode.ErrDatabase, "查询总结配置失败")
	}
	if config.Status != "active" {
		return &SummaryEmbedContextResponse{
			Supported: false,
			Reason:    "config_inactive",
			Message:   fmt.Sprintf("流程「%s」的 AI 总结配置已停用", summary.ProcessType),
			Process:   summary,
		}, nil
	}
	if !config.EmbedEnabled {
		return &SummaryEmbedContextResponse{
			Supported:    false,
			Reason:       "embed_disabled",
			Message:      fmt.Sprintf("流程「%s」未启用 OA 嵌入总结", summary.ProcessType),
			Process:      summary,
			EmbedEnabled: false,
		}, nil
	}

	resp := &SummaryEmbedContextResponse{
		Supported:    true,
		Process:      summary,
		EmbedEnabled: true,
	}
	if running, _ := s.logRepo.GetRunningByProcessID(c, processID); running != nil {
		resp.RunningJobID = running.ID.String()
		if snap, snapErr := s.snapshotRepo.GetByProcessID(c, processID); snapErr == nil && snap != nil {
			if latest, latestErr := s.logRepo.GetByID(c, snap.LatestValidLogID); latestErr == nil {
				resp.HasSummary = true
				resp.LastSummaryAt = apptime.FormatRFC3339(latest.UpdatedAt)
				resp.SummaryResult = s.buildSummaryResultFromLog(latest)
			}
		}
		if !resp.HasSummary {
			resp.SummaryResult = s.buildSummaryResultFromLog(running)
		}
		return resp, nil
	}

	snap, err := s.snapshotRepo.GetByProcessID(c, processID)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "查询总结快照失败")
	}

	var storedAnchor oa.OAContextAnchor
	var latestLog *model.ProcessSummaryLog
	if snap != nil {
		latestLog, err = s.logRepo.GetByID(c, snap.LatestValidLogID)
		if err == nil && latestLog != nil {
			storedAnchor = parseOAContextAnchor(latestLog.OAContextAnchor)
			resp.HasSummary = true
			resp.LastSummaryAt = apptime.FormatRFC3339(latestLog.UpdatedAt)
			resp.SummaryResult = s.buildSummaryResultFromLog(latestLog)
		}
	}

	latestAttempt, latestErr := s.logRepo.GetLatestByProcessID(c, processID)
	if latestErr != nil {
		return nil, newServiceError(errcode.ErrDatabase, "查询最近总结尝试失败")
	}
	if resp.HasSummary && strings.EqualFold(c.Query("prefer_cached"), "true") {
		if latestAttempt != nil &&
			latestAttempt.Status == model.JobStatusFailed &&
			(latestLog == nil || latestAttempt.CreatedAt.After(latestLog.CreatedAt)) {
			resp.AutoRetryBlocked = latestAttempt.AttemptFingerprint != ""
			resp.SummaryResult = s.buildSummaryResultFromLog(latestAttempt)
		}
		return resp, nil
	}

	embedCfg := parseSummaryEmbedConfig(config.EmbedConfig)
	blocks := parseSummaryBlocks(config.SummaryBlocks)
	if len(blocks) == 0 {
		blocks = defaultSummaryBlocks()
	}
	// 可见页变化检查只解析总结块实际使用的浏览字段，不识别附件正文，也不调用 AI。
	currentData, currentAnchor, err := s.fetchCurrentOAState(
		c,
		tenantID,
		processID,
		buildSummaryUnionFieldSet(blocks),
	)
	if err != nil {
		return nil, err
	}
	currentDependencies := buildSummaryBlockDependencyFingerprints(blocks, currentData, currentAnchor, summary)
	resp.CurrentFingerprint = stableJSONFingerprint(currentDependencies)
	storedDependencies := map[string]SummaryBlockDependencyFingerprint{}
	if latestLog != nil {
		storedDependencies = parseSummaryBlockDependencies(latestLog.ProcessSnapshot)
	}
	if resp.HasSummary {
		changes := oa.CompareContextAnchors(storedAnchor, currentAnchor)
		resp.StaleBlockIDs = changedSummaryBlockIDs(blocks, storedDependencies, currentDependencies, changes, embedCfg)
		resp.Stale = len(resp.StaleBlockIDs) > 0
		resp.ShouldAutoSummary = resp.Stale
	} else {
		resp.ShouldAutoSummary = embedCfg.AutoSummaryOnOpen
	}

	if latestAttempt != nil &&
		latestAttempt.Status == model.JobStatusFailed &&
		latestAttempt.AttemptFingerprint != "" &&
		latestAttempt.AttemptFingerprint == resp.CurrentFingerprint &&
		(latestLog == nil || latestAttempt.CreatedAt.After(latestLog.CreatedAt)) {
		resp.AutoRetryBlocked = true
		resp.ShouldAutoSummary = false
		resp.SummaryResult = s.buildSummaryResultFromLog(latestAttempt)
	}
	return resp, nil
}

func (s *ProcessSummaryService) ExecuteEmbed(c *gin.Context, req *SummaryExecuteRequest) (*SummaryExecuteResponse, error) {
	if s.rdb == nil {
		return nil, newServiceError(errcode.ErrInternalServer, "异步队列未初始化（Redis 不可用）")
	}
	trigger := normalizeSummaryTrigger(req.TriggerSource, model.SummaryTriggerEmbedManual)
	if trigger != model.SummaryTriggerEmbedAuto && trigger != model.SummaryTriggerEmbedManual {
		return nil, newServiceError(errcode.ErrParamValidation, "嵌入总结 trigger_source 无效")
	}
	detail, queueKind := normalizeSummaryTriggerDetail(trigger, req.TriggerDetail)
	tenantID, userID, err := s.extractIDs(c)
	if err != nil {
		return nil, err
	}
	release, acquired, lockErr := acquireEmbedCreateLock(
		c.Request.Context(),
		s.rdb,
		embedRefreshModuleSummary,
		tenantID,
		req.ProcessID,
	)
	if lockErr != nil {
		return nil, newServiceError(errcode.ErrRedisConn, "总结任务去重锁获取失败")
	}
	if !acquired {
		return nil, newServiceError(errcode.ErrResourceConflict, "该流程正在创建总结任务")
	}
	defer release()

	if running, runningErr := s.logRepo.GetRunningByProcessID(c, req.ProcessID); runningErr != nil {
		return nil, newServiceError(errcode.ErrDatabase, "查询进行中的总结任务失败")
	} else if running != nil {
		promoteToInteractive := queueKind == model.JobQueueKindInteractive &&
			running.Status == model.JobStatusPending &&
			running.QueueKind != model.JobQueueKindInteractive
		relabelAsManual := trigger == model.SummaryTriggerEmbedManual &&
			running.Status == model.JobStatusPending &&
			running.TriggerDetail != model.SummaryTriggerDetailManual
		if promoteToInteractive || relabelAsManual {
			if err := s.logRepo.UpdateFields(c, running.ID, map[string]interface{}{
				"user_id":            userID,
				"trigger_source":     trigger,
				"trigger_detail":     detail,
				"queue_kind":         queueKind,
				"schedule_config_id": nil,
				"updated_at":         apptime.Now(),
			}); err != nil {
				return nil, newServiceError(errcode.ErrDatabase, "切换总结任务队列失败")
			}
			if promoteToInteractive {
				if _, err := EnqueueSummaryJob(
					c.Request.Context(),
					s.rdb,
					running.ID,
					tenantID,
					userID,
					queueKind,
				); err != nil {
					return nil, newServiceError(errcode.ErrRedisConn, "总结交互任务入队失败: "+err.Error())
				}
			}
			running.TriggerSource = trigger
			running.TriggerDetail = detail
			running.QueueKind = queueKind
		}
		return s.summaryLogToResponse(running), nil
	}

	ctxResp, err := s.GetEmbedContext(c, req.ProcessID)
	if err != nil {
		return nil, err
	}
	if !ctxResp.Supported || !ctxResp.EmbedEnabled {
		return nil, newServiceError(errcode.ErrNoProcessConfig, ctxResp.Message)
	}
	if trigger == model.SummaryTriggerEmbedAuto && !ctxResp.ShouldAutoSummary {
		if ctxResp.AutoRetryBlocked {
			return nil, newServiceError(errcode.ErrResourceConflict, "相同数据和提示词的自动总结已经失败，请手动重新总结")
		}
		return nil, newServiceError(errcode.ErrResourceConflict, "未检测到需要自动刷新的总结块")
	}
	processType := req.ProcessType
	title := req.Title
	if ctxResp.Process != nil {
		if processType == "" {
			processType = ctxResp.Process.ProcessType
		}
		if title == "" {
			title = ctxResp.Process.Title
		}
	}
	if processType == "" {
		return nil, newServiceError(errcode.ErrParamValidation, "无法识别流程类型")
	}
	logID, tenantID, userID, createdAt, err := s.createPendingSummaryLog(
		c,
		req.ProcessID,
		processType,
		title,
		trigger,
		detail,
		queueKind,
		ctxResp.CurrentFingerprint,
		req.ScheduleConfigID,
	)
	if err != nil {
		return nil, err
	}
	if _, err := EnqueueSummaryJob(c.Request.Context(), s.rdb, logID, tenantID, userID, queueKind); err != nil {
		_ = s.logRepo.UpdateFields(c, logID, map[string]interface{}{
			"status":        model.JobStatusFailed,
			"error_message": "任务入队失败: " + err.Error(),
			"updated_at":    apptime.Now(),
		})
		return nil, newServiceError(errcode.ErrRedisConn, "总结任务入队失败: "+err.Error())
	}
	return &SummaryExecuteResponse{
		Status:    model.JobStatusPending,
		ID:        logID.String(),
		TraceID:   fmt.Sprintf("SM-%s", logID.String()[:8]),
		ProcessID: req.ProcessID,
		CreatedAt: apptime.FormatRFC3339(createdAt),
	}, nil
}

func (s *ProcessSummaryService) createPendingSummaryLog(
	c *gin.Context,
	processID, processType, title, trigger, detail string,
	queueKind string,
	attemptFingerprint string,
	scheduleConfigID *uuid.UUID,
) (uuid.UUID, uuid.UUID, uuid.UUID, time.Time, error) {
	tenantID, userID, err := s.extractIDs(c)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, time.Time{}, err
	}
	tenant, err := s.tenantRepo.FindByID(tenantID)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, time.Time{}, newServiceError(errcode.ErrDatabase, "获取租户信息失败")
	}
	if tenant.PrimaryModelID == nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, time.Time{}, newServiceError(errcode.ErrNoAIModelConfig, "租户未配置主用 AI 模型")
	}
	if _, err := s.aiModelRepo.FindByID(*tenant.PrimaryModelID); err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, time.Time{}, newServiceError(errcode.ErrNoAIModelConfig, "AI 模型配置不存在")
	}
	config, err := s.configRepo.GetByProcessType(c, processType)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, time.Time{}, newServiceError(errcode.ErrNoProcessConfig, fmt.Sprintf("流程 '%s' 的总结配置不存在", processType))
	}
	if config.Status != "active" {
		return uuid.Nil, uuid.Nil, uuid.Nil, time.Time{}, newServiceError(errcode.ErrNoProcessConfig, fmt.Sprintf("流程 '%s' 的总结配置已停用", processType))
	}
	if !config.EmbedEnabled {
		return uuid.Nil, uuid.Nil, uuid.Nil, time.Time{}, newServiceError(errcode.ErrPermissionDenied, fmt.Sprintf("流程 '%s' 未启用 OA 嵌入总结", processType))
	}
	id := uuid.New()
	now := apptime.Now()
	row := &model.ProcessSummaryLog{
		ID:                 id,
		TenantID:           tenantID,
		UserID:             userID,
		ProcessID:          processID,
		Title:              title,
		ProcessType:        processType,
		Status:             model.JobStatusPending,
		SummaryResult:      datatypes.JSON([]byte("{}")),
		ProcessSnapshot:    datatypes.JSON([]byte("{}")),
		TriggerSource:      trigger,
		TriggerDetail:      detail,
		QueueKind:          queueKind,
		AttemptFingerprint: attemptFingerprint,
		ScheduleConfigID:   scheduleConfigID,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := s.logRepo.Create(row); err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, time.Time{}, newServiceError(errcode.ErrDatabase, "总结日志写入失败")
	}
	return id, tenantID, userID, now, nil
}

func (s *ProcessSummaryService) processSummaryJob(
	ctx context.Context,
	summaryLogID, tenantID, userID uuid.UUID,
	queueKind string,
) error {
	ctx, cancel := context.WithTimeout(ctx, summaryProcessTimeout)
	defer cancel()
	c := s.workerGinContext(ctx, tenantID, userID)
	claimed, err := s.logRepo.ClaimPending(c, summaryLogID, queueKind)
	if err != nil {
		return err
	}
	if !claimed {
		return nil
	}
	logEntry, err := s.logRepo.GetByID(c, summaryLogID)
	if err != nil {
		return err
	}
	startTime := apptime.Now()
	tenant, err := s.tenantRepo.FindByID(tenantID)
	if err != nil {
		s.markSummaryFailedDB(tenantID, summaryLogID, "获取租户信息失败")
		return err
	}
	tlog := pkglogger.GetTenantLogger(tenant.Code)
	modelCfg, fallbackCfg, err := s.loadTenantModels(tenant)
	if err != nil {
		s.markSummaryFailedDB(tenantID, summaryLogID, err.Error())
		return err
	}
	config, err := s.configRepo.GetByProcessType(c, logEntry.ProcessType)
	if err != nil {
		s.markSummaryFailedDB(tenantID, summaryLogID, "总结配置不存在")
		return err
	}
	blocks := parseSummaryBlocks(config.SummaryBlocks)
	if len(blocks) == 0 {
		blocks = defaultSummaryBlocks()
	}

	unionFieldSet := buildSummaryUnionFieldSet(blocks)
	processData, err := s.fetchOAData(c, tenant, logEntry.ProcessID, unionFieldSet, false)
	if err != nil {
		s.markSummaryFailedDB(tenantID, summaryLogID, "拉取 OA 流程数据失败: "+err.Error())
		return err
	}
	processSummary, _ := s.fetchRequestSummary(c, tenantID, logEntry.ProcessID)
	currentAnchor, err := s.fetchOAAnchorWithData(c, tenantID, logEntry.ProcessID, processData)
	if err != nil {
		s.markSummaryFailedDB(tenantID, summaryLogID, "读取 OA 变化锚点失败: "+err.Error())
		return err
	}
	currentDependencies := buildSummaryBlockDependencyFingerprints(blocks, processData, currentAnchor, processSummary)
	attemptSnapshot, _ := json.Marshal(map[string]interface{}{
		"block_dependencies": currentDependencies,
	})
	if err := s.logRepo.UpdateFields(c, summaryLogID, map[string]interface{}{
		"attempt_fingerprint": stableJSONFingerprint(currentDependencies),
		"process_snapshot":    datatypes.JSON(attemptSnapshot),
		"updated_at":          apptime.Now(),
	}); err != nil {
		return err
	}

	var previousLog *model.ProcessSummaryLog
	if snap, snapErr := s.snapshotRepo.GetByProcessID(c, logEntry.ProcessID); snapErr == nil && snap != nil {
		if row, rowErr := s.logRepo.GetByID(c, snap.LatestValidLogID); rowErr == nil {
			previousLog = row
		}
	}

	blockIDsToRun := make(map[string]bool)
	if logEntry.TriggerSource == model.SummaryTriggerEmbedAuto && previousLog != nil {
		storedAnchor := parseOAContextAnchor(previousLog.OAContextAnchor)
		storedDependencies := parseSummaryBlockDependencies(previousLog.ProcessSnapshot)
		changes := oa.CompareContextAnchors(storedAnchor, currentAnchor)
		embedCfg := parseSummaryEmbedConfig(config.EmbedConfig)
		for _, id := range changedSummaryBlockIDs(blocks, storedDependencies, currentDependencies, changes, embedCfg) {
			blockIDsToRun[id] = true
		}
	} else {
		for _, block := range blocks {
			if block.Enabled {
				blockIDsToRun[block.ID] = true
			}
		}
	}

	enabledBlocks := make([]model.SummaryBlockConfig, 0, len(blockIDsToRun))
	needsAttachments := false
	for _, block := range blocks {
		if !block.Enabled || !blockIDsToRun[block.ID] {
			continue
		}
		enabledBlocks = append(enabledBlocks, block)
		if summaryBlockUsesVariable(block, "{{attachments}}") {
			needsAttachments = true
		}
	}
	if needsAttachments {
		processData, err = s.fetchOAData(c, tenant, logEntry.ProcessID, unionFieldSet, true)
		if err != nil {
			s.markSummaryFailedDB(tenantID, summaryLogID, "拉取 OA 附件数据失败: "+err.Error())
			return err
		}
	}
	flowSnapshot := s.fetchFlowSnapshot(c, tenant, logEntry.ProcessID)
	if s.sysFlags != nil && s.sysFlags.DataEncryptionEnabled() {
		sanitize.SanitizeProcessData(processData)
		sanitize.SanitizeFlowSnapshot(flowSnapshot)
	}

	if err := s.logRepo.UpdateFields(c, summaryLogID, map[string]interface{}{"status": model.JobStatusReasoning, "updated_at": apptime.Now()}); err != nil {
		return err
	}

	resultsByIndex := make([]model.ProcessSummaryBlockResult, len(enabledBlocks))
	rawPartsByIndex := make([]string, len(enabledBlocks))
	parseErrorsByIndex := make([]string, len(enabledBlocks))

	blockCtx, cancelBlocks := context.WithCancel(ctx)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	for idx, block := range enabledBlocks {
		idx, block := idx, block
		wg.Add(1)
		go func() {
			defer wg.Done()
			blockStart := apptime.Now()
			fieldSet := buildSummaryBlockFieldSet(block)
			externalContextText := ""
			if s.externalCtx != nil && len(block.ContextMounts) > 0 {
				externalContextText = s.externalCtx.ResolveMountsForPrompt(c, tenant, logEntry.ProcessID, processData, block.ContextMounts)
				if s.sysFlags != nil && s.sysFlags.DataEncryptionEnabled() {
					externalContextText = sanitize.SanitizeText(externalContextText)
				}
			}
			req := BuildSummaryBlockPrompt(logEntry.ProcessType, processData, flowSnapshot, block, fieldSet, processSummary, externalContextText)
			req.Temperature = float64(tenant.Temperature)
			req.MaxTokens = tenant.MaxTokensPerRequest
			req.ModelConfig = modelCfg
			req.StreamChunkFunc = func(chunk string) {
				s.publishSummaryBlockChunk(summaryLogID, block.ID, block.Title, chunk)
			}
			processTitle := logEntry.Title
			if processSummary != nil && strings.TrimSpace(processSummary.Title) != "" {
				processTitle = processSummary.Title
			}
			bindLLMProcessContext(req, logEntry.ProcessID, processTitle, summaryLogID)
			blockGinCtx := s.workerGinContext(blockCtx, tenantID, userID)
			resp, err := s.aiCaller.ChatWithFallback(blockGinCtx, tenantID, userID, modelCfg, fallbackCfg, req)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
					cancelBlocks()
				}
				mu.Unlock()
				return
			}
			raw := resp.Content
			if s.sysFlags != nil && s.sysFlags.DataEncryptionEnabled() {
				raw = sanitize.SanitizeText(raw)
			}
			parsed, parseErr := ParseSummaryBlockResult(raw, block)
			if parseErr != nil {
				parseErrorsByIndex[idx] = fmt.Sprintf("%s: %s", block.Title, parseErr.Error())
			}
			parsed.DurationMs = int(time.Since(blockStart).Milliseconds())
			resultsByIndex[idx] = parsed
			rawPartsByIndex[idx] = fmt.Sprintf("## %s\n%s", block.Title, raw)
		}()
	}
	wg.Wait()
	cancelBlocks()
	if firstErr != nil {
		s.markSummaryFailedDB(tenantID, summaryLogID, firstErr.Error())
		tlog.Warn("总结任务执行失败", zap.String("summaryLogID", summaryLogID.String()), zap.Error(firstErr))
		return firstErr
	}

	generatedResults := make(map[string]model.ProcessSummaryBlockResult, len(resultsByIndex))
	parseErrors := make([]string, 0)
	rawParts := make([]string, 0, len(rawPartsByIndex))
	for idx, result := range resultsByIndex {
		if result.BlockID == "" && result.Title == "" && result.Content == "" {
			continue
		}
		generatedResults[result.BlockID] = result
		if rawPartsByIndex[idx] != "" {
			rawParts = append(rawParts, rawPartsByIndex[idx])
		}
		if parseErrorsByIndex[idx] != "" {
			parseErrors = append(parseErrors, parseErrorsByIndex[idx])
		}
	}
	previousResults := make(map[string]model.ProcessSummaryBlockResult)
	if previousLog != nil {
		var previous model.ProcessSummaryResultJSON
		if json.Unmarshal(previousLog.SummaryResult, &previous) == nil {
			for _, result := range previous.Blocks {
				previousResults[result.BlockID] = result
			}
		}
	}
	results := make([]model.ProcessSummaryBlockResult, 0, len(blocks))
	for _, block := range blocks {
		if !block.Enabled {
			continue
		}
		if result, ok := generatedResults[block.ID]; ok {
			results = append(results, result)
			continue
		}
		if result, ok := previousResults[block.ID]; ok {
			result.Title = block.Title
			results = append(results, result)
		}
	}
	if len(results) == 0 {
		results = append(results, model.ProcessSummaryBlockResult{
			BlockID: "fallback",
			Title:   summaryDefaultBlockName,
			Content: "（未生成总结内容，请检查总结块配置）",
			Points:  []string{},
		})
		parseErrors = append(parseErrors, "未生成任何启用的总结块")
	}

	if err := s.logRepo.UpdateFields(c, summaryLogID, map[string]interface{}{"status": model.JobStatusExtracting, "updated_at": apptime.Now()}); err != nil {
		return err
	}

	resultJSON, _ := json.Marshal(model.ProcessSummaryResultJSON{Blocks: results})
	snapshotJSON, _ := json.Marshal(map[string]interface{}{
		"process":               processSummary,
		"process_data":          processData,
		"flow_snapshot":         flowSnapshot,
		"summary_block_ids":     activeSummaryBlockIDs(blocks),
		"block_dependencies":    currentDependencies,
		"regenerated_block_ids": sortedStringSetKeys(blockIDsToRun),
	})
	rawStored := strings.Join(rawParts, "\n\n")
	parseErrText := strings.Join(parseErrors, "\n")
	anchorJSON, _ := json.Marshal(currentAnchor)
	duration := int(time.Since(startTime).Milliseconds())

	if err := s.logRepo.UpdateFields(c, summaryLogID, map[string]interface{}{
		"status":            model.JobStatusCompleted,
		"summary_result":    datatypes.JSON(resultJSON),
		"process_snapshot":  datatypes.JSON(snapshotJSON),
		"duration_ms":       duration,
		"raw_content":       rawStored,
		"parse_error":       parseErrText,
		"oa_context_anchor": datatypes.JSON(anchorJSON),
		"updated_at":        apptime.Now(),
	}); err != nil {
		s.markSummaryFailedDB(tenantID, summaryLogID, "保存总结结果失败: "+err.Error())
		return err
	}
	if err := s.snapshotRepo.UpsertAppendValid(c, tenantID, logEntry.ProcessID, summaryLogID, logEntry.Title, logEntry.ProcessType, len(results)); err != nil {
		return err
	}
	tlog.Info("总结任务执行完成",
		zap.String("summaryLogID", summaryLogID.String()),
		zap.Int("blocks", len(results)),
		zap.Int("regeneratedBlocks", len(enabledBlocks)))
	return nil
}

func (s *ProcessSummaryService) GetJobStatus(c *gin.Context, id uuid.UUID) (*SummaryExecuteResponse, error) {
	logEntry, err := s.logRepo.GetByID(c, id)
	if err != nil {
		return nil, err
	}
	return s.summaryLogToResponse(logEntry), nil
}

func (s *ProcessSummaryService) SubscribeJobStream(ctx context.Context, id uuid.UUID) (<-chan string, func(), error) {
	if s.rdb == nil {
		return nil, nil, newServiceError(errcode.ErrRedisConn, "Redis 未初始化")
	}
	ch := make(chan string, 16)
	key := summaryStreamKeyPrefix + id.String()
	if existing, err := s.rdb.Get(ctx, key).Result(); err == nil && existing != "" {
		for _, line := range strings.Split(existing, "\n") {
			if strings.TrimSpace(line) != "" {
				ch <- line
			}
		}
	}
	pubsub := s.rdb.Subscribe(ctx, summaryPubSubKeyPrefix+id.String())
	go func() {
		defer close(ch)
		for msg := range pubsub.Channel() {
			ch <- msg.Payload
		}
	}()
	return ch, func() { _ = pubsub.Close() }, nil
}

func (s *ProcessSummaryService) ListSnapshots(c *gin.Context, filter repository.ProcessSummarySnapshotFilter, page, pageSize int) ([]repository.ProcessSummarySnapshotListRow, int64, error) {
	return s.snapshotRepo.ListPagedWithUser(c, filter, page, pageSize)
}

func (s *ProcessSummaryService) ListSnapshotsForExport(c *gin.Context, filter repository.ProcessSummarySnapshotFilter) ([]repository.ProcessSummarySnapshotListRow, error) {
	items, _, err := s.snapshotRepo.ListPagedWithUser(c, filter, 1, 5000)
	return items, err
}

func (s *ProcessSummaryService) GetSnapshotStats(c *gin.Context, channel string) (*repository.ProcessSummarySnapshotStats, error) {
	return s.snapshotRepo.CountStats(c, channel)
}

func (s *ProcessSummaryService) GetSnapshotChain(c *gin.Context, processID string) ([]repository.ProcessSummaryLogWithUser, error) {
	snap, err := s.snapshotRepo.GetByProcessID(c, processID)
	if err != nil {
		return nil, err
	}
	if snap == nil {
		return []repository.ProcessSummaryLogWithUser{}, nil
	}
	ids := parseSummarySnapshotValidIDs(snap.ValidLogIDs)
	chain, err := s.logRepo.ListByIDsWithUserOrdered(c, ids)
	if err != nil {
		return nil, err
	}
	sort.Slice(chain, func(i, j int) bool {
		return chain[i].CreatedAt.After(chain[j].CreatedAt)
	})
	return chain, nil
}

func (s *ProcessSummaryService) summaryLogToResponse(log *model.ProcessSummaryLog) *SummaryExecuteResponse {
	resp := &SummaryExecuteResponse{
		Status:       log.Status,
		ID:           log.ID.String(),
		TraceID:      fmt.Sprintf("SM-%s", log.ID.String()[:8]),
		ProcessID:    log.ProcessID,
		DurationMs:   log.DurationMs,
		CreatedAt:    apptime.FormatRFC3339(log.CreatedAt),
		ParseError:   log.ParseError,
		RawContent:   log.RawContent,
		ErrorMessage: log.ErrorMessage,
	}
	var parsed model.ProcessSummaryResultJSON
	if err := json.Unmarshal(log.SummaryResult, &parsed); err == nil {
		resp.Blocks = parsed.Blocks
	}
	return resp
}

func (s *ProcessSummaryService) buildSummaryResultFromLog(log *model.ProcessSummaryLog) map[string]interface{} {
	resp := s.summaryLogToResponse(log)
	b, _ := json.Marshal(resp)
	var out map[string]interface{}
	_ = json.Unmarshal(b, &out)
	return out
}

func (s *ProcessSummaryService) publishSummaryBlockChunk(id uuid.UUID, blockID, title, chunk string) {
	if s.rdb == nil || chunk == "" {
		return
	}
	payload, err := json.Marshal(summaryStreamChunk{
		BlockID: blockID,
		Title:   title,
		Chunk:   chunk,
	})
	if err != nil {
		return
	}
	message := string(payload)
	key := summaryStreamKeyPrefix + id.String()
	s.rdb.Append(context.Background(), key, message+"\n")
	s.rdb.Expire(context.Background(), key, 24*time.Hour)
	s.rdb.Publish(context.Background(), summaryPubSubKeyPrefix+id.String(), message)
}

func (s *ProcessSummaryService) loadTenantModels(tenant *model.Tenant) (*model.AIModelConfig, *model.AIModelConfig, error) {
	if tenant.PrimaryModelID == nil {
		return nil, nil, newServiceError(errcode.ErrNoAIModelConfig, "租户未配置主用 AI 模型")
	}
	modelCfg, err := s.aiModelRepo.FindByID(*tenant.PrimaryModelID)
	if err != nil {
		return nil, nil, newServiceError(errcode.ErrNoAIModelConfig, "AI 模型配置不存在")
	}
	if modelCfg.APIKey != "" {
		decrypted, err := crypto.Decrypt(modelCfg.APIKey)
		if err != nil {
			return nil, nil, newServiceError(errcode.ErrInternalServer, "API Key 解密失败")
		}
		modelCfg.APIKey = decrypted
	}
	var fallbackCfg *model.AIModelConfig
	if tenant.FallbackModelID != nil {
		if fb, err := s.aiModelRepo.FindByID(*tenant.FallbackModelID); err == nil {
			if fb.APIKey != "" {
				if decrypted, err := crypto.Decrypt(fb.APIKey); err == nil {
					fb.APIKey = decrypted
				}
			}
			fallbackCfg = fb
		}
	}
	return modelCfg, fallbackCfg, nil
}

func (s *ProcessSummaryService) fetchOAData(c *gin.Context, tenant *model.Tenant, processID string, fieldSet SelectedFieldSet, withAttachments bool) (*oa.ProcessData, error) {
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
	var attachmentSvc oa.AttachmentRecognitionService
	if withAttachments && s.attachmentSvc != nil {
		attachmentSvc = s.attachmentSvc
	}
	adapter, err := s.oaConnections.GetAdapter(c.Request.Context(), conn.OAType, conn, attachmentSvc)
	if err != nil {
		return nil, newServiceError(errcode.ErrOAConnectionFailed, "创建 OA 适配器失败: "+err.Error())
	}
	fetchCtx := c.Request.Context()
	if withAttachments && fieldSet != nil {
		allowedMainFields := fieldSet["main"]
		if allowedMainFields == nil {
			allowedMainFields = map[string]bool{}
		}
		fetchCtx = oa.WithAttachmentFieldFilter(fetchCtx, allowedMainFields)
	}
	data, err := adapter.FetchProcessData(fetchCtx, processID)
	if err != nil {
		return nil, newServiceError(errcode.ErrOAQueryFailed, err.Error())
	}
	if resolver, ok := adapter.(oa.BrowseValueResolver); ok {
		if err := resolver.ResolveBrowseDisplayValues(c.Request.Context(), processID, data, fieldSet); err != nil {
			return nil, newServiceError(errcode.ErrOAQueryFailed, "解析 OA 浏览按钮显示值失败: "+err.Error())
		}
	}
	return data, nil
}

func (s *ProcessSummaryService) fetchFlowSnapshot(c *gin.Context, tenant *model.Tenant, processID string) *oa.ProcessFlowSnapshot {
	adapter, err := s.getOAAdapter(c.Request.Context(), tenant.ID, false)
	if err != nil {
		return nil
	}
	snapshot, err := adapter.FetchProcessFlow(c.Request.Context(), processID)
	if err != nil {
		return nil
	}
	return snapshot
}

func (s *ProcessSummaryService) fetchRequestSummary(c *gin.Context, tenantID uuid.UUID, processID string) (*oa.ProcessRequestSummary, error) {
	adapter, err := s.getOAAdapter(c.Request.Context(), tenantID, false)
	if err != nil {
		return nil, err
	}
	return adapter.FetchProcessRequestSummary(c.Request.Context(), processID)
}

func (s *ProcessSummaryService) fetchCurrentOAState(
	c *gin.Context,
	tenantID uuid.UUID,
	processID string,
	fieldSet SelectedFieldSet,
) (*oa.ProcessData, oa.OAContextAnchor, error) {
	tenant, err := s.tenantRepo.FindByID(tenantID)
	if err != nil {
		return nil, oa.OAContextAnchor{}, newServiceError(errcode.ErrDatabase, "获取租户信息失败")
	}
	pd, err := s.fetchOAData(c, tenant, processID, fieldSet, false)
	if err != nil {
		return nil, oa.OAContextAnchor{}, err
	}
	anchor, err := s.fetchOAAnchorWithData(c, tenantID, processID, pd)
	return pd, anchor, err
}

func (s *ProcessSummaryService) fetchOAAnchorWithData(c *gin.Context, tenantID uuid.UUID, processID string, pd *oa.ProcessData) (oa.OAContextAnchor, error) {
	adapter, err := s.getOAAdapter(c.Request.Context(), tenantID, false)
	if err != nil {
		return oa.OAContextAnchor{}, err
	}
	anchor, err := adapter.FetchProcessContextAnchor(c.Request.Context(), processID, pd)
	if err != nil {
		return oa.OAContextAnchor{}, newServiceError(errcode.ErrOAQueryFailed, err.Error())
	}
	return *anchor, nil
}

func (s *ProcessSummaryService) buildOAContextAnchorForJob(c *gin.Context, tenantID uuid.UUID, processID string) []byte {
	_, anchor, err := s.fetchCurrentOAState(c, tenantID, processID, nil)
	if err != nil {
		return []byte("{}")
	}
	b, _ := json.Marshal(anchor)
	return b
}

func (s *ProcessSummaryService) getOAAdapter(
	ctx context.Context,
	tenantID uuid.UUID,
	withAttachment bool,
) (oa.OAAdapter, error) {
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
	if withAttachment && s.attachmentSvc != nil {
		return s.oaConnections.GetAdapter(ctx, conn.OAType, conn, s.attachmentSvc)
	}
	return s.oaConnections.GetAdapter(ctx, conn.OAType, conn)
}

func (s *ProcessSummaryService) decryptOAConn(conn *model.OADatabaseConnection) error {
	password, err := crypto.Decrypt(conn.Password)
	if err != nil {
		return newServiceError(errcode.ErrInternalServer, "OA 数据库密码解密失败")
	}
	conn.Password = password
	return nil
}

func (s *ProcessSummaryService) extractIDs(c *gin.Context) (uuid.UUID, uuid.UUID, error) {
	tidVal, exists := c.Get("tenant_id")
	if !exists {
		return uuid.Nil, uuid.Nil, newServiceError(errcode.ErrNoAuthToken, "租户ID缺失")
	}
	tenantID, err := uuid.Parse(fmt.Sprintf("%v", tidVal))
	if err != nil {
		return uuid.Nil, uuid.Nil, newServiceError(errcode.ErrNoAuthToken, "租户ID格式无效")
	}
	if embedUID, ok := c.Get("embed_user_id"); ok {
		if uid, ok2 := embedUID.(uuid.UUID); ok2 && uid != uuid.Nil {
			return tenantID, uid, nil
		}
	}
	claimsVal, _ := c.Get("jwt_claims")
	claims, ok := claimsVal.(*jwtpkg.JWTClaims)
	if !ok || claims.Sub == "" {
		return uuid.Nil, uuid.Nil, newServiceError(errcode.ErrNoAuthToken, "用户信息缺失")
	}
	userID, err := uuid.Parse(claims.Sub)
	if err != nil {
		return uuid.Nil, uuid.Nil, newServiceError(errcode.ErrNoAuthToken, "用户ID格式无效")
	}
	return tenantID, userID, nil
}

func (s *ProcessSummaryService) workerGinContext(ctx context.Context, tenantID, userID uuid.UUID) *gin.Context {
	rec := httptest.NewRecorder()
	gc, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPost, "/", nil).WithContext(ctx)
	gc.Request = req
	gc.Set("tenant_id", tenantID.String())
	gc.Set("jwt_claims", &jwtpkg.JWTClaims{Sub: userID.String(), Username: ""})
	return gc
}

func (s *ProcessSummaryService) markSummaryFailedDB(tenantID, id uuid.UUID, message string) error {
	return s.db.Model(&model.ProcessSummaryLog{}).
		Where("id = ? AND tenant_id = ?", id, tenantID).
		Updates(map[string]interface{}{
			"status":        model.JobStatusFailed,
			"error_message": message,
			"updated_at":    apptime.Now(),
		}).Error
}

func (s *ProcessSummaryService) FailStaleSummaryJobs(ctx context.Context) (int64, error) {
	cutoff := apptime.Now().Add(-summaryJobMaxAge)
	res := s.db.WithContext(ctx).Model(&model.ProcessSummaryLog{}).
		Where("status IN ? AND updated_at < ?", []string{
			model.JobStatusPending,
			model.JobStatusAssembling,
			model.JobStatusReasoning,
			model.JobStatusExtracting,
		}, cutoff).
		Updates(map[string]interface{}{
			"status":        model.JobStatusFailed,
			"error_message": summaryErrStaleMessage,
			"updated_at":    apptime.Now(),
		})
	return res.RowsAffected, res.Error
}

func parseSummaryEmbedConfig(raw datatypes.JSON) model.SummaryEmbedConfigData {
	cfg := model.SummaryEmbedConfigData{
		AutoSummaryOnOpen:           true,
		AutoSummaryOnDataChange:     true,
		AutoSummaryOnReturnResubmit: true,
		AutoSummaryOnFlowChange:     false,
		ScheduledLookbackDays:       3,
		ScheduledIntervalMinutes:    5,
	}
	if len(raw) == 0 {
		return cfg
	}
	_ = json.Unmarshal(raw, &cfg)
	normalizeScheduledRefreshConfig(&cfg.ScheduledLookbackDays, &cfg.ScheduledIntervalMinutes)
	return cfg
}

func normalizeSummaryTrigger(source, fallback string) string {
	switch source {
	case model.SummaryTriggerEmbedAuto, model.SummaryTriggerEmbedManual:
		return source
	default:
		return fallback
	}
}

func normalizeSummaryTriggerDetail(trigger, detail string) (string, string) {
	if trigger == model.SummaryTriggerEmbedManual {
		return model.SummaryTriggerDetailManual, model.JobQueueKindInteractive
	}
	switch strings.TrimSpace(detail) {
	case model.SummaryTriggerDetailVisibleOpen:
		return model.SummaryTriggerDetailVisibleOpen, model.JobQueueKindInteractive
	case model.SummaryTriggerDetailScheduled:
		return model.SummaryTriggerDetailScheduled, model.JobQueueKindScheduled
	case model.SummaryTriggerDetailSaveRequested:
		return model.SummaryTriggerDetailSaveRequested, model.JobQueueKindBackground
	case model.SummaryTriggerDetailSubmitRequested:
		return model.SummaryTriggerDetailSubmitRequested, model.JobQueueKindBackground
	default:
		// 兼容旧版可见嵌入页：未传详细来源的自动请求按前台打开处理。
		return model.SummaryTriggerDetailVisibleOpen, model.JobQueueKindInteractive
	}
}

func parseSummaryBlocks(raw datatypes.JSON) []model.SummaryBlockConfig {
	var blocks []model.SummaryBlockConfig
	_ = json.Unmarshal(raw, &blocks)
	out := make([]model.SummaryBlockConfig, 0, len(blocks))
	for i, block := range blocks {
		out = append(out, normalizeSummaryBlock(block, i))
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].SortOrder < out[j].SortOrder })
	return out
}

func buildSummaryBlockFieldSet(block model.SummaryBlockConfig) SelectedFieldSet {
	if block.FieldMode == "all" {
		return nil
	}
	set := SelectedFieldSet{"main": map[string]bool{}}
	for _, ref := range block.SelectedFields {
		table, field := parseSummaryFieldRef(ref)
		if table == "" || field == "" {
			continue
		}
		if set[table] == nil {
			set[table] = map[string]bool{}
		}
		set[table][strings.ToLower(field)] = true
	}
	return set
}

func buildSummaryUnionFieldSet(blocks []model.SummaryBlockConfig) SelectedFieldSet {
	union := SelectedFieldSet{"main": map[string]bool{}}
	for _, block := range blocks {
		if !block.Enabled {
			continue
		}
		if block.FieldMode == "all" {
			return nil
		}
		for _, ref := range block.SelectedFields {
			table, field := parseSummaryFieldRef(ref)
			if table == "" || field == "" {
				continue
			}
			if union[table] == nil {
				union[table] = map[string]bool{}
			}
			union[table][strings.ToLower(field)] = true
		}
	}
	return union
}

func parseSummaryFieldRef(ref string) (string, string) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", ""
	}
	parts := strings.SplitN(ref, ":", 2)
	if len(parts) == 1 {
		return "main", parts[0]
	}
	return parts[0], parts[1]
}

func activeSummaryBlockIDs(blocks []model.SummaryBlockConfig) []string {
	out := make([]string, 0, len(blocks))
	for _, block := range blocks {
		if block.Enabled {
			out = append(out, block.ID)
		}
	}
	return out
}

func parseSummarySnapshotValidIDs(raw datatypes.JSON) []uuid.UUID {
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

func isSummaryJobRunningStatus(status string) bool {
	switch status {
	case model.JobStatusPending, model.JobStatusAssembling, model.JobStatusReasoning, model.JobStatusExtracting:
		return true
	default:
		return false
	}
}
