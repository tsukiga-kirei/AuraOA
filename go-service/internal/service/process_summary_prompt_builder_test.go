package service

import (
	"strings"
	"testing"

	"auraoa/go-service/internal/model"
)

func TestBuildSummaryUserPromptAllDataMode(t *testing.T) {
	payload := summaryPromptPayload{
		meta:           "meta-content",
		mainTable:      "main-content",
		detailTables:   "detail-content",
		attachments:    "attach-content",
		flowHistory:    "history-content",
		flowGraph:      "graph-content",
		userPrompt:     "请总结重点",
		includeAllData: true,
	}
	got := buildSummaryUserPrompt("流程摘要", payload)
	for _, want := range []string{"流程基础信息：", "meta-content", "主表字段：", "main-content", "本总结块的用户要求：", "请总结重点"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected all-data prompt to contain %q, got:\n%s", want, got)
		}
	}
}

func TestBuildSummaryUserPromptCustomModeSelectedOnly(t *testing.T) {
	payload := summaryPromptPayload{
		meta:         "meta-content",
		mainTable:    "main-content",
		detailTables: "detail-content",
		attachments:  "attach-content",
		userPrompt:   "请总结重点",
		enabledKeys: map[string]struct{}{
			"{{process_meta}}": {},
			"{{attachments}}":  {},
		},
		includeAllData: false,
	}
	got := buildSummaryUserPrompt("流程摘要", payload)
	if !strings.Contains(got, "meta-content") {
		t.Fatalf("expected custom prompt to include meta, got:\n%s", got)
	}
	if !strings.Contains(got, "attach-content") {
		t.Fatalf("expected custom prompt to include attachments when selected")
	}
	if strings.Contains(got, "main-content") {
		t.Fatalf("expected custom prompt to exclude unselected main table data")
	}
}

func TestBuildSummaryUserPromptCustomModeEmptySelection(t *testing.T) {
	payload := summaryPromptPayload{
		meta:           "meta-content",
		mainTable:      "main-content",
		userPrompt:     "请总结重点",
		enabledKeys:    map[string]struct{}{},
		includeAllData: false,
	}
	got := buildSummaryUserPrompt("流程摘要", payload)
	if !strings.Contains(got, "（当前总结块未选择任何数据变量）") {
		t.Fatalf("expected empty-selection placeholder, got:\n%s", got)
	}
	if strings.Contains(got, "meta-content") {
		t.Fatalf("expected no data content when nothing selected")
	}
}

func TestSummaryBlockIncludeAllDataDefault(t *testing.T) {
	block := model.SummaryBlockConfig{}
	if !summaryBlockIncludeAllData(block) {
		t.Fatal("nil IncludeMeta should default to all-data mode")
	}
}

func TestNormalizeSummaryEnabledDataVariables(t *testing.T) {
	block := model.SummaryBlockConfig{
		EnabledDataVariables: []string{"{{main_table}}", "{{unknown}}", " {{flow_graph}} "},
	}
	got := normalizeSummaryEnabledDataVariables(block)
	if len(got) != 2 {
		t.Fatalf("expected 2 valid keys, got %d", len(got))
	}
	if _, ok := got["{{main_table}}"]; !ok {
		t.Fatal("expected main_table to be enabled")
	}
	if _, ok := got["{{flow_graph}}"]; !ok {
		t.Fatal("expected flow_graph to be enabled")
	}
}
