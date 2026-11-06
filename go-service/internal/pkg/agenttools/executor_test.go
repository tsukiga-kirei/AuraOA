package agenttools

import (
	"context"
	"testing"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/oa"
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
func (a executionAdapter) ResolveProcessID(ctx context.Context, identifier string) (string, error) {
	if identifier == "CODE-123" {
		return "1001", nil
	}
	return identifier, nil
}
func (a executionAdapter) FetchWorkflowCodes(ctx context.Context, processIDs []string) (map[string]string, error) {
	res := make(map[string]string)
	for _, id := range processIDs {
		if id == "1001" {
			res[id] = "CODE-123"
		}
	}
	return res, nil
}
func (a executionAdapter) FetchProcessData(ctx context.Context, processID string) (*oa.ProcessData, error) {
	return &oa.ProcessData{
		ProcessID: processID,
		MainData:  map[string]interface{}{"field1": "val1"},
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

func TestExecuteGetProcessWithWorkflowCode(t *testing.T) {
	executor := &SystemToolExecutor{}
	ctx := &ExecutionContext{Ctx: context.Background(), Username: "user"}
	adapt := executionAdapter{visible: true}

	// 1. 通过流程编号反查并获取详情，验证输出 workflow_code
	res, uiKind, err := executor.executeGetProcess(`{"process_id":"CODE-123"}`, ctx, adapt)
	if err != nil {
		t.Fatalf("executeGetProcess failed: %v", err)
	}
	if uiKind != "process_detail" {
		t.Fatalf("expected uiKind process_detail, got %s", uiKind)
	}
	m := res.(map[string]interface{})
	if m["process_id"] != "1001" {
		t.Fatalf("expected resolved process_id 1001, got %v", m["process_id"])
	}
	if m["workflow_code"] != "CODE-123" {
		t.Fatalf("expected workflow_code CODE-123, got %v", m["workflow_code"])
	}

	// 2. 没有流程编号的流程，确保“有值放，没有值不放”
	res2, _, err := executor.executeGetProcess(`{"process_id":"9999"}`, ctx, adapt)
	if err != nil {
		t.Fatalf("executeGetProcess for 9999 failed: %v", err)
	}
	m2 := res2.(map[string]interface{})
	if _, exists := m2["workflow_code"]; exists {
		t.Fatalf("expected no workflow_code field when empty, got %v", m2["workflow_code"])
	}
}

type draftCreatorAdapter struct {
	oa.OAAdapter
	calledDraft *oa.CreateWorkflowDraftInput
}

func (d *draftCreatorAdapter) CreateWorkflowDraft(ctx context.Context, loginID string, draft *oa.CreateWorkflowDraftInput) (*oa.CreateWorkflowDraftOutput, error) {
	d.calledDraft = draft
	return &oa.CreateWorkflowDraftOutput{
		RequestID:   "8848",
		RequestName: draft.RequestName,
		ProcessURL:  "http://oa.example.com/workflow/request/ViewRequestForwardSPA.jsp?requestid=8848",
	}, nil
}

func TestGetAllToolSpecsCoversBuiltinTools(t *testing.T) {
	specs := GetAllToolSpecs()
	seen := make(map[string]bool, len(specs))
	for _, spec := range specs {
		if seen[spec.Code] {
			t.Fatalf("系统工具目录重复登记 %s", spec.Code)
		}
		seen[spec.Code] = true
	}
	for code := range BuiltinTools {
		if !seen[code] {
			t.Fatalf("能力装配目录缺少系统工具 %s", code)
		}
	}
	if len(seen) != len(BuiltinTools) {
		t.Fatalf("目录数量 %d 与内置工具 %d 不一致", len(seen), len(BuiltinTools))
	}
}

func TestExecuteCreateOADraft(t *testing.T) {
	executor := &SystemToolExecutor{}
	ctx := &ExecutionContext{Ctx: context.Background(), Username: "zhangsan"}
	mockAdapter := &draftCreatorAdapter{}

	argsJSON := `{
		"workflow_id": "102",
		"workflow_name": "物品领用申请",
		"request_name": "张三-领用文具-20261103",
		"request_level": "0",
		"main_data": {"sqr": "1058", "reason": "日常办公文具"}
	}`

	res, uiKind, err := executor.executeCreateOADraft(argsJSON, ctx, nil, mockAdapter)
	if err != nil {
		t.Fatalf("executeCreateOADraft failed: %v", err)
	}
	if uiKind != "create_workflow_draft" {
		t.Fatalf("expected uiKind create_workflow_draft, got %s", uiKind)
	}
	m := res.(map[string]interface{})
	if m["request_id"] != "8848" {
		t.Fatalf("expected request_id 8848, got %v", m["request_id"])
	}
	if m["process_url"] == "" {
		t.Fatalf("expected non-empty process_url")
	}
	if mockAdapter.calledDraft == nil || mockAdapter.calledDraft.WorkflowID != "102" {
		t.Fatalf("mock adapter did not receive expected workflow_id 102")
	}
}

func TestExecuteSafeQuerySecurityGuards(t *testing.T) {
	executor := &SystemToolExecutor{}
	ctx := &ExecutionContext{Ctx: context.Background(), Username: "user"}
	conn := &model.OADatabaseConnection{
		Driver: "mysql",
		OAType: "weaver_e9",
	}

	// 1. 非 SELECT 语句拦截
	badSQLs := []string{
		"UPDATE hrmresource SET lastname='hacked'",
		"DELETE FROM workflow_requestbase",
		"DROP TABLE hrmdepartment",
		"INSERT INTO workflow_requestbase (requestid) VALUES (1)",
		"ALTER TABLE hrmresource DROP COLUMN password",
		"TRUNCATE TABLE hrmresource",
	}
	for _, sql := range badSQLs {
		_, _, err := executor.executeSafeQuery(`{"sql":"`+sql+`"}`, ctx, conn)
		if err == nil {
			t.Fatalf("SQL %s 应被拦截，但未报错", sql)
		}
	}

	// 2. 注入注释符拦截
	commentSQLs := []string{
		"SELECT * FROM hrmresource WHERE 1=1 -- 注释绕过",
		"SELECT * FROM hrmresource /* 多行注释 */ WHERE 1=1",
	}
	for _, sql := range commentSQLs {
		_, _, err := executor.executeSafeQuery(`{"sql":"`+sql+`"}`, ctx, conn)
		if err == nil {
			t.Fatalf("包含注释符的 SQL %s 应被拦截，但未报错", sql)
		}
	}

	// 3. 敏感凭证字段拦截
	sensitiveSQLs := []string{
		"SELECT password FROM hrmresource",
		"SELECT user_token FROM sys_tokens",
		"SELECT secret FROM ecology_biz_ec",
		"SELECT license_key FROM sys_license",
	}
	for _, sql := range sensitiveSQLs {
		_, _, err := executor.executeSafeQuery(`{"sql":"`+sql+`"}`, ctx, conn)
		if err == nil {
			t.Fatalf("包含敏感词的 SQL %s 应被拦截，但未报错", sql)
		}
	}
}

type mockSchemaAdapter struct {
	oa.OAAdapter
}

func (m *mockSchemaAdapter) SearchWorkflowDefinitions(ctx context.Context, keyword string) ([]oa.ProcessInfo, error) {
	return []oa.ProcessInfo{
		{
			WorkflowID:  "102",
			ProcessName: "物品领用申请",
		},
	}, nil
}

func (m *mockSchemaAdapter) FetchFieldsByWorkflowID(ctx context.Context, workflowID string) (*oa.ProcessFields, error) {
	return &oa.ProcessFields{
		MainFields: []oa.FieldDef{
			{
				FieldKey:  "sqr",
				FieldName: "申请人",
				FieldType: "browser",
			},
			{
				FieldKey:  "wplx",
				FieldName: "物品类型",
				FieldType: "select",
				Options: []oa.FieldChoiceOption{
					{Value: "0", Label: "办公文具"},
					{Value: "1", Label: "数码耗材"},
				},
			},
		},
	}, nil
}

func TestExecuteGetWorkflowSchema(t *testing.T) {
	executor := &SystemToolExecutor{}
	ctx := &ExecutionContext{Ctx: context.Background(), Username: "zhangsan"}
	mockAdapter := &mockSchemaAdapter{}

	// 按流程名称搜索并拉取表单选项
	res, uiKind, err := executor.executeGetWorkflowSchema(`{"workflow_name":"领用"}`, ctx, nil, mockAdapter)
	if err != nil {
		t.Fatalf("executeGetWorkflowSchema failed: %v", err)
	}
	if uiKind != "workflow_schema" {
		t.Fatalf("expected uiKind workflow_schema, got %s", uiKind)
	}
	m := res.(map[string]interface{})
	if m["workflow_id"] != "102" {
		t.Fatalf("expected workflow_id 102, got %v", m["workflow_id"])
	}
	mainFields := m["main_fields"].([]oa.FieldDef)
	if len(mainFields) != 2 {
		t.Fatalf("expected 2 main fields, got %d", len(mainFields))
	}
	if mainFields[1].FieldKey != "wplx" || len(mainFields[1].Options) != 2 {
		t.Fatalf("wplx options not populated properly: %+v", mainFields[1])
	}
}

type permissionSchemaAdapter struct {
	mockSchemaAdapter
	items []oa.CreatableWorkflow
}

func (p *permissionSchemaAdapter) ListCreatableWorkflows(context.Context, string, oa.CreatableWorkflowQuery) ([]oa.CreatableWorkflow, error) {
	return p.items, nil
}

func TestGetWorkflowSchemaRequiresCreatePermission(t *testing.T) {
	executor := &SystemToolExecutor{}
	ctx := &ExecutionContext{Ctx: context.Background(), Username: "zhangsan"}

	denied := &permissionSchemaAdapter{}
	res, _, err := executor.executeGetWorkflowSchema(`{"workflow_name":"领用"}`, ctx, nil, denied)
	if err != nil {
		t.Fatalf("无权时应返回未匹配结果，而不是错误: %v", err)
	}
	if res.(map[string]interface{})["matched"] != false {
		t.Fatal("没有创建权限时不应读取表单")
	}
	if _, ok := res.(map[string]interface{})["main_fields"]; ok {
		t.Fatal("没有创建权限时不应返回表单字段")
	}

	allowed := &permissionSchemaAdapter{items: []oa.CreatableWorkflow{{
		WorkflowID:   "102",
		WorkflowName: "物品领用申请",
	}}}
	res, _, err = executor.executeGetWorkflowSchema(`{"workflow_name":"领用"}`, ctx, nil, allowed)
	if err != nil {
		t.Fatalf("有权限时读取表单失败: %v", err)
	}
	if res.(map[string]interface{})["workflow_id"] != "102" {
		t.Fatalf("应使用权限列表中的流程，得到 %v", res)
	}

	multiple := &permissionSchemaAdapter{items: []oa.CreatableWorkflow{
		{WorkflowID: "102", WorkflowName: "物品领用申请"},
		{WorkflowID: "103", WorkflowName: "固定资产领用"},
	}}
	res, _, err = executor.executeGetWorkflowSchema(`{"workflow_name":"领用"}`, ctx, nil, multiple)
	if err != nil {
		t.Fatalf("多条匹配不应报错: %v", err)
	}
	got := res.(map[string]interface{})
	if got["matched"] != false {
		t.Fatal("多条可创建流程时应先让用户确认，不读取表单")
	}
	if len(got["workflows"].([]map[string]string)) != 2 {
		t.Fatalf("应返回两条候选流程，得到 %#v", got["workflows"])
	}
}
