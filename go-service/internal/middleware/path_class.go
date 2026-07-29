package middleware

import "strings"

// IsLowValuePollingPath 与 Logger 中间件一致：轮询类接口频率高、审计价值低，可跳过或降级处理。
func IsLowValuePollingPath(path string) bool {
	pollingPrefixes := []string{
		"/api/audit/jobs/",
		"/api/archive/jobs/",
		"/api/embed/jobs/",
		"/api/embed/summary/jobs/",
		"/api/auth/notifications/unread-count",
		"/api/audit/stats",
		"/api/archive/stats",
	}
	for _, prefix := range pollingPrefixes {
		if strings.HasPrefix(path, prefix) || strings.Contains(path, prefix) {
			return true
		}
	}
	return false
}
