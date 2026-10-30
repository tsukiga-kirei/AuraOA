package service

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/ai"
	"auraoa/go-service/internal/pkg/sanitize"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/datatypes"
)

// newReviewStageStream 按阶段发送完整累计文本，断线重连及重试时不会混入上一轮半截JSON。
func newReviewStageStream(rdb *redis.Client, module string, id uuid.UUID, stage string, scrub bool) (func(string), func()) {
	return newReviewStageAccumulator(func(value string) {
		if scrub {
			value = sanitize.SanitizeText(value)
		}
		if rdb != nil {
			ctx := context.Background()
			rdb.Set(ctx, module+":"+stage+":"+id.String(), value, 24*time.Hour)
			event := stage
			if stage == "reasoning" {
				event = "message"
			}
			payload, _ := json.Marshal(JobStreamEvent{Event: event, Data: value})
			rdb.Publish(ctx, module+":stream:"+id.String(), string(payload))
		}
	})
}

// newReviewStageAccumulator 只管理阶段累计与重试清空，空增量用于立即刷新末尾。
func newReviewStageAccumulator(emit func(string)) (func(string), func()) {
	var mu sync.Mutex
	var text strings.Builder
	var lastPublished time.Time
	publish := func() { emit(text.String()); lastPublished = time.Now() }

	chunk := func(value string) {
		mu.Lock()
		defer mu.Unlock()
		text.WriteString(value)
		if value == "" || lastPublished.IsZero() || time.Since(lastPublished) >= 80*time.Millisecond {
			publish()
		}
	}
	reset := func() { mu.Lock(); defer mu.Unlock(); text.Reset(); publish(); lastPublished = time.Time{} }
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

// bindReviewReasoningStream 推理与思考均发送累计快照，重试或切换模型时同步清空旧尝试。
func bindReviewReasoningStream(req *ai.ChatRequest, rdb *redis.Client, module string, id uuid.UUID, scrub bool) {
	contentChunk, contentReset := newReviewStageStream(rdb, module, id, "reasoning", scrub)
	thinkingChunk, thinkingReset := newReviewStageStream(rdb, module, id, "thinking", scrub)
	req.StreamChunkFunc = contentChunk
	req.StreamReasoningChunkFunc = thinkingChunk
	req.StreamResetFunc = func() { contentReset(); thinkingReset() }
}
