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

// BuiltinTools 一期内置的 9 大系统工具规格清单
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
					"description": "OA 流程实例 ID (requestid)",
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
		Description: "获取 AuraOA 平台对该流程最近一次 AI 审核的结论、风险点与建议（若已进行过审核）。",
		UIKind:      "audit_result",
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
	"get_latest_summary": {
		Code:        "get_latest_summary",
		Name:        "获取最新流程总结",
		Description: "获取 AuraOA 平台对该流程最近一次生成的 AI 流程总结要点（若已进行过总结）。",
		UIKind:      "summary_result",
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
		"get_process",
		"get_approval_flow",
		"get_latest_audit",
		"get_latest_summary",
		"draft_comment",
		"run_audit",
		"run_summary",
		"resolve_oa_url",
	}
	specs := make([]ToolSpec, 0, len(orderedCodes))
	for _, code := range orderedCodes {
		if spec, ok := BuiltinTools[code]; ok {
			specs = append(specs, spec)
		}
	}
	return specs
}
