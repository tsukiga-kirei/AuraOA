package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"auraoa/go-service/internal/model"
	"github.com/google/uuid"
)

// reviewResultObject 区分缺失、null 与合法零值，避免将无效模型响应生成业务结论。
func reviewResultObject(raw string) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(cleanJSONResponse(raw)), &fields); err != nil || len(fields) == 0 {
		return nil, fmt.Errorf("AI 未返回有效的结构化审核结果")
	}
	return fields, nil
}

func reviewRequiredValue(fields map[string]json.RawMessage, key string) (json.RawMessage, error) {
	raw := bytes.TrimSpace(fields[key])
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil, fmt.Errorf("AI 结果缺少有效字段：%s", key)
	}
	return raw, nil
}

func reviewString(fields map[string]json.RawMessage, key string, nonempty bool) (string, error) {
	raw, err := reviewRequiredValue(fields, key)
	var value string
	if err != nil {
		return "", err
	}
	if json.Unmarshal(raw, &value) != nil || (nonempty && strings.TrimSpace(value) == "") {
		return "", fmt.Errorf("AI 结果文字字段无效：%s", key)
	}
	return value, nil
}

func reviewPercent(fields map[string]json.RawMessage, key string) (float64, error) {
	raw, err := reviewRequiredValue(fields, key)
	var value float64
	if err != nil {
		return 0, err
	}
	if json.Unmarshal(raw, &value) != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 100 {
		return 0, fmt.Errorf("AI 结果数值须在0至100之间：%s", key)
	}
	return value, nil
}

func reviewArray(fields map[string]json.RawMessage, key string) ([]json.RawMessage, error) {
	raw, err := reviewRequiredValue(fields, key)
	if err != nil {
		return nil, err
	}
	var items []json.RawMessage
	if raw[0] != '[' || json.Unmarshal(raw, &items) != nil {
		return nil, fmt.Errorf("AI 结果数组字段无效：%s", key)
	}
	return items, nil
}

// validateReviewRows 校验每一条检查记录，显式 false 可以保留，缺少布尔值不能当作不通过。
func validateReviewRows(fields map[string]json.RawMessage, key, nameKey, boolKey, reasonKey string) error {
	items, err := reviewArray(fields, key)
	if err != nil {
		return err
	}
	for _, raw := range items {
		var row map[string]json.RawMessage
		if json.Unmarshal(raw, &row) != nil || row == nil {
			return fmt.Errorf("AI 结果检查记录无效：%s", key)
		}
		if _, err := reviewString(row, nameKey, true); err != nil {
			return err
		}
		value, err := reviewRequiredValue(row, boolKey)
		var passed bool
		if err != nil {
			return err
		}
		if json.Unmarshal(value, &passed) != nil {
			return fmt.Errorf("AI 结果判断字段无效：%s", boolKey)
		}
		if _, err := reviewString(row, reasonKey, false); err != nil {
			return err
		}
	}
	return nil
}

func validateReviewCommon(fields map[string]json.RawMessage) (float64, error) {
	scoreKey := "overall_score"
	if _, ok := fields[scoreKey]; !ok {
		scoreKey = "score" // 兼容旧模板，但显式零分不回退到别名。
	}
	score, err := reviewPercent(fields, scoreKey)
	if err != nil {
		return 0, err
	}
	if _, err := reviewPercent(fields, "confidence"); err != nil {
		return 0, err
	}
	for _, key := range []string{"risk_points", "suggestions"} {
		items, err := reviewArray(fields, key)
		if err != nil {
			return 0, err
		}
		for _, item := range items {
			var text string
			if bytes.Equal(bytes.TrimSpace(item), []byte("null")) || json.Unmarshal(item, &text) != nil {
				return 0, fmt.Errorf("AI 结果数组须包含文字：%s", key)
			}
		}
	}
	return score, nil
}

func validateArchiveReviewStructure(fields map[string]json.RawMessage) error {
	raw, err := reviewRequiredValue(fields, "flow_audit")
	if err != nil {
		return err
	}
	var flow map[string]json.RawMessage
	if json.Unmarshal(raw, &flow) != nil || flow == nil {
		return fmt.Errorf("AI 结果审批流核验结构无效")
	}
	value, err := reviewRequiredValue(flow, "is_complete")
	var complete bool
	if err != nil {
		return err
	}
	if json.Unmarshal(value, &complete) != nil {
		return fmt.Errorf("AI 结果审批流完整性字段无效")
	}
	missing, err := reviewArray(flow, "missing_nodes")
	if err != nil {
		return err
	}
	for _, item := range missing {
		var name string
		if bytes.Equal(bytes.TrimSpace(item), []byte("null")) || json.Unmarshal(item, &name) != nil {
			return fmt.Errorf("AI 结果缺失节点须为文字")
		}
	}
	if err := validateReviewRows(flow, "node_results", "node_name", "compliant", "reasoning"); err != nil {
		return err
	}
	if err := validateReviewRows(fields, "field_audit", "field_name", "passed", "reasoning"); err != nil {
		return err
	}
	key := "ai_summary"
	if _, ok := fields[key]; !ok {
		key = "summary"
	}
	_, err = reviewString(fields, key, false)
	return err
}

// validateReviewRuleCoverage 以本次启用规则为准检查覆盖，不将遗漏规则解释为通过。
func validateReviewRuleCoverage(expected, actual []string) error {
	want, got := map[string]int{}, map[string]int{}
	for _, name := range expected {
		want[strings.TrimSpace(stripRuleScopePrefix(name))]++
	}
	for _, name := range actual {
		got[strings.TrimSpace(stripRuleScopePrefix(name))]++
	}
	// 个人规则也会进入提示词，允许返回额外条目；租户已启用规则不能被遗漏或合并掉。
	for name, count := range want {
		if got[name] < count {
			return fmt.Errorf("AI 结果未完整覆盖本次启用的审核规则")
		}
	}
	return nil
}

func validateAuditRuleCoverage(result *model.AuditResultJSON, rules []model.AuditRule) error {
	var expected, actual []string
	for _, rule := range rules {
		if rule.IsEnabled() {
			expected = append(expected, rule.RuleContent)
		}
	}
	for _, rule := range result.RuleResults {
		actual = append(actual, rule.RuleContent)
	}
	return validateReviewRuleCoverage(expected, actual)
}

func validateArchiveRuleCoverage(result *model.ArchiveResultJSON, rules []model.ArchiveRule) error {
	var expected, actual []string
	byID := map[string]string{}
	for _, rule := range rules {
		if rule.IsEnabled() {
			expected = append(expected, rule.RuleContent)
			if rule.ID != uuid.Nil {
				byID[rule.ID.String()] = rule.RuleContent
			}
		}
	}
	for _, rule := range result.RuleAudit {
		// 归档输出同时有规则 ID 和展示名，优先以已知 ID 识别，兼容合理简写的展示名。
		if content, ok := byID[rule.RuleID]; ok {
			actual = append(actual, content)
		} else {
			actual = append(actual, rule.RuleName)
		}
	}
	return validateReviewRuleCoverage(expected, actual)
}
