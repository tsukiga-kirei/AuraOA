package dto

// ===================== 管理端用户配置 DTO =====================

// AdminUserConfigListItem 管理员视图：单个用户的个人配置摘要（含成员信息）
type AdminUserConfigListItem struct {
	UserID              string               `json:"user_id"`
	MemberID            string               `json:"member_id"`
	Username            string               `json:"username"`
	DisplayName         string               `json:"display_name"`
	Department          string               `json:"department"`
	RoleNames           []string             `json:"role_names"`
	AuditProcessCount   int                  `json:"audit_process_count"`
	CronEmailCount      int                  `json:"cron_email_count"`
	ArchiveProcessCount int                  `json:"archive_process_count"`
	LastModified        string               `json:"last_modified"`
	AuditDetails        []AdminProcessDetail `json:"audit_details"`
	CronDetails         AdminCronDetail      `json:"cron_details"`
	ArchiveDetails      []AdminProcessDetail `json:"archive_details"`
}

// AdminProcessDetail 单个流程的用户个性化配置详情（审核工作台/归档复盘共用）
type AdminProcessDetail struct {
	ProcessType         string               `json:"process_type"`
	StrictnessOverride  string               `json:"strictness_override"`
	CustomRules         []AdminCustomRule    `json:"custom_rules"`
	FieldOverrides      []string             `json:"field_overrides"`
	RuleToggleOverrides []AdminRuleToggleItem `json:"rule_toggle_overrides"`
}

// AdminCustomRule 用户自定义规则（管理员视图）
type AdminCustomRule struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Enabled bool   `json:"enabled"`
}

// AdminRuleToggleItem 用户对租户规则的开关覆盖（管理员视图）
type AdminRuleToggleItem struct {
	RuleID       string `json:"rule_id"`
	RuleContent  string `json:"rule_content"`
	RuleScope    string `json:"rule_scope"`    // mandatory | default_on | default_off
	AdminEnabled bool   `json:"admin_enabled"` // 管理员当前启用状态（租户默认值）
	Enabled      bool   `json:"enabled"`       // 用户覆盖后的启用状态
}

// AdminCronDetail 用户定时任务偏好（管理员视图）
type AdminCronDetail struct {
	DefaultEmail string `json:"default_email"`
	EmailCount   int    `json:"email_count"`
}
