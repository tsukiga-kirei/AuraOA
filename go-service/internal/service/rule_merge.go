package service

import (
	"sort"

	"oa-smart-audit/go-service/internal/model"
)

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
func MergeRules(tenantRules []model.AuditRule, userDetail *model.AuditDetailItem) []MergedRule {
	var result []MergedRule

	// 构建用户 toggle 覆盖映射
	toggleMap := make(map[string]bool)
	if userDetail != nil {
		for _, toggle := range userDetail.RuleConfig.RuleToggleOverrides {
			toggleMap[toggle.RuleID] = toggle.Enabled
		}
	}

	// 处理租户规则
	for _, rule := range tenantRules {
		if !rule.Enabled {
			continue
		}

		merged := MergedRule{
			RuleID:  rule.ID.String(),
			Content: rule.RuleContent,
			Scope:   rule.RuleScope,
			Source:  "tenant",
		}

		switch rule.RuleScope {
		case "mandatory":
			// 强制规则始终生效，忽略用户 toggle
			merged.Enabled = true
		case "default_on":
			// 默认开启，用户可通过 toggle 关闭
			merged.Enabled = true
			if userEnabled, exists := toggleMap[rule.ID.String()]; exists {
				merged.Enabled = userEnabled
			}
		case "default_off":
			// 默认关闭，用户可通过 toggle 开启
			merged.Enabled = false
			if userEnabled, exists := toggleMap[rule.ID.String()]; exists {
				merged.Enabled = userEnabled
			}
		default:
			merged.Enabled = true
		}

		result = append(result, merged)
	}

	// 添加用户私有规则
	if userDetail != nil {
		for _, customRule := range userDetail.RuleConfig.CustomRules {
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
