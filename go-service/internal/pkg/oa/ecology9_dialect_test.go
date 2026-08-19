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
