package dto

// ===================== 归档规则 DTO =====================

// CreateArchiveRuleRequest 创建归档规则请求
type CreateArchiveRuleRequest struct {
	ConfigID    string `json:"config_id"`
	ProcessType string `json:"process_type" binding:"required"`
	RuleContent string `json:"rule_content" binding:"required"`
	RuleScope   string `json:"rule_scope"`
	Enabled     *bool  `json:"enabled"`
	Source      string `json:"source"`
	RelatedFlow bool   `json:"related_flow"`
}

// UpdateArchiveRuleRequest 更新归档规则请求
type UpdateArchiveRuleRequest struct {
	RuleContent string `json:"rule_content"`
	RuleScope   string `json:"rule_scope"`
	Enabled     *bool  `json:"enabled"`
	RelatedFlow *bool  `json:"related_flow"`
}

// ListArchiveRulesQuery 归档规则列表查询参数
type ListArchiveRulesQuery struct {
	ConfigID  string `form:"config_id" binding:"required"`
	RuleScope string `form:"rule_scope"`
	Enabled   *bool  `form:"enabled"`
}
