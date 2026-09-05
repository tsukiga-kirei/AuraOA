package agenttools

import (
	"auraoa/go-service/internal/pkg/oa"
	"context"
	"testing"
)

type executionAdapter struct {
	oa.OAAdapter
	visible bool
}

func (a executionAdapter) CheckProcessVisibility(context.Context, string, string) (bool, error) {
	return a.visible, nil
}
func (a executionAdapter) FetchProcessRequestSummary(context.Context, string) (*oa.ProcessRequestSummary, error) {
	return &oa.ProcessRequestSummary{ProcessID: "42", ProcessType: "expense", Title: "费用报销"}, nil
}

func TestExecutionToolsCallRealServiceAndRespectVisibility(t *testing.T) {
	count := 0
	callback := func(ctx *ExecutionContext, process *oa.ProcessRequestSummary) (interface{}, error) {
		count++
		if process.ProcessType != "expense" {
			t.Fatal("未从 OA 获取流程类型")
		}
		return map[string]string{"id": "real-job"}, nil
	}
	executor := &SystemToolExecutor{RunAudit: callback, RunSummary: callback}
	ctx := &ExecutionContext{Ctx: context.Background(), Username: "user"}
	for _, run := range []func(string, *ExecutionContext, oa.OAAdapter) (interface{}, string, error){executor.executeRunAudit, executor.executeRunSummary} {
		result, _, err := run(`{"process_id":"42"}`, ctx, executionAdapter{visible: true})
		if err != nil || result.(map[string]string)["id"] != "real-job" {
			t.Fatal(result, err)
		}
		if _, _, err := run(`{"process_id":"42"}`, ctx, executionAdapter{visible: false}); err == nil {
			t.Fatal("不可见流程应拒绝执行")
		}
	}
	if count != 2 {
		t.Fatalf("错误调用次数 %d", count)
	}
}
