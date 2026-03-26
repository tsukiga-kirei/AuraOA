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
	archiveRedisStream       = "archive:jobs"
	archiveRedisConsumerGrp  = "archive-review-workers"
	archiveRedisFieldPayload = "payload"
)

type archiveJobMsg struct {
	ArchiveLogID string `json:"archive_log_id"`
	TenantID     string `json:"tenant_id"`
	UserID       string `json:"user_id"`
}

func EnqueueArchiveJob(ctx context.Context, rdb *redis.Client, archiveLogID, tenantID, userID uuid.UUID) (string, error) {
	if rdb == nil {
		return "", fmt.Errorf("redis client is nil")
	}
	b, err := json.Marshal(archiveJobMsg{
		ArchiveLogID: archiveLogID.String(),
		TenantID:     tenantID.String(),
		UserID:       userID.String(),
	})
	if err != nil {
		return "", err
	}
	return rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: archiveRedisStream,
		MaxLen: 100000,
		Approx: true,
		Values: map[string]interface{}{archiveRedisFieldPayload: string(b)},
	}).Result()
}

func ensureArchiveConsumerGroup(ctx context.Context, rdb *redis.Client) error {
	err := rdb.XGroupCreateMkStream(ctx, archiveRedisStream, archiveRedisConsumerGrp, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}

func StartArchiveStreamWorker(ctx context.Context, rdb *redis.Client, svc *ArchiveReviewService, logger *zap.Logger, concurrency int) error {
	if rdb == nil || svc == nil {
		return nil
	}
	if err := ensureArchiveConsumerGroup(ctx, rdb); err != nil {
		return err
	}
	if concurrency < 1 {
		concurrency = 2
	}
	host, _ := os.Hostname()
	consumerBase := fmt.Sprintf("%s-%d", host, time.Now().UnixNano())

	for i := 0; i < concurrency; i++ {
		consumerName := fmt.Sprintf("%s-%d", consumerBase, i)
		go runArchiveConsumerLoop(ctx, rdb, svc, logger, consumerName)
	}
	if logger != nil {
		logger.Info("archive stream worker started", zap.Int("concurrency", concurrency))
	}
	return nil
}

func runArchiveConsumerLoop(ctx context.Context, rdb *redis.Client, svc *ArchiveReviewService, logger *zap.Logger, consumerName string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		streams, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    archiveRedisConsumerGrp,
			Consumer: consumerName,
			Streams:  []string{archiveRedisStream, ">"},
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
			if logger != nil {
				logger.Error("archive stream worker error", zap.Error(err))
			}
			time.Sleep(time.Second)
			continue
		}
		for _, stream := range streams {
			for _, msg := range stream.Messages {
				svc.handleArchiveStreamMessage(ctx, rdb, msg.ID, msg.Values, logger)
			}
		}
	}
}

func (s *ArchiveReviewService) handleArchiveStreamMessage(ctx context.Context, rdb *redis.Client, msgID string, values map[string]interface{}, logger *zap.Logger) {
	raw, _ := values[archiveRedisFieldPayload].(string)
	var job archiveJobMsg
	if err := json.Unmarshal([]byte(raw), &job); err != nil {
		_ = rdb.XAck(ctx, archiveRedisStream, archiveRedisConsumerGrp, msgID).Err()
		return
	}

	archiveLogID, err := uuid.Parse(job.ArchiveLogID)
	if err != nil {
		_ = rdb.XAck(ctx, archiveRedisStream, archiveRedisConsumerGrp, msgID).Err()
		return
	}
	tenantID, err := uuid.Parse(job.TenantID)
	if err != nil {
		_ = rdb.XAck(ctx, archiveRedisStream, archiveRedisConsumerGrp, msgID).Err()
		return
	}
	userID, err := uuid.Parse(job.UserID)
	if err != nil {
		_ = rdb.XAck(ctx, archiveRedisStream, archiveRedisConsumerGrp, msgID).Err()
		return
	}

	if err := s.processArchiveJob(ctx, archiveLogID, tenantID, userID); err != nil && logger != nil {
		logger.Warn("archive review job failed", zap.String("archive_log_id", archiveLogID.String()), zap.Error(err))
	}
	_ = rdb.XAck(ctx, archiveRedisStream, archiveRedisConsumerGrp, msgID).Err()
}

const archiveStaleReconcileInterval = 30 * time.Second

func StartArchiveStaleReconciler(ctx context.Context, svc *ArchiveReviewService, logger *zap.Logger, interval time.Duration) {
	if svc == nil {
		return
	}
	if interval < 5*time.Second {
		interval = archiveStaleReconcileInterval
	}
	go func() {
		run := func() {
			n, err := svc.FailStaleArchiveJobs(context.Background())
			if err != nil {
				if logger != nil {
					logger.Warn("fail stale archive jobs", zap.Error(err))
				}
				return
			}
			if n > 0 && logger != nil {
				logger.Info("marked stale archive jobs as failed", zap.Int64("count", n))
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
		logger.Info("archive stale reconciler started", zap.Duration("interval", interval))
	}
}
