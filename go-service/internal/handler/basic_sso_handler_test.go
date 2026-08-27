package handler

import (
	"strings"
	"testing"
)

func TestRenderBasicSSOBridgeLoadsMenusAndOpensBusinessWorkbench(t *testing.T) {
	html := renderBasicSSOBridge("e30=")
	for _, expected := range []string{"/api/auth/menu", "menus = result.data.menus", "window.location.replace('/dashboard')"} {
		if !strings.Contains(html, expected) {
			t.Fatalf("单点登录桥接页缺少 %q", expected)
		}
	}
	if strings.Contains(html, "window.location.replace('/overview')") {
		t.Fatal("单点登录成功后不应再固定跳转到概览页")
	}
}
