package oa

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestFetchAttachmentVersionAnchorsKeepsTableAndRawIDs(t *testing.T) {
	db := setupTestE9DB(t)
	for _, query := range []string{
		`CREATE TABLE workflow_requestbase (requestid INTEGER, workflowid INTEGER);`,
		`CREATE TABLE workflow_bill (id INTEGER, tablename TEXT);`,
		`CREATE TABLE DocImageFile (docid INTEGER, versionid INTEGER, imagefileid INTEGER);`,
		`CREATE TABLE ImageFile (imagefileid INTEGER, imagefilename TEXT, filesize INTEGER);`,
		`INSERT INTO workflow_requestbase VALUES (9001, 101);`,
		`INSERT INTO workflow_bill VALUES (201, 'formtable_main_900003');`,
		`INSERT INTO workflow_billfield (id, billid, fieldname, fieldhtmltype, type, detailtable) VALUES
		 (10, 201, 'fj', '6', 0, ''), (11, 201, 'fj', '6', 0, '1'), (12, 201, 'fj', '3', 9, '2');`,
		`INSERT INTO DocImageFile VALUES (10, 1, 100), (10, 2, 101), (20, 1, 200), (30, 1, 300);`,
		`INSERT INTO ImageFile VALUES (100, '旧合同.pdf', 10), (101, '合同.pdf', 20), (200, '明细.pdf', 30), (300, '文档.pdf', 40);`,
	} {
		if err := db.Exec(query).Error; err != nil {
			t.Fatal(err)
		}
	}
	adapter := &Ecology9Adapter{db: db, driver: "mysql"}
	pd := &ProcessData{MainData: map[string]interface{}{"fj": "10"}, DetailTables: map[string][]map[string]interface{}{
		"formtable_main_900003_dt1": {{"fj": "20"}},
		"formtable_main_900003_dt2": {{"fj": "30"}},
	}}
	before, err := adapter.fetchAttachmentVersionAnchors(context.Background(), "9001", pd)
	if err != nil {
		t.Fatal(err)
	}
	global, fields := ComputeAttachmentFingerprints(before)
	for _, key := range []string{"fj", "formtable_main_900003_dt1:fj", "formtable_main_900003_dt2:fj"} {
		if fields[key] == "" {
			t.Fatalf("版本指纹缺少独立的主表/明细字段 %q：%v", key, fields)
		}
	}
	for _, item := range before {
		if item.DocID == "10" && (item.VersionID != 2 || item.ImageFileID != "101" || item.FileName != "合同.pdf") {
			t.Fatalf("应读取最新附件版本：%+v", item)
		}
	}
	versions, err := adapter.fetchLatestAttachmentVersions(context.Background(), []string{"10"})
	if err != nil || mapGetInt(versions["10"], "file_size") != 20 {
		t.Fatalf("查询表达式返回的文件大小也应正确解包：%v, %v", versions, err)
	}
	applyAttachmentFileNamesToRow(pd.MainData, "fj", "10", []AttachmentFilePayload{{DocID: "10", FileName: "合同.pdf"}})
	applyAttachmentFileNamesToRow(pd.DetailTables["formtable_main_900003_dt1"][0], "fj", "20", []AttachmentFilePayload{{DocID: "20", FileName: "明细.pdf"}})
	after, err := adapter.fetchAttachmentVersionAnchors(context.Background(), "9001", pd)
	if err != nil {
		t.Fatal(err)
	}
	got, gotFields := ComputeAttachmentFingerprints(after)
	if global == "" || got != global || !reflect.DeepEqual(fields, gotFields) {
		t.Fatalf("无附件下载与带展示值的执行路径必须读取相同附件版本指纹：before=%+v after=%+v", before, after)
	}
}

func TestFetchWeaverAttachmentsByDocIDs_PerDocIDResilience(t *testing.T) {
	// 模拟泛微服务端：
	// docId = 295774 -> 成功返回
	// docId = 295799 -> 模拟超大文件被 Filter 掐断（直接关闭连接 / 返回空 body）
	// docId = 295798 -> 成功返回
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		docID := r.URL.Query().Get("docIds")
		if docID == "295799" {
			// 模拟泛微端触发 OverLimitSizeException 导致连接中断或空响应
			w.WriteHeader(http.StatusOK)
			// 不写入任何正文，客户端 json.Decode 报 EOF
			return
		}

		resp := map[string]interface{}{
			"code": 0,
			"msg":  "ok",
			"data": []map[string]interface{}{
				{
					"docId":    docID,
					"fileName": "test_" + docID + ".pdf",
					"fileSize": 1024,
					"fileData": "dGVzdA==",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := &Ecology9Adapter{
		weaverAPIURL:  server.URL,
		weaverAppID:   "test_app",
		weaverLoginID: "sysadmin",
		httpClient:    server.Client(),
	}

	ctx := context.Background()
	rawDocIDs := "295774,295799,295798"

	files, failedDocs, err := adapter.fetchWeaverAttachmentsByDocIDs(ctx, "628048", "fjsc", rawDocIDs)
	if err != nil {
		t.Fatalf("预期部分成功时不返回致命错误，但收到 err: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("预期成功拉取 2 个附件，实际: %d", len(files))
	}
	if len(failedDocs) != 1 || failedDocs[0] != "295799" {
		t.Fatalf("预期失败的附件为 [295799]，实际: %v", failedDocs)
	}

	expectedIDs := map[string]bool{"295774": true, "295798": true}
	for _, f := range files {
		if !expectedIDs[f.DocID] {
			t.Errorf("意外拉取到 docId: %s", f.DocID)
		}
	}
}

func TestFetchWeaverAttachmentsByDocIDs_AllFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer server.Close()

	adapter := &Ecology9Adapter{
		weaverAPIURL:  server.URL,
		weaverAppID:   "test_app",
		weaverLoginID: "sysadmin",
		httpClient:    server.Client(),
	}

	ctx := context.Background()
	_, failedDocs, err := adapter.fetchWeaverAttachmentsByDocIDs(ctx, "628048", "fjsc", "111,222")
	if err == nil {
		t.Fatalf("预期全失败时返回错误，但收到 nil")
	}
	if len(failedDocs) != 2 {
		t.Fatalf("预期 failedDocs 为 2，实际: %d", len(failedDocs))
	}
}

func TestMapGetUnwrapsResolvedAttachmentAndBrowseValues(t *testing.T) {
	row := map[string]interface{}{
		"fj": "295726,295727",
	}
	applyAttachmentFileNamesToRow(row, "fj", "295726,295727", []AttachmentFilePayload{
		{DocID: "295726", FileName: "合同.pdf"},
		{DocID: "295727", FileName: "验收报告.pdf"},
	})

	if got := mapGet(row, "FJ"); got != "295726,295727" {
		t.Fatalf("expected unwrapped docIDs, got %q", got)
	}
	docIDs := splitAttachmentDocIDs(mapGet(row, "fj"))
	if len(docIDs) != 2 || docIDs[0] != "295726" || docIDs[1] != "295727" {
		t.Fatalf("unexpected split docIDs: %v", docIDs)
	}

	browseRow := map[string]interface{}{
		"doc_ref": map[string]interface{}{"value": 9981, "display": "关联文档"},
		"weird":   map[string]interface{}{"display": "no value"},
	}
	if got := mapGet(browseRow, "doc_ref"); got != "9981" {
		t.Fatalf("expected unwrapped numeric value 9981, got %q", got)
	}
	if got := mapGet(browseRow, "weird"); got != "" {
		t.Fatalf("expected empty string when value key missing, got %q", got)
	}
	if got := stringifyDBValue(browseRow["doc_ref"]); got != "9981" {
		t.Fatalf("expected stringifyDBValue to unwrap resolved map, got %q", got)
	}
}

func TestFilterNumericDocIDsAndSkipsDBOnInvalidOnly(t *testing.T) {
	valid, skipped := filterNumericDocIDs([]string{"295726", "map[display:x value:1]", " 295727 ", "abc", ""})
	if len(valid) != 2 || valid[0] != "295726" || valid[1] != "295727" {
		t.Fatalf("unexpected valid docIDs: %v", valid)
	}
	if len(skipped) != 2 || skipped[0] != "map[display:x value:1]" || skipped[1] != "abc" {
		t.Fatalf("unexpected skipped docIDs: %v", skipped)
	}

	adapter := &Ecology9Adapter{}
	got, err := adapter.fetchLatestAttachmentVersions(context.Background(), []string{"not-numeric", "map[bad]"})
	if err != nil {
		t.Fatalf("expected nil error when all docIDs are non-numeric, got %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty result map, got %v", got)
	}
}
