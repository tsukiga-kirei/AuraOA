package service

import (
	"encoding/base64"
	"testing"
)

func TestParseBasicSSOCredential(t *testing.T) {
	header := "Basic " + base64.StdEncoding.EncodeToString([]byte("GJCW/zhangsan:shared-secret"))
	tenantCode, username, password, err := parseBasicSSOCredential(header)
	if err != nil {
		t.Fatalf("解析 Basic 凭据失败: %v", err)
	}
	if tenantCode != "GJCW" || username != "zhangsan" || password != "shared-secret" {
		t.Fatalf("解析结果不符合预期: %s %s %s", tenantCode, username, password)
	}
}

func TestMatchesAllowedIP(t *testing.T) {
	if !matchesAllowedIP("10.10.1.8, 10.20.0.0/16", "10.20.3.9") {
		t.Fatal("CIDR 内的来源 IP 应通过校验")
	}
	if matchesAllowedIP("10.10.1.8, 10.20.0.0/16", "10.30.3.9") {
		t.Fatal("白名单外的来源 IP 不应通过校验")
	}
}

func TestMatchesAllowedDomain(t *testing.T) {
	if !matchesAllowedDomain("example.com", "https://oa.example.com", "") {
		t.Fatal("允许域名的子域名应通过校验")
	}
	if matchesAllowedDomain("example.com", "https://example.net", "") {
		t.Fatal("白名单外的来源域名不应通过校验")
	}
}
