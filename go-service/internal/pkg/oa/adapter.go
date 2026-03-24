package oa

import "context"

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

	// FetchTodoList 拉取指定用户的 OA 待审批流程列表
	FetchTodoList(ctx context.Context, username string) ([]TodoItem, error)

	// IsProcessInTodo 判断指定流程是否仍在用户待办中
	IsProcessInTodo(ctx context.Context, username string, processID string) (bool, error)
}

// ProcessInfo 流程基本信息
type ProcessInfo struct {
	ProcessType       string `json:"process_type"`
	ProcessName       string `json:"process_name"`
	ProcessTypeLabel  string `json:"process_type_label,omitempty"`
	MainTable         string `json:"main_table"`
	DetailCount       int    `json:"detail_count"`
	TableMismatch     bool   `json:"table_mismatch,omitempty"`       // 用户填写的主表名与 OA 实际不一致
	ExpectedTable     string `json:"expected_table,omitempty"`       // OA 系统中的正确主表名（仅 mismatch 时返回）
	TypeLabelMismatch bool   `json:"type_label_mismatch,omitempty"`  // 用户填写的流程类型与 OA 实际不一致
	ExpectedTypeLabel string `json:"expected_type_label,omitempty"`  // OA 系统中的正确流程类型（仅 mismatch 时返回）
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
	ProcessID    string                                `json:"process_id"`
	MainData     map[string]interface{}                `json:"main_data"`
	DetailTables map[string][]map[string]interface{}   `json:"detail_tables"`
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
}
