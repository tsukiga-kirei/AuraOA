package repository

// auditOperatorDisplaySQL 返回审核日志操作人展示表达式；通用嵌入优先使用 OA 人员姓名快照。
func auditOperatorDisplaySQL(logTable, userTable string) string {
	return "CASE WHEN " + logTable + ".trigger_source IN ('embed_auto', 'embed_manual') " +
		"AND COALESCE(" + logTable + ".trigger_detail, '') != 'personal_embed_manual' THEN " +
		"CASE WHEN NULLIF(TRIM(COALESCE(" + logTable + ".oa_operator_name, '')), '') IS NOT NULL " +
		"THEN 'OA 嵌入审核（' || TRIM(" + logTable + ".oa_operator_name) || '）' ELSE 'OA 嵌入审核' END " +
		"ELSE COALESCE(" + userTable + ".display_name, " + userTable + ".username, '') END"
}

// summaryOperatorDisplaySQL 返回总结日志操作人展示表达式；通用嵌入优先使用 OA 人员姓名快照。
func summaryOperatorDisplaySQL(logTable, userTable string) string {
	return "CASE WHEN " + logTable + ".trigger_source IN ('summary_embed_auto', 'summary_embed_manual') THEN " +
		"CASE WHEN NULLIF(TRIM(COALESCE(" + logTable + ".oa_operator_name, '')), '') IS NOT NULL " +
		"THEN 'OA 嵌入总结（' || TRIM(" + logTable + ".oa_operator_name) || '）' ELSE 'OA 嵌入总结' END " +
		"ELSE COALESCE(" + userTable + ".display_name, " + userTable + ".username, '') END"
}

// llmOperatorDisplaySQL 根据 AI 调用关联的业务日志返回实际展示操作人。
func llmOperatorDisplaySQL(logTable, userTable, auditTable, summaryTable string) string {
	return "CASE " +
		"WHEN " + logTable + ".request_type = 'audit' AND " + auditTable + ".trigger_source IN ('embed_auto', 'embed_manual') " +
		"AND COALESCE(" + auditTable + ".trigger_detail, '') != 'personal_embed_manual' THEN " +
		"CASE WHEN NULLIF(TRIM(COALESCE(" + auditTable + ".oa_operator_name, '')), '') IS NOT NULL " +
		"THEN 'OA 嵌入审核（' || TRIM(" + auditTable + ".oa_operator_name) || '）' ELSE 'OA 嵌入审核' END " +
		"WHEN " + logTable + ".request_type = 'summary' AND " + summaryTable + ".trigger_source IN ('summary_embed_auto', 'summary_embed_manual') THEN " +
		"CASE WHEN NULLIF(TRIM(COALESCE(" + summaryTable + ".oa_operator_name, '')), '') IS NOT NULL " +
		"THEN 'OA 嵌入总结（' || TRIM(" + summaryTable + ".oa_operator_name) || '）' ELSE 'OA 嵌入总结' END " +
		"ELSE COALESCE(" + userTable + ".display_name, " + userTable + ".username, '') END"
}

// llmTriggerDetailSQL 返回 AI 调用关联业务执行记录的触发动作。
func llmTriggerDetailSQL(logTable, auditTable, summaryTable string) string {
	return "CASE " +
		"WHEN " + logTable + ".request_type = 'audit' AND " + auditTable + ".trigger_source IN ('embed_auto', 'embed_manual') " +
		"AND COALESCE(" + auditTable + ".trigger_detail, '') != 'personal_embed_manual' THEN COALESCE(" + auditTable + ".trigger_detail, '') " +
		"WHEN " + logTable + ".request_type = 'summary' AND " + summaryTable + ".trigger_source IN ('summary_embed_auto', 'summary_embed_manual') " +
		"THEN COALESCE(" + summaryTable + ".trigger_detail, '') ELSE '' END"
}
