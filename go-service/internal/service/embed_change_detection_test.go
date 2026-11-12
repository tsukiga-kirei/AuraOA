package service

import (
	"encoding/json"
	"reflect"
	"testing"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/oa"
)

func TestAuditAttemptFingerprintUsesSelectedRawDependencies(t *testing.T) {
	fields := SelectedFieldSet{"main": {"fj": true, "amount": true}, "dt1": {"fj": false}, "dt2": {}}
	raw := &oa.ProcessData{
		MainData:     map[string]interface{}{"fj": "10", "amount": 100},
		DetailTables: map[string][]map[string]interface{}{"dt1": {{"fj": "20"}}, "dt2": {{"fj": "30"}}},
	}
	resolved := &oa.ProcessData{
		MainData:     map[string]interface{}{"fj": map[string]interface{}{"value": "10", "display": "合同.pdf"}, "amount": 100},
		DetailTables: map[string][]map[string]interface{}{"dt1": {{"fj": "20"}}, "dt2": {{"fj": "30"}}},
	}
	anchor := oa.OAContextAnchor{LastLogID: 100, CurrentNodeID: 7,
		AttachmentFieldFingerprints: map[string]string{"fj": "v1", "dt1:fj": "v1", "dt2:fj": "v1"}}
	stored := auditOAAnchorForFields(anchor, resolved, fields, "config-v13")
	current := auditOAAnchorForFields(anchor, raw, fields, "config-v13")
	// 模拟任务将锚点写库、嵌入页再次读取；与实际失败重试拦截使用同一序列化口径。
	encoded, _ := json.Marshal(stored)
	if stableJSONFingerprint(parseOAContextAnchor(encoded)) != stableJSONFingerprint(current) {
		t.Fatal("相同数据的失败尝试必须可由嵌入页识别，不能因附件展示结构重复执行")
	}
	anchor.AttachmentFieldFingerprints["dt1:fj"] = "v2"
	current = auditOAAnchorForFields(anchor, raw, fields, "config-v13")
	if stableJSONFingerprint(stored) != stableJSONFingerprint(current) {
		t.Fatal("未选中的同名附件换版不能解除失败拦截")
	}
	anchor.AttachmentFieldFingerprints["fj"] = "v2"
	current = auditOAAnchorForFields(anchor, raw, fields, "config-v13")
	if !oa.CompareContextAnchors(stored, current).AttachmentChanged || stableJSONFingerprint(stored) == stableJSONFingerprint(current) {
		t.Fatal("选中附件换版必须可被检测，允许新依赖重新执行")
	}
	raw.MainData["amount"] = 101
	if !oa.CompareContextAnchors(stored, auditOAAnchorForFields(anchor, raw, fields, "config-v13")).DataChanged {
		t.Fatal("真实业务字段变化不能被展示归一化屏蔽")
	}
}

func TestAttachmentDependenciesMatchDownloadAndSummaryInput(t *testing.T) {
	attachments := []oa.AttachmentInfo{
		{FieldKey: "fj", DocID: "10"},
		{FieldKey: "fj", DocID: "20", DetailTable: "dt1"},
		{FieldKey: "fj", DocID: "30", DetailTable: "dt2"},
	}
	anchor := oa.OAContextAnchor{AttachmentFingerprint: "all", AttachmentFieldFingerprints: map[string]string{
		"fj": "main-v1", "dt1:fj": "detail1-v1", "dt2:fj": "detail2-v1",
	}}
	for _, tc := range []struct {
		name   string
		fields SelectedFieldSet
		want   []string
	}{
		{"全部", nil, []string{"10", "20", "30"}},
		{"主表同名字段", SelectedFieldSet{"main": {"fj": true}, "dt1": {}, "dt2": {}}, []string{"10"}},
		{"明细同名字段", SelectedFieldSet{"main": {}, "dt1": {"FJ": true}, "dt2": {}}, []string{"20"}},
		{"单表全选", SelectedFieldSet{"main": {}, "dt1": nil, "dt2": {"fj": false}}, []string{"20"}},
		{"全部拒绝", SelectedFieldSet{"main": {}, "dt1": {}, "dt2": {"fj": false}}, []string{}},
		{"未声明表沿用全选", SelectedFieldSet{"main": {}, "dt1": {}}, []string{"30"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			selected := filterAttachmentsForFieldSet(attachments, tc.fields)
			ids := make([]string, 0, len(selected))
			for _, item := range selected {
				ids = append(ids, item.DocID)
			}
			if !reflect.DeepEqual(ids, tc.want) {
				t.Fatalf("总结块附件正文范围不正确：got %v, want %v", ids, tc.want)
			}
			fingerprints := attachmentFieldFingerprintsForFieldSet(anchor, tc.fields)
			if len(fingerprints) != len(selected) {
				t.Fatalf("附件版本与提示词输入范围不一致：%v vs %v", fingerprints, ids)
			}
			for _, item := range attachments {
				table, key := item.DetailTable, item.FieldKey
				if table == "" {
					table = "main"
				} else {
					key = table + ":" + key
				}
				_, included := fingerprints[key]
				if included != oa.AttachmentFieldSelected(tc.fields, table, item.FieldKey) {
					t.Fatal("附件下载与版本指纹口径不一致")
				}
			}
		})
	}
}

func TestSummaryAttachmentChangeOnlyRefreshesDependentBlock(t *testing.T) {
	blocks := []model.SummaryBlockConfig{
		{ID: "main", Enabled: true, IncludeMeta: boolPointer(false), EnabledDataVariables: []string{"{{attachments}}"}, FieldMode: "selected", SelectedFields: []string{"main:fj", "dt1:amount"}},
		{ID: "detail", Enabled: true, IncludeMeta: boolPointer(false), EnabledDataVariables: []string{"{{attachments}}"}, FieldMode: "selected", SelectedFields: []string{"dt1:fj"}},
	}
	anchor := oa.OAContextAnchor{AttachmentFieldFingerprints: map[string]string{"fj": "v1", "dt1:fj": "v1"}}
	stored := buildSummaryBlockDependencyFingerprints(blocks, nil, anchor, nil)
	anchor.AttachmentFieldFingerprints["dt1:fj"] = "v2"
	current := buildSummaryBlockDependencyFingerprints(blocks, nil, anchor, nil)
	got := changedSummaryBlockIDs(blocks, stored, current, oa.OAContextChanges{AttachmentChanged: true}, model.SummaryEmbedConfigData{AutoSummaryOnDataChange: true})
	if !reflect.DeepEqual(got, []string{"detail"}) {
		t.Fatalf("同名附件换版只能刷新依赖该表的总结块：%v", got)
	}
}

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
