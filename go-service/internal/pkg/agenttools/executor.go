package agenttools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

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
		ProcessID   string `json:"process_id"`
		Title       string `json:"title"`
		Applicant   string `json:"applicant"`
		Department  string `json:"department"`
		CurrentNode string `json:"current_node"`
		SubmitTime  string `json:"submit_time"`
		OAURL       string `json:"oa_url,omitempty"`
	}

	items := make([]TodoItemPayload, 0, len(pagedResult.Items))
	for _, item := range pagedResult.Items {
		oaURL := ""
		if oaConn != nil {
			oaURL = buildProcessURL(oaConn.OABaseURL, oaConn.ProcessURLTemplate, oaConn.OAType, item.ProcessID)
		}
		items = append(items, TodoItemPayload{
			ProcessID:   item.ProcessID,
			Title:       item.Title,
			Applicant:   item.Applicant,
			Department:  item.Department,
			CurrentNode: item.CurrentNode,
			SubmitTime:  item.SubmitTime,
			OAURL:       oaURL,
		})
	}

	return map[string]interface{}{
		"items":     items,
		"total":     pagedResult.Total,
		"page":      args.Page,
		"page_size": args.PageSize,
	}, "todo_list", nil
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
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil || args.ProcessID == "" {
		return nil, "process_detail", fmt.Errorf("缺少必填参数 process_id")
	}

	// 校验可见性
	visible, err := adapter.CheckProcessVisibility(execCtx.Ctx, execCtx.Username, args.ProcessID)
	if err != nil || !visible {
		return nil, "process_detail", fmt.Errorf("无权访问流程 %s 或流程不存在", args.ProcessID)
	}

	data, err := adapter.FetchProcessData(execCtx.Ctx, args.ProcessID)
	if err != nil {
		return nil, "process_detail", fmt.Errorf("读取流程数据失败: %w", err)
	}

	summary, _ := adapter.FetchProcessRequestSummary(execCtx.Ctx, args.ProcessID)
	title := args.ProcessID
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

	return map[string]interface{}{
		"process_id":    args.ProcessID,
		"title":         title,
		"process_type":  processType,
		"applicant":     applicant,
		"department":    department,
		"main_fields":   mainFields,
		"detail_tables": data.DetailTables,
		"attachments":   attachments,
	}, "process_detail", nil
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
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil || args.ProcessID == "" {
		return nil, "approval_flow", fmt.Errorf("缺少必填参数 process_id")
	}

	// 可见性校验
	visible, err := adapter.CheckProcessVisibility(execCtx.Ctx, execCtx.Username, args.ProcessID)
	if err != nil || !visible {
		return nil, "approval_flow", fmt.Errorf("无权访问流程 %s 或流程不存在", args.ProcessID)
	}

	flow, err := adapter.FetchProcessFlow(execCtx.Ctx, args.ProcessID)
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

	return map[string]interface{}{
		"process_id": args.ProcessID,
		"nodes":      nodes,
	}, "approval_flow", nil
}

// 4. 获取最新审核结果
func (e *SystemToolExecutor) executeGetLatestAudit(
	argsJSON string,
	execCtx *ExecutionContext,
	adapter oa.OAAdapter,
) (interface{}, string, error) {
	var args struct {
		ProcessID string `json:"process_id"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil || args.ProcessID == "" {
		return nil, "audit_result", fmt.Errorf("缺少必填参数 process_id")
	}

	if adapter != nil {
		visible, err := adapter.CheckProcessVisibility(execCtx.Ctx, execCtx.Username, args.ProcessID)
		if err != nil || !visible {
			return nil, "audit_result", fmt.Errorf("无权访问流程 %s 或流程不存在", args.ProcessID)
		}
	}

	var log model.AuditLog
	err := e.db.WithContext(execCtx.Ctx).
		Where("tenant_id = ? AND process_id = ?", execCtx.TenantID, args.ProcessID).
		Order("created_at DESC").
		First(&log).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return map[string]interface{}{
			"process_id": args.ProcessID,
			"has_audit":  false,
			"message":    "该流程暂无历史 AI 审核记录",
		}, "audit_result", nil
	}
	if err != nil {
		return nil, "audit_result", fmt.Errorf("查询审核记录失败: %w", err)
	}

	return map[string]interface{}{
		"process_id":     args.ProcessID,
		"has_audit":      true,
		"status":         log.Status,
		"recommendation": log.Recommendation,
		"score":          log.Score,
		"audit_result":   log.AuditResult,
		"created_at":     apptime.FormatRFC3339(log.CreatedAt),
	}, "audit_result", nil
}

// 5. 获取最新流程总结
func (e *SystemToolExecutor) executeGetLatestSummary(
	argsJSON string,
	execCtx *ExecutionContext,
	adapter oa.OAAdapter,
) (interface{}, string, error) {
	var args struct {
		ProcessID string `json:"process_id"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil || args.ProcessID == "" {
		return nil, "summary_result", fmt.Errorf("缺少必填参数 process_id")
	}

	if adapter != nil {
		visible, err := adapter.CheckProcessVisibility(execCtx.Ctx, execCtx.Username, args.ProcessID)
		if err != nil || !visible {
			return nil, "summary_result", fmt.Errorf("无权访问流程 %s 或流程不存在", args.ProcessID)
		}
	}

	var summary model.ProcessSummaryLog
	err := e.db.WithContext(execCtx.Ctx).
		Where("tenant_id = ? AND process_id = ?", execCtx.TenantID, args.ProcessID).
		Order("created_at DESC").
		First(&summary).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return map[string]interface{}{
			"process_id":  args.ProcessID,
			"has_summary": false,
			"message":     "该流程暂无历史总结记录",
		}, "summary_result", nil
	}
	if err != nil {
		return nil, "summary_result", fmt.Errorf("查询总结记录失败: %w", err)
	}

	return map[string]interface{}{
		"process_id":     args.ProcessID,
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

	// 读取流程摘要与最新审核建议
	summary, _ := adapter.FetchProcessRequestSummary(execCtx.Ctx, args.ProcessID)
	var latestLog model.AuditLog
	_ = e.db.WithContext(execCtx.Ctx).
		Where("tenant_id = ? AND process_id = ?", execCtx.TenantID, args.ProcessID).
		Order("created_at DESC").
		First(&latestLog).Error

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
		oaURL = buildProcessURL(oaConn.OABaseURL, oaConn.ProcessURLTemplate, oaConn.OAType, args.ProcessID)
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

	oaURL := buildProcessURL(oaConn.OABaseURL, oaConn.ProcessURLTemplate, oaConn.OAType, args.ProcessID)
	if oaURL == "" {
		return nil, "oa_link", fmt.Errorf("未配置有效的 OA Web 地址或跳转模板")
	}

	return map[string]interface{}{
		"process_id": args.ProcessID,
		"oa_url":     oaURL,
		"label":      "去 OA 办理",
	}, "oa_link", nil
}

func buildProcessURL(baseURL, template, oaType, processID string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	template = strings.TrimSpace(template)
	if template == "" {
		switch oaType {
		case "weaver_e9":
			template = "/spa/workflow/static4form/index.html#/req?requestid={process_id}"
		default:
			template = "/req?requestid={process_id}"
		}
	}
	path := strings.ReplaceAll(template, "{process_id}", processID)
	if !strings.HasPrefix(path, "/") && baseURL != "" {
		path = "/" + path
	}
	return baseURL + path
}
