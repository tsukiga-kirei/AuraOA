package oa

// AttachmentAuthWeaverApp 表示连接沿用泛微应用凭证：
// weaver_api_url、weaver_appid、weaver_default_user。
// 目前只有 Ecology 9 使用。E10 的附件接口未确认前不要复用这三项。
const AttachmentAuthWeaverApp = "weaver_app"

// WorkflowArchiveRule 是数据记忆在「流程表 + 仅已归档」时使用的关联规则。
// 查询器只按这些字段生成 SQL，不在公共层写死某一厂商的表名。
// 已保存的 SQL 不回写；只影响之后重新编译的配置。
//
// 若某 OA 的归档判断不是「状态列等于一个字面量」，先扩展本结构，
// 不要在查询器里按 oa_type 写分支。
type WorkflowArchiveRule struct {
	RequestTable        string `json:"request_table"`
	RequestIDColumn     string `json:"request_id_column"`
	FormRequestIDColumn string `json:"form_request_id_column"`
	StatusColumn        string `json:"status_column"`
	StatusValue         string `json:"status_value"`
}

// OAProfile 是某一 OA 类型在连接、跳转和数据记忆上的档案。
// 待办、表单、审批流等取数仍由各自适配器实现，不放在这里。
type OAProfile struct {
	Type string
	// DisplayName 非空时覆盖数据源展示名。E9 保持「泛微 Ecology 9」。
	DisplayName string
	// Implemented 为 false 时工厂拒绝建连。档案可以先占位。
	Implemented bool
	Drivers     []string
	// DefaultProcessURLTemplate 是连接未填跳转模板时的兜底，须含 {process_id}。
	// 按 OA 类型区分。留空表示该类型没有兜底，调用方不得再套用其他类型的地址。
	DefaultProcessURLTemplate string
	// AttachmentAuth 为空表示连接页不收集附件 HTTP 凭证。
	AttachmentAuth string
	// WorkflowArchive 为空表示该类型尚未定义归档关联，查询器不追加归档条件。
	WorkflowArchive *WorkflowArchiveRule
	// QueryAssistNote 追加进数据记忆 SQL 助手的系统提示词。空则只用通用约束。
	QueryAssistNote string
}

const (
	oaTypeWeaverE9  = "weaver_e9"
	oaTypeWeaverE10 = "weaver_e10"
)

// weaverE9DefaultProcessURLTemplate 是工作台一直使用的 Ecology 9 流程页。
// 对话跳转与工作台共用这一条；连接里另填模板时以连接为准。
const weaverE9DefaultProcessURLTemplate = "/workflow/request/ViewRequestForwardSPA.jsp?requestid={process_id}"

// weaverE9QueryAssistNote 保持原 E9 提示词原文，避免已在用的 SQL 助手口径变化。
const weaverE9QueryAssistNote = "泛微 Ecology 9（weaver_e9/ecology9）的 requestid 只用于关联，不能判断归档；仅在业务明确要求且提供 workflow_requestbase 的真实字段时按 currentnodetype='3' 筛选已归档记录。不得把泛微字段约定套用到其他 OA，也不得臆造流程主表、明细表或建模表关系。"

var profiles = map[string]OAProfile{
	oaTypeWeaverE9: {
		Type:                      oaTypeWeaverE9,
		DisplayName:               "泛微 Ecology 9",
		Implemented:               true,
		Drivers:                   []string{"mysql", "oracle", "dm"},
		DefaultProcessURLTemplate: weaverE9DefaultProcessURLTemplate,
		AttachmentAuth:            AttachmentAuthWeaverApp,
		WorkflowArchive: &WorkflowArchiveRule{
			RequestTable:        "workflow_requestbase",
			RequestIDColumn:     "requestid",
			FormRequestIDColumn: "requestid",
			StatusColumn:        "currentnodetype",
			StatusValue:         "3",
		},
		QueryAssistNote: weaverE9QueryAssistNote,
	},
	// E10 只占位。不要把 E9 的 JSP、workflow_requestbase、currentnodetype 或附件三字段抄过来。
	// 表结构确认后需要补的内容见 ecology10.go。在此之前不要写入 oa_type_options。
	oaTypeWeaverE10: {
		Type:        oaTypeWeaverE10,
		Implemented: false,
	},
}

// Profile 按连接上的 oa_type 精确查找档案。ecology9 这个历史写法不在此列。
func Profile(oaType string) (OAProfile, bool) {
	profile, ok := profiles[oaType]
	return profile, ok
}

// MetadataProfile 查找跳转、归档和提示词使用的档案。
// ecology9 是早期显示名里的别称，元数据与 weaver_e9 相同，但不能据此创建适配器。
func MetadataProfile(oaType string) (OAProfile, bool) {
	if profile, ok := Profile(oaType); ok {
		return profile, true
	}
	if oaType == "ecology9" {
		return Profile(oaTypeWeaverE9)
	}
	return OAProfile{}, false
}

// WorkflowArchive 返回该 OA 类型的归档关联规则。没有规则时第二个返回值为 false。
func WorkflowArchive(oaType string) (WorkflowArchiveRule, bool) {
	profile, ok := MetadataProfile(oaType)
	if !ok || profile.WorkflowArchive == nil {
		return WorkflowArchiveRule{}, false
	}
	return *profile.WorkflowArchive, true
}
