package label

// RecommendationZh 将审核建议英文值映射为中文。
// 未知值原样返回。
func RecommendationZh(val string) string {
	m := map[string]string{
		"approve": "通过",
		"return":  "退回",
		"review":  "人工复核",
	}
	if zh, ok := m[val]; ok {
		return zh
	}
	return val
}

// ComplianceZh 将合规性英文值映射为中文。
// 未知值原样返回。
func ComplianceZh(val string) string {
	m := map[string]string{
		"compliant":           "合规",
		"non_compliant":       "不合规",
		"partially_compliant": "部分合规",
	}
	if zh, ok := m[val]; ok {
		return zh
	}
	return val
}
