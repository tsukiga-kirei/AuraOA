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
	// 1. MySQL 方言测试
	adapterMySQL := &Ecology9Adapter{driver: "mysql"}
	fromWhereMySQL, argsMySQL := adapterMySQL.buildTodoFromJoinWhere(247, TodoListPagedFilter{
		Page:     1,
		PageSize: 20,
	})
	if len(argsMySQL) == 0 || argsMySQL[0] != 247 {
		t.Fatalf("expected args[0] to be 247, got %#v", argsMySQL)
	}
	if !containsStr(fromWhereMySQL, "CAST(co.isremark AS CHAR) IN ('0', '1', 'a', 'h', '9')") {
		t.Errorf("expected CAST(co.isremark AS CHAR) in SQL, got: %s", fromWhereMySQL)
	}
	if !containsStr(fromWhereMySQL, "formtable_main_%") {
		t.Errorf("expected formtable_main_%% in SQL, got: %s", fromWhereMySQL)
	}
	if !containsStr(fromWhereMySQL, "NOT LIKE '%系统提醒%'") {
		t.Errorf("expected NOT LIKE '%%系统提醒%%' in SQL, got: %s", fromWhereMySQL)
	}

	// 2. DM (达梦) 方言测试：确保使用 TO_CHAR 避免数字列抛出 -6128 错误
	adapterDM := &Ecology9Adapter{driver: "dm"}
	fromWhereDM, argsDM := adapterDM.buildTodoFromJoinWhere(247, TodoListPagedFilter{
		Page:     1,
		PageSize: 20,
	})
	if len(argsDM) == 0 || argsDM[0] != 247 {
		t.Fatalf("expected args[0] to be 247, got %#v", argsDM)
	}
	if !containsStr(fromWhereDM, "TO_CHAR(co.ISREMARK) IN ('0', '1', 'a', 'h', '9')") {
		t.Errorf("expected TO_CHAR(co.ISREMARK) in DM SQL, got: %s", fromWhereDM)
	}
}

func TestArchivedVisibilityCondition(t *testing.T) {
	userID := 247
	for _, driver := range []string{"mysql", "dm", "oracle"} {
		t.Run(driver, func(t *testing.T) {
			adapter := &Ecology9Adapter{driver: driver}
			condition, args := adapter.archivedVisibilityCondition(&userID)
			if !containsStr(condition, "visibility_log.") || !containsStr(condition, " = r.") {
				t.Fatalf("已办权限条件缺少审批历史关联: %s", condition)
			}
			if !containsStr(condition, "EXISTS") || !containsStr(condition, adapter.col("creater")+" = ?") {
				t.Fatalf("已办权限条件必须覆盖申请人与历史审批人: %s", condition)
			}
			if len(args) != 2 || args[0] != userID || args[1] != userID {
				t.Fatalf("已办权限参数不正确: %#v", args)
			}
		})
	}

	adapter := &Ecology9Adapter{driver: "mysql"}
	if condition, args := adapter.archivedVisibilityCondition(nil); condition != "" || len(args) != 0 {
		t.Fatalf("后台按流程补充快照时不应附加用户条件: condition=%q args=%#v", condition, args)
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
