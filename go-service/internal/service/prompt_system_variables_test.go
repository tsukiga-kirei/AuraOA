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
