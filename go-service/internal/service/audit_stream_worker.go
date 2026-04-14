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

// 审核任务 Redis Stream 相关常量
const (
	auditRedisStream       = "audit:jobs"
	auditRedisConsumerGrp  = "audit-workers"
	auditRedisFieldPayload = "payload"
)

// auditJobMsg Redis Stream 消息体，携带审核任务的关键 ID 信息
type auditJobMsg struct {
	AuditLogID string `json:"audit_log_id"`
	TenantID   string `json:"tenant_id"`
	UserID     string `json:"user_id"`
}

// EnqueueAuditJob 将审核任务写入 Redis Stream。
// 须在 DB 已写入 pending 记录后调用，确保消费者能查到对应日志行。
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

// ensureAuditConsumerGroup 确保消费者组存在，若已存在则忽略 BUSYGROUP 错误。
func ensureAuditConsumerGroup(ctx context.Context, rdb *redis.Client) error {
	err := rdb.XGroupCreateMkStream(ctx, auditRedisStream, auditRedisConsumerGrp, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}

// StartAuditStreamWorker 启动审核后台消费者，支持多 goroutine 并发消费。
// concurrency 控制并发数，最小为 2。
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

// runAuditConsumerLoop 单个消费者的主循环，阻塞读取 Stream 消息并分发处理。
// context 取消时退出循环，Redis 错误时短暂休眠后重试。
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

// handleAuditStreamMessage 解析单条 Stream 消息并执行审核任务。
// 消息格式非法时直接 ACK 跳过，避免消息积压。
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

// auditStaleReconcileInterval 后台扫描超时任务的默认周期
const auditStaleReconcileInterval = 30 * time.Second

// StartAuditStaleReconciler 定时将长时间未结束的非终态审核任务标记为失败。
// 防止因 Worker 崩溃或网络异常导致任务永久卡在 pending/reasoning 状态。
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
