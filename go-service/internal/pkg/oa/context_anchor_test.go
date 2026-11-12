package oa

import (
	"encoding/json"
	"testing"
)

func TestProcessFingerprintIgnoresAttachmentDisplayChanges(t *testing.T) {
	for _, original := range []interface{}{"10,20", int64(10), []byte("10")} {
		pd := &ProcessData{
			MainData: map[string]interface{}{"value": "业务值", "display": "业务字段", "FJ": original},
			DetailTables: map[string][]map[string]interface{}{
				"formtable_main_900003_dt1": {{"fj": "30", "amount": 100}},
			},
		}
		before := ComputeProcessDataFingerprint(pd)
		files := []AttachmentFilePayload{{DocID: "10", FileName: "合同.pdf"}, {DocID: "20", FileName: "凭证.pdf"}}
		applyAttachmentFileNamesToRow(pd.MainData, "fj", mapGet(pd.MainData, "fj"), files)
		applyAttachmentFileNamesToRow(pd.DetailTables["formtable_main_900003_dt1"][0], "fj", "30", []AttachmentFilePayload{{DocID: "30", FileName: "明细.pdf"}})
		if after := ComputeProcessDataFingerprint(pd); after != before {
			t.Fatalf("附件名称补全不应改变内容指纹（原值类型 %T）：%s != %s", original, before, after)
		}
		// 文件名变更或部分下载失败只影响展示；附件版本变化由独立的附件指纹跟踪。
		applyAttachmentFileNamesToRow(pd.MainData, "fj", mapGet(pd.MainData, "fj"), []AttachmentFilePayload{{DocID: "10", FileName: "新名称.pdf"}})
		snapshot, _ := json.Marshal(pd)
		if ComputeProcessDataFingerprint(pd) != before {
			t.Fatal("附件重命名或下载成功数量变化不应污染内容指纹")
		}
		afterHash, _ := json.Marshal(pd)
		if string(snapshot) != string(afterHash) {
			t.Fatal("计算指纹不能改写供提示词使用的数据")
		}
		pd.DetailTables["formtable_main_900003_dt1"][0]["fj"] = "31"
		if ComputeProcessDataFingerprint(pd) == before {
			t.Fatal("替换明细附件必须改变内容指纹")
		}
	}
}

func TestProcessFingerprintKeepsBusinessChanges(t *testing.T) {
	before := &ProcessData{MainData: map[string]interface{}{
		"value": "原始业务值", "display": "业务字段",
		"browser": map[string]interface{}{"value": "10,20", "display": "人员甲, 人员乙", "items": []interface{}{}},
		"json":    map[string]interface{}{"value": 10, "display": "嵌套业务值", "business": true},
	}}
	after := &ProcessData{MainData: map[string]interface{}{
		"value": "原始业务值", "display": "业务字段", "browser": "10,20",
		"json": map[string]interface{}{"value": 10, "display": "嵌套业务值", "business": true},
	}}
	if ComputeProcessDataFingerprint(before) != ComputeProcessDataFingerprint(after) {
		t.Fatal("浏览框的展示包装也应按原值比较")
	}
	after.MainData["json"].(map[string]interface{})["display"] = "变化后的业务值"
	if ComputeProcessDataFingerprint(before) == ComputeProcessDataFingerprint(after) {
		t.Fatal("带其他业务字段的 JSON 不应被当作展示包装")
	}
	after.MainData = map[string]interface{}{"value": "原始业务值", "display": "另一个业务字段"}
	if ComputeProcessDataFingerprint(&ProcessData{MainData: map[string]interface{}{"value": "原始业务值", "display": "业务字段"}}) == ComputeProcessDataFingerprint(after) {
		t.Fatal("整行的 value/display 字段也必须独立参与指纹")
	}
}

func TestIsAnchorStale(t *testing.T) {
	stored := OAContextAnchor{LastReturnLogID: 100, FlowRevision: 150, ContentFingerprint: "sha256:aaa"}
	currentSame := OAContextAnchor{LastReturnLogID: 100, FlowRevision: 150, ContentFingerprint: "sha256:aaa"}
	if IsAnchorStale(stored, currentSame) {
		t.Fatal("expected same anchor not stale")
	}

	currentNewReturn := OAContextAnchor{LastReturnLogID: 101, FlowRevision: 160}
	if !IsAnchorStale(stored, currentNewReturn) {
		t.Fatal("expected new return to be stale")
	}

	currentNewRevision := OAContextAnchor{LastReturnLogID: 100, FlowRevision: 151}
	if !IsAnchorStale(stored, currentNewRevision) {
		t.Fatal("expected higher flow revision to be stale")
	}

	currentNewFingerprint := OAContextAnchor{LastReturnLogID: 100, FlowRevision: 150, ContentFingerprint: "sha256:bbb"}
	if !IsAnchorStale(stored, currentNewFingerprint) {
		t.Fatal("expected fingerprint change to be stale")
	}
}

func TestCompareContextAnchorsClassifiesOrdinaryApproval(t *testing.T) {
	stored := OAContextAnchor{
		LastLogID:          100,
		LastReturnLogID:    20,
		LastResubmitLogID:  21,
		CurrentNodeID:      7,
		ContentFingerprint: "sha256:data",
	}
	current := stored
	current.LastLogID = 101
	current.LastLogType = "0"
	current.CurrentNodeID = 8

	changes := CompareContextAnchors(stored, current)
	if !changes.FlowChanged || !changes.CurrentNodeChanged {
		t.Fatalf("expected flow and node changes, got %+v", changes)
	}
	if changes.DataChanged || changes.AttachmentChanged || changes.ReturnResubmitChanged {
		t.Fatalf("ordinary approval must not be classified as business data change: %+v", changes)
	}
}

func TestComputeAttachmentFingerprintsTracksVersion(t *testing.T) {
	before, beforeFields := ComputeAttachmentFingerprints([]AttachmentVersionAnchor{
		{FieldKey: "FJ", DocID: "10", VersionID: 1, ImageFileID: "100", FileName: "a.pdf"},
	})
	after, afterFields := ComputeAttachmentFingerprints([]AttachmentVersionAnchor{
		{FieldKey: "fj", DocID: "10", VersionID: 2, ImageFileID: "101", FileName: "a.pdf"},
	})
	if before == "" || after == "" || before == after {
		t.Fatalf("attachment version change must update fingerprint: before=%s after=%s", before, after)
	}
	if beforeFields["fj"] == afterFields["fj"] {
		t.Fatal("field fingerprint must change with version")
	}
}
