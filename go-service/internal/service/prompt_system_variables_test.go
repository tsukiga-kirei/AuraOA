package service

import (
	"strings"
	"testing"
	"time"

	"auraoa/go-service/internal/pkg/apptime"
)

func TestReplaceSystemPromptVariablesAt(t *testing.T) {
	if err := apptime.Configure("Asia/Shanghai"); err != nil {
		t.Fatalf("configure timezone: %v", err)
	}

	fixed := time.Date(2026, 7, 9, 14, 30, 45, 0, apptime.Location())
	input := "日期={{current_date}} 时间={{current_time}} 日期时间={{current_datetime}} 星期={{weekday}}"
	got := replaceSystemPromptVariablesAt(input, fixed)

	want := "日期=2026-07-09 时间=14:30:45 日期时间=2026-07-09 14:30:45 星期=星期四"
	if got != want {
		t.Fatalf("replaceSystemPromptVariablesAt() = %q, want %q", got, want)
	}
}

func TestReplaceSystemPromptVariablesLeavesUnknownPlaceholders(t *testing.T) {
	input := "数据={{main_table}} 系统={{current_date}}"
	got := replaceSystemPromptVariables(input)
	if !strings.Contains(got, "{{main_table}}") {
		t.Fatalf("expected {{main_table}} to remain, got %q", got)
	}
	if strings.Contains(got, "{{current_date}}") {
		t.Fatalf("expected {{current_date}} to be replaced, got %q", got)
	}
}

func TestReplaceSystemPromptVariablesExpandsUIDisplayProtocols(t *testing.T) {
	input := "前文\n" + SystemVariableFormConfirmProtocol + "\n" + SystemVariableChartProtocol + "\n后文"
	got := replaceSystemPromptVariables(input)

	if strings.Contains(got, "{{") {
		t.Fatalf("expected all placeholders expanded, got %q", got)
	}
	for _, want := range []string{":::form-confirm", `"name"`, `"options"`, ":::chart", `"categories"`, `"indicators"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected protocol text to contain %q", want)
		}
	}
	if !strings.Contains(got, "前文") || !strings.Contains(got, "后文") {
		t.Fatalf("expected surrounding text preserved, got %q", got)
	}
}

func TestUIDisplayProtocolsStayAlignedWithRendererContract(t *testing.T) {
	// 协议里承诺的顶层键必须与前端归一化/渲染逻辑一致，避免文档与渲染再次漂移。
	formKeys := []string{"variant", "workflow_id", "workflow_name", "description", "fields", "columns", "rows", "details"}
	for _, key := range formKeys {
		if !strings.Contains(uiFormConfirmProtocol, key) {
			t.Fatalf("form-confirm protocol missing top-level key %q", key)
		}
	}
	chartKeys := []string{"type", "title", "subtitle", "categories", "series", "indicators", "columns", "rows"}
	for _, key := range chartKeys {
		if !strings.Contains(uiChartProtocol, key) {
			t.Fatalf("chart protocol missing top-level key %q", key)
		}
	}
}
