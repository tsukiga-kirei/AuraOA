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
)

const (
	summaryRedisStream       = "summary:jobs"
	summaryRedisConsumerGrp  = "summary-workers"
	summaryRedisFieldPayload = "payload"
)

type summaryJobMsg struct {
	SummaryLogID string `json:"summary_log_id"`
	TenantID     string `json:"tenant_id"`
	UserID       string `json:"user_id"`
}

func EnqueueSummaryJob(ctx context.Context, rdb *redis.Client, summaryLogID, tenantID, userID uuid.UUID) (string, error) {
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
	return rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: summaryRedisStream,
		MaxLen: 100000,
		Approx: true,
		Values: map[string]interface{}{summaryRedisFieldPayload: string(b)},
	}).Result()
}

func ensureSummaryConsumerGroup(ctx context.Context, rdb *redis.Client) error {
	err := rdb.XGroupCreateMkStream(ctx, summaryRedisStream, summaryRedisConsumerGrp, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}

func StartSummaryStreamWorker(ctx context.Context, rdb *redis.Client, svc *ProcessSummaryService, logger *zap.Logger, concurrency int) error {
	if rdb == nil || svc == nil {
		return nil
	}
	if err := ensureSummaryConsumerGroup(ctx, rdb); err != nil {
		return err
	}
	if concurrency < 1 {
		concurrency = 1
	}
	host, _ := os.Hostname()
	consumerBase := fmt.Sprintf("%s-%d", host, time.Now().UnixNano())
	for i := 0; i < concurrency; i++ {
		consumerName := fmt.Sprintf("%s-%d", consumerBase, i)
		go runSummaryConsumerLoop(ctx, rdb, svc, logger, consumerName)
	}
	logger.Info("summary stream worker started", zap.Int("concurrency", concurrency))
	return nil
}

func runSummaryConsumerLoop(ctx context.Context, rdb *redis.Client, svc *ProcessSummaryService, logger *zap.Logger, consumerName string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		streams, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    summaryRedisConsumerGrp,
			Consumer: consumerName,
			Streams:  []string{summaryRedisStream, ">"},
			Count:    1,
			Block:    5 * time.Second,
		}).Result()
		if err != nil {
			if err == redis.Nil {
				continue
			}
			if err == context.Canceled || ctx.Err() != nil {
				return
			}
			logger.Error("summary stream worker error", zap.Error(err))
			time.Sleep(time.Second)
			continue
		}
		for _, stream := range streams {
			for _, msg := range stream.Messages {
				svc.handleSummaryStreamMessage(ctx, rdb, msg.ID, msg.Values, logger)
			}
		}
	}
}

func (s *ProcessSummaryService) handleSummaryStreamMessage(ctx context.Context, rdb *redis.Client, msgID string, values map[string]interface{}, logger *zap.Logger) {
	raw, _ := values[summaryRedisFieldPayload].(string)
	var job summaryJobMsg
	if err := json.Unmarshal([]byte(raw), &job); err != nil {
		_ = rdb.XAck(ctx, summaryRedisStream, summaryRedisConsumerGrp, msgID).Err()
		return
	}
	summaryLogID, err := uuid.Parse(job.SummaryLogID)
	if err != nil {
		_ = rdb.XAck(ctx, summaryRedisStream, summaryRedisConsumerGrp, msgID).Err()
		return
	}
	tenantID, err := uuid.Parse(job.TenantID)
	if err != nil {
		_ = rdb.XAck(ctx, summaryRedisStream, summaryRedisConsumerGrp, msgID).Err()
		return
	}
	userID, err := uuid.Parse(job.UserID)
	if err != nil {
		_ = rdb.XAck(ctx, summaryRedisStream, summaryRedisConsumerGrp, msgID).Err()
		return
	}
	if err := s.processSummaryJob(ctx, summaryLogID, tenantID, userID); err != nil && logger != nil {
		logger.Warn("summary job failed", zap.String("summary_log_id", summaryLogID.String()), zap.Error(err))
	}
	_ = rdb.XAck(ctx, summaryRedisStream, summaryRedisConsumerGrp, msgID).Err()
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
					logger.Warn("fail stale summary jobs", zap.Error(err))
				}
				return
			}
			if n > 0 && logger != nil {
				logger.Info("marked stale summary jobs as failed", zap.Int64("count", n))
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
		logger.Info("summary stale reconciler started", zap.Duration("interval", interval))
	}
}
