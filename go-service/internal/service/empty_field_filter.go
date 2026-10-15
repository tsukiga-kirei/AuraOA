package service

import (
	"strings"

	"auraoa/go-service/internal/pkg/oa"
)

// filterEmptyProcessDataForPrompt 仅为提示词生成过滤空值后的副本，保留原始 OA 数据和明细行位置。
// 统一开关优先生效；关闭时按归属表和字段标识匹配单字段设置，兼容 OA 大写列名。
func filterEmptyProcessDataForPrompt(data *oa.ProcessData, skipAll bool, refs []string) *oa.ProcessData {
	if !skipAll && len(refs) == 0 {
		return data
	}
	fields := make(SelectedFieldSet)
	for _, ref := range refs {
		table, key := parseFieldOverrideKey(strings.ToLower(strings.TrimSpace(ref)))
		if fields[table] == nil {
			fields[table] = make(map[string]bool)
		}
		fields[table][key] = true
	}
	filtered := *data
	filtered.MainData = filterEmptyPromptFields(data.MainData, skipAll, fields["main"])
	filtered.DetailTables = make(map[string][]map[string]interface{}, len(data.DetailTables))
	for table, rows := range data.DetailTables {
		filteredRows := make([]map[string]interface{}, len(rows))
		for i, row := range rows {
			filteredRows[i] = filterEmptyPromptFields(row, skipAll, fields[strings.ToLower(table)])
		}
		filtered.DetailTables[table] = filteredRows
	}
	return &filtered
}

func filterEmptyPromptFields(data map[string]interface{}, skipAll bool, fields map[string]bool) map[string]interface{} {
	if data == nil {
		return nil
	}
	filtered := make(map[string]interface{}, len(data))
	for key, value := range data {
		if (skipAll || fields[strings.ToLower(key)]) && isEmptyPromptField(value) {
			continue
		}
		filtered[key] = value
	}
	return filtered
}

// isEmptyPromptField 按 OA 原始字段值判断空值，数字 0、布尔 false 和字面字符串 "null" 均保留。
// 浏览/选择字段的 value/display 包装优先判断原始 value，避免空值回显后被误传。
func isEmptyPromptField(value interface{}) bool {
	switch v := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(v) == ""
	case []byte:
		return strings.TrimSpace(string(v)) == ""
	case map[string]interface{}:
		if raw, ok := v["value"]; ok {
			return isEmptyPromptField(raw)
		}
	}
	return false
}
