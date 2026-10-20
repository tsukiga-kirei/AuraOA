package oa

import "strings"

// BuildProcessURL 根据 OA Web 地址、跳转模板、OA 类型和流程 ID 拼接流程详情 URL。
// 连接里写了模板时按模板生成，与 OA 类型无关。
// 模板留空时只使用该类型档案中的 DefaultProcessURLTemplate。
// 没有兜底模板时返回空字符串，不套用其他 OA 的地址。
func BuildProcessURL(baseURL, template, oaType, processID string) string {
	processID = strings.TrimSpace(processID)
	if processID == "" {
		return ""
	}

	baseURL = strings.TrimSpace(baseURL)
	template = strings.TrimSpace(template)
	// 连接既没有模板也没有 Web 地址时不生成链接。
	// 只有 Web 地址时才套用该 OA 类型自己的兜底模板。
	usingTypeDefault := false
	if template == "" {
		usingTypeDefault = true
		if profile, ok := MetadataProfile(oaType); ok {
			template = strings.TrimSpace(profile.DefaultProcessURLTemplate)
		}
	}
	if template == "" || (usingTypeDefault && baseURL == "") {
		return ""
	}

	replacer := strings.NewReplacer(
		"{process_id}", processID,
		"{requestid}", processID,
		"{processId}", processID,
		"{requestId}", processID,
		"{PROCESS_ID}", processID,
		"{REQUESTID}", processID,
	)
	target := replacer.Replace(template)
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		return target
	}
	if baseURL == "" {
		return target
	}
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if !strings.HasPrefix(target, "/") {
		target = "/" + target
	}
	return baseURL + target
}
