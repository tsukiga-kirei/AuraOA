package oa

import "testing"

// TestEcology9DialectSQLFragments 固化 Ecology9 各数据库驱动的 SQL 方言差异，
// 避免后续只验证达梦时误把 MySQL 分页或标识符规则改坏。
func TestEcology9DialectSQLFragments(t *testing.T) {
	tests := []struct {
		name             string
		driver           string
		wantTable        string
		wantPage         string
		wantTextCast     string
		oracleCompatible bool
	}{
		{
			name:             "MySQL",
			driver:           "mysql",
			wantTable:        "workflow_requestbase",
			wantPage:         " LIMIT 20 OFFSET 40",
			wantTextCast:     "CAST(RB.id AS CHAR)",
			oracleCompatible: false,
		},
		{
			name:             "达梦",
			driver:           "dm",
			wantTable:        "WORKFLOW_REQUESTBASE",
			wantPage:         " LIMIT 20 OFFSET 40",
			wantTextCast:     "TO_CHAR(RB.ID)",
			oracleCompatible: true,
		},
		{
			name:             "Oracle",
			driver:           "oracle",
			wantTable:        "WORKFLOW_REQUESTBASE",
			wantPage:         " OFFSET 40 ROWS FETCH NEXT 20 ROWS ONLY",
			wantTextCast:     "TO_CHAR(RB.ID)",
			oracleCompatible: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := &Ecology9Adapter{driver: tt.driver}
			if got := adapter.tableName("workflow_requestbase"); got != tt.wantTable {
				t.Fatalf("tableName() = %q, want %q", got, tt.wantTable)
			}
			if got := adapter.limitOffsetClause(20, 40); got != tt.wantPage {
				t.Fatalf("limitOffsetClause() = %q, want %q", got, tt.wantPage)
			}
			if got := adapter.castToTextExpr("RB." + adapter.col("id")); got != tt.wantTextCast {
				t.Fatalf("castToTextExpr() = %q, want %q", got, tt.wantTextCast)
			}
			if got := adapter.isOracleCompatible(); got != tt.oracleCompatible {
				t.Fatalf("isOracleCompatible() = %v, want %v", got, tt.oracleCompatible)
			}
		})
	}
}

func TestMapTodoType(t *testing.T) {
	cases := map[string]string{
		"0": "待审批",
		"1": "待批注",
		"a": "待意见征询",
		"A": "待意见征询",
		"h": "待转办",
		"H": "待转办",
		"9": "抄送待反馈",
		"x": "待办",
	}
	for input, want := range cases {
		if got := mapTodoType(input); got != want {
			t.Errorf("mapTodoType(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestBuildTodoFromJoinWhereFilters(t *testing.T) {
	adapter := &Ecology9Adapter{driver: "mysql"}
	fromWhere, args := adapter.buildTodoFromJoinWhere(247, TodoListPagedFilter{
		Page:     1,
		PageSize: 20,
	})
	if len(args) == 0 || args[0] != 247 {
		t.Fatalf("expected args[0] to be 247, got %#v", args)
	}
	// 验证包含状态码扩展
	if !containsStr(fromWhere, "co.isremark IN ('0', '1', 'a', 'h', '9')") {
		t.Errorf("expected isremark IN ('0', '1', 'a', 'h', '9') in SQL, got: %s", fromWhere)
	}
	// 验证默认包含主表与系统提醒过滤
	if !containsStr(fromWhere, "formtable_main_%") {
		t.Errorf("expected formtable_main_%% in SQL, got: %s", fromWhere)
	}
	if !containsStr(fromWhere, "NOT LIKE '%系统提醒%'") {
		t.Errorf("expected NOT LIKE '%%系统提醒%%' in SQL, got: %s", fromWhere)
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > 0 && len(substr) > 0 && (stringContains(s, substr))))
}

func stringContains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

