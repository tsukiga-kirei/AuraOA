package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	defaultScheduledLookbackDays    = 3
	defaultScheduledIntervalMinutes = 5
)

var allowedScheduledIntervals = map[int]bool{
	5:  true,
	10: true,
	15: true,
	30: true,
	60: true,
}

// normalizeScheduledRefreshConfig 校正流程级定时检查参数。
func normalizeScheduledRefreshConfig(lookbackDays, intervalMinutes *int) {
	if *lookbackDays < 1 || *lookbackDays > 30 {
		*lookbackDays = defaultScheduledLookbackDays
	}
	if !allowedScheduledIntervals[*intervalMinutes] {
		*intervalMinutes = defaultScheduledIntervalMinutes
	}
}

var releaseEmbedCreateLockScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`)

// acquireEmbedCreateLock 防止可见 iframe 与后台协调器同时创建同一流程任务。
func acquireEmbedCreateLock(
	ctx context.Context,
	rdb *redis.Client,
	module string,
	tenantID uuid.UUID,
	processID string,
) (func(), bool, error) {
	if rdb == nil {
		return nil, false, fmt.Errorf("Redis 不可用")
	}
	key := fmt.Sprintf("embed:create:%s:%s:%s", module, tenantID.String(), processID)
	owner := uuid.NewString()
	ok, err := rdb.SetNX(ctx, key, owner, 30*time.Second).Result()
	if err != nil || !ok {
		return nil, ok, err
	}
	release := func() {
		_, _ = releaseEmbedCreateLockScript.Run(context.Background(), rdb, []string{key}, owner).Result()
	}
	return release, true, nil
}
