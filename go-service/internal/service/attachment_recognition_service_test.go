package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"auraoa/go-service/internal/pkg/oa"
)

type mapSystemConfigReader map[string]string

func (r mapSystemConfigReader) FindByKey(key string) (string, error) {
	value, ok := r[key]
	if !ok {
		return "", fmt.Errorf("配置不存在")
	}
	return value, nil
}

func TestExtractMinerUMarkdown(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		want    string
	}{
		{
			name:    "document result",
			payload: `{"document":{"md_content":"from document"}}`,
			want:    "from document",
		},
		{
			name:    "files result",
			payload: `{"files":{"md_content":"from files"}}`,
			want:    "from files",
		},
		{
			name:    "filename keyed result",
			payload: `{"合同.pdf":{"md_content":"from filename"}}`,
			want:    "from filename",
		},
		{
			name:    "direct markdown result",
			payload: `{"md_content":"from direct"}`,
			want:    "from direct",
		},
		{
			name:    "empty result",
			payload: `{}`,
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractMinerUMarkdown(json.RawMessage(tt.payload))
			if got != tt.want {
				t.Fatalf("extractMinerUMarkdown() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLoadRecognitionConfigCompatibilityDefaults(t *testing.T) {
	service := &AttachmentRecognitionService{
		configRepo: mapSystemConfigReader{},
		httpClient: &http.Client{Timeout: time.Second},
	}

	cfg, err := service.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.CompatEndpoint != "http://document-parser:8090" {
		t.Fatalf("CompatEndpoint = %q", cfg.CompatEndpoint)
	}
	if cfg.LegacyOfficeEnabled || cfg.OFDEnabled {
		t.Fatalf("兼容格式开关默认应关闭: %+v", cfg)
	}
	if !cfg.VisualFallbackEnabled {
		t.Fatal("VisualFallbackEnabled 默认应开启")
	}
	wantTypes := []string{
		"pdf", "png", "jpg", "jpeg", "bmp", "gif", "tiff", "webp",
		"txt", "csv", "md", "docx", "xlsx", "pptx", "doc", "xls", "ppt", "ofd",
	}
	if !reflect.DeepEqual(cfg.SupportedTypes, wantTypes) {
		t.Fatalf("SupportedTypes = %v, want %v", cfg.SupportedTypes, wantTypes)
	}
}

func TestRecognizeAttachmentsRoutesAllParsersAndOFDFallback(t *testing.T) {
	var mu sync.Mutex
	requestCounts := map[string]int{}
	var minerUFiles []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		mu.Lock()
		requestCounts[r.URL.Path]++
		mu.Unlock()

		switch r.URL.Path {
		case "/parse":
			fileName, _, err := readMultipartTestFile(r, "file")
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			if strings.HasSuffix(fileName, ".ofd") {
				_, _ = io.WriteString(w, `{"parser":"ofdrw","file_type":"ofd","content":"","has_text_layer":false,"fallback_required":true,"fallback_format":"pdf","warnings":[]}`)
				return
			}
			_, _ = io.WriteString(w, fmt.Sprintf(
				`{"parser":"apache-poi","file_type":"doc","content":%q,"has_text_layer":true,"fallback_required":false,"warnings":[]}`,
				"compat:"+fileName,
			))

		case "/convert/pdf":
			fileName, _, err := readMultipartTestFile(r, "file")
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if !strings.HasSuffix(fileName, ".ofd") {
				http.Error(w, "unexpected file", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/pdf")
			_, _ = w.Write([]byte("%PDF-1.4\nconverted"))

		case "/file_parse":
			fileName, body, err := readMultipartTestFile(r, "files")
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if fileName == "invoice.pdf" && !bytes.HasPrefix(body, []byte("%PDF-")) {
				http.Error(w, "fallback body is not pdf", http.StatusBadRequest)
				return
			}
			mu.Lock()
			minerUFiles = append(minerUFiles, fileName)
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, fmt.Sprintf(
				`{"status":"completed","results":{"document":{"md_content":%q}}}`,
				"mineru:"+fileName,
			))

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	config := mapSystemConfigReader{
		"attachment.recognition_enabled":     "true",
		"attachment.max_file_size_mb":        "10",
		"attachment.supported_types":         "txt,pdf,doc,ofd",
		"attachment.mineru_endpoint":         server.URL,
		"attachment.mineru_api_key":          "test-key",
		"attachment.compat_endpoint":         server.URL,
		"attachment.compat_api_key":          "test-key",
		"attachment.legacy_office_enabled":   "true",
		"attachment.ofd_enabled":             "true",
		"attachment.visual_fallback_enabled": "true",
		"attachment.mineru_enable_formula":   "true",
		"attachment.mineru_enable_table":     "true",
		"attachment.mineru_parse_method":     "ocr",
		"attachment.mineru_language":         "ch",
	}
	service := &AttachmentRecognitionService{
		configRepo: config,
		httpClient: server.Client(),
	}
	files := []oa.AttachmentFilePayload{
		testAttachment("1", "note.txt", []byte("local text")),
		testAttachment("2", "scan.pdf", []byte("%PDF-source")),
		testAttachment("3", "legacy.doc", []byte("legacy source")),
		testAttachment("4", "invoice.ofd", []byte("ofd source")),
	}

	results, err := service.RecognizeAttachments(context.Background(), files, "attachments", "附件")
	if err != nil {
		t.Fatalf("RecognizeAttachments() error = %v", err)
	}
	if len(results) != 4 {
		t.Fatalf("len(results) = %d, want 4", len(results))
	}
	wantContents := []string{"local text", "mineru:scan.pdf", "compat:legacy.doc", "mineru:invoice.pdf"}
	for i, want := range wantContents {
		if results[i].Error != "" {
			t.Fatalf("results[%d].Error = %q", i, results[i].Error)
		}
		if results[i].Content != want {
			t.Fatalf("results[%d].Content = %q, want %q", i, results[i].Content, want)
		}
	}
	if results[3].FileName != "invoice.ofd" || results[3].FileType != "ofd" {
		t.Fatalf("OFD 回退后应保留原附件元数据: %+v", results[3])
	}

	mu.Lock()
	defer mu.Unlock()
	if requestCounts["/parse"] != 2 || requestCounts["/convert/pdf"] != 1 || requestCounts["/file_parse"] != 2 {
		t.Fatalf("requestCounts = %#v", requestCounts)
	}
	if !reflect.DeepEqual(minerUFiles, []string{"scan.pdf", "invoice.pdf"}) {
		t.Fatalf("MinerU files = %v", minerUFiles)
	}
}

func TestRecognizeAttachmentsFailureDoesNotStopBatch(t *testing.T) {
	var minerUCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/file_parse" {
			http.NotFound(w, r)
			return
		}
		minerUCalls++
		_, _ = io.WriteString(w, `{"status":"completed","results":{"document":{"md_content":"parsed"}}}`)
	}))
	defer server.Close()

	service := &AttachmentRecognitionService{
		configRepo: mapSystemConfigReader{
			"attachment.recognition_enabled": "true",
			"attachment.supported_types":     "txt,pdf,doc,ofd",
			"attachment.mineru_endpoint":     server.URL,
		},
		httpClient: server.Client(),
	}
	results, err := service.RecognizeAttachments(context.Background(), []oa.AttachmentFilePayload{
		{DocID: "bad", FileName: "bad.txt", FileSize: 2, FileData: "%%%"},
		testAttachment("disabled-doc", "legacy.doc", []byte("doc")),
		testAttachment("disabled-ofd", "invoice.ofd", []byte("ofd")),
		testAttachment("ok", "ok.pdf", []byte("%PDF")),
	}, "attachments", "附件")
	if err != nil {
		t.Fatalf("RecognizeAttachments() error = %v", err)
	}
	if len(results) != 4 {
		t.Fatalf("len(results) = %d, want 4", len(results))
	}
	if results[0].Error == "" || results[1].Error != "旧版 Office 解析未启用" || results[2].Error != "OFD 解析未启用" {
		t.Fatalf("unexpected failure results: %#v", results[:3])
	}
	if results[3].Content != "parsed" || results[3].Error != "" {
		t.Fatalf("最后一个附件应继续成功解析: %+v", results[3])
	}
	if minerUCalls != 1 {
		t.Fatalf("minerUCalls = %d, want 1", minerUCalls)
	}
}

func TestOFDFallbackFailureKeepsTextLayerContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/parse":
			_, _ = io.WriteString(w, `{"parser":"ofdrw","file_type":"ofd","content":"OFD 文字层","has_text_layer":true,"fallback_required":true,"fallback_format":"pdf","warnings":["版式复杂"]}`)
		case "/convert/pdf":
			http.Error(w, "conversion failed", http.StatusUnprocessableEntity)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	service := &AttachmentRecognitionService{
		configRepo: mapSystemConfigReader{
			"attachment.recognition_enabled":     "true",
			"attachment.supported_types":         "ofd",
			"attachment.compat_endpoint":         server.URL,
			"attachment.ofd_enabled":             "true",
			"attachment.visual_fallback_enabled": "true",
		},
		httpClient: server.Client(),
	}
	results, err := service.RecognizeAttachments(
		context.Background(),
		[]oa.AttachmentFilePayload{testAttachment("1", "invoice.ofd", []byte("ofd"))},
		"attachments",
		"附件",
	)
	if err != nil {
		t.Fatalf("RecognizeAttachments() error = %v", err)
	}
	if len(results) != 1 || results[0].Error != "" || results[0].Content != "OFD 文字层" {
		t.Fatalf("fallback failure should keep direct content: %#v", results)
	}
}

func TestEffectiveSupportedTypesAndRuleImportCapability(t *testing.T) {
	config := mapSystemConfigReader{
		"attachment.recognition_enabled":   "true",
		"attachment.supported_types":       "pdf,txt,doc,xls,ppt,ofd,unknown",
		"attachment.mineru_endpoint":       "http://mineru",
		"attachment.compat_endpoint":       "http://document-parser",
		"attachment.legacy_office_enabled": "false",
		"attachment.ofd_enabled":           "false",
	}
	attachmentService := &AttachmentRecognitionService{
		configRepo: config,
		httpClient: &http.Client{Timeout: time.Second},
	}
	cfg, err := attachmentService.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if got, want := effectiveSupportedTypes(cfg), []string{"pdf", "txt"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("effectiveSupportedTypes() = %v, want %v", got, want)
	}

	importService := &RuleImportService{attachmentService: attachmentService}
	capability, err := importService.Capability()
	if err != nil {
		t.Fatalf("Capability() error = %v", err)
	}
	if !capability.Enabled || !reflect.DeepEqual(capability.SupportedTypes, []string{"pdf", "txt"}) {
		t.Fatalf("Capability() = %+v", capability)
	}

	config["attachment.legacy_office_enabled"] = "true"
	config["attachment.ofd_enabled"] = "true"
	capability, err = importService.Capability()
	if err != nil {
		t.Fatalf("Capability() error = %v", err)
	}
	wantEnabled := []string{"doc", "ofd", "pdf", "ppt", "txt", "xls"}
	if !reflect.DeepEqual(capability.SupportedTypes, wantEnabled) {
		t.Fatalf("enabled compatibility types = %v, want %v", capability.SupportedTypes, wantEnabled)
	}
}

func TestCompatibilityResponseLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, strings.Repeat("x", maxCompatibilityJSONBytes+1))
	}))
	defer server.Close()

	service := &AttachmentRecognitionService{httpClient: server.Client()}
	_, err := service.parseViaCompatibility(
		context.Background(),
		&RecognitionConfig{CompatEndpoint: server.URL},
		"legacy.doc",
		[]byte("doc"),
	)
	if err == nil || !strings.Contains(err.Error(), "响应体超过") {
		t.Fatalf("parseViaCompatibility() error = %v", err)
	}
}

func TestCompatibilityConnectionWithUnsavedConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ready" {
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer unsaved-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_, _ = io.WriteString(w, `{"status":"ok"}`)
	}))
	defer server.Close()

	service := &AttachmentRecognitionService{httpClient: server.Client()}
	err := service.TestCompatibilityConnectionWithConfig(context.Background(), &RecognitionConfig{
		CompatEndpoint: server.URL,
		CompatAPIKey:   "unsaved-key",
	})
	if err != nil {
		t.Fatalf("TestCompatibilityConnectionWithConfig() error = %v", err)
	}
}

func TestCompatibilityConnectionReportsReadyAuthenticationFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ready" {
			http.NotFound(w, r)
			return
		}
		http.Error(w, `{"code":"unauthorized"}`, http.StatusUnauthorized)
	}))
	defer server.Close()

	service := &AttachmentRecognitionService{httpClient: server.Client()}
	err := service.TestCompatibilityConnectionWithConfig(context.Background(), &RecognitionConfig{
		CompatEndpoint: server.URL,
		CompatAPIKey:   "wrong-key",
	})
	if err == nil || !strings.Contains(err.Error(), "/ready 返回 HTTP 401") {
		t.Fatalf("TestCompatibilityConnectionWithConfig() error = %v", err)
	}
}

func testAttachment(docID, fileName string, raw []byte) oa.AttachmentFilePayload {
	return oa.AttachmentFilePayload{
		DocID:    docID,
		FileName: fileName,
		FileSize: int64(len(raw)),
		FileData: base64.StdEncoding.EncodeToString(raw),
	}
}

func readMultipartTestFile(r *http.Request, field string) (string, []byte, error) {
	if err := r.ParseMultipartForm(70 * 1024 * 1024); err != nil {
		return "", nil, err
	}
	file, header, err := r.FormFile(field)
	if err != nil {
		return "", nil, err
	}
	defer file.Close()
	body, err := io.ReadAll(file)
	if err != nil {
		return "", nil, err
	}
	return header.Filename, body, nil
}
