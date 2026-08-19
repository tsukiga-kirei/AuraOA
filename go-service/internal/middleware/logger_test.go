package middleware

import (
	"net/url"
	"testing"
)

func TestRedactSensitiveQuery(t *testing.T) {
	got := redactSensitiveQuery("page=1&token=secret-jwt&embed_token=embed-secret&keyword=%E5%AE%A1%E6%A0%B8")
	values, err := url.ParseQuery(got)
	if err != nil {
		t.Fatalf("解析脱敏查询参数失败: %v", err)
	}
	if values.Get("token") != "***" || values.Get("embed_token") != "***" {
		t.Fatalf("敏感参数未脱敏: %s", got)
	}
	if values.Get("page") != "1" || values.Get("keyword") != "审核" {
		t.Fatalf("普通查询参数被意外修改: %s", got)
	}
}

func TestRedactSensitiveQueryInvalidInput(t *testing.T) {
	if got := redactSensitiveQuery("token=%zz"); got != "" {
		t.Fatalf("非法查询参数应从日志中省略，实际为 %q", got)
	}
}
