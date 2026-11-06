package agenttools

import "testing"

func TestSanitizeToolConfigsClampsAndDropsUnknown(t *testing.T) {
	got := SanitizeToolConfigs(map[string]map[string]int{
		"list_my_todos": {
			"page_size":     80,
			"max_page_size": 30,
			"unknown":       9,
		},
		"execute_safe_query": {
			"max_rows":    -1,
			"max_queries": 99,
		},
		"not_a_tool": {"page_size": 1},
	})
	if got["list_my_todos"]["page_size"] != 30 || got["list_my_todos"]["max_page_size"] != 30 {
		t.Fatalf("分页配置未按上限收拢: %#v", got["list_my_todos"])
	}
	if _, ok := got["list_my_todos"]["unknown"]; ok {
		t.Fatal("未知配置项不应保留")
	}
	if got["execute_safe_query"]["max_rows"] != 0 || got["execute_safe_query"]["max_queries"] != 10 {
		t.Fatalf("查数配置未按边界收拢: %#v", got["execute_safe_query"])
	}
	if _, ok := got["not_a_tool"]; ok {
		t.Fatal("未知工具不应保留配置")
	}
}

func TestResolvePageSizeUsesConfiguredDefault(t *testing.T) {
	ctx := &ExecutionContext{ToolConfigs: map[string]map[string]int{
		"list_my_todos": {"page_size": 15, "max_page_size": 40},
	}}
	if got := ResolvePageSize(ctx, "list_my_todos", 0); got != 15 {
		t.Fatalf("未传条数时应使用配置默认值，得到 %d", got)
	}
	if got := ResolvePageSize(ctx, "list_my_todos", 80); got != 40 {
		t.Fatalf("超过上限时应截断，得到 %d", got)
	}
	if got := ResolvePageSize(nil, "list_my_todos", 0); got != 20 {
		t.Fatalf("没有配置时应回落到 20，得到 %d", got)
	}
}

func TestResolveMaxRowsKeepsExplicitZero(t *testing.T) {
	limited := &ExecutionContext{ToolConfigs: map[string]map[string]int{
		"execute_safe_query": {"max_rows": 0, "max_queries": 3},
	}}
	if got := ResolveMaxRows(limited); got != 0 {
		t.Fatalf("显式 0 应表示不限制，得到 %d", got)
	}
	if got := ResolveMaxQueries(limited); got != 3 {
		t.Fatalf("一次查询条数应为 3，得到 %d", got)
	}
	if got := ResolveMaxRows(nil); got != 100 {
		t.Fatalf("缺省行数应为 100，得到 %d", got)
	}
}

func TestWrapSafeSQL(t *testing.T) {
	if got := WrapSafeSQL("mysql", "SELECT 1", 0); got != "SELECT 1" {
		t.Fatalf("不限制时不应包裹: %s", got)
	}
	if got := WrapSafeSQL("oracle", "SELECT 1", 20); got != "SELECT * FROM (SELECT 1) safe_sub WHERE ROWNUM <= 20" {
		t.Fatalf("Oracle 包裹不正确: %s", got)
	}
}

func TestParseToolConfig(t *testing.T) {
	got := ParseToolConfig([]byte(`{"max_rows":0,"max_queries":4}`))
	if got["max_rows"] != 0 || got["max_queries"] != 4 {
		t.Fatalf("配置解析不正确: %#v", got)
	}
}
