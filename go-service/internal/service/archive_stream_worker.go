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

// 归档复盘 Redis Stream 相关常量
const (
	archiveRedisStream       = "archive:jobs"
	archiveRedisConsumerGrp  = "archive-review-workers"
	archiveRedisFieldPayload = "payload"
)

// archiveJobMsg 归档复盘任务的 Redis Stream 消息体
type archiveJobMsg struct {
	ArchiveLogID string `json:"archive_log_id"`
	TenantID     string `json:"tenant_id"`
	UserID       string `json:"user_id"`
}

// EnqueueArchiveJob 将归档复盘任务写入 Redis Stream。
// 须在 DB 已写入 pending 记录后调用，确保消费者能查到对应日志行。
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

// ensureArchiveConsumerGroup 确保消费者组存在，若已存在则忽略 BUSYGROUP 错误。
func ensureArchiveConsumerGroup(ctx context.Context, rdb *redis.Client) error {
	err := rdb.XGroupCreateMkStream(ctx, archiveRedisStream, archiveRedisConsumerGrp, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}

// StartArchiveStreamWorker 启动归档复盘后台消费者，支持多 goroutine 并发消费。
// concurrency 控制并发数，最小为 2。
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

// runArchiveConsumerLoop 单个消费者的主循环，阻塞读取 Stream 消息并分发处理。
// context 取消时退出循环，Redis 错误时短暂休眠后重试。
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

// handleArchiveStreamMessage 解析单条 Stream 消息并执行归档复盘任务。
// 消息格式非法时直接 ACK 跳过，避免消息积压。
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

// archiveStaleReconcileInterval 后台扫描超时任务的默认周期
const archiveStaleReconcileInterval = 30 * time.Second

// StartArchiveStaleReconciler 定时将长时间未结束的非终态归档任务标记为失败。
// 防止因 Worker 崩溃或网络异常导致任务永久卡在 pending/reasoning 状态。
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
