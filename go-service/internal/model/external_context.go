package model

// ExternalContextMount 描述挂在审核规则、归档规则或总结块上的外部关联数据。
// 系统会在组装提示词前完成查询，AI 只接收格式化后的查询结果。
type ExternalContextMount struct {
	Type        string                         `json:"type"` // workflow=关联流程；model=关联建模表
	Enabled     bool                           `json:"enabled"`
	Name        string                         `json:"name"`
	SourceField string                         `json:"source_field"` // 字段引用，格式为 table:field_key；省略 table 时默认为主表
	Splitter    string                         `json:"source_splitter,omitempty"`
	Workflow    *ExternalWorkflowContextConfig `json:"workflow,omitempty"`
	Model       *ExternalModelContextConfig    `json:"model,omitempty"`
}

// ExternalWorkflowContextConfig 控制 requestid 流程引用如何展开。
type ExternalWorkflowContextConfig struct {
	IncludeBasic       bool     `json:"include_basic"`
	BasicFields        []string `json:"basic_fields"`
	DataMode           string   `json:"data_mode"` // none=不取表单；all_fields=全部字段；selected_fields=指定字段
	TargetProcessType  string   `json:"target_process_type,omitempty"`
	TargetWorkflowID   string   `json:"target_workflow_id,omitempty"`
	TargetProcessLabel string   `json:"target_process_label,omitempty"`
	TargetMainTable    string   `json:"target_main_table,omitempty"`
	SelectedFields     []string `json:"selected_fields,omitempty"` // 字段引用，格式为 main:field 或 detail_table:field
	FallbackStrategy   string   `json:"fallback_strategy,omitempty"`
	MaxRows            int      `json:"max_rows,omitempty"`
}

// ExternalModelContextConfig 控制建模表关联查询。
type ExternalModelContextConfig struct {
	TableName    string   `json:"table_name"`
	JoinField    string   `json:"join_field"`
	Mode         string   `json:"mode"` // exists=是否存在；count=存在条数；rows=返回行数据；custom_sql=自定义 SQL
	ReturnFields []string `json:"return_fields,omitempty"`
	MaxRows      int      `json:"max_rows,omitempty"`
	OrderBy      string   `json:"order_by,omitempty"`
	OrderDir     string   `json:"order_dir,omitempty"`
	CustomSQL    string   `json:"custom_sql,omitempty"`
}
