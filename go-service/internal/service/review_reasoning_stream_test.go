package service

import (
	"context"
	"encoding/json"
	"testing"

	"auraoa/go-service/internal/pkg/ai"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// reviewStreamRecorder 截获缓存与广播命令，验证协议而无需连接真实 Redis。
type reviewStreamRecorder struct {
	cache  map[string]string
	events []JobStreamEvent
}

func (r *reviewStreamRecorder) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (r *reviewStreamRecorder) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(_ context.Context, cmd redis.Cmder) error {
		args := cmd.Args()
		switch cmd.Name() {
		case "set":
			r.cache[args[1].(string)] = args[2].(string)
		case "publish":
			var event JobStreamEvent
			if err := json.Unmarshal([]byte(args[2].(string)), &event); err != nil {
				return err
			}
			r.events = append(r.events, event)
		}
		return nil
	}
}

func (r *reviewStreamRecorder) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

func TestReviewReasoningStreamUsesSnapshotsAndResetsBothChannels(t *testing.T) {
	for _, module := range []string{"audit", "archive"} {
		t.Run(module, func(t *testing.T) {
			recorder := &reviewStreamRecorder{cache: map[string]string{}}
			rdb := redis.NewClient(&redis.Options{})
			t.Cleanup(func() { _ = rdb.Close() })
			rdb.AddHook(recorder)
			id := uuid.New()
			req := &ai.ChatRequest{}
			bindReviewReasoningStream(req, rdb, module, id, false)
			req.StreamChunkFunc("规则核验")
			req.StreamChunkFunc("：通过")
			req.StreamReasoningChunkFunc("旧尝试的思考")
			req.StreamChunkFunc("")
			req.StreamReasoningChunkFunc("")
			assertSnapshot := func(stage, event, want string) {
				t.Helper()
				if got := recorder.cache[module+":"+stage+":"+id.String()]; got != want {
					t.Fatalf("%s 缓存：%q，期望 %q", stage, got, want)
				}
				for i := len(recorder.events) - 1; i >= 0; i-- {
					if recorder.events[i].Event == event {
						if recorder.events[i].Data != want {
							t.Fatalf("%s 广播必须是累计全文：%q", event, recorder.events[i].Data)
						}
						return
					}
				}
				t.Fatalf("缺少 %s 广播", event)
			}
			assertSnapshot("reasoning", "message", "规则核验：通过")
			assertSnapshot("thinking", "thinking", "旧尝试的思考")
			req.StreamResetFunc()
			assertSnapshot("reasoning", "message", "")
			assertSnapshot("thinking", "thinking", "")
			req.StreamChunkFunc("## 规则核验\n通过")
			req.StreamReasoningChunkFunc("本轮思考")
			req.StreamChunkFunc("")
			req.StreamReasoningChunkFunc("")
			assertSnapshot("reasoning", "message", "## 规则核验\n通过")
			assertSnapshot("thinking", "thinking", "本轮思考")
		})
	}
}
