package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"auraoa/go-service/internal/pkg/errcode"
	"auraoa/go-service/internal/pkg/oa"
)

// MinerU 4.0 使用 uploads → parse/jobs → files 协议；4.1 的 Tasks 协议需独立适配。
// 契约依据：MinerU mineru-4.0.0-released 的 HTTP API 文档与官方客户端。
type minerU4Upload struct {
	ID            string            `json:"id"`
	Status        string            `json:"status"`
	UploadURL     string            `json:"upload_url"`
	UploadMethod  string            `json:"upload_method"`
	UploadHeaders map[string]string `json:"upload_headers"`
	File          struct {
		ID string `json:"id"`
	} `json:"file"`
}

type minerU4Job struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
	Files  []struct {
		Status string `json:"status"`
		Error  *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		OutputFiles map[string]struct {
			FileID string `json:"file_id"`
		} `json:"output_files"`
	} `json:"files"`
}

func (s *AttachmentRecognitionService) recognizeOneViaMinerU4(ctx context.Context, cfg *RecognitionConfig, name string, raw []byte, base oa.AttachmentInfo) oa.AttachmentInfo {
	content, err := s.parseMinerU4(ctx, cfg, name, raw)
	if err != nil {
		base.Error = "MinerU 4.0 解析失败: " + err.Error()
	} else {
		base.Content = content
	}
	return base
}

func (s *AttachmentRecognitionService) parseMinerU4(ctx context.Context, cfg *RecognitionConfig, name string, raw []byte) (string, error) {
	if strings.TrimSpace(cfg.MinerU4Endpoint) == "" {
		return "", fmt.Errorf("服务地址未配置")
	}
	tier, mode := cfg.MinerU4Tier, cfg.MinerU4OCRMode
	if tier == "" {
		tier = "standard"
	}
	if mode == "" {
		mode = "auto"
	}
	switch tier {
	case "flash", "basic", "standard", "advanced":
	default:
		return "", fmt.Errorf("不支持的解析档位 %q", tier)
	}
	switch mode {
	case "auto", "txt", "ocr":
	default:
		return "", fmt.Errorf("不支持的 OCR 模式 %q", mode)
	}
	// 原生 Office/OFD 由 Flash 解析，不把 PDF/图片的模型档位强加给原生文档。
	if ext := extractFileExt(name); isType(documentParserTypes, ext) && ext != "pdf" {
		tier = "flash"
	}
	timeout := s.httpClient.Timeout
	if timeout <= 0 {
		timeout = defaultMinerUHTTPTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	root := strings.TrimRight(cfg.MinerU4Endpoint, "/")
	sum := sha256.Sum256(raw)
	contentType := mime.TypeByExtension(filepath.Ext(name))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	var upload minerU4Upload
	if err := s.minerU4JSON(ctx, cfg, http.MethodPost, root+"/v1/uploads", map[string]any{
		"filename": safeAttachmentFileName(name), "bytes": len(raw), "mime_type": contentType,
		"purpose": "parse", "sha256sum": hex.EncodeToString(sum[:]),
	}, &upload); err != nil {
		return "", fmt.Errorf("创建上传失败: %w", err)
	}
	if upload.ID == "" {
		return "", fmt.Errorf("上传响应缺少 id")
	}
	switch upload.Status {
	case "pending":
		if upload.UploadURL == "" {
			return "", fmt.Errorf("上传响应缺少 upload_url")
		}
		method := upload.UploadMethod
		if method == "" {
			method = http.MethodPut
		}
		if method != http.MethodPut {
			return "", fmt.Errorf("不支持的上传方法 %q", method)
		}
		if _, err := s.minerU4Request(ctx, cfg, method, upload.UploadURL, bytes.NewReader(raw), upload.UploadHeaders); err != nil {
			return "", fmt.Errorf("上传文件失败: %w", err)
		}
		id := upload.ID
		upload = minerU4Upload{}
		if err := s.minerU4JSON(ctx, cfg, http.MethodPost, root+"/v1/uploads/"+url.PathEscape(id)+"/complete", nil, &upload); err != nil {
			return "", fmt.Errorf("完成上传失败: %w", err)
		}
		if upload.Status != "completed" {
			return "", fmt.Errorf("完成上传状态异常 %q", upload.Status)
		}
	case "completed":
	default:
		return "", fmt.Errorf("上传状态异常 %q", upload.Status)
	}
	if upload.File.ID == "" {
		return "", fmt.Errorf("上传响应缺少 file.id")
	}
	var job minerU4Job
	if err := s.minerU4JSON(ctx, cfg, http.MethodPost, root+"/v1/parse/jobs", map[string]any{
		"files": []any{map[string]any{"source": map[string]string{"type": "file_id", "file_id": upload.File.ID}}},
		"tier":  tier, "ocr_mode": mode, "output_formats": []string{"markdown"},
	}, &job); err != nil {
		return "", fmt.Errorf("创建解析任务失败: %w", err)
	}
	if job.JobID == "" {
		return "", fmt.Errorf("任务响应缺少 job_id")
	}
	jobID := job.JobID
	for polls := 0; ; polls++ {
		switch job.Status {
		case "completed":
			if len(job.Files) != 1 || job.Files[0].Status != "completed" {
				return "", fmt.Errorf("任务完成但文件结果不完整")
			}
			id := job.Files[0].OutputFiles["markdown"].FileID
			if id == "" {
				return "", fmt.Errorf("任务结果缺少 Markdown 产物")
			}
			body, err := s.minerU4Request(ctx, cfg, http.MethodGet, root+"/v1/files/"+url.PathEscape(id)+"/content", nil, nil)
			if err != nil {
				return "", fmt.Errorf("下载 Markdown 失败: %w", err)
			}
			content := strings.TrimSpace(strings.ToValidUTF8(string(body), "�"))
			if content == "" {
				return "", fmt.Errorf("Markdown 产物内容为空")
			}
			return content, nil
		case "failed", "canceled", "partial":
			for _, file := range job.Files {
				if file.Error != nil {
					return "", fmt.Errorf("任务 %s（%s）: %s", jobID, job.Status, truncate(file.Error.Message, 200))
				}
			}
			return "", fmt.Errorf("任务 %s 结束，状态为 %s", jobID, job.Status)
		case "queued", "running":
		default:
			return "", fmt.Errorf("任务 %s 状态异常 %q", jobID, job.Status)
		}
		if polls >= 300 {
			return "", fmt.Errorf("任务 %s 查询次数超限", jobID)
		}
		if polls > 0 {
			timer := time.NewTimer(time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				return "", fmt.Errorf("等待任务 %s 结束: %w", jobID, ctx.Err())
			case <-timer.C:
			}
		}
		job = minerU4Job{}
		if err := s.minerU4JSON(ctx, cfg, http.MethodGet, root+"/v1/parse/jobs/"+url.PathEscape(jobID), nil, &job); err != nil {
			return "", fmt.Errorf("查询任务 %s 失败: %w", jobID, err)
		}
		if job.JobID != jobID {
			return "", fmt.Errorf("查询返回的任务 ID 不匹配")
		}
	}
}

func (s *AttachmentRecognitionService) minerU4JSON(ctx context.Context, cfg *RecognitionConfig, method, target string, payload, result any) error {
	var reader io.Reader
	if payload != nil {
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(body)
	}
	body, err := s.minerU4Request(ctx, cfg, method, target, reader, map[string]string{"Content-Type": "application/json"})
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("服务返回无效 JSON")
	}
	return nil
}

func minerU4SameOrigin(a, b *url.URL) bool {
	port := func(u *url.URL) string {
		if p := u.Port(); p != "" {
			return p
		}
		if strings.EqualFold(u.Scheme, "https") {
			return "443"
		}
		return "80"
	}
	return strings.EqualFold(a.Scheme, b.Scheme) && strings.EqualFold(a.Hostname(), b.Hostname()) && port(a) == port(b)
}

// minerU4Request 保留部署路径前缀；API 凭证仅发往同源，重定向时逐跳移除跨源凭证。
func (s *AttachmentRecognitionService) minerU4Request(ctx context.Context, cfg *RecognitionConfig, method, target string, body io.Reader, headers map[string]string) ([]byte, error) {
	baseURL, err := url.Parse(strings.TrimRight(cfg.MinerU4Endpoint, "/") + "/")
	if err != nil || baseURL.Host == "" || (baseURL.Scheme != "http" && baseURL.Scheme != "https") || baseURL.User != nil {
		return nil, fmt.Errorf("服务地址无效")
	}
	relative, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("服务返回的地址无效")
	}
	resolved := baseURL.ResolveReference(relative)
	if (resolved.Scheme != "http" && resolved.Scheme != "https") || resolved.User != nil {
		return nil, fmt.Errorf("服务返回的地址无效")
	}
	req, err := http.NewRequestWithContext(ctx, method, resolved.String(), body)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	if minerU4SameOrigin(baseURL, resolved) && cfg.MinerU4APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.MinerU4APIKey)
	}
	client := *s.httpClient
	client.CheckRedirect = func(next *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("重定向次数超限")
		}
		if !minerU4SameOrigin(resolved, next.URL) || !minerU4SameOrigin(via[len(via)-1].URL, next.URL) {
			next.Header.Del("Authorization")
			next.Header.Del("Cookie")
			next.Header.Del("Proxy-Authorization")
		}
		if s.httpClient.CheckRedirect != nil {
			return s.httpClient.CheckRedirect(next, via)
		}
		return nil
	}
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("服务请求失败")
	}
	defer resp.Body.Close()
	limit := int64(maxCompatibilityJSONBytes)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		limit = maxExternalErrorBytes
	}
	data, err := readBoundedResponse(resp.Body, limit)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("服务返回 HTTP %d", resp.StatusCode)
	}
	return data, nil
}

func (s *AttachmentRecognitionService) testMinerU4Connection(ctx context.Context, cfg *RecognitionConfig) error {
	var health struct {
		Status string `json:"status"`
	}
	target := strings.TrimRight(cfg.MinerU4Endpoint, "/") + "/v1/health"
	if strings.TrimSpace(cfg.MinerU4Endpoint) == "" {
		return newServiceError(errcode.ErrInvalidParam, "MinerU 4.0 服务地址未配置")
	}
	if err := s.minerU4JSON(ctx, cfg, http.MethodGet, target, nil, &health); err != nil {
		return newServiceError(errcode.ErrExternal, "连接 MinerU 4.0 失败: "+err.Error())
	}
	if health.Status != "ok" {
		return newServiceError(errcode.ErrExternal, "MinerU 4.0 健康状态异常")
	}
	return nil
}
