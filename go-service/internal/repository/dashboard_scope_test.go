package repository

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestResolvedDepartmentByNameSQLAlignsEmbedAndOrg(t *testing.T) {
	sql := resolvedDepartmentByNameSQL("psl.tenant_id", "d.name", "psl.oa_operator_dept")
	for _, fragment := range []string{
		"NULLIF(TRIM(d.name), '')",
		"NULLIF(TRIM(psl.oa_operator_dept), '')",
		"d.tenant_id = psl.tenant_id",
		"'未分配'",
	} {
		if !strings.Contains(sql, fragment) {
			t.Errorf("部门名称对齐 SQL 缺少 %q: %s", fragment, sql)
		}
	}
}

func TestDashboardPersonalScopeSQL(t *testing.T) {
	if got := auditDashboardPersonalWhere(nil); got != "" {
		t.Fatalf("租户管理员审核范围应为全渠道，实际 %q", got)
	}
	id := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	if got := auditDashboardPersonalWhere(&id); !strings.Contains(got, "embed_personal") || !strings.Contains(got, "workbench") {
		t.Fatalf("个人审核范围应排除通用嵌入: %q", got)
	}
	if got := summaryDashboardPersonalWhere(&id); !strings.Contains(got, "summary_workbench") {
		t.Fatalf("个人总结范围应仅工作台: %q", got)
	}
}
