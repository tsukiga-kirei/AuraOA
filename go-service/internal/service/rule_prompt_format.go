package service

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"auraoa/go-service/internal/model"

	"gorm.io/datatypes"
)

// ruleScopeLabels 规则作用域的中文标签（用于 Prompt，避免模型回显枚举值 default_on）。
var ruleScopeLabels = map[string]string{
	"mandatory":   "强制执行",
	"default_on":  "默认开启",
	"default_off": "默认关闭",
	"custom":      "用户自定义",
}

// ruleScopePrefixRE 剥离规则结果中可能携带的作用域前缀（兼容历史 [default_on] 与新格式 （默认开启））。
var ruleScopePrefixRE = regexp.MustCompile(`^(?:\d+\.\s*)?(?:\[(?:mandatory|default_on|default_off|custom)\]|（(?:强制执行|默认开启|默认关闭|用户自定义)）|\[用户自定义\])\s*`)

func ruleScopeLabel(scope string) string {
	if label, ok := ruleScopeLabels[scope]; ok {
		return label
	}
	return scope
}

// formatRuleLineForPrompt 将单条规则格式化为送入 AI 的文本行。
// scope 为空时仅输出序号与规则正文，避免无意义标签。
func formatRuleLineForPrompt(index int, scope, content string) string {
	content = strings.TrimSpace(content)
	if scope == "" {
		return fmt.Sprintf("%d. %s", index, content)
	}
	if _, known := ruleScopeLabels[scope]; !known {
		return fmt.Sprintf("%d. %s", index, content)
	}
	return fmt.Sprintf("%d. （%s）%s", index, ruleScopeLabel(scope), content)
}

// stripRuleScopePrefix 去除 rule_content / rule_name 中的作用域前缀（兜底，兼容历史数据）。
func stripRuleScopePrefix(content string) string {
	s := strings.TrimSpace(content)
	for {
		next := ruleScopePrefixRE.ReplaceAllString(s, "")
		next = strings.TrimSpace(next)
		if next == s {
			break
		}
		s = next
	}
	return s
}

// normalizeAuditRuleResults 清洗审核规则结果中的 rule_content 字段。
func normalizeAuditRuleResults(rules []model.RuleResultJSON) []model.RuleResultJSON {
	if rules == nil {
		return []model.RuleResultJSON{}
	}
	out := make([]model.RuleResultJSON, len(rules))
	for i, r := range rules {
		out[i] = r
		out[i].RuleContent = stripRuleScopePrefix(r.RuleContent)
	}
	return out
}

// normalizeArchiveRuleAudits 清洗归档规则审计中的 rule_name 字段。
func normalizeArchiveRuleAudits(rules []model.ArchiveRuleAuditJSON) []model.ArchiveRuleAuditJSON {
	if rules == nil {
		return []model.ArchiveRuleAuditJSON{}
	}
	out := make([]model.ArchiveRuleAuditJSON, len(rules))
	for i, r := range rules {
		out[i] = r
		out[i].RuleName = stripRuleScopePrefix(r.RuleName)
		if out[i].RuleID == "" || out[i].RuleID == r.RuleName {
			out[i].RuleID = out[i].RuleName
		} else {
			out[i].RuleID = stripRuleScopePrefix(out[i].RuleID)
		}
	}
	return out
}

// normalizeArchiveLogStoredResult 清洗 archive_logs.archive_result JSONB 中的 rule_audit（读库兜底）。
func normalizeArchiveLogStoredResult(raw datatypes.JSON) datatypes.JSON {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "{}" || trimmed == "null" {
		return raw
	}
	var parsed model.ArchiveResultJSON
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return raw
	}
	parsed.RuleAudit = normalizeArchiveRuleAudits(parsed.RuleAudit)
	out, err := json.Marshal(parsed)
	if err != nil {
		return raw
	}
	return datatypes.JSON(out)
}

// normalizeArchiveLogInPlace 就地清洗单条归档日志的存储结果。
func normalizeArchiveLogInPlace(log *model.ArchiveLog) {
	if log == nil {
		return
	}
	log.ArchiveResult = normalizeArchiveLogStoredResult(log.ArchiveResult)
}

// NormalizeArchiveLogForResponse 对外返回归档日志前清洗 archive_result（历史数据兜底）。
func NormalizeArchiveLogForResponse(log *model.ArchiveLog) {
	normalizeArchiveLogInPlace(log)
}

// normalizeAuditLogStoredResult 清洗 audit_logs.audit_result JSONB 中的 rule_results（读库兜底）。
func normalizeAuditLogStoredResult(raw datatypes.JSON) datatypes.JSON {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "{}" || trimmed == "null" {
		return raw
	}
	var parsed model.AuditResultJSON
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return raw
	}
	parsed.RuleResults = normalizeAuditRuleResults(parsed.RuleResults)
	out, err := json.Marshal(parsed)
	if err != nil {
		return raw
	}
	return datatypes.JSON(out)
}

// normalizeAuditLogInPlace 就地清洗单条审核日志的存储结果。
func normalizeAuditLogInPlace(log *model.AuditLog) {
	if log == nil {
		return
	}
	log.AuditResult = normalizeAuditLogStoredResult(log.AuditResult)
}

// NormalizeAuditLogForResponse 对外返回审核日志前清洗 audit_result（历史数据兜底）。
func NormalizeAuditLogForResponse(log *model.AuditLog) {
	normalizeAuditLogInPlace(log)
}
