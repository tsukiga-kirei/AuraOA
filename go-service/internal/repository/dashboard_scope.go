package repository

import "github.com/google/uuid"

// resolvedDepartmentByNameSQL 将组织当前部门名与 OA 嵌入部门快照按去空格后的名称对齐。
// 嵌入部门名能匹配到同名组织部门时并入该部门；否则保留快照名；都空则「未分配」。
func resolvedDepartmentByNameSQL(tenantExpr, orgNameExpr, embedNameExpr string) string {
	return `COALESCE(
		NULLIF(TRIM(` + orgNameExpr + `), ''),
		(SELECT d.name FROM departments d
		  WHERE d.tenant_id = ` + tenantExpr + `
		    AND NULLIF(TRIM(d.name), '') IS NOT NULL
		    AND NULLIF(TRIM(d.name), '') = NULLIF(TRIM(` + embedNameExpr + `), '')
		  ORDER BY d.id
		  LIMIT 1),
		NULLIF(TRIM(` + embedNameExpr + `), ''),
		'未分配')`
}

// auditDashboardPersonalWhere 个人视角只统计工作台与个人嵌入审核，排除 OA 通用嵌入。
func auditDashboardPersonalWhere(userID *uuid.UUID) string {
	if userID == nil {
		return ""
	}
	return "AND al.user_id = ? AND aps.channel IN ('workbench', 'embed_personal')"
}

// summaryDashboardPersonalWhere 个人视角只统计流程总结工作台，排除 OA 嵌入总结。
func summaryDashboardPersonalWhere(userID *uuid.UUID) string {
	if userID == nil {
		return ""
	}
	return "AND psl.user_id = ? AND psl.trigger_source = 'summary_workbench'"
}
