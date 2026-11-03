package agenttools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/apptime"
	"auraoa/go-service/internal/pkg/crypto"
	"auraoa/go-service/internal/pkg/oa"
	"auraoa/go-service/internal/repository"
)

// ExecutionContext 工具调用执行上下文
type ExecutionContext struct {
	Ctx      context.Context
	GinCtx   *gin.Context
	TenantID uuid.UUID
	UserID   uuid.UUID
	Username string
}

// ToolExecutor 系统工具执行器接口
type ToolExecutor interface {
	Execute(toolCode string, argumentsJSON string, execCtx *ExecutionContext) (interface{}, string, error)
}

// SystemToolExecutor 系统工具调度与执行器实现
type SystemToolExecutor struct {
	RunAudit      func(*ExecutionContext, *oa.ProcessRequestSummary) (interface{}, error)
	RunSummary    func(*ExecutionContext, *oa.ProcessRequestSummary) (interface{}, error)
	db            *gorm.DB
	tenantRepo    *repository.TenantRepo
	oaConnRepo    *repository.OAConnectionRepo
	oaConnections *oa.ConnectionManager
	auditLogRepo  *repository.AuditLogRepo
	summaryRepo   *repository.ProcessSummaryLogRepo
}

// NewSystemToolExecutor 创建系统工具执行器
func NewSystemToolExecutor(
	db *gorm.DB,
	tenantRepo *repository.TenantRepo,
	oaConnRepo *repository.OAConnectionRepo,
	oaConnections *oa.ConnectionManager,
	auditLogRepo *repository.AuditLogRepo,
	summaryRepo *repository.ProcessSummaryLogRepo,
) *SystemToolExecutor {
	return &SystemToolExecutor{
		db:            db,
		tenantRepo:    tenantRepo,
		oaConnRepo:    oaConnRepo,
		oaConnections: oaConnections,
		auditLogRepo:  auditLogRepo,
		summaryRepo:   summaryRepo,
	}
}

// Execute 执行具体的工具调用
func (e *SystemToolExecutor) Execute(toolCode string, argumentsJSON string, execCtx *ExecutionContext) (interface{}, string, error) {
	spec, ok := BuiltinTools[toolCode]
	if !ok {
		return nil, "mcp_generic", fmt.Errorf("未知的系统工具: %s", toolCode)
	}

	// 1. OA 连接前置校验
	var tenant *model.Tenant
	var oaConn *model.OADatabaseConnection
	var adapter oa.OAAdapter

	if spec.OARequired || toolCode == "resolve_oa_url" {
		var err error
		tenant, err = e.tenantRepo.FindByID(execCtx.TenantID)
		if err != nil || tenant == nil {
			return nil, spec.UIKind, fmt.Errorf("租户不存在")
		}
		if tenant.OADBConnectionID == nil {
			return nil, spec.UIKind, fmt.Errorf("当前租户尚未配置 OA 数据库连接，无法执行 OA 相关工具")
		}
		oaConn, err = e.oaConnRepo.FindByID(*tenant.OADBConnectionID)
		if err != nil || oaConn == nil {
			return nil, spec.UIKind, fmt.Errorf("未找到租户绑定的 OA 连接")
		}
		if oaConn.Password != "" {
			decrypted, decryptErr := crypto.Decrypt(oaConn.Password)
			if decryptErr != nil {
				return nil, spec.UIKind, fmt.Errorf("OA 数据库密码解密失败: %w", decryptErr)
			}
			oaConn.Password = decrypted
		}
		if oaConn.WeaverAppSecret != "" {
			decryptedSecret, decryptErr := crypto.Decrypt(oaConn.WeaverAppSecret)
			if decryptErr == nil {
				oaConn.WeaverAppSecret = decryptedSecret
			}
		}
		if spec.OARequired {
			adapter, err = e.oaConnections.GetAdapter(execCtx.Ctx, oaConn.OAType, oaConn)
			if err != nil {
				return nil, spec.UIKind, fmt.Errorf("获取 OA 适配器失败: %w", err)
			}
		}
	}

	// 2. 分发具体执行
	switch toolCode {
	case "list_my_todos":
		return e.executeListMyTodos(argumentsJSON, execCtx, oaConn, adapter)
	case "list_my_requests":
		return e.executeListMyRequests(argumentsJSON, execCtx, oaConn, adapter)
	case "get_process":
		return e.executeGetProcess(argumentsJSON, execCtx, adapter)
	case "get_approval_flow":
		return e.executeGetApprovalFlow(argumentsJSON, execCtx, adapter)
	case "get_latest_audit":
		return e.executeGetLatestAudit(argumentsJSON, execCtx, adapter)
	case "get_latest_summary":
		return e.executeGetLatestSummary(argumentsJSON, execCtx, adapter)
	case "draft_comment":
		return e.executeDraftComment(argumentsJSON, execCtx, oaConn, adapter)
	case "run_audit":
		return e.executeRunAudit(argumentsJSON, execCtx, adapter)
	case "run_summary":
		return e.executeRunSummary(argumentsJSON, execCtx, adapter)
	case "resolve_oa_url":
		return e.executeResolveOAURL(argumentsJSON, execCtx, oaConn)
	case "create_oa_draft":
		return e.executeCreateOADraft(argumentsJSON, execCtx, oaConn, adapter)
	case "execute_safe_query":
		return e.executeSafeQuery(argumentsJSON, execCtx, oaConn)
	case "get_workflow_schema":
		return e.executeGetWorkflowSchema(argumentsJSON, execCtx, adapter)
	default:
		return nil, spec.UIKind, fmt.Errorf("工具 %s 暂未实现", toolCode)
	}
}

// 1. 查询我的待办
func (e *SystemToolExecutor) executeListMyTodos(
	argsJSON string,
	execCtx *ExecutionContext,
	oaConn *model.OADatabaseConnection,
	adapter oa.OAAdapter,
) (interface{}, string, error) {
	var args struct {
		Keyword    string `json:"keyword"`
		Applicant  string `json:"applicant"`
		Department string `json:"department"`
		Page       int    `json:"page"`
		PageSize   int    `json:"page_size"`
	}
	_ = json.Unmarshal([]byte(argsJSON), &args)
	if args.Page <= 0 {
		args.Page = 1
	}
	if args.PageSize <= 0 {
		args.PageSize = 20
	}
	if args.PageSize > 50 {
		args.PageSize = 50
	}

	filter := oa.TodoListPagedFilter{
		Keyword:    args.Keyword,
		Applicant:  args.Applicant,
		Department: args.Department,
		Page:       args.Page,
		PageSize:   args.PageSize,
	}

	pagedResult, err := adapter.FetchTodoListPaged(execCtx.Ctx, execCtx.Username, filter)
	if err != nil {
		return nil, "todo_list", fmt.Errorf("拉取待办列表失败: %w", err)
	}

	type TodoItemPayload struct {
		ProcessID    string `json:"process_id"`
		WorkflowCode string `json:"workflow_code,omitempty"`
		Title        string `json:"title"`
		Applicant    string `json:"applicant"`
		Department   string `json:"department"`
		CurrentNode  string `json:"current_node"`
		SubmitTime   string `json:"submit_time"`
		OAURL        string `json:"oa_url,omitempty"`
		TodoType     string `json:"todo_type,omitempty"`
	}

	pids := make([]string, 0, len(pagedResult.Items))
	for _, item := range pagedResult.Items {
		pids = append(pids, item.ProcessID)
	}
	var codes map[string]string
	if adapter != nil && len(pids) > 0 {
		codes, _ = adapter.FetchWorkflowCodes(execCtx.Ctx, pids)
	}

	items := make([]TodoItemPayload, 0, len(pagedResult.Items))
	for _, item := range pagedResult.Items {
		oaURL := ""
		if oaConn != nil {
			oaURL = oa.BuildProcessURL(oaConn.OABaseURL, oaConn.ProcessURLTemplate, oaConn.OAType, item.ProcessID)
		}
		var code string
		if codes != nil {
			code = codes[item.ProcessID]
		}
		items = append(items, TodoItemPayload{
			ProcessID:    item.ProcessID,
			WorkflowCode: code,
			Title:        item.Title,
			Applicant:    item.Applicant,
			Department:   item.Department,
			CurrentNode:  item.CurrentNode,
			SubmitTime:   item.SubmitTime,
			OAURL:        oaURL,
			TodoType:     item.TodoType,
		})
	}

	return map[string]interface{}{
		"items":     items,
		"total":     pagedResult.Total,
		"page":      args.Page,
		"page_size": args.PageSize,
	}, "todo_list", nil
}

// 查询我发起的流程
func (e *SystemToolExecutor) executeListMyRequests(
	argsJSON string,
	execCtx *ExecutionContext,
	oaConn *model.OADatabaseConnection,
	adapter oa.OAAdapter,
) (interface{}, string, error) {
	var args struct {
		Keyword  string `json:"keyword"`
		Status   string `json:"status"` // all | processing | archived
		Page     int    `json:"page"`
		PageSize int    `json:"page_size"`
	}
	_ = json.Unmarshal([]byte(argsJSON), &args)
	if args.Page <= 0 {
		args.Page = 1
	}
	if args.PageSize <= 0 {
		args.PageSize = 20
	}
	if args.PageSize > 50 {
		args.PageSize = 50
	}
	if args.Status == "" {
		args.Status = "all"
	}

	filter := oa.MyRequestPagedFilter{
		Keyword:  args.Keyword,
		Status:   args.Status,
		Page:     args.Page,
		PageSize: args.PageSize,
	}

	pagedResult, err := adapter.FetchMyRequestsPaged(execCtx.Ctx, execCtx.Username, filter)
	if err != nil {
		return nil, "my_request_list", fmt.Errorf("拉取我发起的流程失败: %w", err)
	}

	type MyRequestPayload struct {
		ProcessID        string `json:"process_id"`
		WorkflowCode     string `json:"workflow_code,omitempty"`
		Title            string `json:"title"`
		ProcessType      string `json:"process_type"`
		ProcessTypeLabel string `json:"process_type_label"`
		CurrentNode      string `json:"current_node"`
		SubmitTime       string `json:"submit_time"`
		Status           string `json:"status"`
		OAURL            string `json:"oa_url,omitempty"`
	}

	pids := make([]string, 0, len(pagedResult.Items))
	for _, item := range pagedResult.Items {
		pids = append(pids, item.ProcessID)
	}
	var codes map[string]string
	if adapter != nil && len(pids) > 0 {
		codes, _ = adapter.FetchWorkflowCodes(execCtx.Ctx, pids)
	}

	items := make([]MyRequestPayload, 0, len(pagedResult.Items))
	for _, item := range pagedResult.Items {
		oaURL := ""
		if oaConn != nil {
			oaURL = oa.BuildProcessURL(oaConn.OABaseURL, oaConn.ProcessURLTemplate, oaConn.OAType, item.ProcessID)
		}
		var code string
		if codes != nil {
			code = codes[item.ProcessID]
		}
		items = append(items, MyRequestPayload{
			ProcessID:        item.ProcessID,
			WorkflowCode:     code,
			Title:            item.Title,
			ProcessType:      item.ProcessType,
			ProcessTypeLabel: item.ProcessTypeLabel,
			CurrentNode:      item.CurrentNode,
			SubmitTime:       item.SubmitTime,
			Status:           item.Status,
			OAURL:            oaURL,
		})
	}

	return map[string]interface{}{
		"items":     items,
		"total":     pagedResult.Total,
		"page":      args.Page,
		"page_size": args.PageSize,
	}, "my_request_list", nil
}

// 2. 获取流程详情
func (e *SystemToolExecutor) executeGetProcess(
	argsJSON string,
	execCtx *ExecutionContext,
	adapter oa.OAAdapter,
) (interface{}, string, error) {
	var args struct {
		ProcessID string `json:"process_id"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil || strings.TrimSpace(args.ProcessID) == "" {
		return nil, "process_detail", fmt.Errorf("缺少必填参数 process_id")
	}

	targetID := strings.TrimSpace(args.ProcessID)
	if adapter != nil {
		if realID, err := adapter.ResolveProcessID(execCtx.Ctx, targetID); err == nil && realID != "" {
			targetID = realID
		}
	}

	// 校验可见性
	visible, err := adapter.CheckProcessVisibility(execCtx.Ctx, execCtx.Username, targetID)
	if err != nil || !visible {
		return nil, "process_detail", fmt.Errorf("无权访问流程 %s 或流程不存在", args.ProcessID)
	}

	data, err := adapter.FetchProcessData(execCtx.Ctx, targetID)
	if err != nil {
		return nil, "process_detail", fmt.Errorf("读取流程数据失败: %w", err)
	}

	summary, _ := adapter.FetchProcessRequestSummary(execCtx.Ctx, targetID)
	title := targetID
	applicant := ""
	department := ""
	processType := ""
	if summary != nil {
		if summary.Title != "" {
			title = summary.Title
		}
		applicant = summary.Applicant
		department = summary.Department
		processType = summary.ProcessType
	}

	mainFields := make([]map[string]interface{}, 0, len(data.MainData))
	for k, v := range data.MainData {
		mainFields = append(mainFields, map[string]interface{}{
			"field": k,
			"value": fmt.Sprintf("%v", v),
		})
	}

	attachments := make([]string, 0, len(data.Attachments))
	for _, att := range data.Attachments {
		attachments = append(attachments, att.FileName)
	}

	resp := map[string]interface{}{
		"process_id":    targetID,
		"title":         title,
		"process_type":  processType,
		"applicant":     applicant,
		"department":    department,
		"main_fields":   mainFields,
		"detail_tables": data.DetailTables,
		"attachments":   attachments,
	}

	// 动态装配流程编号（有值放，没有值不放）
	if adapter != nil {
		if codes, err := adapter.FetchWorkflowCodes(execCtx.Ctx, []string{targetID}); err == nil {
			if code := strings.TrimSpace(codes[targetID]); code != "" {
				resp["workflow_code"] = code
			}
		}
	}

	return resp, "process_detail", nil
}

// 3. 获取审批流
func (e *SystemToolExecutor) executeGetApprovalFlow(
	argsJSON string,
	execCtx *ExecutionContext,
	adapter oa.OAAdapter,
) (interface{}, string, error) {
	var args struct {
		ProcessID string `json:"process_id"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil || strings.TrimSpace(args.ProcessID) == "" {
		return nil, "approval_flow", fmt.Errorf("缺少必填参数 process_id")
	}

	targetID := strings.TrimSpace(args.ProcessID)
	if adapter != nil {
		if realID, err := adapter.ResolveProcessID(execCtx.Ctx, targetID); err == nil && realID != "" {
			targetID = realID
		}
	}

	// 可见性校验
	visible, err := adapter.CheckProcessVisibility(execCtx.Ctx, execCtx.Username, targetID)
	if err != nil || !visible {
		return nil, "approval_flow", fmt.Errorf("无权访问流程 %s 或流程不存在", args.ProcessID)
	}

	flow, err := adapter.FetchProcessFlow(execCtx.Ctx, targetID)
	if err != nil {
		return nil, "approval_flow", fmt.Errorf("获取审批流失败: %w", err)
	}

	type FlowNodePayload struct {
		NodeName     string `json:"node_name"`
		OperatorName string `json:"operator_name"`
		Action       string `json:"action"`
		Opinion      string `json:"opinion"`
		OperatedAt   string `json:"operated_at"`
	}

	nodes := make([]FlowNodePayload, 0, len(flow.Nodes))
	for _, n := range flow.Nodes {
		nodes = append(nodes, FlowNodePayload{
			NodeName:     n.NodeName,
			OperatorName: n.Approver,
			Action:       n.Action,
			Opinion:      n.Opinion,
			OperatedAt:   n.ActionTime,
		})
	}

	resp := map[string]interface{}{
		"process_id": targetID,
		"nodes":      nodes,
	}

	if adapter != nil {
		if codes, err := adapter.FetchWorkflowCodes(execCtx.Ctx, []string{targetID}); err == nil {
			if code := strings.TrimSpace(codes[targetID]); code != "" {
				resp["workflow_code"] = code
			}
		}
	}

	return resp, "approval_flow", nil
}

// 4. 获取最新审核结果 (支持单条 process_id 或批量 process_ids)
func (e *SystemToolExecutor) executeGetLatestAudit(
	argsJSON string,
	execCtx *ExecutionContext,
	adapter oa.OAAdapter,
) (interface{}, string, error) {
	var args struct {
		ProcessID  string   `json:"process_id"`
		ProcessIDs []string `json:"process_ids"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil, "audit_result", fmt.Errorf("解析参数失败: %w", err)
	}

	// 收集待查询的目标流程 ID
	targetIDs := make([]string, 0)
	if strings.TrimSpace(args.ProcessID) != "" {
		targetIDs = append(targetIDs, strings.TrimSpace(args.ProcessID))
	}
	for _, pid := range args.ProcessIDs {
		pid = strings.TrimSpace(pid)
		if pid != "" && !containsString(targetIDs, pid) {
			targetIDs = append(targetIDs, pid)
		}
	}
	if len(targetIDs) == 0 {
		return nil, "audit_result", fmt.Errorf("缺少必填参数 process_id 或 process_ids")
	}

	if adapter == nil {
		return nil, "audit_result", fmt.Errorf("无法校验 OA 流程访问权限")
	}

	// 限制单次批量最大查询 50 条，防止超载
	if len(targetIDs) > 50 {
		targetIDs = targetIDs[:50]
	}

	// 单条流程查询，保持与之前完全一致的 DTO 结构，向后兼容
	if len(targetIDs) == 1 {
		pid := targetIDs[0]
		visible, err := adapter.CheckProcessVisibility(execCtx.Ctx, execCtx.Username, pid)
		if err != nil || !visible {
			return nil, "audit_result", fmt.Errorf("无权访问流程 %s 或流程不存在", pid)
		}
		return e.querySingleLatestAudit(execCtx, pid)
	}

	// 多条流程批量查询，直接用 IN (?) 批量一次性捞取审核记录
	items := make([]map[string]interface{}, 0, len(targetIDs))
	for _, pid := range targetIDs {
		visible, err := adapter.CheckProcessVisibility(execCtx.Ctx, execCtx.Username, pid)
		if err != nil || !visible {
			continue // 忽略无权访问或不存在的流程
		}
		single, _, err := e.querySingleLatestAudit(execCtx, pid)
		if err == nil && single != nil {
			if m, ok := single.(map[string]interface{}); ok {
				items = append(items, m)
			}
		}
	}

	return map[string]interface{}{
		"items": items,
		"total": len(items),
	}, "audit_result", nil
}

func (e *SystemToolExecutor) querySingleLatestAudit(
	execCtx *ExecutionContext,
	pid string,
) (interface{}, string, error) {
	var log model.AuditLog
	// 1. 优先查询当前用户的个人视角定制审核结论
	err := e.db.WithContext(execCtx.Ctx).
		Where("tenant_id = ? AND process_id = ? AND user_id = ? AND trigger_detail = 'personal_embed_manual' AND status = ? AND COALESCE(parse_error, '') = ''",
			execCtx.TenantID, pid, execCtx.UserID, model.JobStatusCompleted).
		Order("created_at DESC").
		First(&log).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 2. 其次查询当前用户在工作台执行的审核结论
		err = e.db.WithContext(execCtx.Ctx).
			Where("tenant_id = ? AND process_id = ? AND user_id = ? AND trigger_source NOT IN ('embed_auto', 'embed_manual') AND COALESCE(trigger_detail, '') != 'personal_embed_manual' AND status = ? AND COALESCE(parse_error, '') = ''",
				execCtx.TenantID, pid, execCtx.UserID, model.JobStatusCompleted).
			Order("created_at DESC").
			First(&log).Error
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 3. 兜底查询公共/通用嵌入审核结论（排除任何人的私有定制结论）
		err = e.db.WithContext(execCtx.Ctx).
			Where("tenant_id = ? AND process_id = ? AND trigger_source IN ('embed_auto', 'embed_manual') AND COALESCE(trigger_detail, '') != 'personal_embed_manual' AND status = ? AND COALESCE(parse_error, '') = ''",
				execCtx.TenantID, pid, model.JobStatusCompleted).
			Order("created_at DESC").
			First(&log).Error
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return map[string]interface{}{
			"process_id": pid,
			"has_audit":  false,
			"message":    "该流程暂无历史 AI 审核记录",
		}, "audit_result", nil
	}
	if err != nil {
		return nil, "audit_result", fmt.Errorf("查询审核记录失败: %w", err)
	}

	return map[string]interface{}{
		"process_id":     pid,
		"has_audit":      true,
		"status":         log.Status,
		"recommendation": log.Recommendation,
		"score":          log.Score,
		"audit_result":   log.AuditResult,
		"created_at":     apptime.FormatRFC3339(log.CreatedAt),
	}, "audit_result", nil
}

// 5. 获取最新流程总结 (支持单条 process_id 或批量 process_ids)
func (e *SystemToolExecutor) executeGetLatestSummary(
	argsJSON string,
	execCtx *ExecutionContext,
	adapter oa.OAAdapter,
) (interface{}, string, error) {
	var args struct {
		ProcessID  string   `json:"process_id"`
		ProcessIDs []string `json:"process_ids"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil, "summary_result", fmt.Errorf("解析参数失败: %w", err)
	}

	// 收集待查询的目标流程 ID
	targetIDs := make([]string, 0)
	if strings.TrimSpace(args.ProcessID) != "" {
		targetIDs = append(targetIDs, strings.TrimSpace(args.ProcessID))
	}
	for _, pid := range args.ProcessIDs {
		pid = strings.TrimSpace(pid)
		if pid != "" && !containsString(targetIDs, pid) {
			targetIDs = append(targetIDs, pid)
		}
	}
	if len(targetIDs) == 0 {
		return nil, "summary_result", fmt.Errorf("缺少必填参数 process_id 或 process_ids")
	}

	if adapter == nil {
		return nil, "summary_result", fmt.Errorf("无法校验 OA 流程访问权限")
	}

	if len(targetIDs) > 50 {
		targetIDs = targetIDs[:50]
	}

	if len(targetIDs) == 1 {
		pid := targetIDs[0]
		visible, err := adapter.CheckProcessVisibility(execCtx.Ctx, execCtx.Username, pid)
		if err != nil || !visible {
			return nil, "summary_result", fmt.Errorf("无权访问流程 %s 或流程不存在", pid)
		}
		return e.querySingleLatestSummary(execCtx, pid)
	}

	// 批量查询
	items := make([]map[string]interface{}, 0, len(targetIDs))
	for _, pid := range targetIDs {
		visible, err := adapter.CheckProcessVisibility(execCtx.Ctx, execCtx.Username, pid)
		if err != nil || !visible {
			continue
		}
		single, _, err := e.querySingleLatestSummary(execCtx, pid)
		if err == nil && single != nil {
			if m, ok := single.(map[string]interface{}); ok {
				items = append(items, m)
			}
		}
	}

	return map[string]interface{}{
		"items": items,
		"total": len(items),
	}, "summary_result", nil
}

func (e *SystemToolExecutor) querySingleLatestSummary(
	execCtx *ExecutionContext,
	pid string,
) (interface{}, string, error) {
	var summary model.ProcessSummaryLog
	// 1. 优先查询当前用户针对该流程的总结记录
	err := e.db.WithContext(execCtx.Ctx).
		Where("tenant_id = ? AND process_id = ? AND user_id = ? AND trigger_source = 'summary_workbench' AND status = ? AND COALESCE(parse_error, '') = ''",
			execCtx.TenantID, pid, execCtx.UserID, model.JobStatusCompleted).
		Order("created_at DESC").
		First(&summary).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 2. 兜底查询租户通用总结记录
		err = e.db.WithContext(execCtx.Ctx).
			Where("tenant_id = ? AND process_id = ? AND trigger_source IN ('summary_embed_auto', 'summary_embed_manual') AND status = ? AND COALESCE(parse_error, '') = ''",
				execCtx.TenantID, pid, model.JobStatusCompleted).
			Order("created_at DESC").
			First(&summary).Error
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return map[string]interface{}{
			"process_id":  pid,
			"has_summary": false,
			"message":     "该流程暂无历史总结记录",
		}, "summary_result", nil
	}
	if err != nil {
		return nil, "summary_result", fmt.Errorf("查询流程总结记录失败: %w", err)
	}

	return map[string]interface{}{
		"process_id":     pid,
		"has_summary":    true,
		"status":         summary.Status,
		"summary_result": summary.SummaryResult,
		"created_at":     apptime.FormatRFC3339(summary.CreatedAt),
	}, "summary_result", nil
}

// 6. 起草审批意见
func (e *SystemToolExecutor) executeDraftComment(
	argsJSON string,
	execCtx *ExecutionContext,
	oaConn *model.OADatabaseConnection,
	adapter oa.OAAdapter,
) (interface{}, string, error) {
	var args struct {
		ProcessID string `json:"process_id"`
		Intent    string `json:"intent"` // approve | return
		Note      string `json:"note"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil || args.ProcessID == "" {
		return nil, "opinion_draft", fmt.Errorf("缺少必填参数 process_id")
	}
	if args.Intent != "approve" && args.Intent != "return" {
		args.Intent = "approve"
	}

	// 校验可见性
	visible, err := adapter.CheckProcessVisibility(execCtx.Ctx, execCtx.Username, args.ProcessID)
	if err != nil || !visible {
		return nil, "opinion_draft", fmt.Errorf("无权访问流程 %s 或流程不存在", args.ProcessID)
	}

	// 读取流程摘要与最新审核建议（遵循个人优先防泄露原则）
	summary, _ := adapter.FetchProcessRequestSummary(execCtx.Ctx, args.ProcessID)
	var latestLog model.AuditLog
	err = e.db.WithContext(execCtx.Ctx).
		Where("tenant_id = ? AND process_id = ? AND user_id = ? AND trigger_detail = 'personal_embed_manual' AND status = ? AND COALESCE(parse_error, '') = ''",
			execCtx.TenantID, args.ProcessID, execCtx.UserID, model.JobStatusCompleted).
		Order("created_at DESC").
		First(&latestLog).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = e.db.WithContext(execCtx.Ctx).
			Where("tenant_id = ? AND process_id = ? AND user_id = ? AND trigger_source NOT IN ('embed_auto', 'embed_manual') AND COALESCE(trigger_detail, '') != 'personal_embed_manual' AND status = ? AND COALESCE(parse_error, '') = ''",
				execCtx.TenantID, args.ProcessID, execCtx.UserID, model.JobStatusCompleted).
			Order("created_at DESC").
			First(&latestLog).Error
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		_ = e.db.WithContext(execCtx.Ctx).
			Where("tenant_id = ? AND process_id = ? AND trigger_source IN ('embed_auto', 'embed_manual') AND COALESCE(trigger_detail, '') != 'personal_embed_manual' AND status = ? AND COALESCE(parse_error, '') = ''",
				execCtx.TenantID, args.ProcessID, model.JobStatusCompleted).
			Order("created_at DESC").
			First(&latestLog).Error
	}

	var draft strings.Builder
	if args.Intent == "approve" {
		draft.WriteString("【同意】经审核，该单据符合规范，同意办理。")
		if args.Note != "" {
			draft.WriteString(" 补充说明：" + args.Note)
		}
		if latestLog.Recommendation == "approve" {
			draft.WriteString("（AI 规则合规审核通过）")
		}
	} else {
		draft.WriteString("【退回】经审查，当前单据存在待完善事项，予以退回。")
		if args.Note != "" {
			draft.WriteString(" 退回原因：" + args.Note)
		} else if latestLog.Recommendation == "return" {
			draft.WriteString(" 退回原因：AI 合规审核未通过，请检查相关明细。")
		} else {
			draft.WriteString(" 请核实相关明细与附件材料后重新提交。")
		}
	}

	oaURL := ""
	if oaConn != nil {
		oaURL = oa.BuildProcessURL(oaConn.OABaseURL, oaConn.ProcessURLTemplate, oaConn.OAType, args.ProcessID)
	}

	flowTitle := args.ProcessID
	if summary != nil && summary.Title != "" {
		flowTitle = summary.Title
	}

	return map[string]interface{}{
		"process_id":    args.ProcessID,
		"process_title": flowTitle,
		"intent":        args.Intent,
		"comment":       draft.String(),
		"oa_url":        oaURL,
	}, "opinion_draft", nil
}

// 7. 触发智能审核任务
func (e *SystemToolExecutor) executeRunAudit(
	argsJSON string,
	execCtx *ExecutionContext,
	adapter oa.OAAdapter,
) (interface{}, string, error) {
	var args struct {
		ProcessID string `json:"process_id"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil || args.ProcessID == "" {
		return nil, "audit_job", fmt.Errorf("缺少必填参数 process_id")
	}

	visible, err := adapter.CheckProcessVisibility(execCtx.Ctx, execCtx.Username, args.ProcessID)
	if err != nil || !visible {
		return nil, "audit_job", fmt.Errorf("无权访问流程 %s 或流程不存在", args.ProcessID)
	}

	if e.RunAudit == nil {
		return nil, "audit_job", fmt.Errorf("执行服务未初始化")
	}
	summary, err := adapter.FetchProcessRequestSummary(execCtx.Ctx, args.ProcessID)
	if err != nil {
		return nil, "audit_job", err
	}
	result, err := e.RunAudit(execCtx, summary)
	return result, "audit_job", err
}

// 8. 触发智能总结任务
func (e *SystemToolExecutor) executeRunSummary(
	argsJSON string,
	execCtx *ExecutionContext,
	adapter oa.OAAdapter,
) (interface{}, string, error) {
	var args struct {
		ProcessID string `json:"process_id"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil || args.ProcessID == "" {
		return nil, "summary_job", fmt.Errorf("缺少必填参数 process_id")
	}

	visible, err := adapter.CheckProcessVisibility(execCtx.Ctx, execCtx.Username, args.ProcessID)
	if err != nil || !visible {
		return nil, "summary_job", fmt.Errorf("无权访问流程 %s 或流程不存在", args.ProcessID)
	}

	if e.RunSummary == nil {
		return nil, "summary_job", fmt.Errorf("执行服务未初始化")
	}
	summary, err := adapter.FetchProcessRequestSummary(execCtx.Ctx, args.ProcessID)
	if err != nil {
		return nil, "summary_job", err
	}
	result, err := e.RunSummary(execCtx, summary)
	return result, "summary_job", err
}

// 9. 生成 OA 办理链接
func (e *SystemToolExecutor) executeResolveOAURL(
	argsJSON string,
	execCtx *ExecutionContext,
	oaConn *model.OADatabaseConnection,
) (interface{}, string, error) {
	var args struct {
		ProcessID string `json:"process_id"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil || args.ProcessID == "" {
		return nil, "oa_link", fmt.Errorf("缺少必填参数 process_id")
	}
	if oaConn == nil {
		return nil, "oa_link", fmt.Errorf("当前租户尚未配置 OA 连接，无法生成跳转链接")
	}

	oaURL := oa.BuildProcessURL(oaConn.OABaseURL, oaConn.ProcessURLTemplate, oaConn.OAType, args.ProcessID)
	if oaURL == "" {
		return nil, "oa_link", fmt.Errorf("未配置有效的 OA Web 地址或跳转模板")
	}

	return map[string]interface{}{
		"process_id": args.ProcessID,
		"oa_url":     oaURL,
		"label":      "去 OA 办理",
	}, "oa_link", nil
}

func containsString(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

var (
	forbiddenDMLRegex   = regexp.MustCompile(`(?i)\b(INSERT|UPDATE|DELETE|DROP|ALTER|TRUNCATE|EXEC|EXECUTE|GRANT|REVOKE|MERGE|CREATE|REPLACE)\b`)
	sensitiveWordsRegex = regexp.MustCompile(`(?i)\b(password|license|token|secret|key|passwd|pwd)\b`)
)

// 11. 创建 OA 流程草稿
func (e *SystemToolExecutor) executeCreateOADraft(
	argsJSON string,
	execCtx *ExecutionContext,
	oaConn *model.OADatabaseConnection,
	adapter oa.OAAdapter,
) (interface{}, string, error) {
	creator, ok := adapter.(oa.WorkflowDraftCreator)
	if !ok {
		return nil, "create_workflow_draft", fmt.Errorf("当前 OA 适配器不支持自动创建流程草稿")
	}

	var args struct {
		WorkflowID   string                 `json:"workflow_id"`
		WorkflowName string                 `json:"workflow_name"`
		RequestName  string                 `json:"request_name"`
		RequestLevel string                 `json:"request_level"`
		MainData     map[string]interface{} `json:"main_data"`
		DetailData   []interface{}          `json:"detail_data"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil, "create_workflow_draft", fmt.Errorf("解析参数失败: %w", err)
	}

	if strings.TrimSpace(args.WorkflowID) == "" {
		return nil, "create_workflow_draft", fmt.Errorf("缺少流程定义 ID (workflow_id)")
	}

	loginID := strings.TrimSpace(execCtx.Username)
	if loginID == "" && oaConn != nil {
		loginID = strings.TrimSpace(oaConn.WeaverDefaultUser)
	}

	input := &oa.CreateWorkflowDraftInput{
		WorkflowID:   strings.TrimSpace(args.WorkflowID),
		WorkflowName: strings.TrimSpace(args.WorkflowName),
		RequestName:  strings.TrimSpace(args.RequestName),
		RequestLevel: strings.TrimSpace(args.RequestLevel),
		MainData:     args.MainData,
		DetailData:   args.DetailData,
	}

	out, err := creator.CreateWorkflowDraft(execCtx.Ctx, loginID, input)
	if err != nil {
		return nil, "create_workflow_draft", fmt.Errorf("创建流程草稿失败: %w", err)
	}

	result := map[string]interface{}{
		"request_id":   out.RequestID,
		"request_name": out.RequestName,
		"process_url":  out.ProcessURL,
		"workflow_id":  input.WorkflowID,
		"status":       "draft_created",
		"message":      "流程草稿创建成功，请点击链接前往 OA 确认并提交",
	}

	return result, "create_workflow_draft", nil
}

// 12. 执行安全数据查询
func (e *SystemToolExecutor) executeSafeQuery(
	argsJSON string,
	execCtx *ExecutionContext,
	oaConn *model.OADatabaseConnection,
) (interface{}, string, error) {
	if oaConn == nil {
		return nil, "data_query_table", fmt.Errorf("当前租户尚未配置 OA 连接，无法执行查数分析")
	}

	var args struct {
		SQL   string `json:"sql"`
		Title string `json:"title"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil, "data_query_table", fmt.Errorf("解析参数失败: %w", err)
	}

	rawSQL := strings.TrimSpace(args.SQL)
	if rawSQL == "" {
		return nil, "data_query_table", fmt.Errorf("SQL 查询语句不能为空")
	}

	// 1. 五层安全拦截
	// 层级一：去除首尾分号后检查是否以 SELECT 开头
	cleanSQL := strings.TrimRight(rawSQL, ";")
	cleanSQL = strings.TrimSpace(cleanSQL)

	upper := strings.ToUpper(cleanSQL)
	if !strings.HasPrefix(upper, "SELECT") || (len(upper) > 6 && upper[6] != ' ' && upper[6] != '\t' && upper[6] != '\n') {
		return nil, "data_query_table", fmt.Errorf("安全守卫拦截：仅允许执行 SELECT 只读查询语句")
	}

	// 层级二：严禁包含增删改查以外的 DML/DDL 关键字
	if forbiddenDMLRegex.MatchString(cleanSQL) {
		return nil, "data_query_table", fmt.Errorf("安全守卫拦截：查询语句中包含禁止的 DDL/DML 关键字")
	}

	// 层级三：严禁包含 SQL 注释符（防绕过注入）
	if strings.Contains(cleanSQL, "--") || strings.Contains(cleanSQL, "/*") || strings.Contains(cleanSQL, "*/") {
		return nil, "data_query_table", fmt.Errorf("安全守卫拦截：禁止在 SQL 查询中使用注释符号（-- 或 /* */）")
	}

	// 层级四：敏感凭证表与字段黑名单
	if sensitiveWordsRegex.MatchString(cleanSQL) {
		return nil, "data_query_table", fmt.Errorf("安全守卫拦截：禁止查询包含密码、令牌或密钥等敏感字段")
	}

	// 层级五：强制行数限制（100 行截断保护）
	// 根据数据库方言使用子查询包裹，杜绝全表扫描与巨量输出
	var wrappedSQL string
	if oaConn.Driver == "oracle" {
		wrappedSQL = fmt.Sprintf("SELECT * FROM (%s) safe_sub WHERE ROWNUM <= 100", cleanSQL)
	} else {
		wrappedSQL = fmt.Sprintf("SELECT * FROM (%s) AS safe_sub LIMIT 100", cleanSQL)
	}

	db, err := e.oaConnections.GetDB(execCtx.Ctx, oaConn.OAType, oaConn)
	if err != nil {
		return nil, "data_query_table", fmt.Errorf("获取 OA 数据库连接失败: %w", err)
	}

	// 执行只读查询，超时时间 10 秒
	queryCtx, cancel := context.WithTimeout(execCtx.Ctx, 10*time.Second)
	defer cancel()

	rows, err := db.WithContext(queryCtx).Raw(wrappedSQL).Rows()
	if err != nil {
		return nil, "data_query_table", fmt.Errorf("执行 SQL 查询失败: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, "data_query_table", fmt.Errorf("读取列信息失败: %w", err)
	}

	var results []map[string]interface{}
	for rows.Next() {
		// 创建动态切片承载列值
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range cols {
			valuePtrs[i] = &values[i]
		}
		if scanErr := rows.Scan(valuePtrs...); scanErr != nil {
			return nil, "data_query_table", fmt.Errorf("扫描查询数据行失败: %w", scanErr)
		}

		rowMap := make(map[string]interface{})
		for i, colName := range cols {
			val := values[i]
			// 如果是字节数组转成字符串，便于 JSON 序列化
			if b, ok := val.([]byte); ok {
				rowMap[colName] = string(b)
			} else {
				rowMap[colName] = val
			}
		}
		results = append(results, rowMap)
	}

	title := strings.TrimSpace(args.Title)
	if title == "" {
		title = "OA 业务数据查询结果"
	}

	res := map[string]interface{}{
		"title":      title,
		"columns":    cols,
		"rows":       results,
		"total_rows": len(results),
		"sql":        cleanSQL,
		"truncated":  len(results) >= 100,
	}

	return res, "data_query_table", nil
}

// 13. 获取流程表单定义与字段选项（支持自动匹配 workflow_id 或名称搜索）
func (e *SystemToolExecutor) executeGetWorkflowSchema(
	argsJSON string,
	execCtx *ExecutionContext,
	adapter oa.OAAdapter,
) (interface{}, string, error) {
	var args struct {
		WorkflowID   string `json:"workflow_id"`
		WorkflowName string `json:"workflow_name"`
	}
	_ = json.Unmarshal([]byte(argsJSON), &args)

	targetID := strings.TrimSpace(args.WorkflowID)
	targetName := strings.TrimSpace(args.WorkflowName)

	selector, ok := adapter.(oa.WorkflowDefinitionSelector)
	if !ok {
		return nil, "workflow_schema", fmt.Errorf("当前 OA 适配器不支持检索流程定义与表单字段")
	}

	ctx := context.Background()
	if targetID == "" {
		if targetName == "" {
			return nil, "workflow_schema", fmt.Errorf("请提供流程定义 ID (workflow_id) 或流程名称关键词 (workflow_name)")
		}
		// 检索流程定义
		defs, err := selector.SearchWorkflowDefinitions(ctx, targetName)
		if err != nil {
			return nil, "workflow_schema", fmt.Errorf("检索流程定义失败: %w", err)
		}
		if len(defs) == 0 {
			return map[string]interface{}{
				"matched":   false,
				"message":   fmt.Sprintf("未在 OA 系统中检索到包含关键词 '%s' 的流程定义", targetName),
				"workflows": []interface{}{},
			}, "workflow_schema", nil
		}
		targetID = defs[0].WorkflowID
		targetName = defs[0].ProcessName
	}

	fields, err := selector.FetchFieldsByWorkflowID(ctx, targetID)
	if err != nil {
		return nil, "workflow_schema", fmt.Errorf("获取流程字段结构失败: %w", err)
	}

	return map[string]interface{}{
		"matched":       true,
		"workflow_id":   targetID,
		"workflow_name": targetName,
		"main_fields":   fields.MainFields,
		"detail_tables": fields.DetailTables,
	}, "workflow_schema", nil
}
