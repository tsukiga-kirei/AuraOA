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
func (a executionAdapter) FetchMyRequestsPaged(context.Context, string, oa.MyRequestPagedFilter) (*oa.PagedResult[oa.MyRequestItem], error) {
	return &oa.PagedResult[oa.MyRequestItem]{
		Items: []oa.MyRequestItem{
			{ProcessID: "1001", Title: "采购申请", Status: "流转中"},
		},
		Total: 1,
	}, nil
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

func TestExecuteListMyRequests(t *testing.T) {
	executor := &SystemToolExecutor{}
	ctx := &ExecutionContext{Ctx: context.Background(), Username: "user"}
	res, uiKind, err := executor.executeListMyRequests(`{"status":"processing"}`, ctx, nil, executionAdapter{})
	if err != nil {
		t.Fatalf("executeListMyRequests failed: %v", err)
	}
	if uiKind != "my_request_list" {
		t.Fatalf("expected uiKind my_request_list, got %s", uiKind)
	}
	m := res.(map[string]interface{})
	if m["total"] != 1 {
		t.Fatalf("expected total 1, got %v", m["total"])
	}
}

// 结果读取必须在查询数据库前拒绝未参与的流程及缺失的 OA 权限校验。
func TestResultToolsRequireParticipation(t *testing.T) {
	executor := &SystemToolExecutor{}
	ctx := &ExecutionContext{Ctx: context.Background(), Username: "user"}
	for _, code := range []string{"get_latest_audit", "get_latest_summary"} {
		if !BuiltinTools[code].OARequired {
			t.Fatalf("%s 必须初始化 OA 适配器", code)
		}
	}
	for _, run := range []func(string, *ExecutionContext, oa.OAAdapter) (interface{}, string, error){executor.executeGetLatestAudit, executor.executeGetLatestSummary} {
		for _, adapter := range []oa.OAAdapter{nil, executionAdapter{visible: false}} {
			if _, _, err := run(`{"process_id":"42"}`, ctx, adapter); err == nil {
				t.Fatal("缺少参与权限应在数据库查询前拒绝")
			}
		}
	}
}
