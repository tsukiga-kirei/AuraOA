package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"auraoa/go-service/internal/model"
)

const (
	auditRedisWorkbenchStream   = "audit:jobs"
	auditRedisWorkbenchGroup    = "audit-workers"
	auditRedisBackgroundStream  = "audit:jobs:background"
	auditRedisBackgroundGroup   = "audit-background-workers"
	auditRedisScheduledStream   = "audit:jobs:scheduled"
	auditRedisScheduledGroup    = "audit-scheduled-workers"
	auditRedisInteractiveStream = "audit:jobs:interactive"
	auditRedisInteractiveGroup  = "audit-interactive-workers"
	auditRedisFieldPayload      = "payload"
	auditRedisReclaimMinIdle    = 30 * time.Second
	auditRedisReclaimInterval   = 30 * time.Second
)

type auditJobMsg struct {
	AuditLogID string `json:"audit_log_id"`
	TenantID   string `json:"tenant_id"`
	UserID     string `json:"user_id"`
}

// EnqueueAuditJob 按队列类型将审核任务写入工作台、OA 交互、保存/提交后台或定时 Redis Stream。
func EnqueueAuditJob(
	ctx context.Context,
	rdb *redis.Client,
	auditLogID, tenantID, userID uuid.UUID,
	queueKind string,
) (string, error) {
	if rdb == nil {
		return "", fmt.Errorf("redis client is nil")
	}
	b, err := json.Marshal(auditJobMsg{
		AuditLogID: auditLogID.String(),
		TenantID:   tenantID.String(),
		UserID:     userID.String(),
	})
	if err != nil {
		return "", err
	}
	queueKind = model.NormalizeAuditJobQueueKind(queueKind)
	stream := auditRedisWorkbenchStream
	if queueKind == model.JobQueueKindInteractive {
		stream = auditRedisInteractiveStream
	} else if queueKind == model.JobQueueKindBackground {
		stream = auditRedisBackgroundStream
	} else if queueKind == model.JobQueueKindScheduled {
		stream = auditRedisScheduledStream
	}
	return rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		MaxLen: 100000,
		Approx: true,
		Values: map[string]interface{}{auditRedisFieldPayload: string(b)},
	}).Result()
}

func ensureAuditConsumerGroup(ctx context.Context, rdb *redis.Client, stream, group string) error {
	err := rdb.XGroupCreateMkStream(ctx, stream, group, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}

// StartAuditStreamWorker 启动审核工作台、OA 交互、保存/提交后台和定时扫描四类独立队列。
func StartAuditStreamWorker(
	ctx context.Context,
	rdb *redis.Client,
	svc *AuditExecuteService,
	logger *zap.Logger,
	workbenchConcurrency, interactiveConcurrency, backgroundConcurrency, scheduledConcurrency, totalConcurrency int,
) error {
	if rdb == nil || svc == nil {
		return nil
	}
	for _, item := range []struct {
		stream string
		group  string
	}{
		{auditRedisWorkbenchStream, auditRedisWorkbenchGroup},
		{auditRedisBackgroundStream, auditRedisBackgroundGroup},
		{auditRedisScheduledStream, auditRedisScheduledGroup},
		{auditRedisInteractiveStream, auditRedisInteractiveGroup},
	} {
		if err := ensureAuditConsumerGroup(ctx, rdb, item.stream, item.group); err != nil {
			return err
		}
	}
	if workbenchConcurrency < 1 {
		workbenchConcurrency = 2
	}
	if backgroundConcurrency < 1 {
		backgroundConcurrency = 1
	}
	if interactiveConcurrency < 1 {
		interactiveConcurrency = 1
	}
	if scheduledConcurrency < 1 {
		scheduledConcurrency = 1
	}
	if totalConcurrency < 1 {
		totalConcurrency = 3
	}
	host, _ := os.Hostname()
	consumerBase := fmt.Sprintf("%s-%d", host, time.Now().UnixNano())
	limiter := newJobExecutionLimiter(totalConcurrency)
	svc.executionLimiter = limiter
	startAuditQueueConsumers(ctx, rdb, svc, logger, limiter, consumerBase, "workbench",
		auditRedisWorkbenchStream, auditRedisWorkbenchGroup, workbenchConcurrency)
	startAuditQueueConsumers(ctx, rdb, svc, logger, limiter, consumerBase, "interactive",
		auditRedisInteractiveStream, auditRedisInteractiveGroup, interactiveConcurrency)
	startAuditQueueConsumers(ctx, rdb, svc, logger, limiter, consumerBase, "background",
		auditRedisBackgroundStream, auditRedisBackgroundGroup, backgroundConcurrency)
	startAuditQueueConsumers(ctx, rdb, svc, logger, limiter, consumerBase, "scheduled",
		auditRedisScheduledStream, auditRedisScheduledGroup, scheduledConcurrency)
	if logger != nil {
		logger.Info("审核任务队列处理器已启动",
			zap.Int("workbenchConcurrency", workbenchConcurrency),
			zap.Int("backgroundConcurrency", backgroundConcurrency),
			zap.Int("interactiveConcurrency", interactiveConcurrency),
			zap.Int("scheduledConcurrency", scheduledConcurrency),
			zap.Int("totalConcurrency", totalConcurrency))
	}
	return nil
}

func startAuditQueueConsumers(
	ctx context.Context,
	rdb *redis.Client,
	svc *AuditExecuteService,
	logger *zap.Logger,
	limiter *jobExecutionLimiter,
	consumerBase, queueName, stream, group string,
	concurrency int,
) {
	for i := 0; i < concurrency; i++ {
		go runAuditConsumerLoop(
			ctx,
			rdb,
			svc,
			logger,
			limiter,
			stream,
			group,
			fmt.Sprintf("%s-%s-%d", consumerBase, queueName, i),
		)
	}
}

func runAuditConsumerLoop(
	ctx context.Context,
	rdb *redis.Client,
	svc *AuditExecuteService,
	logger *zap.Logger,
	limiter *jobExecutionLimiter,
	stream, group, consumerName string,
) {
	reclaimIdleAuditMessages(ctx, rdb, svc, logger, limiter, stream, group, consumerName)
	lastReclaim := time.Now()
	for {
		if ctx.Err() != nil {
			return
		}
		if time.Since(lastReclaim) >= auditRedisReclaimInterval {
			reclaimIdleAuditMessages(ctx, rdb, svc, logger, limiter, stream, group, consumerName)
			lastReclaim = time.Now()
		}
		_, err := consumeOneAuditMessage(
			ctx,
			rdb,
			svc,
			logger,
			limiter,
			stream,
			group,
			consumerName,
			5*time.Second,
		)
		if err != nil {
			logAuditConsumerError(ctx, logger, stream, err)
		}
	}
}

// reclaimIdleAuditMessages 接管旧容器遗留的 pending 消息；数据库状态会阻止终态任务重跑。
func reclaimIdleAuditMessages(
	ctx context.Context,
	rdb *redis.Client,
	svc *AuditExecuteService,
	logger *zap.Logger,
	limiter *jobExecutionLimiter,
	stream, group, consumerName string,
) {
	start := "0-0"
	for ctx.Err() == nil {
		messages, next, err := rdb.XAutoClaim(ctx, &redis.XAutoClaimArgs{
			Stream:   stream,
			Group:    group,
			Consumer: consumerName,
			MinIdle:  auditRedisReclaimMinIdle,
			Start:    start,
			Count:    20,
		}).Result()
		if err != nil {
			if logger != nil && err != redis.Nil && ctx.Err() == nil {
				logger.Warn("接管遗留审核队列消息失败",
					zap.String("stream", stream),
					zap.Error(err))
			}
			return
		}
		for _, msg := range messages {
			svc.handleAuditStreamMessage(ctx, rdb, limiter, stream, group, msg.ID, msg.Values, logger)
		}
		if next == "0-0" {
			return
		}
		start = next
	}
}

func consumeOneAuditMessage(
	ctx context.Context,
	rdb *redis.Client,
	svc *AuditExecuteService,
	logger *zap.Logger,
	limiter *jobExecutionLimiter,
	stream, group, consumerName string,
	block time.Duration,
) (bool, error) {
	streams, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumerName,
		Streams:  []string{stream, ">"},
		Count:    1,
		Block:    block,
	}).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	consumed := false
	for _, result := range streams {
		for _, msg := range result.Messages {
			consumed = true
			svc.handleAuditStreamMessage(ctx, rdb, limiter, result.Stream, group, msg.ID, msg.Values, logger)
		}
	}
	return consumed, nil
}

func logAuditConsumerError(ctx context.Context, logger *zap.Logger, stream string, err error) {
	if err == nil || err == context.Canceled || ctx.Err() != nil {
		return
	}
	if logger != nil {
		logger.Error("审核任务队列读取失败",
			zap.String("stream", stream),
			zap.Error(err))
	}
	time.Sleep(time.Second)
}

func (s *AuditExecuteService) handleAuditStreamMessage(
	ctx context.Context,
	rdb *redis.Client,
	limiter *jobExecutionLimiter,
	stream, group, msgID string,
	values map[string]interface{},
	logger *zap.Logger,
) {
	raw, _ := values[auditRedisFieldPayload].(string)
	var job auditJobMsg
	if err := json.Unmarshal([]byte(raw), &job); err != nil {
		_ = rdb.XAck(ctx, stream, group, msgID).Err()
		return
	}
	auditLogID, err := uuid.Parse(job.AuditLogID)
	if err != nil {
		_ = rdb.XAck(ctx, stream, group, msgID).Err()
		return
	}
	tenantID, err := uuid.Parse(job.TenantID)
	if err != nil {
		_ = rdb.XAck(ctx, stream, group, msgID).Err()
		return
	}
	userID, err := uuid.Parse(job.UserID)
	if err != nil {
		_ = rdb.XAck(ctx, stream, group, msgID).Err()
		return
	}
	queueKind := auditQueueKindFromStream(stream)
	rerouted, rerouteErr := s.rerouteAuditJobIfNeeded(
		ctx,
		rdb,
		auditLogID,
		tenantID,
		userID,
		queueKind,
	)
	if rerouteErr != nil {
		if logger != nil {
			logger.Warn("审核任务按队列类型转投失败",
				zap.String("audit_log_id", auditLogID.String()),
				zap.String("source_stream", stream),
				zap.Error(rerouteErr))
		}
		// 不 ACK，保留 pending 消息供后续接管重试。
		return
	}
	if rerouted {
		_ = rdb.XAck(ctx, stream, group, msgID).Err()
		return
	}
	release, acquired := limiter.Acquire(ctx, queueKind)
	if !acquired {
		return
	}
	defer release()
	if err := s.processAuditJob(ctx, auditLogID, tenantID, userID, queueKind); err != nil && logger != nil {
		logger.Warn("审核任务执行失败",
			zap.String("audit_log_id", auditLogID.String()),
			zap.Error(err))
	}
	_ = rdb.XAck(ctx, stream, group, msgID).Err()
}

// rerouteAuditJobIfNeeded 将升级前或任务切换队列后遗留在旧 Stream 的 pending 消息转投到正确队列。
func (s *AuditExecuteService) rerouteAuditJobIfNeeded(
	ctx context.Context,
	rdb *redis.Client,
	auditLogID, tenantID, userID uuid.UUID,
	streamQueueKind string,
) (bool, error) {
	c := s.workerGinContext(ctx, tenantID, userID)
	logEntry, err := s.auditLogRepo.GetByID(c, auditLogID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return true, nil
		}
		return false, err
	}
	targetQueueKind := model.NormalizeAuditJobQueueKind(logEntry.QueueKind)
	if targetQueueKind == streamQueueKind {
		return false, nil
	}
	if logEntry.Status != model.JobStatusPending {
		return true, nil
	}
	_, err = EnqueueAuditJob(ctx, rdb, auditLogID, tenantID, logEntry.UserID, targetQueueKind)
	return true, err
}

func auditQueueKindFromStream(stream string) string {
	switch stream {
	case auditRedisInteractiveStream:
		return model.JobQueueKindInteractive
	case auditRedisBackgroundStream:
		return model.JobQueueKindBackground
	case auditRedisScheduledStream:
		return model.JobQueueKindScheduled
	default:
		return model.JobQueueKindWorkbench
	}
}

const auditStaleReconcileInterval = 30 * time.Second

// StartAuditStaleReconciler 定时将长时间未结束的非终态审核任务标记为失败。
func StartAuditStaleReconciler(
	ctx context.Context,
	svc *AuditExecuteService,
	logger *zap.Logger,
	interval time.Duration,
) {
	if svc == nil {
		return
	}
	if interval < 5*time.Second {
		interval = auditStaleReconcileInterval
	}
	go func() {
		run := func() {
			n, err := svc.FailStaleAuditJobs(context.Background())
			if err != nil {
				if logger != nil {
					logger.Warn("清理超时审核任务失败", zap.Error(err))
				}
				return
			}
			if n > 0 && logger != nil {
				logger.Info("超时审核任务已标记失败", zap.Int64("count", n))
			}
		}
		run()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()
	if logger != nil {
		logger.Info("审核超时任务协调器已启动", zap.Duration("interval", interval))
	}
}
