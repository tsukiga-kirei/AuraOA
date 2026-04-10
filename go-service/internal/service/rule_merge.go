package service

import (
	"sort"
)

// MergeableRule 可合并规则的通用接口，支持 AuditRule 和 ArchiveRule。
type MergeableRule interface {
	GetID() string
	GetRuleContent() string
	GetRuleScope() string
	IsEnabled() bool
}

// UserRuleOverride 用户规则配置覆盖，抽象自 AuditDetailItem / ArchiveDetailItem。
type UserRuleOverride struct {
	CustomRules         []CustomRuleItem
	RuleToggleOverrides []RuleToggleItem
}

// CustomRuleItem 用户自定义规则条目。
type CustomRuleItem struct {
	ID      string
	Content string
	Enabled bool
}

// RuleToggleItem 用户对租户规则的开关覆盖条目。
type RuleToggleItem struct {
	RuleID  string
	Enabled bool
}

// MergedRule 合并后的最终生效规则。
type MergedRule struct {
	RuleID  string `json:"rule_id"`
	Content string `json:"content"`
	Scope   string `json:"scope"`   // mandatory | default_on | default_off | custom
	Enabled bool   `json:"enabled"`
	Source  string `json:"source"`  // tenant | user
}

// MergeRules 合并租户规则和用户个性化配置，返回最终生效的规则列表。
// 优先级：mandatory 始终生效 > 用户私有规则 > 用户 toggle 覆盖 > 租户默认规则
func MergeRules(tenantRules []MergeableRule, userOverride *UserRuleOverride) []MergedRule {
	var result []MergedRule

	// 构建用户 toggle 覆盖映射
	toggleMap := make(map[string]bool)
	if userOverride != nil {
		for _, toggle := range userOverride.RuleToggleOverrides {
			toggleMap[toggle.RuleID] = toggle.Enabled
		}
	}

	// 处理租户规则
	for _, rule := range tenantRules {
		if !rule.IsEnabled() {
			continue
		}

		merged := MergedRule{
			RuleID:  rule.GetID(),
			Content: rule.GetRuleContent(),
			Scope:   rule.GetRuleScope(),
			Source:  "tenant",
		}

		switch rule.GetRuleScope() {
		case "mandatory":
			// 强制规则始终生效，忽略用户 toggle
			merged.Enabled = true
		case "default_on":
			// 默认开启，用户可通过 toggle 关闭
			merged.Enabled = true
			if userEnabled, exists := toggleMap[rule.GetID()]; exists {
				merged.Enabled = userEnabled
			}
		case "default_off":
			// 默认关闭，用户可通过 toggle 开启
			merged.Enabled = false
			if userEnabled, exists := toggleMap[rule.GetID()]; exists {
				merged.Enabled = userEnabled
			}
		default:
			merged.Enabled = true
		}

		result = append(result, merged)
	}

	// 添加用户私有规则
	if userOverride != nil {
		for _, customRule := range userOverride.CustomRules {
			result = append(result, MergedRule{
				RuleID:  customRule.ID,
				Content: customRule.Content,
				Scope:   "custom",
				Enabled: customRule.Enabled,
				Source:  "user",
			})
		}
	}

	// 按优先级排序：mandatory > custom > default_on > default_off
	scopePriority := map[string]int{
		"mandatory":   0,
		"custom":      1,
		"default_on":  2,
		"default_off": 3,
	}

	sort.SliceStable(result, func(i, j int) bool {
		pi := scopePriority[result[i].Scope]
		pj := scopePriority[result[j].Scope]
		return pi < pj
	})

	if result == nil {
		result = []MergedRule{}
	}
	return result
}
