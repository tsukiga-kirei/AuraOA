package service

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"auraoa/go-service/internal/pkg/oa"
)

func buildMinerU4ResultZip(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %s: %v", name, err)
		}
		if _, err := io.WriteString(w, content); err != nil {
			t.Fatalf("write zip entry %s: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
}

func TestMinerU4TaskMarkdown(t *testing.T) {
	t.Run("upload_needed_and_polling", func(t *testing.T) {
		rawPDF := []byte("%PDF-1.7 contract body")
		sum := sha256.Sum256(rawPDF)
		wantSHA := hex.EncodeToString(sum[:])
		resultZip := buildMinerU4ResultZip(t, map[string]string{
			"markdown.md":        "# 合同正文\n甲方：测试单位",
			"middle_json.json":   "{}",
			"images/figure1.png": "png",
		})

		for _, tc := range []struct {
			name        string
			crossOrigin bool
		}{
			{name: "same_origin_upload"},
			{name: "cross_origin_upload", crossOrigin: true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				var uploadCalled atomic.Bool
				var completeCalled atomic.Bool
				var pollCount atomic.Int32
				var uploadMu sync.Mutex
				var uploadBody []byte
				var uploadHeader string
				var uploadAuth string

				extUpload := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method != http.MethodPut {
						t.Fatalf("expected PUT upload, got %s", r.Method)
					}
					body, _ := io.ReadAll(r.Body)
					uploadMu.Lock()
					uploadBody = body
					uploadHeader = r.Header.Get("X-Upload-Token")
					uploadAuth = r.Header.Get("Authorization")
					uploadMu.Unlock()
					uploadCalled.Store(true)
					w.WriteHeader(http.StatusOK)
				}))
				defer extUpload.Close()

				var apiServer *httptest.Server
				apiServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					switch {
					case r.Method == http.MethodPost && r.URL.Path == "/mineru/v1/tasks":
						if got := r.Header.Get("Authorization"); got != "Bearer v4-key" {
							t.Fatalf("missing bearer token on POST /v1/tasks: %q", got)
						}
						var rawReq map[string]any
						if err := json.NewDecoder(r.Body).Decode(&rawReq); err != nil {
							t.Fatalf("decode task req: %v", err)
						}
						for _, forbidden := range []string{"file_id", "client_file_id", "manifest", "filename"} {
							if _, exists := rawReq[forbidden]; exists {
								t.Fatalf("unexpected top-level field %q in MinerU 4.1 request: %v", forbidden, rawReq)
							}
						}
						if rawReq["tier"] != "advanced" || rawReq["ocr_mode"] != "ocr" {
							t.Fatalf("unexpected tier/ocr_mode: %v", rawReq)
						}
						files, _ := rawReq["files"].([]any)
						if len(files) != 1 {
							t.Fatalf("expected 1 file in request, got %v", rawReq["files"])
						}
						f0 := files[0].(map[string]any)
						for _, forbidden := range []string{"file_id", "client_file_id", "filename"} {
							if _, exists := f0[forbidden]; exists {
								t.Fatalf("unexpected file field %q in MinerU 4.1 request: %v", forbidden, f0)
							}
						}
						if f0["file_name"] != "合同.pdf" || int(f0["bytes"].(float64)) != len(rawPDF) || f0["sha256sum"] != wantSHA {
							t.Fatalf("unexpected file manifest: %v", f0)
						}

						uploadURL := apiServer.URL + "/mineru/upload/file-1"
						if tc.crossOrigin {
							uploadURL = extUpload.URL + "/presigned-put"
						}
						w.WriteHeader(http.StatusAccepted)
						_ = json.NewEncoder(w).Encode(map[string]any{
							"task_id": "task-41-1",
							"status":  "waiting-file",
							"files": []map[string]any{{
								"file_id":   "file-1",
								"file_name": "合同.pdf",
								"status":    "waiting-file",
								"upload": map[string]any{
									"url":     uploadURL,
									"headers": map[string]string{"X-Upload-Token": "sig-ok"},
								},
							}},
						})
					case r.Method == http.MethodPut && r.URL.Path == "/mineru/upload/file-1":
						body, _ := io.ReadAll(r.Body)
						uploadMu.Lock()
						uploadBody = body
						uploadHeader = r.Header.Get("X-Upload-Token")
						uploadAuth = r.Header.Get("Authorization")
						uploadMu.Unlock()
						uploadCalled.Store(true)
						_ = json.NewEncoder(w).Encode(map[string]any{
							"task_id": "task-41-1",
							"file_id": "file-1",
							"status":  "pending",
						})
					case strings.HasSuffix(r.URL.Path, "/complete"):
						completeCalled.Store(true)
						http.NotFound(w, r)
					case r.Method == http.MethodGet && r.URL.Path == "/mineru/v1/tasks/task-41-1":
						n := pollCount.Add(1)
						if n == 1 {
							_ = json.NewEncoder(w).Encode(map[string]any{
								"task_id": "task-41-1",
								"status":  "running",
								"files": []map[string]any{{
									"file_id":   "file-1",
									"file_name": "合同.pdf",
									"status":    "running",
								}},
							})
							return
						}
						_ = json.NewEncoder(w).Encode(map[string]any{
							"task_id": "task-41-1",
							"status":  "completed",
							"files": []map[string]any{{
								"file_id":    "file-1",
								"file_name":  "合同.pdf",
								"status":     "completed",
								"result_url": "/mineru/v1/results/task-41-1/file-1.zip",
							}},
						})
					case r.Method == http.MethodGet && r.URL.Path == "/mineru/v1/results/task-41-1/file-1.zip":
						if got := r.Header.Get("Authorization"); got != "Bearer v4-key" {
							t.Fatalf("missing bearer token on same-origin result download: %q", got)
						}
						w.Header().Set("Content-Type", "application/zip")
						_, _ = w.Write(resultZip)
					default:
						http.NotFound(w, r)
					}
				}))
				defer apiServer.Close()

				s := &AttachmentRecognitionService{httpClient: &http.Client{Timeout: 5 * time.Second}}
				cfg := &RecognitionConfig{
					MinerU4Endpoint: apiServer.URL + "/mineru",
					MinerU4APIKey:   "v4-key",
					MinerU4Tier:     "advanced",
					MinerU4OCRMode:  "ocr",
				}
				content, err := s.parseMinerU4(context.Background(), cfg, "合同.pdf", rawPDF)
				if err != nil {
					t.Fatalf("parseMinerU4 failed: %v", err)
				}
				if content != "# 合同正文\n甲方：测试单位" {
					t.Fatalf("unexpected markdown content: %q", content)
				}
				if !uploadCalled.Load() {
					t.Fatal("expected PUT upload to be called")
				}
				if completeCalled.Load() {
					t.Fatal("MinerU 4.1 must not call legacy /complete endpoint")
				}
				uploadMu.Lock()
				gotBody, gotHeader, gotAuth := uploadBody, uploadHeader, uploadAuth
				uploadMu.Unlock()
				if !bytes.Equal(gotBody, rawPDF) || gotHeader != "sig-ok" {
					t.Fatalf("upload body/header mismatch: len=%d token=%q", len(gotBody), gotHeader)
				}
				if tc.crossOrigin && gotAuth != "" {
					t.Fatalf("Authorization leaked to cross-origin upload URL: %q", gotAuth)
				}
				if !tc.crossOrigin && gotAuth != "Bearer v4-key" {
					t.Fatalf("same-origin upload expected Bearer key, got %q", gotAuth)
				}
			})
		}
	})

	t.Run("cache_hit_and_fallback_md_and_sanitized_filename", func(t *testing.T) {
		var putCalled atomic.Bool
		var gotSubmittedName string
		resultZip := buildMinerU4ResultZip(t, map[string]string{
			"__MACOSX/._markdown.md": "ignored",
			"nested/output.md":       "缓存命中正文",
		})
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPost && r.URL.Path == "/v1/tasks":
				var req minerU4CreateTaskRequest
				_ = json.NewDecoder(r.Body).Decode(&req)
				gotSubmittedName = req.Files[0].FileName
				_ = json.NewEncoder(w).Encode(map[string]any{
					"task_id": "task-cache",
					"status":  "completed",
					"files": []map[string]any{{
						"file_id":    "f-cache",
						"file_name":  gotSubmittedName,
						"status":     "completed",
						"result_url": "/v1/files/f-cache/result.zip",
					}},
				})
			case r.Method == http.MethodPut:
				putCalled.Store(true)
				w.WriteHeader(http.StatusOK)
			case r.Method == http.MethodGet && r.URL.Path == "/v1/files/f-cache/result.zip":
				_, _ = w.Write(resultZip)
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()

		s := &AttachmentRecognitionService{httpClient: &http.Client{Timeout: 5 * time.Second}}
		content, err := s.parseMinerU4(context.Background(), &RecognitionConfig{MinerU4Endpoint: server.URL}, `dir\sub:../evil..file.pdf`, []byte("%PDF"))
		if err != nil {
			t.Fatalf("cache hit parseMinerU4 failed: %v", err)
		}
		if content != "缓存命中正文" {
			t.Fatalf("unexpected content: %q", content)
		}
		if putCalled.Load() {
			t.Fatal("cache hit should not upload bytes")
		}
		if strings.ContainsAny(gotSubmittedName, `/\:`) || strings.Contains(gotSubmittedName, "..") {
			t.Fatalf("unsafe submitted file_name: %q", gotSubmittedName)
		}
	})
}

func TestMinerU4TerminalFailuresAndInvalidResponses(t *testing.T) {
	cases := []struct {
		name      string
		raw       []byte
		handler   http.HandlerFunc
		wantInErr string
	}{
		{
			name:      "empty_file",
			raw:       nil,
			wantInErr: "为空",
		},
		{
			name: "http_422_structured_error",
			raw:  []byte("pdf"),
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnprocessableEntity)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]any{
						"code":    "invalid_request",
						"message": "bad file name",
						"param":   "files[0].file_name",
					},
				})
			},
			wantInErr: "[invalid_request] bad file name",
		},
		{
			name: "http_404_plain",
			raw:  []byte("pdf"),
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.NotFound(w, r)
			},
			wantInErr: "HTTP 404",
		},
		{
			name: "async_file_failed",
			raw:  []byte("pdf"),
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					_ = json.NewEncoder(w).Encode(map[string]any{
						"task_id": "t1",
						"status":  "pending",
						"files":   []map[string]any{{"file_id": "f1", "file_name": "a.pdf", "status": "pending"}},
					})
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{
					"task_id": "t1",
					"status":  "failed",
					"files": []map[string]any{{
						"file_id":   "f1",
						"file_name": "a.pdf",
						"status":    "failed",
						"error":     map[string]any{"code": "parse_failed", "message": "corrupt pdf"},
					}},
				})
			},
			wantInErr: "[parse_failed] corrupt pdf",
		},
		{
			name: "task_canceled",
			raw:  []byte("pdf"),
			handler: func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"task_id": "t1",
					"status":  "canceled",
					"error":   map[string]any{"code": "canceled", "message": "user canceled"},
				})
			},
			wantInErr: "已取消",
		},
		{
			name: "task_id_mismatch",
			raw:  []byte("pdf"),
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					_ = json.NewEncoder(w).Encode(map[string]any{
						"task_id": "t1",
						"status":  "pending",
						"files":   []map[string]any{{"file_id": "f1", "file_name": "a.pdf", "status": "pending"}},
					})
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{
					"task_id": "other-id",
					"status":  "completed",
					"files":   []map[string]any{{"file_id": "f1", "file_name": "a.pdf", "status": "completed", "result_url": "/r.zip"}},
				})
			},
			wantInErr: "不匹配",
		},
		{
			name: "invalid_zip_result",
			raw:  []byte("pdf"),
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					_ = json.NewEncoder(w).Encode(map[string]any{
						"task_id": "t1",
						"status":  "completed",
						"files":   []map[string]any{{"file_id": "f1", "file_name": "a.pdf", "status": "completed", "result_url": "/bad.zip"}},
					})
					return
				}
				_, _ = io.WriteString(w, "not-a-zip")
			},
			wantInErr: "压缩包",
		},
		{
			name: "empty_markdown_in_zip",
			raw:  []byte("pdf"),
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					_ = json.NewEncoder(w).Encode(map[string]any{
						"task_id": "t1",
						"status":  "completed",
						"files":   []map[string]any{{"file_id": "f1", "file_name": "a.pdf", "status": "completed", "result_url": "/empty.zip"}},
					})
					return
				}
				_, _ = w.Write(buildMinerU4ResultZip(t, map[string]string{"markdown.md": "   \n  "}))
			},
			wantInErr: "有效内容",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			endpoint := "http://unused.local"
			if tc.handler != nil {
				server := httptest.NewServer(tc.handler)
				defer server.Close()
				endpoint = server.URL
			}
			s := &AttachmentRecognitionService{httpClient: &http.Client{Timeout: 5 * time.Second}}
			_, err := s.parseMinerU4(context.Background(), &RecognitionConfig{MinerU4Endpoint: endpoint}, "a.pdf", tc.raw)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantInErr)
			}
			if !strings.Contains(err.Error(), tc.wantInErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantInErr, err)
			}
		})
	}
}

func TestMinerU4PollingHonorsTotalTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"task_id": "t-slow",
			"status":  "pending",
			"files":   []map[string]any{{"file_id": "f1", "file_name": "a.pdf", "status": "pending"}},
		})
	}))
	defer server.Close()

	s := &AttachmentRecognitionService{httpClient: &http.Client{Timeout: 30 * time.Millisecond}}
	start := time.Now()
	_, err := s.parseMinerU4(context.Background(), &RecognitionConfig{MinerU4Endpoint: server.URL}, "a.pdf", []byte("pdf"))
	elapsed := time.Since(start)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("polling ignored timeout, elapsed=%v", elapsed)
	}
}

func TestMinerU4CredentialsIsolatedAcrossOriginsAndRedirects(t *testing.T) {
	var cdnAuth string
	var redirectAuth string

	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cdnAuth = r.Header.Get("Authorization")
		redirectAuth = cdnAuth
		_, _ = io.WriteString(w, "ok")
	}))
	defer cdn.Close()

	var apiAuth string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiAuth = r.Header.Get("Authorization")
		if r.URL.Path == "/redirect-cdn" {
			http.Redirect(w, r, cdn.URL+"/asset", http.StatusFound)
			return
		}
		_, _ = io.WriteString(w, "ok")
	}))
	defer api.Close()

	s := &AttachmentRecognitionService{httpClient: &http.Client{Timeout: 5 * time.Second}}
	cfg := &RecognitionConfig{MinerU4Endpoint: api.URL, MinerU4APIKey: "secret"}

	if _, err := s.minerU4Request(context.Background(), cfg, http.MethodGet, api.URL+"/same", nil, nil); err != nil {
		t.Fatalf("same-origin request failed: %v", err)
	}
	if apiAuth != "Bearer secret" {
		t.Fatalf("expected Bearer secret on same-origin, got %q", apiAuth)
	}

	if _, err := s.minerU4Request(context.Background(), cfg, http.MethodGet, cdn.URL+"/direct", nil, nil); err != nil {
		t.Fatalf("cross-origin request failed: %v", err)
	}
	if cdnAuth != "" {
		t.Fatalf("expected empty Authorization on cross-origin, got %q", cdnAuth)
	}

	if _, err := s.minerU4Request(context.Background(), cfg, http.MethodGet, api.URL+"/redirect-cdn", nil, nil); err != nil {
		t.Fatalf("redirect request failed: %v", err)
	}
	if redirectAuth != "" {
		t.Fatalf("expected empty Authorization after cross-origin redirect, got %q", redirectAuth)
	}

	a, _ := url.Parse("http://127.0.0.1:8080/a")
	b, _ := url.Parse("http://127.0.0.1:8080/b")
	c, _ := url.Parse("http://127.0.0.1:8081/a")
	if !minerU4SameOrigin(a, b) || minerU4SameOrigin(a, c) {
		t.Fatal("minerU4SameOrigin check mismatch")
	}
}

func TestMinerU4RoutingAndHealth(t *testing.T) {
	var mu sync.Mutex
	seenTiers := map[string]string{}
	resultZip := buildMinerU4ResultZip(t, map[string]string{"markdown.md": "文档解析内容"})

	var healthPayload atomic.Value
	healthPayload.Store(`{"status":"ok","version":"4.1.2"}`)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/health":
			_, _ = io.WriteString(w, healthPayload.Load().(string))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/tasks":
			var req minerU4CreateTaskRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req.OCRMode != "txt" {
				t.Fatalf("expected ocr_mode=txt, got %q", req.OCRMode)
			}
			mu.Lock()
			seenTiers[req.Files[0].FileName] = req.Tier
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{
				"task_id": "t-" + req.Files[0].FileName,
				"status":  "completed",
				"files": []map[string]any{{
					"file_id":    "f-" + req.Files[0].FileName,
					"file_name":  req.Files[0].FileName,
					"status":     "completed",
					"result_url": "/res/" + url.PathEscape(req.Files[0].FileName) + ".zip",
				}},
			})
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/res/"):
			_, _ = w.Write(resultZip)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	s := &AttachmentRecognitionService{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		configRepo: mapSystemConfigReader{
			"attachment.recognition_enabled": "true",
			"attachment.ocr_provider":        "mineru4",
			"attachment.mineru4_endpoint":    server.URL,
			"attachment.mineru4_tier":        "advanced",
			"attachment.mineru4_ocr_mode":    "txt",
		},
	}
	cfg, err := s.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.MinerU4Tier != "advanced" || cfg.MinerU4OCRMode != "txt" {
		t.Fatalf("unexpected MinerU4 config: %+v", cfg)
	}

	for _, ext := range []string{"pdf", "docx", "doc", "xls", "ppt", "ofd"} {
		fileName := "a." + ext
		got := s.recognizeAttachment(context.Background(), cfg, oa.AttachmentFilePayload{FileName: fileName}, []byte("raw"), oa.AttachmentInfo{DocID: "1", FieldKey: "field", ArchiveFileName: "a.zip"})
		if got.Error != "" || got.Content != "文档解析内容" {
			t.Fatalf("ext %s failed: %+v", ext, got)
		}
		wantTier := "flash"
		if ext == "pdf" {
			wantTier = "advanced"
		}
		mu.Lock()
		gotTier := seenTiers[fileName]
		mu.Unlock()
		if gotTier != wantTier {
			t.Fatalf("ext %s want tier %q, got %q", ext, wantTier, gotTier)
		}
	}

	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	w, _ := zw.Create("doc.docx")
	_, _ = io.WriteString(w, "raw")
	_ = zw.Close()

	zipped, err := s.RecognizeAttachments(context.Background(), []oa.AttachmentFilePayload{testAttachment("zip-id", "资料.zip", archive.Bytes())}, "fj", "附件")
	if err != nil || len(zipped) != 1 || zipped[0].Content != "文档解析内容" {
		t.Fatalf("RecognizeAttachments zip failed: err=%v res=%+v", err, zipped)
	}

	fallback := s.recognizeAttachment(context.Background(), cfg, oa.AttachmentFilePayload{FileName: "scan.pdf"}, []byte("%PDF"), oa.AttachmentInfo{FileType: "pdf", DocID: "scan"})
	if fallback.Error != "" || fallback.Content != "文档解析内容" {
		t.Fatalf("unexpected fallback result: %+v", fallback)
	}

	// 健康检查与 4.0 版本不兼容提示。
	if err := s.testMinerU4Connection(context.Background(), cfg); err != nil {
		t.Fatalf("expected 4.1 health check to pass, got %v", err)
	}
	healthPayload.Store(`{"status":"ok"}`)
	if err := s.testMinerU4Connection(context.Background(), cfg); err != nil {
		t.Fatalf("expected health check without version field to pass, got %v", err)
	}
	healthPayload.Store(`{"status":"ok","version":"4.0.0"}`)
	if err := s.testMinerU4Connection(context.Background(), cfg); err == nil || !strings.Contains(err.Error(), "4.1") {
		t.Fatalf("expected 4.0 mismatch error mentioning 4.1, got %v", err)
	}
	healthPayload.Store(`{"status":"error"}`)
	if err := s.testMinerU4Connection(context.Background(), cfg); err == nil {
		t.Fatal("expected unhealthy error")
	}

	emptyCfg := *cfg
	emptyCfg.MinerU4Endpoint = ""
	if err := s.testMinerU4Connection(context.Background(), &emptyCfg); err == nil {
		t.Fatal("expected error for empty endpoint")
	}
	got := s.recognizeOneViaMinerU(context.Background(), &emptyCfg, oa.AttachmentFilePayload{FileName: "scan.pdf", FileData: base64.StdEncoding.EncodeToString([]byte("raw"))}, oa.AttachmentInfo{DocID: "fallback"})
	if got.Error == "" {
		t.Fatal("expected error when mineru4 endpoint is empty")
	}
}
