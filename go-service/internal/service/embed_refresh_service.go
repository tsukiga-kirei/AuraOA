package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/apptime"
	pkglogger "auraoa/go-service/internal/pkg/logger"
	"auraoa/go-service/internal/pkg/oa"
	"auraoa/go-service/internal/repository"
)

const (
	embedRefreshDueKey        = "embed:refresh:due"
	embedRefreshPayloadTTL    = 48 * time.Hour
	embedRefreshInitialDelay  = 2 * time.Second
	embedRefreshRunningDelay  = 15 * time.Second
	embedRefreshRunningMaxAge = 40 * time.Minute
	embedRefreshScanLimit     = 500
	embedRefreshScheduleTTL   = 40 * time.Minute
	embedRefreshScheduleTopic = "embed:refresh:schedule:changed"

	embedRefreshModuleAudit   = "audit"
	embedRefreshModuleSummary = "summary"
	embedRefreshModuleResolve = "resolve"
)

var scheduleEmbedRefreshScript = redis.NewScript(`
redis.call("SET", KEYS[2], ARGV[1], "EX", ARGV[4])
redis.call("SET", KEYS[3], ARGV[2], "EX", ARGV[4])
redis.call("ZADD", KEYS[1], ARGV[3], ARGV[5])
return 1
`)

var retryEmbedRefreshScript = redis.NewScript(`
if redis.call("GET", KEYS[3]) ~= ARGV[2] then
  return 0
end
redis.call("SET", KEYS[2], ARGV[1], "EX", ARGV[4])
redis.call("ZADD", KEYS[1], ARGV[3], ARGV[5])
return 1
`)

var scheduleIdleEmbedRefreshScript = redis.NewScript(`
if redis.call("ZSCORE", KEYS[1], ARGV[5]) then
  return 0
end
redis.call("SET", KEYS[2], ARGV[1], "EX", ARGV[4])
redis.call("SET", KEYS[3], ARGV[2], "EX", ARGV[4])
redis.call("ZADD", KEYS[1], ARGV[3], ARGV[5])
return 1
`)

var claimEmbedRefreshScript = redis.NewScript(`
local score = redis.call("ZSCORE", KEYS[1], ARGV[1])
if not score or tonumber(score) > tonumber(ARGV[2]) then
  return ""
end
local payload = redis.call("GET", KEYS[2])
redis.call("ZREM", KEYS[1], ARGV[1])
return payload or ""
`)

// ErrInvalidEmbedRefreshAction 表示 OA 嵌入刷新动作不是当前版本支持的保存或提交请求。
var ErrInvalidEmbedRefreshAction = errors.New("action 仅支持 save_requested 或 submit_requested")

// ErrInvalidEmbedRefreshContext 表示首次新建流程缺少用于解析 requestid 的流程定义。
var ErrInvalidEmbedRefreshContext = errors.New("首次新建流程缺少 workflow_id")

// EmbedRefreshEventRequest OA 保存/提交前的轻量刷新事件；首次新建流程允许 process_id 为空。
type EmbedRefreshEventRequest struct {
	ProcessID       string `json:"process_id"`
	WorkflowID      string `json:"workflow_id"`
	OABelongUserID  string `json:"oa_belong_user_id"`
	OACurrentUserID string `json:"oa_current_user_id"`
	OccurredAtMS    int64  `json:"occurred_at_ms"`
	Action          string `json:"action" binding:"required"`
	EventID         string `json:"event_id"`
}

// EmbedRefreshEventResponse 事件接收结果。
type EmbedRefreshEventResponse struct {
	ProcessID         string   `json:"process_id"`
	Action            string   `json:"action"`
	EventID           string   `json:"event_id"`
	ScheduledModules  []string `json:"scheduled_modules"`
	ResolutionPending bool     `json:"resolution_pending"`
}

type embedRefreshPayload struct {
	TenantID          uuid.UUID `json:"tenant_id"`
	UserID            uuid.UUID `json:"user_id"`
	ProcessID         string    `json:"process_id"`
	Module            string    `json:"module"`
	Action            string    `json:"action"`
	EventID           string    `json:"event_id"`
	Generation        string    `json:"generation"`
	Attempt           int       `json:"attempt"`
	FirstReceived     time.Time `json:"first_received"`
	ConfigID          uuid.UUID `json:"config_id,omitempty"`
	ScheduleID        uuid.UUID `json:"schedule_id,omitempty"`
	WorkflowID        string    `json:"workflow_id,omitempty"`
	OABelongUserID    string    `json:"oa_belong_user_id,omitempty"`
	OACurrentUserID   string    `json:"oa_current_user_id,omitempty"`
	BaselineRequestID int64     `json:"baseline_request_id,omitempty"`
	OccurredAtMS      int64     `json:"occurred_at_ms,omitempty"`
}

type embedRefreshResult int

const (
	embedRefreshDone embedRefreshResult = iota
	embedRefreshRetry
	embedRefreshRunning
)

type embedRefreshCheckOutcome struct {
	Result            embedRefreshResult
	Reason            string
	JobID             string
	ResolvedProcessID string
}

// EmbedRefreshService 编排 OA 保存/提交事件、首次 requestid 解析、延迟指纹检查和流程级定时扫描。
type EmbedRefreshService struct {
	rdb             *redis.Client
	auditSvc        *AuditExecuteService
	summarySvc      *ProcessSummaryService
	auditRepo       *repository.ProcessAuditConfigRepo
	summaryRepo     *repository.ProcessSummaryConfigRepo
	scheduleRepo    *repository.EmbedRefreshScheduleRepo
	eventRepo       *repository.EmbedRefreshEventRepo
	tenantRepo      *repository.TenantRepo
	scheduleCron    *cron.Cron
	scheduleMu      sync.Mutex
	scheduleEntries map[uuid.UUID]cron.EntryID
	scheduleConfigs map[string]uuid.UUID
	scheduleExprs   map[uuid.UUID]string
	logger          *zap.Logger
}

// NewEmbedRefreshService 创建嵌入刷新协调服务。
func NewEmbedRefreshService(
	rdb *redis.Client,
	auditSvc *AuditExecuteService,
	summarySvc *ProcessSummaryService,
	auditRepo *repository.ProcessAuditConfigRepo,
	summaryRepo *repository.ProcessSummaryConfigRepo,
	scheduleRepo *repository.EmbedRefreshScheduleRepo,
	eventRepo *repository.EmbedRefreshEventRepo,
	tenantRepo *repository.TenantRepo,
	logger *zap.Logger,
) *EmbedRefreshService {
	return &EmbedRefreshService{
		rdb:          rdb,
		auditSvc:     auditSvc,
		summarySvc:   summarySvc,
		auditRepo:    auditRepo,
		summaryRepo:  summaryRepo,
		scheduleRepo: scheduleRepo,
		eventRepo:    eventRepo,
		tenantRepo:   tenantRepo,
		scheduleCron: cron.New(
			cron.WithParser(newCronParser()),
			cron.WithLocation(apptime.Location()),
			cron.WithChain(cron.Recover(cron.DefaultLogger)),
		),
		scheduleEntries: make(map[uuid.UUID]cron.EntryID),
		scheduleConfigs: make(map[string]uuid.UUID),
		scheduleExprs:   make(map[uuid.UUID]string),
		logger:          logger,
	}
}

// ScheduleEvent 接收 OA 保存/提交事件；已有 requestid 直接安排检查，首次新建流程先记录高水位再异步解析。
func (s *EmbedRefreshService) ScheduleEvent(
	ctx context.Context,
	tenantID, userID uuid.UUID,
	req EmbedRefreshEventRequest,
) (*EmbedRefreshEventResponse, error) {
	if s.rdb == nil {
		return nil, fmt.Errorf("Redis 不可用")
	}
	if s.eventRepo == nil {
		return nil, fmt.Errorf("嵌入刷新事件存储不可用")
	}
	req.ProcessID = strings.TrimSpace(req.ProcessID)
	req.WorkflowID = strings.TrimSpace(req.WorkflowID)
	req.OABelongUserID = strings.TrimSpace(req.OABelongUserID)
	req.OACurrentUserID = strings.TrimSpace(req.OACurrentUserID)
	req.Action = strings.TrimSpace(req.Action)
	if req.Action != model.SummaryTriggerDetailSaveRequested &&
		req.Action != model.SummaryTriggerDetailSubmitRequested {
		return nil, ErrInvalidEmbedRefreshAction
	}
	if strings.TrimSpace(req.EventID) == "" {
		req.EventID = uuid.NewString()
	}

	now := apptime.Now()
	baselineRequestID := int64(0)
	status := model.EmbedRefreshEventScheduled
	resolutionPending := req.ProcessID == ""
	if resolutionPending {
		if req.WorkflowID == "" {
			return nil, ErrInvalidEmbedRefreshContext
		}
		adapter, err := s.auditSvc.getOAAdapter(ctx, tenantID)
		if err != nil {
			return nil, err
		}
		resolver, ok := adapter.(oa.ProcessRequestWatermarkResolver)
		if !ok {
			return nil, fmt.Errorf("当前 OA 适配器不支持 requestid 高水位解析")
		}
		baselineRequestID, err = resolver.CaptureProcessRequestHighWatermark(ctx)
		if err != nil {
			return nil, err
		}
		status = model.EmbedRefreshEventPending
	}
	nextAttemptAt := now.Add(embedRefreshInitialDelay)
	event, created, err := s.eventRepo.CreateOrGet(ctx, &model.EmbedRefreshEvent{
		TenantID:          tenantID,
		UserID:            userID,
		EventID:           req.EventID,
		Action:            req.Action,
		ProcessID:         req.ProcessID,
		WorkflowID:        req.WorkflowID,
		OABelongUserID:    req.OABelongUserID,
		OACurrentUserID:   req.OACurrentUserID,
		OccurredAtMS:      req.OccurredAtMS,
		BaselineRequestID: baselineRequestID,
		Status:            status,
		NextAttemptAt:     &nextAttemptAt,
		ReceivedAt:        now,
	})
	if err != nil {
		return nil, err
	}
	if !created {
		return &EmbedRefreshEventResponse{
			ProcessID:         event.ProcessID,
			Action:            event.Action,
			EventID:           event.EventID,
			ScheduledModules:  []string{embedRefreshModuleAudit, embedRefreshModuleSummary},
			ResolutionPending: event.Status == model.EmbedRefreshEventPending,
		}, nil
	}

	modules := []string{embedRefreshModuleAudit, embedRefreshModuleSummary}
	if resolutionPending {
		payload := embedRefreshPayload{
			TenantID:          tenantID,
			UserID:            userID,
			Module:            embedRefreshModuleResolve,
			Action:            req.Action,
			EventID:           req.EventID,
			Generation:        uuid.NewString(),
			FirstReceived:     now,
			WorkflowID:        req.WorkflowID,
			OABelongUserID:    req.OABelongUserID,
			OACurrentUserID:   req.OACurrentUserID,
			BaselineRequestID: baselineRequestID,
			OccurredAtMS:      req.OccurredAtMS,
		}
		if _, err := s.schedule(ctx, payload, embedRefreshInitialDelay, false); err != nil {
			return nil, err
		}
	} else {
		for _, module := range modules {
			payload := embedRefreshPayload{
				TenantID:      tenantID,
				UserID:        userID,
				ProcessID:     req.ProcessID,
				Module:        module,
				Action:        req.Action,
				EventID:       req.EventID,
				Generation:    uuid.NewString(),
				FirstReceived: now,
				OccurredAtMS:  req.OccurredAtMS,
			}
			if _, err := s.schedule(ctx, payload, embedRefreshInitialDelay, false); err != nil {
				return nil, err
			}
		}
	}
	if s.logger != nil {
		s.logger.Info("OA 嵌入刷新事件已接收",
			zap.String("tenantID", tenantID.String()),
			zap.String("processID", req.ProcessID),
			zap.String("action", req.Action),
			zap.String("eventID", req.EventID),
			zap.String("workflowID", req.WorkflowID),
			zap.Strings("scheduledModules", modules),
			zap.Bool("resolutionPending", resolutionPending),
			zap.Int64("baselineRequestID", baselineRequestID),
			zap.Int64("clientDelayMs", embedRefreshClientDelay(req.OccurredAtMS, now)),
			zap.Int64("initialDelayMs", embedRefreshInitialDelay.Milliseconds()))
	}
	return &EmbedRefreshEventResponse{
		ProcessID:         req.ProcessID,
		Action:            req.Action,
		EventID:           req.EventID,
		ScheduledModules:  modules,
		ResolutionPending: resolutionPending,
	}, nil
}

// Start 启动延迟事件消费者，并从数据库恢复流程级精确定时任务。
func (s *EmbedRefreshService) Start(ctx context.Context) {
	if s == nil || s.rdb == nil || s.scheduleRepo == nil {
		return
	}
	if removed, err := s.purgeObsoletePayloads(ctx); err != nil {
		if s.logger != nil {
			s.logger.Warn("清理旧版嵌入刷新队列失败", zap.Error(err))
		}
	} else if removed > 0 && s.logger != nil {
		s.logger.Info("已清理旧版嵌入刷新队列", zap.Int64("count", removed))
	}
	if err := s.reconcileSchedules(ctx); err != nil && s.logger != nil {
		s.logger.Warn("重建 OA 嵌入刷新调度记录失败", zap.Error(err))
	}
	if err := s.restoreSchedules(ctx); err != nil && s.logger != nil {
		s.logger.Warn("恢复 OA 嵌入刷新定时任务失败", zap.Error(err))
	}
	if err := s.restorePendingEvents(ctx); err != nil && s.logger != nil {
		s.logger.Warn("恢复 OA requestid 待解析事件失败", zap.Error(err))
	}
	s.scheduleMu.Lock()
	scheduleCount := len(s.scheduleEntries)
	s.scheduleMu.Unlock()
	s.scheduleCron.Start()
	go s.runDueLoop(ctx)
	go s.runScheduleChangeSubscriber(ctx)
	go func() {
		<-ctx.Done()
		s.scheduleCron.Stop()
	}()
	if s.logger != nil {
		s.logger.Info("OA 嵌入后台刷新协调器已启动",
			zap.Int("schedules", scheduleCount))
	}
}

// purgeObsoletePayloads 清理旧动作及缺少 config_id 的定时候选；启用中的 Cron 会重新生成。
func (s *EmbedRefreshService) purgeObsoletePayloads(ctx context.Context) (int64, error) {
	members, err := s.rdb.ZRange(ctx, embedRefreshDueKey, 0, -1).Result()
	if err != nil {
		return 0, err
	}
	var removed int64
	for _, member := range members {
		raw, getErr := s.rdb.Get(ctx, embedRefreshPayloadKey(member)).Result()
		if getErr != nil {
			continue
		}
		var payload embedRefreshPayload
		if json.Unmarshal([]byte(raw), &payload) != nil {
			continue
		}
		shouldRemove := isObsoleteEmbedRefreshAction(payload.Action) ||
			(payload.Action == model.SummaryTriggerDetailScheduled && payload.ConfigID == uuid.Nil)
		if !shouldRemove {
			continue
		}
		pipe := s.rdb.TxPipeline()
		pipe.ZRem(ctx, embedRefreshDueKey, member)
		pipe.Del(ctx, embedRefreshPayloadKey(member), embedRefreshGenerationKey(member))
		if _, execErr := pipe.Exec(ctx); execErr != nil {
			return removed, execErr
		}
		removed++
	}
	return removed, nil
}

func (s *EmbedRefreshService) runDueLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.processDue(ctx)
		}
	}
}

func (s *EmbedRefreshService) processDue(ctx context.Context) {
	members, err := s.rdb.ZRangeByScore(ctx, embedRefreshDueKey, &redis.ZRangeBy{
		Min:    "-inf",
		Max:    fmt.Sprintf("%d", apptime.Now().UnixMilli()),
		Offset: 0,
		Count:  20,
	}).Result()
	if err != nil {
		return
	}
	for _, member := range members {
		raw, err := claimEmbedRefreshScript.Run(
			ctx,
			s.rdb,
			[]string{embedRefreshDueKey, embedRefreshPayloadKey(member)},
			member,
			apptime.Now().UnixMilli(),
		).Text()
		if err != nil || raw == "" {
			continue
		}
		var payload embedRefreshPayload
		if json.Unmarshal([]byte(raw), &payload) != nil {
			continue
		}
		checkAttempt := payload.Attempt
		outcome, processErr := s.checkAndTrigger(ctx, payload)
		retryScheduled := false
		retryDelay := time.Duration(0)
		retryErr := error(nil)
		switch outcome.Result {
		case embedRefreshRunning:
			if shouldRetryEmbedEvent(payload.Action) &&
				apptime.Now().Sub(payload.FirstReceived) < embedRefreshRunningMaxAge {
				payload.Attempt++
				retryDelay = embedRefreshRunningDelay
				retryScheduled, retryErr = s.schedule(ctx, payload, retryDelay, true)
			}
		case embedRefreshRetry:
			if shouldRetryEmbedEvent(payload.Action) && payload.Attempt < 2 {
				payload.Attempt++
				retryDelay = time.Duration(2*payload.Attempt+1) * time.Second
				retryScheduled, retryErr = s.schedule(ctx, payload, retryDelay, true)
			}
		}
		s.logCheckOutcome(payload, checkAttempt, outcome, processErr, retryScheduled, retryDelay, retryErr)
	}
}

func (s *EmbedRefreshService) checkAndTrigger(
	ctx context.Context,
	payload embedRefreshPayload,
) (embedRefreshCheckOutcome, error) {
	if isObsoleteEmbedRefreshAction(payload.Action) {
		// 升级时未被启动清理捕获的旧动作不再执行。
		return embedRefreshCheckOutcome{Result: embedRefreshDone, Reason: "obsolete_action"}, nil
	}
	if payload.Module == embedRefreshModuleResolve {
		return s.resolveProcessRequest(ctx, payload)
	}
	if payload.Action == model.SummaryTriggerDetailScheduled && payload.ConfigID == uuid.Nil {
		return embedRefreshCheckOutcome{Result: embedRefreshDone, Reason: "schedule_config_missing"}, nil
	}
	if payload.Action == model.SummaryTriggerDetailScheduled && payload.ConfigID != uuid.Nil {
		schedule, err := s.scheduleRepo.GetByConfig(ctx, payload.Module, payload.ConfigID)
		if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && !schedule.IsActive) {
			return embedRefreshCheckOutcome{Result: embedRefreshDone, Reason: "schedule_disabled"}, nil
		}
		if err != nil {
			return embedRefreshCheckOutcome{Result: embedRefreshRetry, Reason: "schedule_lookup_failed"}, err
		}
	}
	gc := buildWorkerContext(ctx, payload.TenantID, payload.UserID, "embed_scheduler")
	switch payload.Module {
	case embedRefreshModuleAudit:
		embedCtx, err := s.auditSvc.GetEmbedContext(gc, payload.ProcessID)
		if err != nil {
			return embedRefreshCheckOutcome{Result: embedRefreshRetry, Reason: "context_failed"}, err
		}
		if embedCtx.RunningJobID != "" {
			return embedRefreshCheckOutcome{
				Result: embedRefreshRunning,
				Reason: "job_running",
				JobID:  embedCtx.RunningJobID,
			}, nil
		}
		if !embedCtx.Supported {
			if embedCtx.Reason == "not_found_in_oa" {
				return embedRefreshCheckOutcome{Result: embedRefreshRetry, Reason: "not_found_in_oa"}, nil
			}
			return embedRefreshCheckOutcome{
				Result: embedRefreshDone,
				Reason: "unsupported_" + strings.TrimSpace(embedCtx.Reason),
			}, nil
		}
		if !embedCtx.ShouldAutoAudit {
			reason := "unchanged"
			if embedCtx.AutoRetryBlocked {
				reason = "auto_retry_blocked"
			} else if isEmbedOperationAction(payload.Action) && payload.Attempt < 2 {
				return embedRefreshCheckOutcome{Result: embedRefreshRetry, Reason: "unchanged_waiting_commit"}, nil
			}
			return embedRefreshCheckOutcome{Result: embedRefreshDone, Reason: reason}, nil
		}
		executeResp, err := s.auditSvc.ExecuteEmbed(gc, &EmbedExecuteRequest{
			ProcessID:        payload.ProcessID,
			TriggerSource:    model.AuditTriggerEmbedAuto,
			TriggerDetail:    payload.Action,
			ScheduleConfigID: nullableUUID(payload.ConfigID),
		})
		if err != nil {
			return embedRefreshCheckOutcome{Result: embedRefreshRetry, Reason: "execute_failed"}, err
		}
		jobID := ""
		if executeResp != nil {
			jobID = executeResp.ID
		}
		return embedRefreshCheckOutcome{Result: embedRefreshDone, Reason: "triggered", JobID: jobID}, nil

	case embedRefreshModuleSummary:
		embedCtx, err := s.summarySvc.GetEmbedContext(gc, payload.ProcessID)
		if err != nil {
			return embedRefreshCheckOutcome{Result: embedRefreshRetry, Reason: "context_failed"}, err
		}
		if embedCtx.RunningJobID != "" {
			return embedRefreshCheckOutcome{
				Result: embedRefreshRunning,
				Reason: "job_running",
				JobID:  embedCtx.RunningJobID,
			}, nil
		}
		if !embedCtx.Supported {
			if embedCtx.Reason == "not_found_in_oa" {
				return embedRefreshCheckOutcome{Result: embedRefreshRetry, Reason: "not_found_in_oa"}, nil
			}
			return embedRefreshCheckOutcome{
				Result: embedRefreshDone,
				Reason: "unsupported_" + strings.TrimSpace(embedCtx.Reason),
			}, nil
		}
		if !embedCtx.ShouldAutoSummary {
			reason := "unchanged"
			if embedCtx.AutoRetryBlocked {
				reason = "auto_retry_blocked"
			} else if isEmbedOperationAction(payload.Action) && payload.Attempt < 2 {
				return embedRefreshCheckOutcome{Result: embedRefreshRetry, Reason: "unchanged_waiting_commit"}, nil
			}
			return embedRefreshCheckOutcome{Result: embedRefreshDone, Reason: reason}, nil
		}
		executeResp, err := s.summarySvc.ExecuteEmbed(gc, &SummaryExecuteRequest{
			ProcessID:        payload.ProcessID,
			TriggerSource:    model.SummaryTriggerEmbedAuto,
			TriggerDetail:    payload.Action,
			ScheduleConfigID: nullableUUID(payload.ConfigID),
		})
		if err != nil {
			return embedRefreshCheckOutcome{Result: embedRefreshRetry, Reason: "execute_failed"}, err
		}
		jobID := ""
		if executeResp != nil {
			jobID = executeResp.ID
		}
		return embedRefreshCheckOutcome{Result: embedRefreshDone, Reason: "triggered", JobID: jobID}, nil
	default:
		return embedRefreshCheckOutcome{Result: embedRefreshDone, Reason: "unknown_module"}, nil
	}
}

// resolveProcessRequest 按“操作前高水位 + 流程定义”解析首次新建流程的 requestid。
// OA 当前操作人和归属用户不等同于流程创建人，只在出现多个候选时辅助消歧。
func (s *EmbedRefreshService) resolveProcessRequest(
	ctx context.Context,
	payload embedRefreshPayload,
) (embedRefreshCheckOutcome, error) {
	event, err := s.eventRepo.GetByEventID(ctx, payload.TenantID, payload.EventID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return embedRefreshCheckOutcome{Result: embedRefreshDone, Reason: "event_not_found"}, nil
		}
		return embedRefreshCheckOutcome{Result: embedRefreshRetry, Reason: "event_lookup_failed"}, err
	}
	if event.Status != model.EmbedRefreshEventPending {
		return embedRefreshCheckOutcome{Result: embedRefreshDone, Reason: "event_" + event.Status}, nil
	}
	adapter, err := s.auditSvc.getOAAdapter(ctx, payload.TenantID)
	if err != nil {
		if s.updateResolutionFailure(ctx, event, payload, err) {
			return embedRefreshCheckOutcome{Result: embedRefreshRetry, Reason: "oa_adapter_failed"}, err
		}
		return embedRefreshCheckOutcome{Result: embedRefreshDone, Reason: "oa_adapter_failed_final"}, err
	}
	resolver, ok := adapter.(oa.ProcessRequestWatermarkResolver)
	if !ok {
		err = fmt.Errorf("当前 OA 适配器不支持 requestid 高水位解析")
		_ = s.eventRepo.UpdateResolution(ctx, event.ID, model.EmbedRefreshEventFailed, "", payload.Attempt, nil, err.Error(), nil)
		return embedRefreshCheckOutcome{Result: embedRefreshDone, Reason: "resolver_unsupported"}, err
	}
	candidates, err := resolver.FindCreatedProcessRequestsAfter(
		ctx,
		payload.WorkflowID,
		payload.BaselineRequestID,
		20,
	)
	if err != nil {
		if s.updateResolutionFailure(ctx, event, payload, err) {
			return embedRefreshCheckOutcome{Result: embedRefreshRetry, Reason: "resolve_query_failed"}, err
		}
		return embedRefreshCheckOutcome{Result: embedRefreshDone, Reason: "resolve_query_failed_final"}, err
	}
	if len(candidates) == 0 {
		if payload.Attempt >= 2 {
			_ = s.eventRepo.UpdateResolution(ctx, event.ID, model.EmbedRefreshEventExpired, "", payload.Attempt, nil, "未发现匹配的新建流程", nil)
			return embedRefreshCheckOutcome{Result: embedRefreshDone, Reason: "requestid_expired"}, nil
		} else {
			nextAttempt := payload.Attempt + 1
			nextAt := apptime.Now().Add(time.Duration(2*nextAttempt+1) * time.Second)
			_ = s.eventRepo.UpdateResolution(ctx, event.ID, model.EmbedRefreshEventPending, "", nextAttempt, &nextAt, "", nil)
		}
		return embedRefreshCheckOutcome{Result: embedRefreshRetry, Reason: "requestid_not_ready"}, nil
	}
	candidate, peopleMatchCount := selectResolvedProcessCandidate(
		candidates,
		payload.OABelongUserID,
		payload.OACurrentUserID,
	)
	if candidate == nil {
		lastError := fmt.Sprintf(
			"高水位后出现 %d 个同流程候选，人员辅助匹配 %d 个，未自动猜测",
			len(candidates),
			peopleMatchCount,
		)
		_ = s.eventRepo.UpdateResolution(ctx, event.ID, model.EmbedRefreshEventAmbiguous, "", payload.Attempt, nil, lastError, nil)
		return embedRefreshCheckOutcome{Result: embedRefreshDone, Reason: "requestid_ambiguous"}, nil
	}

	processID := strings.TrimSpace(candidate.ProcessID)
	for _, module := range []string{embedRefreshModuleAudit, embedRefreshModuleSummary} {
		child := embedRefreshPayload{
			TenantID:      payload.TenantID,
			UserID:        payload.UserID,
			ProcessID:     processID,
			Module:        module,
			Action:        payload.Action,
			EventID:       payload.EventID,
			Generation:    uuid.NewString(),
			FirstReceived: payload.FirstReceived,
			OccurredAtMS:  payload.OccurredAtMS,
		}
		if _, scheduleErr := s.schedule(ctx, child, 0, false); scheduleErr != nil {
			if s.updateResolutionFailure(ctx, event, payload, scheduleErr) {
				return embedRefreshCheckOutcome{Result: embedRefreshRetry, Reason: "child_schedule_failed"}, scheduleErr
			}
			return embedRefreshCheckOutcome{Result: embedRefreshDone, Reason: "child_schedule_failed_final"}, scheduleErr
		}
	}
	resolvedAt := apptime.Now()
	if err := s.eventRepo.UpdateResolution(
		ctx,
		event.ID,
		model.EmbedRefreshEventScheduled,
		processID,
		payload.Attempt,
		nil,
		"",
		&resolvedAt,
	); err != nil {
		return embedRefreshCheckOutcome{Result: embedRefreshDone, Reason: "requestid_resolved"}, err
	}
	return embedRefreshCheckOutcome{
		Result:            embedRefreshDone,
		Reason:            "requestid_resolved",
		ResolvedProcessID: processID,
	}, nil
}

// selectResolvedProcessCandidate 先按流程定义得到候选；仅在多候选时使用 OA 人员标识辅助消歧。
func selectResolvedProcessCandidate(
	candidates []oa.ProcessRequestCandidate,
	peopleIDs ...string,
) (*oa.ProcessRequestCandidate, int) {
	if len(candidates) == 1 {
		return &candidates[0], 0
	}
	people := make(map[string]struct{}, len(peopleIDs))
	for _, raw := range peopleIDs {
		if value := strings.TrimSpace(raw); value != "" {
			people[value] = struct{}{}
		}
	}
	matches := make([]int, 0, len(candidates))
	for index := range candidates {
		creatorID := strings.TrimSpace(candidates[index].CreatorID)
		if _, ok := people[creatorID]; ok && creatorID != "" {
			matches = append(matches, index)
		}
	}
	if len(matches) == 1 {
		return &candidates[matches[0]], 1
	}
	return nil, len(matches)
}

func (s *EmbedRefreshService) updateResolutionFailure(
	ctx context.Context,
	event *model.EmbedRefreshEvent,
	payload embedRefreshPayload,
	resolveErr error,
) bool {
	if s.eventRepo == nil || event == nil {
		return false
	}
	status := model.EmbedRefreshEventPending
	attempt := payload.Attempt + 1
	var nextAt *time.Time
	if payload.Attempt >= 2 {
		status = model.EmbedRefreshEventFailed
		attempt = payload.Attempt
	} else {
		next := apptime.Now().Add(time.Duration(2*attempt+1) * time.Second)
		nextAt = &next
	}
	_ = s.eventRepo.UpdateResolution(ctx, event.ID, status, "", attempt, nextAt, resolveErr.Error(), nil)
	return status == model.EmbedRefreshEventPending
}

// restorePendingEvents 从数据库恢复因服务重启而尚未完成的首次新建流程解析事件。
func (s *EmbedRefreshService) restorePendingEvents(ctx context.Context) error {
	if s.eventRepo == nil {
		return nil
	}
	events, err := s.eventRepo.ListPending(ctx, 1000)
	if err != nil {
		return err
	}
	for _, event := range events {
		delay := time.Duration(0)
		if event.NextAttemptAt != nil && event.NextAttemptAt.After(apptime.Now()) {
			delay = event.NextAttemptAt.Sub(apptime.Now())
		}
		payload := embedRefreshPayload{
			TenantID:          event.TenantID,
			UserID:            event.UserID,
			Module:            embedRefreshModuleResolve,
			Action:            event.Action,
			EventID:           event.EventID,
			Generation:        uuid.NewString(),
			Attempt:           event.Attempt,
			FirstReceived:     event.ReceivedAt,
			WorkflowID:        event.WorkflowID,
			OABelongUserID:    event.OABelongUserID,
			OACurrentUserID:   event.OACurrentUserID,
			BaselineRequestID: event.BaselineRequestID,
			OccurredAtMS:      event.OccurredAtMS,
		}
		if _, err := s.schedule(ctx, payload, delay, false); err != nil {
			return err
		}
	}
	return nil
}

// logCheckOutcome 记录保存/提交检查的完整结论；定时扫描降为 DEBUG，避免生产 INFO 被候选检查淹没。
func (s *EmbedRefreshService) logCheckOutcome(
	payload embedRefreshPayload,
	checkAttempt int,
	outcome embedRefreshCheckOutcome,
	processErr error,
	retryScheduled bool,
	retryDelay time.Duration,
	retryErr error,
) {
	if s == nil || s.logger == nil {
		return
	}
	fields := []zap.Field{
		zap.String("tenantID", payload.TenantID.String()),
		zap.String("processID", payload.ProcessID),
		zap.String("module", payload.Module),
		zap.String("action", payload.Action),
		zap.String("eventID", payload.EventID),
		zap.Int("attempt", checkAttempt),
		zap.String("result", embedRefreshResultName(outcome.Result)),
		zap.String("reason", outcome.Reason),
		zap.Bool("retryScheduled", retryScheduled),
	}
	if outcome.JobID != "" {
		fields = append(fields, zap.String("jobID", outcome.JobID))
	}
	if outcome.ResolvedProcessID != "" {
		fields = append(fields, zap.String("resolvedProcessID", outcome.ResolvedProcessID))
	}
	if payload.OccurredAtMS > 0 {
		fields = append(fields, zap.Int64(
			"clientDelayMs",
			embedRefreshClientDelay(payload.OccurredAtMS, payload.FirstReceived),
		))
	}
	if retryDelay > 0 {
		fields = append(fields,
			zap.Int("nextAttempt", payload.Attempt),
			zap.Int64("retryDelayMs", retryDelay.Milliseconds()))
	}
	if processErr != nil {
		fields = append(fields, zap.Error(processErr))
	}
	if retryErr != nil {
		fields = append(fields, zap.NamedError("retryError", retryErr))
	}
	if processErr != nil || retryErr != nil {
		s.logger.Warn("OA 嵌入刷新检查失败", fields...)
		return
	}
	if payload.Action == model.SummaryTriggerDetailSaveRequested ||
		payload.Action == model.SummaryTriggerDetailSubmitRequested {
		s.logger.Info("OA 嵌入刷新检查完成", fields...)
		return
	}
	s.logger.Debug("OA 嵌入刷新检查完成", fields...)
}

func embedRefreshResultName(result embedRefreshResult) string {
	switch result {
	case embedRefreshRetry:
		return "retry"
	case embedRefreshRunning:
		return "running"
	default:
		return "done"
	}
}

// embedRefreshClientDelay 返回浏览器点击到服务端接收事件的耗时，负值按客户端时钟偏差归零。
func embedRefreshClientDelay(occurredAtMS int64, receivedAt time.Time) int64 {
	if occurredAtMS <= 0 || receivedAt.IsZero() {
		return 0
	}
	delay := receivedAt.UnixMilli() - occurredAtMS
	if delay < 0 {
		return 0
	}
	return delay
}

func (s *EmbedRefreshService) schedule(
	ctx context.Context,
	payload embedRefreshPayload,
	delay time.Duration,
	onlyIfCurrent bool,
) (bool, error) {
	member := embedRefreshPayloadMember(payload)
	raw, err := json.Marshal(payload)
	if err != nil {
		return false, err
	}
	keys := []string{
		embedRefreshDueKey,
		embedRefreshPayloadKey(member),
		embedRefreshGenerationKey(member),
	}
	args := []interface{}{
		string(raw),
		payload.Generation,
		apptime.Now().Add(delay).UnixMilli(),
		int(embedRefreshPayloadTTL.Seconds()),
		member,
	}
	if onlyIfCurrent {
		result, runErr := retryEmbedRefreshScript.Run(ctx, s.rdb, keys, args...).Int64()
		return result == 1, runErr
	}
	result, runErr := scheduleEmbedRefreshScript.Run(ctx, s.rdb, keys, args...).Int64()
	return result == 1, runErr
}

// EmbedRefreshScheduleManager 由流程配置服务调用，负责即时同步持久化调度。
type EmbedRefreshScheduleManager interface {
	SyncAuditConfig(ctx context.Context, cfg *model.ProcessAuditConfig) error
	SyncSummaryConfig(ctx context.Context, cfg *model.ProcessSummaryConfig) error
	DeleteConfig(ctx context.Context, module string, configID uuid.UUID) error
}

type embedRefreshScheduleChange struct {
	Module   string    `json:"module"`
	ConfigID uuid.UUID `json:"config_id"`
}

// SyncAuditConfig 保存审核流程配置对应的调度记录，并立即更新当前实例的 Cron。
func (s *EmbedRefreshService) SyncAuditConfig(ctx context.Context, cfg *model.ProcessAuditConfig) error {
	embedCfg := parseEmbedConfig(cfg.EmbedConfig)
	schedule := buildEmbedRefreshSchedule(
		embedRefreshModuleAudit,
		cfg.ID,
		cfg.TenantID,
		cfg.ProcessType,
		cfg.Status == "active" && cfg.EmbedEnabled && embedCfg.ScheduledRefreshEnabled,
		embedCfg.ScheduledLookbackDays,
		embedCfg.ScheduledIntervalMinutes,
	)
	return s.persistAndActivateSchedule(ctx, schedule, true)
}

// SyncSummaryConfig 保存总结流程配置对应的调度记录，并立即更新当前实例的 Cron。
func (s *EmbedRefreshService) SyncSummaryConfig(ctx context.Context, cfg *model.ProcessSummaryConfig) error {
	embedCfg := parseSummaryEmbedConfig(cfg.EmbedConfig)
	schedule := buildEmbedRefreshSchedule(
		embedRefreshModuleSummary,
		cfg.ID,
		cfg.TenantID,
		cfg.ProcessType,
		cfg.Status == "active" && cfg.EmbedEnabled && embedCfg.ScheduledRefreshEnabled,
		embedCfg.ScheduledLookbackDays,
		embedCfg.ScheduledIntervalMinutes,
	)
	return s.persistAndActivateSchedule(ctx, schedule, true)
}

// DeleteConfig 删除流程配置对应的持久化调度，并通知其他服务实例移除 Cron。
func (s *EmbedRefreshService) DeleteConfig(ctx context.Context, module string, configID uuid.UUID) error {
	existing, err := s.scheduleRepo.GetByConfig(ctx, module, configID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err := s.scheduleRepo.DeleteByConfig(ctx, module, configID); err != nil {
		return err
	}
	if existing != nil {
		if err := s.cancelScheduledWork(ctx, existing); err != nil {
			return err
		}
		s.removeSchedule(existing.ID, module, configID)
	} else {
		s.removeSchedule(uuid.Nil, module, configID)
	}
	s.publishScheduleChange(ctx, module, configID)
	return nil
}

func buildEmbedRefreshSchedule(
	module string,
	configID, tenantID uuid.UUID,
	processType string,
	active bool,
	lookbackDays, intervalMinutes int,
) *model.EmbedRefreshSchedule {
	normalizeScheduledRefreshConfig(&lookbackDays, &intervalMinutes)
	return &model.EmbedRefreshSchedule{
		ID:              uuid.New(),
		TenantID:        tenantID,
		Module:          module,
		ConfigID:        configID,
		ProcessType:     processType,
		IsActive:        active,
		LookbackDays:    lookbackDays,
		IntervalMinutes: intervalMinutes,
		CronExpression:  fmt.Sprintf("0 */%d * * * *", intervalMinutes),
	}
}

func (s *EmbedRefreshService) persistAndActivateSchedule(
	ctx context.Context,
	schedule *model.EmbedRefreshSchedule,
	notify bool,
) error {
	if err := s.scheduleRepo.Upsert(ctx, schedule); err != nil {
		return err
	}
	if err := s.addOrUpdateSchedule(schedule); err != nil {
		return err
	}
	if notify {
		s.publishScheduleChange(ctx, schedule.Module, schedule.ConfigID)
	}
	return nil
}

// reconcileSchedules 仅在启动时执行一次，用流程配置修复缺失或过期的调度记录。
func (s *EmbedRefreshService) reconcileSchedules(ctx context.Context) error {
	auditConfigs, err := s.auditRepo.ListAllTenants(ctx)
	if err != nil {
		return err
	}
	for i := range auditConfigs {
		cfg := &auditConfigs[i]
		embedCfg := parseEmbedConfig(cfg.EmbedConfig)
		schedule := buildEmbedRefreshSchedule(
			embedRefreshModuleAudit,
			cfg.ID,
			cfg.TenantID,
			cfg.ProcessType,
			cfg.Status == "active" && cfg.EmbedEnabled && embedCfg.ScheduledRefreshEnabled,
			embedCfg.ScheduledLookbackDays,
			embedCfg.ScheduledIntervalMinutes,
		)
		if err := s.scheduleRepo.Upsert(ctx, schedule); err != nil {
			return err
		}
		if !schedule.IsActive {
			if err := s.cancelScheduledWork(ctx, schedule); err != nil {
				return err
			}
		}
	}

	summaryConfigs, err := s.summaryRepo.ListAllTenants(ctx)
	if err != nil {
		return err
	}
	for i := range summaryConfigs {
		cfg := &summaryConfigs[i]
		embedCfg := parseSummaryEmbedConfig(cfg.EmbedConfig)
		schedule := buildEmbedRefreshSchedule(
			embedRefreshModuleSummary,
			cfg.ID,
			cfg.TenantID,
			cfg.ProcessType,
			cfg.Status == "active" && cfg.EmbedEnabled && embedCfg.ScheduledRefreshEnabled,
			embedCfg.ScheduledLookbackDays,
			embedCfg.ScheduledIntervalMinutes,
		)
		if err := s.scheduleRepo.Upsert(ctx, schedule); err != nil {
			return err
		}
		if !schedule.IsActive {
			if err := s.cancelScheduledWork(ctx, schedule); err != nil {
				return err
			}
		}
	}
	return s.scheduleRepo.DeleteOrphans(ctx)
}

func (s *EmbedRefreshService) restoreSchedules(ctx context.Context) error {
	schedules, err := s.scheduleRepo.ListActive(ctx)
	if err != nil {
		return err
	}
	for i := range schedules {
		if err := s.addOrUpdateSchedule(&schedules[i]); err != nil && s.logger != nil {
			s.logger.Warn("恢复 OA 嵌入刷新定时任务失败",
				zap.String("scheduleID", schedules[i].ID.String()),
				zap.Error(err))
		}
	}
	return nil
}

func (s *EmbedRefreshService) addOrUpdateSchedule(schedule *model.EmbedRefreshSchedule) error {
	s.scheduleMu.Lock()
	if old, ok := s.scheduleEntries[schedule.ID]; ok {
		s.scheduleCron.Remove(old)
		delete(s.scheduleEntries, schedule.ID)
	}
	delete(s.scheduleExprs, schedule.ID)
	configKey := embedRefreshScheduleConfigKey(schedule.Module, schedule.ConfigID)
	s.scheduleConfigs[configKey] = schedule.ID

	if !schedule.IsActive {
		s.scheduleMu.Unlock()
		_ = s.scheduleRepo.UpdateNextRun(context.Background(), schedule.ID, nil)
		return s.cancelScheduledWork(context.Background(), schedule)
	}

	scheduleID := schedule.ID
	entryID, err := s.scheduleCron.AddFunc(schedule.CronExpression, func() {
		s.executeScheduledScan(context.Background(), scheduleID)
	})
	if err != nil {
		s.scheduleMu.Unlock()
		return err
	}
	s.scheduleEntries[schedule.ID] = entryID
	s.scheduleExprs[schedule.ID] = schedule.CronExpression
	s.scheduleMu.Unlock()

	nextRun := ParseNextRun(schedule.CronExpression)
	schedule.NextRunAt = nextRun
	return s.scheduleRepo.UpdateNextRun(context.Background(), schedule.ID, nextRun)
}

func (s *EmbedRefreshService) removeSchedule(scheduleID uuid.UUID, module string, configID uuid.UUID) {
	s.scheduleMu.Lock()
	defer s.scheduleMu.Unlock()
	if scheduleID == uuid.Nil {
		scheduleID = s.scheduleConfigs[embedRefreshScheduleConfigKey(module, configID)]
	}
	if entryID, ok := s.scheduleEntries[scheduleID]; ok {
		s.scheduleCron.Remove(entryID)
	}
	delete(s.scheduleEntries, scheduleID)
	delete(s.scheduleExprs, scheduleID)
	if module != "" {
		delete(s.scheduleConfigs, embedRefreshScheduleConfigKey(module, configID))
		return
	}
	for key, id := range s.scheduleConfigs {
		if id == scheduleID {
			delete(s.scheduleConfigs, key)
		}
	}
}

// cancelScheduledWork 清除指定流程配置尚未触发的 Redis 检查，并取消已入库但未领取的总结任务。
func (s *EmbedRefreshService) cancelScheduledWork(
	ctx context.Context,
	schedule *model.EmbedRefreshSchedule,
) error {
	if schedule == nil {
		return nil
	}
	removedDue := int64(0)
	members, err := s.rdb.ZRange(ctx, embedRefreshDueKey, 0, -1).Result()
	if err != nil {
		return err
	}
	for _, member := range members {
		raw, getErr := s.rdb.Get(ctx, embedRefreshPayloadKey(member)).Result()
		if getErr != nil {
			continue
		}
		var payload embedRefreshPayload
		if json.Unmarshal([]byte(raw), &payload) != nil ||
			payload.Action != model.SummaryTriggerDetailScheduled ||
			payload.ConfigID != schedule.ConfigID ||
			payload.Module != schedule.Module {
			continue
		}
		pipe := s.rdb.TxPipeline()
		pipe.ZRem(ctx, embedRefreshDueKey, member)
		pipe.Del(ctx, embedRefreshPayloadKey(member), embedRefreshGenerationKey(member))
		if _, execErr := pipe.Exec(ctx); execErr != nil {
			return execErr
		}
		removedDue++
	}

	cancelledLogs := int64(0)
	if schedule.Module == embedRefreshModuleSummary && s.summarySvc != nil {
		var err error
		cancelledLogs, err = s.summarySvc.logRepo.CancelPendingScheduled(
			schedule.TenantID,
			schedule.ConfigID,
			"对应流程的定时检查已关闭，任务已取消",
		)
		if err != nil {
			return err
		}
	} else if schedule.Module == embedRefreshModuleAudit && s.auditSvc != nil {
		var err error
		cancelledLogs, err = s.auditSvc.auditLogRepo.CancelPendingScheduled(
			schedule.TenantID,
			schedule.ConfigID,
			"对应流程的定时检查已关闭，任务已取消",
		)
		if err != nil {
			return err
		}
	}
	if (removedDue > 0 || cancelledLogs > 0) && s.logger != nil {
		s.logger.Info("已清理关闭配置的定时检查",
			zap.String("tenantID", schedule.TenantID.String()),
			zap.String("module", schedule.Module),
			zap.String("configID", schedule.ConfigID.String()),
			zap.Int64("redisDueCount", removedDue),
			zap.Int64("cancelledJobCount", cancelledLogs))
	}
	return nil
}

func (s *EmbedRefreshService) executeScheduledScan(ctx context.Context, scheduleID uuid.UUID) {
	schedule, err := s.scheduleRepo.GetByID(ctx, scheduleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.removeSchedule(scheduleID, "", uuid.Nil)
		}
		return
	}
	if !schedule.IsActive {
		s.removeSchedule(schedule.ID, schedule.Module, schedule.ConfigID)
		return
	}

	s.scheduleMu.Lock()
	registeredExpr := s.scheduleExprs[schedule.ID]
	s.scheduleMu.Unlock()
	if registeredExpr != schedule.CronExpression {
		_ = s.addOrUpdateSchedule(schedule)
		return
	}

	release, acquired, lockErr := s.acquireScheduleRunLock(ctx, schedule.ID)
	if lockErr != nil || !acquired {
		return
	}
	defer release()

	startedAt := apptime.Now()
	sourceActive, sourceErr := s.isScheduleSourceActive(ctx, schedule)
	if sourceErr != nil {
		s.finishScheduledScan(ctx, schedule, startedAt, sourceErr)
		return
	}
	if !sourceActive {
		schedule.IsActive = false
		if err := s.persistAndActivateSchedule(ctx, schedule, true); err != nil {
			s.finishScheduledScan(ctx, schedule, startedAt, fmt.Errorf("停用失配的流程级定时检查失败: %w", err))
			return
		}
		if s.logger != nil {
			s.logger.Warn("流程级定时检查与源配置不一致，已自动停用",
				zap.String("scheduleID", schedule.ID.String()),
				zap.String("tenantID", schedule.TenantID.String()),
				zap.String("module", schedule.Module),
				zap.String("configID", schedule.ConfigID.String()),
				zap.String("processType", schedule.ProcessType))
		}
		return
	}

	s.finishScheduledScan(ctx, schedule, startedAt, s.scanSchedule(ctx, schedule))
}

func (s *EmbedRefreshService) finishScheduledScan(
	ctx context.Context,
	schedule *model.EmbedRefreshSchedule,
	startedAt time.Time,
	runErr error,
) {
	status := "success"
	lastError := ""
	if runErr != nil {
		status = "failed"
		lastError = runErr.Error()
		if len(lastError) > 2000 {
			lastError = lastError[:2000]
		}
		if s.logger != nil {
			s.logger.Warn("流程级嵌入定时检查执行失败",
				zap.String("scheduleID", schedule.ID.String()),
				zap.String("tenantID", schedule.TenantID.String()),
				zap.String("module", schedule.Module),
				zap.String("processType", schedule.ProcessType),
				zap.Error(runErr))
		}
	}
	nextRun := ParseNextRun(schedule.CronExpression)
	_ = s.scheduleRepo.UpdateRunResult(ctx, schedule.ID, startedAt, nextRun, status, lastError)
}

func (s *EmbedRefreshService) isScheduleSourceActive(
	ctx context.Context,
	schedule *model.EmbedRefreshSchedule,
) (bool, error) {
	switch schedule.Module {
	case embedRefreshModuleAudit:
		cfg, err := s.auditRepo.GetByIDForSchedule(ctx, schedule.TenantID, schedule.ConfigID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return isAuditScheduleConfigActive(cfg), nil
	case embedRefreshModuleSummary:
		cfg, err := s.summaryRepo.GetByIDForSchedule(ctx, schedule.TenantID, schedule.ConfigID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return isSummaryScheduleConfigActive(cfg), nil
	default:
		return false, fmt.Errorf("不支持的流程级定时检查模块: %s", schedule.Module)
	}
}

func isAuditScheduleConfigActive(cfg *model.ProcessAuditConfig) bool {
	if cfg == nil {
		return false
	}
	embedCfg := parseEmbedConfig(cfg.EmbedConfig)
	return cfg.Status == "active" && cfg.EmbedEnabled && embedCfg.ScheduledRefreshEnabled
}

func isSummaryScheduleConfigActive(cfg *model.ProcessSummaryConfig) bool {
	if cfg == nil {
		return false
	}
	embedCfg := parseSummaryEmbedConfig(cfg.EmbedConfig)
	return cfg.Status == "active" && cfg.EmbedEnabled && embedCfg.ScheduledRefreshEnabled
}

func (s *EmbedRefreshService) acquireScheduleRunLock(
	ctx context.Context,
	scheduleID uuid.UUID,
) (func(), bool, error) {
	key := "embed:refresh:schedule:run:" + scheduleID.String()
	owner := uuid.NewString()
	acquired, err := s.rdb.SetNX(ctx, key, owner, embedRefreshScheduleTTL).Result()
	if err != nil || !acquired {
		return func() {}, acquired, err
	}
	release := func() {
		_, _ = releaseEmbedCreateLockScript.Run(context.Background(), s.rdb, []string{key}, owner).Result()
	}
	return release, true, nil
}

func (s *EmbedRefreshService) scanSchedule(
	ctx context.Context,
	schedule *model.EmbedRefreshSchedule,
) error {
	tenant, err := s.tenantRepo.FindByID(schedule.TenantID)
	if err != nil {
		return err
	}
	if tenant.Status != "active" || !tenant.EmbedEnabled || tenant.AdminUserID == nil {
		return nil
	}
	adapter, err := s.auditSvc.getOAAdapter(ctx, schedule.TenantID)
	if err != nil {
		return err
	}
	scanner, ok := adapter.(oa.RecentProcessScanner)
	if !ok {
		return fmt.Errorf("当前 OA 适配器不支持流程级定时检查")
	}
	since := apptime.Now().AddDate(0, 0, -schedule.LookbackDays)
	items, err := scanner.FetchRecentProcessSummaries(
		ctx,
		schedule.ProcessType,
		since,
		embedRefreshScanLimit,
	)
	if err != nil {
		return err
	}

	var firstErr error
	for _, item := range items {
		payload := embedRefreshPayload{
			TenantID:      schedule.TenantID,
			UserID:        *tenant.AdminUserID,
			ProcessID:     item.ProcessID,
			Module:        schedule.Module,
			Action:        "scheduled_scan",
			EventID:       uuid.NewString(),
			Generation:    uuid.NewString(),
			FirstReceived: apptime.Now(),
			ConfigID:      schedule.ConfigID,
			ScheduleID:    schedule.ID,
		}
		if err := s.scheduleIfIdle(ctx, payload); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	pkglogger.GetTenantLogger(tenant.Code).Info("流程级定时检查已安排",
		zap.String("module", schedule.Module),
		zap.String("configID", schedule.ConfigID.String()),
		zap.String("processType", schedule.ProcessType),
		zap.Int("lookbackDays", schedule.LookbackDays),
		zap.Int("intervalMinutes", schedule.IntervalMinutes),
		zap.Int("candidateCount", len(items)))
	return firstErr
}

func (s *EmbedRefreshService) publishScheduleChange(
	ctx context.Context,
	module string,
	configID uuid.UUID,
) {
	raw, _ := json.Marshal(embedRefreshScheduleChange{Module: module, ConfigID: configID})
	_ = s.rdb.Publish(ctx, embedRefreshScheduleTopic, raw).Err()
}

func (s *EmbedRefreshService) runScheduleChangeSubscriber(ctx context.Context) {
	pubsub := s.rdb.Subscribe(ctx, embedRefreshScheduleTopic)
	defer pubsub.Close()
	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			var change embedRefreshScheduleChange
			if json.Unmarshal([]byte(msg.Payload), &change) != nil {
				continue
			}
			s.reloadScheduleByConfig(ctx, change.Module, change.ConfigID)
		}
	}
}

func (s *EmbedRefreshService) reloadScheduleByConfig(
	ctx context.Context,
	module string,
	configID uuid.UUID,
) {
	schedule, err := s.scheduleRepo.GetByConfig(ctx, module, configID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.removeSchedule(uuid.Nil, module, configID)
		}
		return
	}
	_ = s.addOrUpdateSchedule(schedule)
}

func embedRefreshScheduleConfigKey(module string, configID uuid.UUID) string {
	return module + ":" + configID.String()
}

// scheduleIfIdle 只在当前流程没有待处理检查时安排定时候选，避免覆盖保存/提交事件。
func (s *EmbedRefreshService) scheduleIfIdle(ctx context.Context, payload embedRefreshPayload) error {
	member := embedRefreshPayloadMember(payload)
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = scheduleIdleEmbedRefreshScript.Run(
		ctx,
		s.rdb,
		[]string{
			embedRefreshDueKey,
			embedRefreshPayloadKey(member),
			embedRefreshGenerationKey(member),
		},
		string(raw),
		payload.Generation,
		apptime.Now().UnixMilli(),
		int(embedRefreshPayloadTTL.Seconds()),
		member,
	).Result()
	return err
}

func isObsoleteEmbedRefreshAction(action string) bool {
	switch strings.TrimSpace(action) {
	case model.SummaryTriggerDetailSaveRequested,
		model.SummaryTriggerDetailSubmitRequested,
		model.SummaryTriggerDetailScheduled:
		return false
	default:
		return true
	}
}

func shouldRetryEmbedEvent(action string) bool {
	return isEmbedOperationAction(action)
}

func isEmbedOperationAction(action string) bool {
	return action == model.SummaryTriggerDetailSaveRequested ||
		action == model.SummaryTriggerDetailSubmitRequested
}

func nullableUUID(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}

func embedRefreshMember(tenantID uuid.UUID, module, processID string) string {
	return tenantID.String() + ":" + module + ":" + processID
}

func embedRefreshPayloadMember(payload embedRefreshPayload) string {
	key := payload.ProcessID
	if payload.Module == embedRefreshModuleResolve {
		key = payload.EventID
	}
	return embedRefreshMember(payload.TenantID, payload.Module, key)
}

func embedRefreshPayloadKey(member string) string {
	return "embed:refresh:payload:" + member
}

func embedRefreshGenerationKey(member string) string {
	return "embed:refresh:generation:" + member
}
