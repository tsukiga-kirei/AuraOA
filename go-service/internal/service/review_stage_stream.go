package service

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/sanitize"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/datatypes"
)

// newReviewStageStream 按阶段发送完整累计文本，断线重连及重试时不会混入上一轮半截JSON。
func newReviewStageStream(rdb *redis.Client, module string, id uuid.UUID, stage string, scrub bool) (func(string), func()) {
	var mu sync.Mutex
	var text strings.Builder
	publish := func() {
		value := text.String()
		if scrub {
			value = sanitize.SanitizeText(value)
		}
		if rdb != nil {
			ctx := context.Background()
			rdb.Set(ctx, module+":"+stage+":"+id.String(), value, 24*time.Hour)
			payload, _ := json.Marshal(JobStreamEvent{Event: stage, Data: value})
			rdb.Publish(ctx, module+":stream:"+id.String(), string(payload))
		}
	}
	chunk := func(value string) { mu.Lock(); defer mu.Unlock(); text.WriteString(value); publish() }
	reset := func() { mu.Lock(); defer mu.Unlock(); text.Reset(); publish() }
	reset()
	return chunk, reset
}

// appendReviewStageSnapshots 轮询与SSE重连共用相同阶段缓存，结果仍受原有任务权限检查约束。
func appendReviewStageSnapshots(ctx context.Context, rdb *redis.Client, module string, id uuid.UUID, out map[string]interface{}) {
	if rdb == nil {
		return
	}
	for stage, field := range map[string]string{"structured": "structured_output", "recheck": "recheck_output"} {
		if value, err := rdb.Get(ctx, module+":"+stage+":"+id.String()).Result(); err == nil {
			out[field] = value
		}
	}
}

// pendingAIRecheck 从本次绑定的不可变版本读取开关，让进度从入队起显示待复核。
func pendingAIRecheck(version *model.ExecutionConfigVersion) datatypes.JSON {
	if version == nil {
		return datatypes.JSON(`{}`)
	}
	var snapshot struct {
		AIConfig struct {
			Recheck *model.AIRecheckConfig `json:"recheck"`
		} `json:"ai_config"`
	}
	if json.Unmarshal(version.ConfigSnapshot, &snapshot) == nil && snapshot.AIConfig.Recheck != nil && snapshot.AIConfig.Recheck.Enabled {
		return datatypes.JSON(`{"enabled":true}`)
	}
	return datatypes.JSON(`{}`)
}
