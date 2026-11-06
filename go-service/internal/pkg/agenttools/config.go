package agenttools

import (
	"encoding/json"
	"fmt"

	"auraoa/go-service/internal/pkg/ai"
)

type toolConfigField struct {
	Key      string
	Fallback int
	Min      int
	Max      int
}

// 各工具可在智能体上单独配置的数值。未保存时运行时使用 Fallback。
var toolConfigFields = map[string][]toolConfigField{
	"list_my_todos": {
		{Key: "page_size", Fallback: 20, Min: 1, Max: 200},
		{Key: "max_page_size", Fallback: 50, Min: 1, Max: 200},
	},
	"list_my_requests": {
		{Key: "page_size", Fallback: 20, Min: 1, Max: 200},
		{Key: "max_page_size", Fallback: 50, Min: 1, Max: 200},
	},
	"execute_safe_query": {
		{Key: "max_rows", Fallback: 100, Min: 0, Max: 5000},
		{Key: "max_queries", Fallback: 5, Min: 1, Max: 10},
	},
	"get_latest_audit": {
		{Key: "max_batch", Fallback: 50, Min: 1, Max: 100},
	},
	"get_latest_summary": {
		{Key: "max_batch", Fallback: 50, Min: 1, Max: 100},
	},
}

// ClampLoopSteps 把单轮对话步数收拢到 1-20。未配置或 0 使用默认 8。
func ClampLoopSteps(steps int) int {
	if steps <= 0 {
		return 8
	}
	if steps > 20 {
		return 20
	}
	return steps
}

// ParseToolConfig 读取工具绑定上的 JSON 配置。无法识别的值忽略。
func ParseToolConfig(raw []byte) map[string]int {
	out := map[string]int{}
	if len(raw) == 0 || string(raw) == "null" || string(raw) == "{}" {
		return out
	}
	var generic map[string]interface{}
	if err := json.Unmarshal(raw, &generic); err != nil {
		return out
	}
	for key, value := range generic {
		switch n := value.(type) {
		case float64:
			out[key] = int(n)
		case int:
			out[key] = n
		}
	}
	return out
}

// SanitizeToolConfigs 只保留已知工具的已知字段，并把数值收拢到允许范围。
func SanitizeToolConfigs(raw map[string]map[string]int) map[string]map[string]int {
	if raw == nil {
		return nil
	}
	out := map[string]map[string]int{}
	for code, fields := range raw {
		cleaned := sanitizeOneToolConfig(code, fields)
		if len(cleaned) > 0 {
			out[code] = cleaned
		}
	}
	return out
}

func sanitizeOneToolConfig(code string, raw map[string]int) map[string]int {
	spec, ok := toolConfigFields[code]
	if !ok || raw == nil {
		return nil
	}
	out := map[string]int{}
	for _, field := range spec {
		value, exists := raw[field.Key]
		if !exists {
			value = field.Fallback
		}
		if value < field.Min {
			value = field.Min
		}
		if value > field.Max {
			value = field.Max
		}
		out[field.Key] = value
	}
	if page, ok := out["page_size"]; ok {
		if maxPage, ok := out["max_page_size"]; ok && page > maxPage {
			out["page_size"] = maxPage
		}
	}
	return out
}

// ConfigInt 读取配置。键不存在时返回 fallback，显式 0 会保留。
func ConfigInt(cfg map[string]int, key string, fallback int) int {
	if cfg == nil {
		return fallback
	}
	value, ok := cfg[key]
	if !ok {
		return fallback
	}
	return value
}

func toolConfigOf(execCtx *ExecutionContext, toolCode string) map[string]int {
	if execCtx == nil || execCtx.ToolConfigs == nil {
		return nil
	}
	return execCtx.ToolConfigs[toolCode]
}

// ResolvePageSize 按智能体配置决定本次查询条数。模型未传时用配置的默认值，超过上限时截断。
func ResolvePageSize(execCtx *ExecutionContext, toolCode string, requested int) int {
	cfg := toolConfigOf(execCtx, toolCode)
	maxSize := ConfigInt(cfg, "max_page_size", 50)
	if maxSize < 1 {
		maxSize = 50
	}
	if maxSize > 200 {
		maxSize = 200
	}
	size := requested
	if size <= 0 {
		size = ConfigInt(cfg, "page_size", 20)
	}
	if size < 1 {
		size = 1
	}
	if size > maxSize {
		size = maxSize
	}
	return size
}

// ResolveBatchLimit 返回审核或总结一次最多查询的流程数。
func ResolveBatchLimit(execCtx *ExecutionContext, toolCode string) int {
	limit := ConfigInt(toolConfigOf(execCtx, toolCode), "max_batch", 50)
	if limit < 1 {
		return 50
	}
	if limit > 100 {
		return 100
	}
	return limit
}

// ResolveMaxRows 返回单条 SQL 的行数上限。0 表示不截断。
func ResolveMaxRows(execCtx *ExecutionContext) int {
	cfg := toolConfigOf(execCtx, "execute_safe_query")
	if cfg == nil {
		return 100
	}
	value, ok := cfg["max_rows"]
	if !ok {
		return 100
	}
	if value < 0 {
		return 100
	}
	if value > 5000 {
		return 5000
	}
	return value
}

// ResolveMaxQueries 返回一次工具调用最多接受的 SQL 条数。
func ResolveMaxQueries(execCtx *ExecutionContext) int {
	limit := ConfigInt(toolConfigOf(execCtx, "execute_safe_query"), "max_queries", 5)
	if limit < 1 {
		return 5
	}
	if limit > 10 {
		return 10
	}
	return limit
}

// WrapSafeSQL 按数据库方言给只读查询加上行数限制。maxRows 为 0 时原样执行。
func WrapSafeSQL(driver, cleanSQL string, maxRows int) string {
	if maxRows <= 0 {
		return cleanSQL
	}
	if driver == "oracle" {
		return fmt.Sprintf("SELECT * FROM (%s) safe_sub WHERE ROWNUM <= %d", cleanSQL, maxRows)
	}
	return fmt.Sprintf("SELECT * FROM (%s) AS safe_sub LIMIT %d", cleanSQL, maxRows)
}

// AnnotateToolDefinition 把该智能体的工具限额写进模型看到的工具说明。
func AnnotateToolDefinition(def ai.ToolDefinition, cfg map[string]int) ai.ToolDefinition {
	note := toolConfigNote(def.Function.Name, cfg)
	if note == "" {
		return def
	}
	def.Function.Description = def.Function.Description + " " + note
	return def
}

func toolConfigNote(code string, cfg map[string]int) string {
	switch code {
	case "list_my_todos", "list_my_requests":
		pageSize := ConfigInt(cfg, "page_size", 20)
		maxSize := ConfigInt(cfg, "max_page_size", 50)
		return fmt.Sprintf("当前智能体配置：未指定 page_size 时每页 %d 条，单页最多 %d 条。", pageSize, maxSize)
	case "execute_safe_query":
		rows := 100
		if cfg != nil {
			if value, ok := cfg["max_rows"]; ok {
				rows = value
			}
		}
		queries := ConfigInt(cfg, "max_queries", 5)
		rowText := fmt.Sprintf("每条 SQL 最多返回 %d 行", rows)
		if rows == 0 {
			rowText = "每条 SQL 不限制行数，仍受查询超时约束"
		}
		return fmt.Sprintf("当前智能体配置：%s，一次调用最多 %d 条 SQL。需要多组数据时把它们放进 queries 数组一次提交，不要拆成多次调用。", rowText, queries)
	case "get_latest_audit", "get_latest_summary":
		return fmt.Sprintf("当前智能体配置：process_ids 一次最多 %d 条。", ConfigInt(cfg, "max_batch", 50))
	default:
		return ""
	}
}
