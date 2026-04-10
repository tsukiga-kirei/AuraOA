package oa

import (
	"context"
	"time"
)

// TodoListFilter 控制待办列表在 OA 侧的查询条件（由 SQL 直接过滤 workflow_requestbase.createdate）。
type TodoListFilter struct {
	SubmitDateStart *time.Time
	// SubmitDateEndExclusive 上界（不含）
	SubmitDateEndExclusive *time.Time
}

// TodoListPagedFilter 待办列表分页查询条件，将 keyword/applicant/department 等筛选下推到 OA SQL。
type TodoListPagedFilter struct {
	TodoListFilter
	// Keyword 模糊匹配流程标题（requestname）
	Keyword string
	// Applicant 模糊匹配申请人姓名（hrmresource.lastname）
	Applicant string
	// Department 精确匹配部门名称（hrmdepartment.departmentname）
	Department string
	// MainTableNames 允许的主表名列表（小写），为空则不过滤
	MainTableNames []string
	// Page 页码（从 1 开始）
	Page int
	// PageSize 每页条数
	PageSize int
}

// PagedResult 分页查询通用结果。
type PagedResult[T any] struct {
	Items []T
	Total int
}

// ArchivedListFilter 控制已归档流程列表在 OA 侧的查询条件（由 SQL 直接过滤）。
type ArchivedListFilter struct {
	// ArchiveDateStart 归档时间下界（含），与适配器内用于排序的归档时间表达式一致（如 COALESCE(lastoperatedate, createdate)）。
	ArchiveDateStart *time.Time
	// ArchiveDateEndExclusive 归档时间上界（不含），通常为「结束日期」次日 0 点。
	ArchiveDateEndExclusive *time.Time
}

// ArchivedListPagedFilter 已归档流程分页查询条件，将筛选下推到 OA SQL。
type ArchivedListPagedFilter struct {
	ArchivedListFilter
	// Keyword 模糊匹配流程标题（requestname）
	Keyword string
	// Applicant 模糊匹配申请人姓名
	Applicant string
	// Department 精确匹配部门名称
	Department string
	// MainTableNames 允许的主表名列表（小写），为空则不过滤
	MainTableNames []string
	// ProcessTypes 允许的流程类型名列表（小写），为空则不过滤
	ProcessTypes []string
	// Page 页码（从 1 开始）
	Page int
	// PageSize 每页条数
	PageSize int
}

// OAAdapter 定义 OA 系统适配器接口。
// 不同 OA 类型（泛微 E9、致远、钉钉等）各自实现该接口。
type OAAdapter interface {
	// ValidateProcess 验证流程类型是否存在于 OA 系统中
	ValidateProcess(ctx context.Context, processType string) (*ProcessInfo, error)

	// FetchFields 拉取指定流程的全部字段定义（主表 + 明细表）
	FetchFields(ctx context.Context, processType string) (*ProcessFields, error)

	// CheckUserPermission 检查用户在 OA 中是否具有指定流程的审批权限
	CheckUserPermission(ctx context.Context, userID string, processType string) (bool, error)

	// FetchProcessData 拉取指定流程实例的业务数据（用于审核执行）
	FetchProcessData(ctx context.Context, processID string) (*ProcessData, error)

	// FetchTodoList 拉取指定用户的 OA 待审批流程列表（filter 中日期条件在 SQL 中生效）
	FetchTodoList(ctx context.Context, username string, filter TodoListFilter) ([]TodoItem, error)

	// FetchTodoListPaged 分页拉取指定用户的 OA 待审批流程列表，将筛选条件下推到 SQL，返回当前页数据和总数。
	FetchTodoListPaged(ctx context.Context, username string, filter TodoListPagedFilter) (*PagedResult[TodoItem], error)

	// FetchArchivedList 拉取已归档流程列表（filter 中日期条件在 SQL 中生效）
	FetchArchivedList(ctx context.Context, username string, filter ArchivedListFilter) ([]ArchivedItem, error)

	// FetchArchivedListPaged 分页拉取已归档流程列表，将筛选条件下推到 SQL，返回当前页数据和总数。
	FetchArchivedListPaged(ctx context.Context, username string, filter ArchivedListPagedFilter) (*PagedResult[ArchivedItem], error)

	// FetchProcessFlow 拉取流程审批流快照
	FetchProcessFlow(ctx context.Context, processID string) (*ProcessFlowSnapshot, error)

	// IsProcessInTodo 判断指定流程是否仍在用户待办中
	IsProcessInTodo(ctx context.Context, username string, processID string) (bool, error)

	// FetchAllTodoItems 拉取所有待审批流程（不过滤用户，供调度器批处理使用）
	// limit <= 0 表示不限制条数
	FetchAllTodoItems(ctx context.Context, limit int) ([]TodoItem, error)
}

// ProcessInfo 流程基本信息
type ProcessInfo struct {
	ProcessType       string `json:"process_type"`
	ProcessName       string `json:"process_name"`
	ProcessTypeLabel  string `json:"process_type_label,omitempty"`
	MainTable         string `json:"main_table"`
	DetailCount       int    `json:"detail_count"`
	TableMismatch     bool   `json:"table_mismatch,omitempty"`      // 用户填写的主表名与 OA 实际不一致
	ExpectedTable     string `json:"expected_table,omitempty"`      // OA 系统中的正确主表名（仅 mismatch 时返回）
	TypeLabelMismatch bool   `json:"type_label_mismatch,omitempty"` // 用户填写的流程类型与 OA 实际不一致
	ExpectedTypeLabel string `json:"expected_type_label,omitempty"` // OA 系统中的正确流程类型（仅 mismatch 时返回）
}

// FieldDef 字段定义
type FieldDef struct {
	FieldKey  string `json:"field_key"`
	FieldName string `json:"field_name"`
	FieldType string `json:"field_type"`
}

// DetailTableDef 明细表定义
type DetailTableDef struct {
	TableName  string     `json:"table_name"`
	TableLabel string     `json:"table_label"`
	Fields     []FieldDef `json:"fields"`
}

// ProcessFields 流程字段集合
type ProcessFields struct {
	MainFields   []FieldDef       `json:"main_fields"`
	DetailTables []DetailTableDef `json:"detail_tables"`
}

// ProcessData 流程实例业务数据
type ProcessData struct {
	ProcessID    string                              `json:"process_id"`
	MainData     map[string]interface{}              `json:"main_data"`
	DetailTables map[string][]map[string]interface{} `json:"detail_tables"`
}

// TodoItem OA 待办流程条目
type TodoItem struct {
	ProcessID        string `json:"process_id"`
	Title            string `json:"title"`
	Applicant        string `json:"applicant"`
	Department       string `json:"department"`
	ProcessType      string `json:"process_type"`
	ProcessTypeLabel string `json:"process_type_label"`
	CurrentNode      string `json:"current_node"`
	SubmitTime       string `json:"submit_time"`
	Urgency          string `json:"urgency"`
	MainTableName    string `json:"main_table_name"`
}

// ArchivedItem OA 已归档流程条目。
type ArchivedItem struct {
	ProcessID        string `json:"process_id"`
	Title            string `json:"title"`
	Applicant        string `json:"applicant"`
	Department       string `json:"department"`
	ProcessType      string `json:"process_type"`
	ProcessTypeLabel string `json:"process_type_label"`
	CurrentNode      string `json:"current_node"`
	SubmitTime       string `json:"submit_time"`
	ArchiveTime      string `json:"archive_time"`
	MainTableName    string `json:"main_table_name"`
}

// ProcessFlowNode 审批流节点快照。
type ProcessFlowNode struct {
	NodeID     string `json:"node_id"`
	NodeName   string `json:"node_name"`
	Approver   string `json:"approver"`
	Action     string `json:"action"`
	ActionTime string `json:"action_time"`
	Opinion    string `json:"opinion"`
}

// ProcessFlowSnapshot 审批流快照。
type ProcessFlowSnapshot struct {
	IsComplete   bool              `json:"is_complete"`
	MissingNodes []string          `json:"missing_nodes"`
	Nodes        []ProcessFlowNode `json:"nodes"`
	HistoryText  string            `json:"history_text"`
	GraphText    string            `json:"graph_text"`
}
