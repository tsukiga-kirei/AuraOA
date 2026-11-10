package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"auraoa/go-service/internal/pkg/oa"
)

func TestMinerU4UploadJobMarkdown(t *testing.T) {
	for _, cached := range []bool{false, true} {
		t.Run(map[bool]string{false: "upload", true: "cached"}[cached], func(t *testing.T) {
			var calls []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls = append(calls, r.Method+" "+r.URL.Path)
				if r.Header.Get("Authorization") != "Bearer v4-key" {
					t.Error("缺少新版鉴权")
				}
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/mineru/v1/uploads":
					var payload map[string]any
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Error(err)
					}
					if payload["filename"] != "合同.pdf" || payload["bytes"] != float64(4) || len(payload["sha256sum"].(string)) != 64 {
						t.Errorf("上传参数异常: %v", payload)
					}
					if cached {
						io.WriteString(w, `{"id":"upload-1","status":"completed","file":{"id":"source-1"}}`)
					} else {
						io.WriteString(w, `{"id":"upload-1","status":"pending","upload_url":"/mineru/upload","upload_method":"PUT","upload_headers":{"X-Upload":"yes"}}`)
					}
				case "/mineru/upload":
					data, _ := io.ReadAll(r.Body)
					if string(data) != "%PDF" || r.Header.Get("X-Upload") != "yes" {
						t.Error("上传内容或返回的请求头丢失")
					}
					w.WriteHeader(http.StatusNoContent)
				case "/mineru/v1/uploads/upload-1/complete":
					io.WriteString(w, `{"status":"completed","file":{"id":"source-1"}}`)
				case "/mineru/v1/parse/jobs":
					var payload map[string]any
					json.NewDecoder(r.Body).Decode(&payload)
					if payload["tier"] != "advanced" || payload["ocr_mode"] != "ocr" {
						t.Errorf("任务参数异常: %v", payload)
					}
					source := payload["files"].([]any)[0].(map[string]any)["source"].(map[string]any)
					if source["type"] != "file_id" || source["file_id"] != "source-1" {
						t.Errorf("文件来源异常: %v", source)
					}
					for _, old := range []string{"backend", "language", "lang_list", "formula_enable", "table_enable"} {
						if _, ok := payload[old]; ok {
							t.Errorf("不应发送旧版参数 %s", old)
						}
					}
					io.WriteString(w, `{"job_id":"job-1","status":"queued"}`)
				case "/mineru/v1/parse/jobs/job-1":
					io.WriteString(w, `{"job_id":"job-1","status":"completed","files":[{"status":"completed","output_files":{"markdown":{"file_id":"md-1"}}}]}`)
				case "/mineru/v1/files/md-1/content":
					io.WriteString(w, "# 合同正文")
				default:
					t.Errorf("意外请求 %s", r.URL.Path)
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			s := &AttachmentRecognitionService{httpClient: server.Client()}
			cfg := &RecognitionConfig{MinerU4Endpoint: server.URL + "/mineru", MinerU4APIKey: "v4-key", MinerU4Tier: "advanced", MinerU4OCRMode: "ocr"}
			content, err := s.parseMinerU4(context.Background(), cfg, "合同.pdf", []byte("%PDF"))
			if err != nil || content != "# 合同正文" {
				t.Fatalf("解析结果 = %q, %v", content, err)
			}
			want := []string{"POST /mineru/v1/uploads"}
			if !cached {
				want = append(want, "PUT /mineru/upload", "POST /mineru/v1/uploads/upload-1/complete")
			}
			want = append(want, "POST /mineru/v1/parse/jobs", "GET /mineru/v1/parse/jobs/job-1", "GET /mineru/v1/files/md-1/content")
			if !reflect.DeepEqual(calls, want) {
				t.Fatalf("请求流程 = %v", calls)
			}
		})
	}
}

func TestMinerU4TerminalFailuresAndInvalidResponses(t *testing.T) {
	tests := []struct{ name, upload, job, want string }{
		{"missing upload id", `{"status":"pending"}`, "", "缺少 id"},
		{"missing file id", `{"id":"u","status":"completed"}`, "", "缺少 file.id"},
		{"failed", `{"id":"u","status":"completed","file":{"id":"f"}}`, `{"job_id":"j","status":"failed","files":[{"status":"failed","error":{"message":"模型不可用"}}]}`, "模型不可用"},
		{"canceled", `{"id":"u","status":"completed","file":{"id":"f"}}`, `{"job_id":"j","status":"canceled"}`, "canceled"},
		{"partial", `{"id":"u","status":"completed","file":{"id":"f"}}`, `{"job_id":"j","status":"partial"}`, "partial"},
		{"missing artifact", `{"id":"u","status":"completed","file":{"id":"f"}}`, `{"job_id":"j","status":"completed","files":[{"status":"completed"}]}`, "缺少 Markdown"},
		{"unknown status", `{"id":"u","status":"completed","file":{"id":"f"}}`, `{"job_id":"j","status":"unknown"}`, "状态异常"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/v1/uploads":
					io.WriteString(w, tt.upload)
				case "/v1/parse/jobs":
					io.WriteString(w, tt.job)
				default:
					t.Errorf("失败响应后不应继续请求 %s", r.URL.Path)
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			s := &AttachmentRecognitionService{httpClient: server.Client()}
			_, err := s.parseMinerU4(context.Background(), &RecognitionConfig{MinerU4Endpoint: server.URL}, "a.pdf", []byte("pdf"))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestMinerU4PollingHonorsTotalTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/uploads":
			io.WriteString(w, `{"id":"u","status":"completed","file":{"id":"f"}}`)
		default:
			io.WriteString(w, `{"job_id":"j","status":"running"}`)
		}
	}))
	defer server.Close()
	s := &AttachmentRecognitionService{httpClient: &http.Client{Timeout: 30 * time.Millisecond}}
	start := time.Now()
	_, err := s.parseMinerU4(context.Background(), &RecognitionConfig{MinerU4Endpoint: server.URL}, "a.pdf", []byte("pdf"))
	if err == nil || !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Fatalf("超时结果: %v", err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("轮询未及时停止")
	}
}

func TestMinerU4CredentialsIsolatedAcrossOriginsAndRedirects(t *testing.T) {
	storage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
			t.Error("跨源泄露凭证")
		}
		io.WriteString(w, "markdown")
	}))
	defer storage.Close()
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Error("API 缺少鉴权")
		}
		http.Redirect(w, r, storage.URL, http.StatusFound)
	}))
	defer api.Close()
	s := &AttachmentRecognitionService{httpClient: api.Client()}
	cfg := &RecognitionConfig{MinerU4Endpoint: api.URL, MinerU4APIKey: "secret"}
	for _, target := range []string{api.URL + "/download", storage.URL} {
		if _, err := s.minerU4Request(context.Background(), cfg, http.MethodGet, target, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	a, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	b, _ := http.NewRequest(http.MethodGet, "http://example.com:80/path", nil)
	if !minerU4SameOrigin(a.URL, b.URL) {
		t.Error("默认端口应同源")
	}
}

func TestMinerU4RoutingAndHealth(t *testing.T) {
	var tier string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/health":
			io.WriteString(w, `{"status":"ok"}`)
		case "/parse":
			io.WriteString(w, `{"content":"","fallback_required":true,"fallback_format":"pdf"}`)
		case "/v1/uploads":
			var upload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&upload); err != nil || upload["bytes"] == float64(0) {
				t.Errorf("附件字节不可为空: %v, %v", upload, err)
			}
			io.WriteString(w, `{"id":"u","status":"completed","file":{"id":"f"}}`)
		case "/v1/parse/jobs":
			var p map[string]any
			json.NewDecoder(r.Body).Decode(&p)
			tier = p["tier"].(string)
			io.WriteString(w, `{"job_id":"j","status":"completed","files":[{"status":"completed","output_files":{"markdown":{"file_id":"md"}}}]}`)
		case "/v1/files/md/content":
			io.WriteString(w, "正文")
		default:
			t.Errorf("请求不应走旧版接口 %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	s := &AttachmentRecognitionService{httpClient: server.Client(), configRepo: mapSystemConfigReader{
		"attachment.recognition_enabled": "true", "attachment.ocr_provider": "mineru4", "attachment.mineru4_endpoint": server.URL,
		"attachment.mineru4_tier": "advanced", "attachment.mineru4_ocr_mode": "txt",
	}}
	cfg, err := s.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MinerU4Tier != "advanced" || cfg.MinerU4OCRMode != "txt" {
		t.Fatalf("配置未加载: %+v", cfg)
	}
	if err := s.TestConnectionWithConfig(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	for _, ext := range []string{"pdf", "docx", "doc", "xls", "ppt", "ofd"} {
		found := false
		for _, v := range effectiveSupportedTypes(cfg) {
			if v == ext {
				found = true
			}
		}
		if !found {
			t.Errorf("未声明新版支持格式 %s", ext)
		}
		got := s.recognizeAttachment(context.Background(), cfg, oa.AttachmentFilePayload{FileName: "a." + ext}, []byte("raw"), oa.AttachmentInfo{DocID: "1", FieldKey: "field", ArchiveFileName: "a.zip"})
		if got.Error != "" || got.Content != "正文" || got.ArchiveFileName != "a.zip" || got.FieldKey != "field" {
			t.Fatalf("格式 %s 路由结果: %+v", ext, got)
		}
		wantTier := "flash"
		if ext == "pdf" {
			wantTier = "advanced"
		}
		if tier != wantTier {
			t.Errorf("%s tier=%s", ext, tier)
		}
	}
	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	entry, err := zw.Create("folder/scan.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("%PDF")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	zipped, err := s.RecognizeAttachments(context.Background(), []oa.AttachmentFilePayload{testAttachment("zip-id", "资料.zip", archive.Bytes())}, "fj", "附件")
	if err != nil || len(zipped) != 1 || zipped[0].Content != "正文" || zipped[0].ArchiveFileName != "资料.zip" {
		t.Fatalf("ZIP 子附件未进入新版解析器: %+v, %v", zipped, err)
	}
	cfg.CompatEndpoint = server.URL
	cfg.DocumentParserTypes = []string{"pdf"}
	cfg.VisualFallbackEnabled = true
	fallback := s.recognizeAttachment(context.Background(), cfg, oa.AttachmentFilePayload{FileName: "scan.pdf"}, []byte("%PDF"), oa.AttachmentInfo{FileType: "pdf", DocID: "scan"})
	if fallback.Error != "" || fallback.Content != "正文" || fallback.DocID != "scan" {
		t.Fatalf("代码解析后的视觉回退失败: %+v", fallback)
	}
	// 代码解析仍优先；禁用新版端点时能力列表不能错误地借用旧版地址。
	cfg.DocumentParserTypes = []string{"pdf"}
	cfg.CompatEndpoint = ""
	cfg.MinerU4Endpoint = ""
	cfg.MinerUEndpoint = server.URL
	if len(effectiveSupportedTypes(cfg)) != 3 {
		t.Fatalf("无端点时应仅保留内置文本格式: %v", effectiveSupportedTypes(cfg))
	}
	// 新版视觉回退使用与直接解析相同的入口。
	cfg.MinerU4Endpoint = server.URL
	got := s.recognizeOneViaMinerU(context.Background(), cfg, oa.AttachmentFilePayload{FileName: "scan.pdf", FileData: base64.StdEncoding.EncodeToString([]byte("raw"))}, oa.AttachmentInfo{DocID: "fallback"})
	if got.Error != "" || got.Content != "正文" || got.DocID != "fallback" {
		t.Fatalf("新版回退失败: %+v", got)
	}
}
