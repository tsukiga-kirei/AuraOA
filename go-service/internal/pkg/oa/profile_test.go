package oa

import (
	"strings"
	"testing"

	"auraoa/go-service/internal/model"
)

func TestMetadataProfileKeepsE9JumpAndArchive(t *testing.T) {
	profile, ok := MetadataProfile("weaver_e9")
	if !ok || !profile.Implemented {
		t.Fatal("weaver_e9 档案应已实现")
	}
	if profile.DefaultProcessURLTemplate != "/workflow/request/ViewRequestForwardSPA.jsp?requestid={process_id}" {
		t.Fatalf("E9 兜底模板 = %q", profile.DefaultProcessURLTemplate)
	}
	rule, ok := WorkflowArchive("ecology9")
	if !ok || rule.RequestTable != "workflow_requestbase" || rule.StatusColumn != "currentnodetype" || rule.StatusValue != "3" {
		t.Fatalf("ecology9 别称应沿用 E9 归档规则: %+v ok=%v", rule, ok)
	}
	if _, ok := Profile("ecology9"); ok {
		t.Fatal("ecology9 不能作为可创建适配器的类型")
	}
}

func TestE10AdapterIsRejected(t *testing.T) {
	err := validateOAAdapterConfig("weaver_e10", &model.OADatabaseConnection{Driver: "mysql"})
	if err == nil || !strings.Contains(err.Error(), "尚未实现") {
		t.Fatalf("E10 应拒绝建连，得到 %v", err)
	}
}

func TestE10ProfileStaysUnimplemented(t *testing.T) {
	profile, ok := Profile("weaver_e10")
	if !ok {
		t.Fatal("缺少 weaver_e10 档案")
	}
	if profile.Implemented || profile.DefaultProcessURLTemplate != "" || profile.AttachmentAuth != "" || profile.WorkflowArchive != nil || profile.QueryAssistNote != "" {
		t.Fatalf("E10 档案应保持空白占位: %+v", profile)
	}
	if _, ok := WorkflowArchive("weaver_e10"); ok {
		t.Fatal("E10 不应提供归档规则")
	}
}

func TestBuildProcessURLUsesTypeDefaultOnly(t *testing.T) {
	e9 := BuildProcessURL("http://oa.example.com", "", "weaver_e9", "613446")
	want := "http://oa.example.com/workflow/request/ViewRequestForwardSPA.jsp?requestid=613446"
	if e9 != want {
		t.Fatalf("E9 URL = %q, want %q", e9, want)
	}
	if got := BuildProcessURL("http://oa.example.com", "", "weaver_e10", "613446"); got != "" {
		t.Fatalf("E10 未配置模板时不应套用 E9 地址，得到 %q", got)
	}
	if got := BuildProcessURL("http://oa.example.com", "", "zhiyuan_a8", "613446"); got != "" {
		t.Fatalf("未登记兜底的类型应返回空地址，得到 %q", got)
	}
	if got := BuildProcessURL("", "", "weaver_e9", "100"); got != "" {
		t.Fatalf("没有 Web 地址时不应生成相对链接，得到 %q", got)
	}
	custom := BuildProcessURL("https://oa.example.com", "/flow/{requestid}", "weaver_e10", "9")
	if custom != "https://oa.example.com/flow/9" {
		t.Fatalf("显式模板应原样使用，得到 %q", custom)
	}
}
