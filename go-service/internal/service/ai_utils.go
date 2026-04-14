package service

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// SelectedFieldSet 描述用户最终生效的字段选择集合。
// key 为 "main"（主表）或明细表名（如 "formtable_main_151_dt1"），value 为字段 key 的集合。
// value 为 nil 时表示该表全选所有字段，不做过滤。
type SelectedFieldSet map[string]map[string]bool

// ── JSON 清理 ──

// cleanJSONResponse 对 AI 返回的原始文本做多步清理，提取出可解析的 JSON 对象。
// 处理顺序：去除首尾空白 → 剥离省略号前缀 → 提取 markdown 代码块 → 截取首尾花括号范围。
func cleanJSONResponse(raw string) string {
	s := strings.TrimSpace(raw)
	s = stripLeadingEllipsisPrefix(s)
	s = extractJSONFromMarkdownFence(s)
	s = strings.TrimSpace(s)
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		s = s[start : end+1]
	}
	return s
}

// extractJSONFromMarkdownFence 从 markdown 代码块中提取 JSON 正文。
// 支持 ```json ... ``` 和 ``` ... ``` 两种格式，避免模型包裹代码块导致解析失败。
func extractJSONFromMarkdownFence(s string) string {
	lower := strings.ToLower(s)
	if idx := strings.Index(lower, "```json"); idx >= 0 {
		inner := s[idx+7:]
		inner = strings.TrimLeft(inner, " \t\r\n")
		if end := strings.Index(inner, "```"); end >= 0 {
			inner = inner[:end]
		}
		return strings.TrimSpace(inner)
	}
	if idx := strings.Index(s, "```"); idx >= 0 {
		inner := s[idx+3:]
		inner = strings.TrimLeft(inner, " \t\r\n")
		if end := strings.Index(inner, "```"); end >= 0 {
			inner = inner[:end]
		}
		return strings.TrimSpace(inner)
	}
	return s
}

// stripLeadingEllipsisPrefix 去掉模型在 JSON 前附加的省略号（...、…、连续英文句点），
// 防止首字符不是 { 导致后续截取错位。
func stripLeadingEllipsisPrefix(s string) string {
	for {
		t := strings.TrimSpace(s)
		if t == "" {
			return t
		}
		if strings.HasPrefix(t, "...") {
			s = t[3:]
			continue
		}
		if strings.HasPrefix(t, "…") {
			s = t[len("…"):]
			continue
		}
		// 文首连续英文句点（如 .. 或单独 .）
		i := 0
		for i < len(t) && t[i] == '.' {
			i++
		}
		if i > 0 {
			s = t[i:]
			continue
		}
		break
	}
	return strings.TrimSpace(s)
}

// ── 数值处理 ──

// pickOverallScoreInt 优先取 overall 字段，为零时回退到 score 字段，结果钳制到 [0, 100]。
func pickOverallScoreInt(overall, score float64) int {
	if overall != 0 {
		return clampPercentInt(overall)
	}
	return clampPercentInt(score)
}

// clampPercentInt 将浮点数四舍五入并钳制到 [0, 100] 整数范围。
func clampPercentInt(v float64) int {
	if v <= 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return int(v + 0.5)
}

// ── 集合工具 ──

// coalesceStringSlice 将 nil 切片转为空切片，确保 JSON 序列化输出 [] 而非 null。
func coalesceStringSlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// firstNonEmpty 返回参数列表中第一个非空白字符串，全为空时返回空字符串。
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// ── 字段过滤 ──

// filterFields 从 map 中只保留 allowedKeys 指定的字段（大小写不敏感匹配）。
// allowedKeys 为 nil 时直接返回原始 data，表示全选不过滤。
func filterFields(data map[string]interface{}, allowedKeys map[string]bool) map[string]interface{} {
	if data == nil {
		return nil
	}
	if allowedKeys == nil {
		return data
	}
	filtered := make(map[string]interface{})
	for k, v := range data {
		normalKey := strings.ToLower(k)
		if allowedKeys[normalKey] || allowedKeys[k] {
			filtered[k] = v
		}
	}
	return filtered
}

// filterRowFields 对一组明细行批量执行字段过滤，allowedKeys 为 nil 时原样返回。
func filterRowFields(rows []map[string]interface{}, allowedKeys map[string]bool) []map[string]interface{} {
	if allowedKeys == nil {
		return rows
	}
	result := make([]map[string]interface{}, len(rows))
	for i, row := range rows {
		result[i] = filterFields(row, allowedKeys)
	}
	return result
}

// ── 格式化 ──

// formatMainData 将主表数据格式化为缩进 JSON 字符串，数据为空时返回占位提示。
func formatMainData(data map[string]interface{}) string {
	if len(data) == 0 {
		return "（无主表数据）"
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", data)
	}
	return string(b)
}

// formatGroupedDetailData 将按表分组的明细数据格式化为带表名标签的文本。
func formatGroupedDetailData(detailTables map[string][]map[string]interface{}, fieldSet SelectedFieldSet) string {
	if len(detailTables) == 0 {
		return "（无明细表数据）"
	}
	// 按表名排序保证输出稳定
	tableNames := make([]string, 0, len(detailTables))
	for name := range detailTables {
		tableNames = append(tableNames, name)
	}
	sort.Strings(tableNames)

	var sb strings.Builder
	for _, tableName := range tableNames {
		rows := detailTables[tableName]
		// 从表名提取友好标签（如 formtable_main_151_dt1 → 明细表1）
		label := tableName
		if idx := strings.LastIndex(tableName, "_dt"); idx != -1 && idx+3 < len(tableName) {
			label = "明细表" + tableName[idx+3:]
		}

		// 按用户选择的字段过滤
		var allowedKeys map[string]bool
		if fieldSet != nil {
			allowedKeys = fieldSet[tableName]
		}
		filteredRows := filterRowFields(rows, allowedKeys)

		sb.WriteString(fmt.Sprintf("### %s（%s）共 %d 行\n", label, tableName, len(filteredRows)))
		b, err := json.MarshalIndent(filteredRows, "", "  ")
		if err != nil {
			sb.WriteString(fmt.Sprintf("%v\n", filteredRows))
		} else {
			sb.Write(b)
			sb.WriteByte('\n')
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

// ── 通用工具 ──

// truncate 截断字符串到指定字节长度，超出时追加省略号，用于日志和错误信息中的内容预览。
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
