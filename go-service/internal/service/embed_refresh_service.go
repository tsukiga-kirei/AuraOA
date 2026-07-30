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

// EmbedRefreshEventRequest OA 页面提交的轻量刷新事件。
type EmbedRefreshEventRequest struct {
	ProcessID string `json:"process_id" binding:"required"`
	Action    string `json:"action"`
	EventID   string `json:"event_id"`
}

// EmbedRefreshEventResponse 事件接收结果。
type EmbedRefreshEventResponse struct {
	ProcessID        string   `json:"process_id"`
	Action           string   `json:"action"`
	EventID          string   `json:"event_id"`
	ScheduledModules []string `json:"scheduled_modules"`
}

type embedRefreshPayload struct {
	TenantID      uuid.UUID `json:"tenant_id"`
	UserID        uuid.UUID `json:"user_id"`
	ProcessID     string    `json:"process_id"`
	Module        string    `json:"module"`
	Action        string    `json:"action"`
	EventID       string    `json:"event_id"`
	Generation    string    `json:"generation"`
	Attempt       int       `json:"attempt"`
	FirstReceived time.Time `json:"first_received"`
	ConfigID      uuid.UUID `json:"config_id,omitempty"`
	ScheduleID    uuid.UUID `json:"schedule_id,omitempty"`
}

type embedRefreshResult int

const (
	embedRefreshDone embedRefreshResult = iota
	embedRefreshRetry
	embedRefreshRunning
)

// EmbedRefreshService 编排 OA 保存事件、延迟指纹检查和流程级定时扫描。
type EmbedRefreshService struct {
	rdb             *redis.Client
	auditSvc        *AuditExecuteService
	summarySvc      *ProcessSummaryService
	auditRepo       *repository.ProcessAuditConfigRepo
	summaryRepo     *repository.ProcessSummaryConfigRepo
	scheduleRepo    *repository.EmbedRefreshScheduleRepo
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

// ScheduleEvent 接收 OA 页面事件，并分别安排审核和总结检查。
func (s *EmbedRefreshService) ScheduleEvent(
	ctx context.Context,
	tenantID, userID uuid.UUID,
	req EmbedRefreshEventRequest,
) (*EmbedRefreshEventResponse, error) {
	if s.rdb == nil {
		return nil, fmt.Errorf("Redis 不可用")
	}
	req.ProcessID = strings.TrimSpace(req.ProcessID)
	if req.ProcessID == "" {
		return nil, fmt.Errorf("process_id 不能为空")
	}
	req.Action = normalizeEmbedRefreshAction(req.Action)
	if strings.TrimSpace(req.EventID) == "" {
		req.EventID = uuid.NewString()
	}

	modules := []string{embedRefreshModuleAudit, embedRefreshModuleSummary}
	if req.Action == "page_open" {
		// 兼容旧版 runner：打开 OA 页面不再创建后台任务。
		return &EmbedRefreshEventResponse{
			ProcessID:        req.ProcessID,
			Action:           req.Action,
			EventID:          req.EventID,
			ScheduledModules: []string{},
		}, nil
	}
	for _, module := range modules {
		payload := embedRefreshPayload{
			TenantID:      tenantID,
			UserID:        userID,
			ProcessID:     req.ProcessID,
			Module:        module,
			Action:        req.Action,
			EventID:       req.EventID,
			Generation:    uuid.NewString(),
			FirstReceived: apptime.Now(),
		}
		if err := s.schedule(ctx, payload, embedRefreshInitialDelay, false); err != nil {
			return nil, err
		}
	}
	return &EmbedRefreshEventResponse{
		ProcessID:        req.ProcessID,
		Action:           req.Action,
		EventID:          req.EventID,
		ScheduledModules: modules,
	}, nil
}

// Start 启动延迟事件消费者，并从数据库恢复流程级精确定时任务。
func (s *EmbedRefreshService) Start(ctx context.Context) {
	if s == nil || s.rdb == nil || s.scheduleRepo == nil {
		return
	}
	if removed, err := s.purgeLegacyScheduledPayloads(ctx); err != nil {
		if s.logger != nil {
			s.logger.Warn("清理旧版定时检查队列失败", zap.Error(err))
		}
	} else if removed > 0 && s.logger != nil {
		s.logger.Info("已清理旧版定时检查队列", zap.Int64("count", removed))
	}
	if err := s.reconcileSchedules(ctx); err != nil && s.logger != nil {
		s.logger.Warn("重建 OA 嵌入刷新调度记录失败", zap.Error(err))
	}
	if err := s.restoreSchedules(ctx); err != nil && s.logger != nil {
		s.logger.Warn("恢复 OA 嵌入刷新定时任务失败", zap.Error(err))
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

// purgeLegacyScheduledPayloads 清理旧版打开页事件及缺少 config_id 的定时候选；启用中的 Cron 会重新生成。
func (s *EmbedRefreshService) purgeLegacyScheduledPayloads(ctx context.Context) (int64, error) {
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
		shouldRemove := payload.Action == "page_open" ||
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
		result, processErr := s.checkAndTrigger(ctx, payload)
		if processErr != nil && s.logger != nil {
			s.logger.Warn("OA 嵌入后台刷新检查失败",
				zap.String("tenantID", payload.TenantID.String()),
				zap.String("processID", payload.ProcessID),
				zap.String("module", payload.Module),
				zap.Error(processErr))
		}

		switch result {
		case embedRefreshRunning:
			if shouldRetryEmbedEvent(payload.Action) &&
				apptime.Now().Sub(payload.FirstReceived) < embedRefreshRunningMaxAge {
				payload.Attempt++
				_ = s.schedule(ctx, payload, embedRefreshRunningDelay, true)
			}
		case embedRefreshRetry:
			if shouldRetryEmbedEvent(payload.Action) && payload.Attempt < 2 {
				payload.Attempt++
				delay := time.Duration(2*payload.Attempt+1) * time.Second
				_ = s.schedule(ctx, payload, delay, true)
			}
		}
	}
}

func (s *EmbedRefreshService) checkAndTrigger(
	ctx context.Context,
	payload embedRefreshPayload,
) (embedRefreshResult, error) {
	if payload.Action == "page_open" {
		// 兼容升级前已经排入 Redis 的打开页事件，确认后直接丢弃。
		return embedRefreshDone, nil
	}
	if payload.Action == model.SummaryTriggerDetailScheduled && payload.ConfigID != uuid.Nil {
		schedule, err := s.scheduleRepo.GetByConfig(ctx, payload.Module, payload.ConfigID)
		if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && !schedule.IsActive) {
			return embedRefreshDone, nil
		}
		if err != nil {
			return embedRefreshRetry, err
		}
	}
	gc := buildWorkerContext(ctx, payload.TenantID, payload.UserID, "embed_scheduler")
	switch payload.Module {
	case embedRefreshModuleAudit:
		embedCtx, err := s.auditSvc.GetEmbedContext(gc, payload.ProcessID)
		if err != nil {
			return embedRefreshRetry, err
		}
		if embedCtx.RunningJobID != "" {
			return embedRefreshRunning, nil
		}
		if !embedCtx.Supported {
			if embedCtx.Reason == "not_found_in_oa" {
				return embedRefreshRetry, nil
			}
			return embedRefreshDone, nil
		}
		if !embedCtx.ShouldAutoAudit {
			return embedRefreshDone, nil
		}
		_, err = s.auditSvc.ExecuteEmbed(gc, &EmbedExecuteRequest{
			ProcessID:        payload.ProcessID,
			TriggerSource:    model.AuditTriggerEmbedAuto,
			TriggerDetail:    payload.Action,
			ScheduleConfigID: nullableUUID(payload.ConfigID),
		})
		if err != nil {
			return embedRefreshRetry, err
		}
		return embedRefreshDone, nil

	case embedRefreshModuleSummary:
		embedCtx, err := s.summarySvc.GetEmbedContext(gc, payload.ProcessID)
		if err != nil {
			return embedRefreshRetry, err
		}
		if embedCtx.RunningJobID != "" {
			return embedRefreshRunning, nil
		}
		if !embedCtx.Supported {
			if embedCtx.Reason == "not_found_in_oa" {
				return embedRefreshRetry, nil
			}
			return embedRefreshDone, nil
		}
		if !embedCtx.ShouldAutoSummary {
			return embedRefreshDone, nil
		}
		_, err = s.summarySvc.ExecuteEmbed(gc, &SummaryExecuteRequest{
			ProcessID:        payload.ProcessID,
			TriggerSource:    model.SummaryTriggerEmbedAuto,
			TriggerDetail:    payload.Action,
			ScheduleConfigID: nullableUUID(payload.ConfigID),
		})
		if err != nil {
			return embedRefreshRetry, err
		}
		return embedRefreshDone, nil
	default:
		return embedRefreshDone, nil
	}
}

func (s *EmbedRefreshService) schedule(
	ctx context.Context,
	payload embedRefreshPayload,
	delay time.Duration,
	onlyIfCurrent bool,
) error {
	member := embedRefreshMember(payload.TenantID, payload.Module, payload.ProcessID)
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
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
		_, err = retryEmbedRefreshScript.Run(ctx, s.rdb, keys, args...).Result()
		return err
	}
	_, err = scheduleEmbedRefreshScript.Run(ctx, s.rdb, keys, args...).Result()
	return err
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
	runErr := s.scanSchedule(ctx, schedule)
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
	member := embedRefreshMember(payload.TenantID, payload.Module, payload.ProcessID)
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

func normalizeEmbedRefreshAction(action string) string {
	switch strings.TrimSpace(action) {
	case "page_open":
		return "page_open"
	case "save", "submit", "save_or_submit":
		return "save_or_submit"
	default:
		return "save_or_submit"
	}
}

func shouldRetryEmbedEvent(action string) bool {
	return action == "save_or_submit"
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

func embedRefreshPayloadKey(member string) string {
	return "embed:refresh:payload:" + member
}

func embedRefreshGenerationKey(member string) string {
	return "embed:refresh:generation:" + member
}
