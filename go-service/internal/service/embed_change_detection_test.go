package service

import (
	"testing"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/oa"
)

func boolPointer(value bool) *bool {
	return &value
}

func TestChangedSummaryBlockIDsUsesBlockDependencies(t *testing.T) {
	blocks := []model.SummaryBlockConfig{
		{
			ID:                   "attachment",
			Title:                "附件摘要",
			IncludeMeta:          boolPointer(false),
			EnabledDataVariables: []string{"{{attachments}}"},
			FieldMode:            "selected",
			SelectedFields:       []string{"main:fj"},
			Enabled:              true,
		},
		{
			ID:                   "flow",
			Title:                "审批进度",
			IncludeMeta:          boolPointer(false),
			EnabledDataVariables: []string{"{{flow_graph}}"},
			FieldMode:            "selected",
			SelectedFields:       []string{},
			Enabled:              true,
		},
	}
	stored := map[string]SummaryBlockDependencyFingerprint{
		"attachment": {Config: stableJSONFingerprint(blocks[0]), Attachments: "v1"},
		"flow":       {Config: stableJSONFingerprint(blocks[1]), Flow: "log-1"},
	}
	current := map[string]SummaryBlockDependencyFingerprint{
		"attachment": {Config: stableJSONFingerprint(blocks[0]), Attachments: "v2"},
		"flow":       {Config: stableJSONFingerprint(blocks[1]), Flow: "log-2"},
	}
	cfg := model.SummaryEmbedConfigData{
		AutoSummaryOnDataChange:     true,
		AutoSummaryOnReturnResubmit: true,
		AutoSummaryOnFlowChange:     false,
	}
	got := changedSummaryBlockIDs(
		blocks,
		stored,
		current,
		oa.OAContextChanges{AttachmentChanged: true, FlowChanged: true},
		cfg,
	)
	if len(got) != 1 || got[0] != "attachment" {
		t.Fatalf("expected only attachment block, got %v", got)
	}

	cfg.AutoSummaryOnFlowChange = true
	got = changedSummaryBlockIDs(
		blocks,
		stored,
		current,
		oa.OAContextChanges{AttachmentChanged: true, FlowChanged: true},
		cfg,
	)
	if len(got) != 2 {
		t.Fatalf("expected attachment and flow blocks, got %v", got)
	}
}

func TestAuditRefreshRequiredIgnoresOrdinaryApprovalByDefault(t *testing.T) {
	cfg := model.EmbedConfigData{
		AutoAuditOnDataChange:     true,
		AutoAuditOnReturnResubmit: true,
		AutoAuditOnFlowChange:     false,
	}
	if auditRefreshRequired(oa.OAContextChanges{FlowChanged: true, CurrentNodeChanged: true}, cfg) {
		t.Fatal("ordinary approval must not refresh when flow change switch is disabled")
	}
	if !auditRefreshRequired(oa.OAContextChanges{ReturnResubmitChanged: true}, cfg) {
		t.Fatal("return/resubmit must refresh when its switch is enabled")
	}
}

func TestAuditRefreshRequiredIgnoresExecutionConfigChange(t *testing.T) {
	cfg := model.EmbedConfigData{AutoAuditOnDataChange: true}
	if auditRefreshRequired(oa.OAContextChanges{ExecutionConfigChanged: true}, cfg) {
		t.Fatal("审核规则、尺度或提示词变化不得借用业务数据变化开关自动重审")
	}
}

func TestChangedSummaryBlockIDsIgnoresBlockConfigChange(t *testing.T) {
	blocks := []model.SummaryBlockConfig{{ID: "overview", Title: "新版标题", Enabled: true}}
	stored := map[string]SummaryBlockDependencyFingerprint{
		"overview": {Config: "old", Data: "same"},
	}
	current := map[string]SummaryBlockDependencyFingerprint{
		"overview": {Config: "new", Data: "same"},
	}
	got := changedSummaryBlockIDs(
		blocks,
		stored,
		current,
		oa.OAContextChanges{ExecutionConfigChanged: true},
		model.SummaryEmbedConfigData{AutoSummaryOnDataChange: true},
	)
	if len(got) != 0 {
		t.Fatalf("总结块配置变化不得自动重新总结，got %v", got)
	}
}
