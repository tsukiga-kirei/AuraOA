package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"auraoa/go-service/internal/model"
)

const (
	summaryRedisBackgroundStream  = "summary:jobs"
	summaryRedisBackgroundGroup   = "summary-workers"
	summaryRedisScheduledStream   = "summary:jobs:scheduled"
	summaryRedisScheduledGroup    = "summary-scheduled-workers"
	summaryRedisInteractiveStream = "summary:jobs:interactive"
	summaryRedisInteractiveGroup  = "summary-interactive-workers"
	summaryRedisFieldPayload      = "payload"
	summaryRedisReclaimMinIdle    = 30 * time.Second
	summaryRedisReclaimInterval   = 30 * time.Second
)

type summaryJobMsg struct {
	SummaryLogID string `json:"summary_log_id"`
	TenantID     string `json:"tenant_id"`
	UserID       string `json:"user_id"`
}

func EnqueueSummaryJob(
	ctx context.Context,
	rdb *redis.Client,
	summaryLogID, tenantID, userID uuid.UUID,
	priority int,
) (string, error) {
	if rdb == nil {
		return "", fmt.Errorf("redis client is nil")
	}
	b, err := json.Marshal(summaryJobMsg{
		SummaryLogID: summaryLogID.String(),
		TenantID:     tenantID.String(),
		UserID:       userID.String(),
	})
	if err != nil {
		return "", err
	}
	stream := summaryRedisBackgroundStream
	if model.IsSummaryInteractivePriority(priority) {
		stream = summaryRedisInteractiveStream
	} else if priority <= model.SummaryPriorityScheduled {
		stream = summaryRedisScheduledStream
	}
	return rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		MaxLen: 100000,
		Approx: true,
		Values: map[string]interface{}{summaryRedisFieldPayload: string(b)},
	}).Result()
}

func ensureSummaryConsumerGroup(ctx context.Context, rdb *redis.Client, stream, group string) error {
	err := rdb.XGroupCreateMkStream(ctx, stream, group, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}

func StartSummaryStreamWorker(ctx context.Context, rdb *redis.Client, svc *ProcessSummaryService, logger *zap.Logger, concurrency int) error {
	if rdb == nil || svc == nil {
		return nil
	}
	if err := ensureSummaryConsumerGroup(ctx, rdb, summaryRedisBackgroundStream, summaryRedisBackgroundGroup); err != nil {
		return err
	}
	if err := ensureSummaryConsumerGroup(ctx, rdb, summaryRedisInteractiveStream, summaryRedisInteractiveGroup); err != nil {
		return err
	}
	if err := ensureSummaryConsumerGroup(ctx, rdb, summaryRedisScheduledStream, summaryRedisScheduledGroup); err != nil {
		return err
	}
	if concurrency < 1 {
		concurrency = 1
	}
	host, _ := os.Hostname()
	consumerBase := fmt.Sprintf("%s-%d", host, time.Now().UnixNano())
	for i := 0; i < concurrency; i++ {
		consumerName := fmt.Sprintf("%s-background-%d", consumerBase, i)
		go runSummaryBackgroundConsumerLoop(
			ctx,
			rdb,
			svc,
			logger,
			consumerName,
		)
	}
	go runSummaryConsumerLoop(
		ctx,
		rdb,
		svc,
		logger,
		summaryRedisInteractiveStream,
		summaryRedisInteractiveGroup,
		fmt.Sprintf("%s-interactive", consumerBase),
	)
	if logger != nil {
		logger.Info("总结任务队列处理器已启动",
			zap.Int("backgroundConcurrency", concurrency),
			zap.Int("interactiveConcurrency", 1))
	}
	return nil
}

func runSummaryBackgroundConsumerLoop(
	ctx context.Context,
	rdb *redis.Client,
	svc *ProcessSummaryService,
	logger *zap.Logger,
	consumerName string,
) {
	reclaimIdleSummaryMessages(
		ctx,
		rdb,
		svc,
		logger,
		summaryRedisBackgroundStream,
		summaryRedisBackgroundGroup,
		consumerName,
	)
	reclaimIdleSummaryMessages(
		ctx,
		rdb,
		svc,
		logger,
		summaryRedisScheduledStream,
		summaryRedisScheduledGroup,
		consumerName,
	)
	lastReclaim := time.Now()
	for {
		if ctx.Err() != nil {
			return
		}
		if time.Since(lastReclaim) >= summaryRedisReclaimInterval {
			reclaimIdleSummaryMessages(
				ctx,
				rdb,
				svc,
				logger,
				summaryRedisBackgroundStream,
				summaryRedisBackgroundGroup,
				consumerName,
			)
			reclaimIdleSummaryMessages(
				ctx,
				rdb,
				svc,
				logger,
				summaryRedisScheduledStream,
				summaryRedisScheduledGroup,
				consumerName,
			)
			lastReclaim = time.Now()
		}
		consumed, err := consumeOneSummaryMessage(
			ctx,
			rdb,
			svc,
			logger,
			summaryRedisBackgroundStream,
			summaryRedisBackgroundGroup,
			consumerName,
			300*time.Millisecond,
		)
		if err != nil {
			logSummaryConsumerError(ctx, logger, summaryRedisBackgroundStream, err)
			continue
		}
		if consumed {
			continue
		}
		_, err = consumeOneSummaryMessage(
			ctx,
			rdb,
			svc,
			logger,
			summaryRedisScheduledStream,
			summaryRedisScheduledGroup,
			consumerName,
			2*time.Second,
		)
		if err != nil {
			logSummaryConsumerError(ctx, logger, summaryRedisScheduledStream, err)
		}
	}
}

func runSummaryConsumerLoop(
	ctx context.Context,
	rdb *redis.Client,
	svc *ProcessSummaryService,
	logger *zap.Logger,
	stream, group, consumerName string,
) {
	reclaimIdleSummaryMessages(ctx, rdb, svc, logger, stream, group, consumerName)
	lastReclaim := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if time.Since(lastReclaim) >= summaryRedisReclaimInterval {
			reclaimIdleSummaryMessages(ctx, rdb, svc, logger, stream, group, consumerName)
			lastReclaim = time.Now()
		}
		_, err := consumeOneSummaryMessage(
			ctx,
			rdb,
			svc,
			logger,
			stream,
			group,
			consumerName,
			5*time.Second,
		)
		if err != nil {
			logSummaryConsumerError(ctx, logger, stream, err)
		}
	}
}

// reclaimIdleSummaryMessages 接管旧容器遗留的 pending 消息，数据库状态会再次阻止已完成或已失败任务重跑。
func reclaimIdleSummaryMessages(
	ctx context.Context,
	rdb *redis.Client,
	svc *ProcessSummaryService,
	logger *zap.Logger,
	stream, group, consumerName string,
) {
	start := "0-0"
	for ctx.Err() == nil {
		messages, next, err := rdb.XAutoClaim(ctx, &redis.XAutoClaimArgs{
			Stream:   stream,
			Group:    group,
			Consumer: consumerName,
			MinIdle:  summaryRedisReclaimMinIdle,
			Start:    start,
			Count:    20,
		}).Result()
		if err != nil {
			if logger != nil && err != redis.Nil && ctx.Err() == nil {
				logger.Warn("接管遗留总结队列消息失败",
					zap.String("stream", stream),
					zap.Error(err))
			}
			return
		}
		for _, msg := range messages {
			svc.handleSummaryStreamMessage(ctx, rdb, stream, group, msg.ID, msg.Values, logger)
		}
		if next == "0-0" {
			return
		}
		start = next
	}
}

func consumeOneSummaryMessage(
	ctx context.Context,
	rdb *redis.Client,
	svc *ProcessSummaryService,
	logger *zap.Logger,
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
			svc.handleSummaryStreamMessage(ctx, rdb, result.Stream, group, msg.ID, msg.Values, logger)
		}
	}
	return consumed, nil
}

func logSummaryConsumerError(ctx context.Context, logger *zap.Logger, stream string, err error) {
	if err == nil || err == context.Canceled || ctx.Err() != nil {
		return
	}
	if logger != nil {
		logger.Error("总结任务队列读取失败",
			zap.String("stream", stream),
			zap.Error(err))
	}
	time.Sleep(time.Second)
}

func (s *ProcessSummaryService) handleSummaryStreamMessage(
	ctx context.Context,
	rdb *redis.Client,
	stream, group, msgID string,
	values map[string]interface{},
	logger *zap.Logger,
) {
	raw, _ := values[summaryRedisFieldPayload].(string)
	var job summaryJobMsg
	if err := json.Unmarshal([]byte(raw), &job); err != nil {
		_ = rdb.XAck(ctx, stream, group, msgID).Err()
		return
	}
	summaryLogID, err := uuid.Parse(job.SummaryLogID)
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
	interactive := stream == summaryRedisInteractiveStream
	if err := s.processSummaryJob(ctx, summaryLogID, tenantID, userID, interactive); err != nil && logger != nil {
		logger.Warn("总结任务执行失败", zap.String("summary_log_id", summaryLogID.String()), zap.Error(err))
	}
	_ = rdb.XAck(ctx, stream, group, msgID).Err()
}

func StartSummaryStaleReconciler(ctx context.Context, svc *ProcessSummaryService, logger *zap.Logger, interval time.Duration) {
	if svc == nil {
		return
	}
	if interval < 5*time.Second {
		interval = 30 * time.Second
	}
	go func() {
		run := func() {
			n, err := svc.FailStaleSummaryJobs(context.Background())
			if err != nil {
				if logger != nil {
					logger.Warn("清理超时总结任务失败", zap.Error(err))
				}
				return
			}
			if n > 0 && logger != nil {
				logger.Info("超时总结任务已标记失败", zap.Int64("count", n))
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
		logger.Info("总结超时任务协调器已启动", zap.Duration("interval", interval))
	}
}
