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
	auditRedisStream       = "audit:jobs"
	auditRedisConsumerGrp  = "audit-workers"
	auditRedisFieldPayload = "payload"
)

// auditJobMsg Redis Stream 消息体
type auditJobMsg struct {
	AuditLogID string `json:"audit_log_id"`
	TenantID   string `json:"tenant_id"`
	UserID     string `json:"user_id"`
}

// EnqueueAuditJob 将审核任务写入 Redis Stream（需在 DB 已写入 pending 记录后调用）。
func EnqueueAuditJob(ctx context.Context, rdb *redis.Client, auditLogID, tenantID, userID uuid.UUID) (string, error) {
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
	return rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: auditRedisStream,
		MaxLen: 100000,
		Approx: true,
		Values: map[string]interface{}{auditRedisFieldPayload: string(b)},
	}).Result()
}

func ensureAuditConsumerGroup(ctx context.Context, rdb *redis.Client) error {
	err := rdb.XGroupCreateMkStream(ctx, auditRedisStream, auditRedisConsumerGrp, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}

// StartAuditStreamWorker 启动后台消费者（可多 goroutine 并发消费）。
func StartAuditStreamWorker(ctx context.Context, rdb *redis.Client, svc *AuditExecuteService, logger *zap.Logger, concurrency int) error {
	if rdb == nil || svc == nil {
		return nil
	}
	if err := ensureAuditConsumerGroup(ctx, rdb); err != nil {
		return err
	}
	if concurrency < 1 {
		concurrency = 2
	}
	host, _ := os.Hostname()
	consumerBase := fmt.Sprintf("%s-%d", host, time.Now().UnixNano())

	for i := 0; i < concurrency; i++ {
		consumerName := fmt.Sprintf("%s-%d", consumerBase, i)
		go runAuditConsumerLoop(ctx, rdb, svc, logger, consumerName)
	}
	logger.Info("audit stream worker started", zap.Int("concurrency", concurrency))
	return nil
}

func runAuditConsumerLoop(ctx context.Context, rdb *redis.Client, svc *AuditExecuteService, logger *zap.Logger, consumerName string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		streams, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    auditRedisConsumerGrp,
			Consumer: consumerName,
			Streams:  []string{auditRedisStream, ">"},
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
			logger.Error("audit stream worker error", zap.Error(err))
			time.Sleep(time.Second)
			continue
		}
		for _, stream := range streams {
			for _, msg := range stream.Messages {
				svc.handleAuditStreamMessage(ctx, rdb, msg.ID, msg.Values, logger)
			}
		}
	}
}

func (s *AuditExecuteService) handleAuditStreamMessage(ctx context.Context, rdb *redis.Client, msgID string, values map[string]interface{}, logger *zap.Logger) {
	raw, _ := values[auditRedisFieldPayload].(string)
	var job auditJobMsg
	if err := json.Unmarshal([]byte(raw), &job); err != nil {
		_ = rdb.XAck(ctx, auditRedisStream, auditRedisConsumerGrp, msgID).Err()
		return
	}
	auditLogID, err := uuid.Parse(job.AuditLogID)
	if err != nil {
		_ = rdb.XAck(ctx, auditRedisStream, auditRedisConsumerGrp, msgID).Err()
		return
	}
	tenantID, err := uuid.Parse(job.TenantID)
	if err != nil {
		_ = rdb.XAck(ctx, auditRedisStream, auditRedisConsumerGrp, msgID).Err()
		return
	}
	userID, err := uuid.Parse(job.UserID)
	if err != nil {
		_ = rdb.XAck(ctx, auditRedisStream, auditRedisConsumerGrp, msgID).Err()
		return
	}

	if err := s.processAuditJob(ctx, auditLogID, tenantID, userID); err != nil && logger != nil {
		logger.Warn("audit job failed", zap.String("audit_log_id", auditLogID.String()), zap.Error(err))
	}
	_ = rdb.XAck(ctx, auditRedisStream, auditRedisConsumerGrp, msgID).Err()
}

// auditStaleReconcileInterval 后台扫描「非终态超时」任务的周期。
const auditStaleReconcileInterval = 30 * time.Second

// StartAuditStaleReconciler 定时将 pending/reasoning/extracting 过久记录标为 failed，避免列表与轮询永远卡在 pending。
func StartAuditStaleReconciler(ctx context.Context, svc *AuditExecuteService, logger *zap.Logger, interval time.Duration) {
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
					logger.Warn("fail stale audit jobs", zap.Error(err))
				}
				return
			}
			if n > 0 && logger != nil {
				logger.Info("marked stale audit jobs as failed", zap.Int64("count", n))
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
		logger.Info("audit stale reconciler started", zap.Duration("interval", interval))
	}
}
