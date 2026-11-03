package agenttools

import (
	"auraoa/go-service/internal/pkg/ai"
)

// ToolSpec 系统工具的静态规格定义
type ToolSpec struct {
	Code        string                 `json:"code"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	UIKind      string                 `json:"ui_kind"`
	OARequired  bool                   `json:"oa_required"`
	Risk        string                 `json:"risk"` // read | assist | write
	Parameters  map[string]interface{} `json:"parameters"`
}

// BuiltinTools 一期内置的 10 大系统工具规格清单
var BuiltinTools = map[string]ToolSpec{
	"list_my_todos": {
		Code:        "list_my_todos",
		Name:        "查询我的待办",
		Description: "分页查询当前登录用户在 OA 系统中的待审批流程列表，支持根据关键词、申请人、部门及分页参数筛选。",
		UIKind:      "todo_list",
		OARequired:  true,
		Risk:        "read",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"keyword": map[string]interface{}{
					"type":        "string",
					"description": "流程标题模糊搜索关键词",
				},
				"applicant": map[string]interface{}{
					"type":        "string",
					"description": "流程发起人/申请人姓名",
				},
				"department": map[string]interface{}{
					"type":        "string",
					"description": "申请人所在部门名称",
				},
				"page": map[string]interface{}{
					"type":        "integer",
					"description": "页码，从 1 开始，默认 1",
					"default":     1,
				},
				"page_size": map[string]interface{}{
					"type":        "integer",
					"description": "每页记录条数，默认 20，最大 50",
					"default":     20,
				},
			},
		},
	},
	"list_my_requests": {
		Code:        "list_my_requests",
		Name:        "查询我发起的流程",
		Description: "分页查询当前登录用户在 OA 系统中发起的申请与审批流程，支持根据关键词、流转状态（流转中/已归档）及分页参数筛选。",
		UIKind:      "my_request_list",
		OARequired:  true,
		Risk:        "read",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"keyword": map[string]interface{}{
					"type":        "string",
					"description": "流程标题模糊搜索关键词",
				},
				"status": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"all", "processing", "archived"},
					"description": "流程状态：all (全部，默认)，processing (流转中/审批中)，archived (已办结/已归档)",
					"default":     "all",
				},
				"page": map[string]interface{}{
					"type":        "integer",
					"description": "页码，从 1 开始，默认 1",
					"default":     1,
				},
				"page_size": map[string]interface{}{
					"type":        "integer",
					"description": "每页记录条数，默认 20，最大 50",
					"default":     20,
				},
			},
		},
	},
	"get_process": {
		Code:        "get_process",
		Name:        "获取流程表单详情",
		Description: "获取指定流程实例的主表数据、明细表数据以及附件列表摘要。仅限当前用户具有可见性的流程。",
		UIKind:      "process_detail",
		OARequired:  true,
		Risk:        "read",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"process_id": map[string]interface{}{
					"type":        "string",
					"description": "OA 流程实例 ID (requestid) 或流程编号 (如 ZJ182026090162)",
				},
			},
			"required": []string{"process_id"},
		},
	},
	"get_approval_flow": {
		Code:        "get_approval_flow",
		Name:        "获取审批轨迹",
		Description: "获取指定流程实例的历史审批流轨迹（包括节点名称、审批人、操作类型、处理意见及时间）。",
		UIKind:      "approval_flow",
		OARequired:  true,
		Risk:        "read",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"process_id": map[string]interface{}{
					"type":        "string",
					"description": "OA 流程实例 ID (requestid)",
				},
			},
			"required": []string{"process_id"},
		},
	},
	"get_latest_audit": {
		Code:        "get_latest_audit",
		Name:        "获取最新审核结果",
		Description: "获取 AuraOA 平台对流程最近一次 AI 审核的结论、风险点与建议。支持单条流程（process_id）或批量多条流程（process_ids）。【特别注意】当待办/流程列表有多条且需要查询审核结果时，严禁多次循环调用，必须将全部 ID 组装在 process_ids 数组中一次性调用完成！",
		UIKind:      "audit_result",
		OARequired:  true,
		Risk:        "read",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"process_id": map[string]interface{}{
					"type":        "string",
					"description": "单个 OA 流程实例 ID (requestid)，单条查询时使用",
				},
				"process_ids": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
					},
					"description": "OA 流程实例 ID 列表 (requestid 数组)。当存在多个流程时，务必使用此参数一次性批量查询，禁止多次单条循环调用！",
				},
			},
		},
	},
	"get_latest_summary": {
		Code:        "get_latest_summary",
		Name:        "获取最新流程总结",
		Description: "获取 AuraOA 平台对流程最近一次生成的 AI 流程总结要点。支持单条流程（process_id）或批量多条流程（process_ids）。【特别注意】当待办/流程列表有多条且需要查询流程总结时，严禁多次循环调用，必须将全部 ID 组装在 process_ids 数组中一次性调用完成！",
		UIKind:      "summary_result",
		OARequired:  true,
		Risk:        "read",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"process_id": map[string]interface{}{
					"type":        "string",
					"description": "单个 OA 流程实例 ID (requestid)，单条查询时使用",
				},
				"process_ids": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
					},
					"description": "OA 流程实例 ID 列表 (requestid 数组)。当存在多个流程时，务必使用此参数一次性批量查询，禁止多次单条循环调用！",
				},
			},
		},
	},
	"draft_comment": {
		Code:        "draft_comment",
		Name:        "起草审批意见",
		Description: "针对指定流程，结合审批意图（同意/批准 或 退回/驳回）及补充说明，起草专业规范的审批处理意见文本。",
		UIKind:      "opinion_draft",
		OARequired:  true,
		Risk:        "assist",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"process_id": map[string]interface{}{
					"type":        "string",
					"description": "OA 流程实例 ID (requestid)",
				},
				"intent": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"approve", "return"},
					"description": "审批意图：approve (同意/通过) 或 return (退回/驳回)",
				},
				"note": map[string]interface{}{
					"type":        "string",
					"description": "补充说明或关注要点（可选）",
				},
			},
			"required": []string{"process_id", "intent"},
		},
	},
	"run_audit": {
		Code:        "run_audit",
		Name:        "触发智能审核",
		Description: "在后台为指定流程提交一次 AuraOA 规则审核任务，返回提交状态及任务 ID。前端可订阅进度。",
		UIKind:      "audit_job",
		OARequired:  true,
		Risk:        "assist",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"process_id": map[string]interface{}{
					"type":        "string",
					"description": "OA 流程实例 ID (requestid)",
				},
			},
			"required": []string{"process_id"},
		},
	},
	"run_summary": {
		Code:        "run_summary",
		Name:        "触发流程总结",
		Description: "在后台为指定流程提交一次 AuraOA 智能总结任务，返回提交状态及任务 ID。前端可订阅进度。",
		UIKind:      "summary_job",
		OARequired:  true,
		Risk:        "assist",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"process_id": map[string]interface{}{
					"type":        "string",
					"description": "OA 流程实例 ID (requestid)",
				},
			},
			"required": []string{"process_id"},
		},
	},
	"resolve_oa_url": {
		Code:        "resolve_oa_url",
		Name:        "生成 OA 办理链接",
		Description: "根据当前租户配置的 OA Web 访问地址与跳转模板，生成直达该流程详情的办理跳转链接。",
		UIKind:      "oa_link",
		OARequired:  false,
		Risk:        "read",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"process_id": map[string]interface{}{
					"type":        "string",
					"description": "OA 流程实例 ID (requestid)",
				},
			},
			"required": []string{"process_id"},
		},
	},
	"create_oa_draft": {
		Code:        "create_oa_draft",
		Name:        "创建 OA 流程草稿",
		Description: "在泛微 OA 中创建一条指定流程类型的草稿单据（草稿状态，不自动流转，需申请人在 OA 中确认提交）。调用成功后返回创建的流程 ID 与直达办理的 Web 链接。",
		UIKind:      "create_workflow_draft",
		OARequired:  true,
		Risk:        "write",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"workflow_id": map[string]interface{}{
					"type":        "string",
					"description": "泛微流程定义 ID (workflowId)，如 '102'",
				},
				"workflow_name": map[string]interface{}{
					"type":        "string",
					"description": "流程名称（如 '物品领用申请'，用于辅助生成单据标题）",
				},
				"request_name": map[string]interface{}{
					"type":        "string",
					"description": "单据标题（可选，留空时系统自动按 申请人-流程名-日期 规则生成）",
				},
				"request_level": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"0", "1", "2"},
					"description": "紧急程度：0 (正常，默认)，1 (重要)，2 (紧急)",
					"default":     "0",
				},
				"main_data": map[string]interface{}{
					"type":        "object",
					"description": "主表字段数据键值对映射，如 {\"sqrq\": \"2026-11-02\", \"wp_type\": \"文具\", \"reason\": \"日常办公用品\"}",
				},
				"detail_data": map[string]interface{}{
					"type":        "array",
					"description": "明细表数据数组，无明细时固定为空数组 []",
				},
			},
			"required": []string{"workflow_id", "main_data"},
		},
	},
	"execute_safe_query": {
		Code:        "execute_safe_query",
		Name:        "执行安全数据查询",
		Description: "在 OA 数据库中执行只读的数据统计与分析 SELECT SQL 查询。系统内置五层安全守卫拦截（仅限 SELECT、严禁增删改与注释符、敏感表黑名单、强制 LIMIT 100 截断），返回二维结构化数据供分析及表格渲染。",
		UIKind:      "data_query_table",
		OARequired:  true,
		Risk:        "read",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"sql": map[string]interface{}{
					"type":        "string",
					"description": "待执行的只读 SELECT SQL 查询语句（严禁包含 INSERT/UPDATE/DELETE/DROP/ALTER/TRUNCATE 等写入或变更语句，严禁使用注释符）",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "查询主题或统计指标说明（可选，如 '本月各部门流程发起数量统计'）",
				},
			},
			"required": []string{"sql"},
		},
	},
	"get_workflow_schema": {
		Code:        "get_workflow_schema",
		Name:        "获取流程表单定义与字段选项",
		Description: "查询指定流程（workflow_id 或流程名称关键词）的表单字段结构、中文标签、字段类型以及下拉选择框（Select）的完整候选项列表。在填单前用于主动了解表单字段并为用户匹配合适选项。",
		UIKind:      "workflow_schema",
		OARequired:  true,
		Risk:        "read",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"workflow_id": map[string]interface{}{
					"type":        "string",
					"description": "泛微流程定义 ID (workflow_id)，如 '102'（与 workflow_name 二选一）",
				},
				"workflow_name": map[string]interface{}{
					"type":        "string",
					"description": "流程名称模糊搜索关键词，如 '领用'、'请假'（当不知道具体 workflow_id 时使用）",
				},
			},
		},
	},
}

// ToToolDefinition 将 ToolSpec 转为 AI 统一工具定义
func (ts ToolSpec) ToToolDefinition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Type: "function",
		Function: ai.FunctionSpec{
			Name:        ts.Code,
			Description: ts.Description,
			Parameters:  ts.Parameters,
		},
	}
}

// GetAllToolSpecs 返回按稳定顺序排列的内置工具规格列表
func GetAllToolSpecs() []ToolSpec {
	orderedCodes := []string{
		"list_my_todos",
		"list_my_requests",
		"get_process",
		"get_approval_flow",
		"get_latest_audit",
		"get_latest_summary",
		"draft_comment",
		"run_audit",
		"run_summary",
		"resolve_oa_url",
		"create_oa_draft",
		"execute_safe_query",
	}
	specs := make([]ToolSpec, 0, len(orderedCodes))
	for _, code := range orderedCodes {
		if spec, ok := BuiltinTools[code]; ok {
			specs = append(specs, spec)
		}
	}
	return specs
}
