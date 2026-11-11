package service

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
	"unicode"

	"auraoa/go-service/internal/pkg/errcode"
	pkglogger "auraoa/go-service/internal/pkg/logger"
	"auraoa/go-service/internal/pkg/oa"

	"go.uber.org/zap"
)

// MinerU 4.1 使用统一的异步任务协议：POST /v1/tasks -> (可选) PUT upload.url -> GET /v1/tasks/{task_id} -> 下载 result_url ZIP 并读取 markdown.md。
// 契约依据：MinerU 4.1 HTTP Tasks API 文档与官方客户端 api_client.py。

const (
	minerU4PollInterval    = 500 * time.Millisecond
	minerU4MaxPollAttempts = 1200
)

type minerU4CreateTaskRequest struct {
	Files         []minerU4TaskFileSpec `json:"files"`
	Tier          string                `json:"tier,omitempty"`
	OCRMode       string                `json:"ocr_mode,omitempty"`
	OutputFormats []string              `json:"output_formats,omitempty"`
}

type minerU4TaskFileSpec struct {
	FileName  string `json:"file_name"`
	Bytes     int64  `json:"bytes"`
	SHA256Sum string `json:"sha256sum"`
}

type minerU4TaskError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Param   string `json:"param,omitempty"`
}

type minerU4TaskUpload struct {
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers"`
	ExpiresAt any               `json:"expires_at,omitempty"`
}

type minerU4TaskFile struct {
	FileID    string             `json:"file_id"`
	FileName  string             `json:"file_name"`
	Status    string             `json:"status"`
	Upload    *minerU4TaskUpload `json:"upload,omitempty"`
	ResultURL string             `json:"result_url,omitempty"`
	Error     *minerU4TaskError  `json:"error,omitempty"`
}

type minerU4Task struct {
	TaskID string            `json:"task_id"`
	Status string            `json:"status"`
	Files  []minerU4TaskFile `json:"files"`
	Error  *minerU4TaskError `json:"error,omitempty"`
}

// recognizeOneViaMinerU4 使用 MinerU 4.1 Tasks API 解析单个附件，并将失败原因记录到结果与日志中。
func (s *AttachmentRecognitionService) recognizeOneViaMinerU4(ctx context.Context, cfg *RecognitionConfig, name string, raw []byte, base oa.AttachmentInfo) oa.AttachmentInfo {
	content, err := s.parseMinerU4(ctx, cfg, name, raw)
	if err != nil {
		pkglogger.Global().Warn("附件识别：MinerU 4.1 解析失败",
			zap.String("fileName", safeAttachmentFileName(name)),
			zap.Int("bytes", len(raw)),
			zap.Error(err))
		base.Error = "MinerU 4.1 解析失败: " + err.Error()
	} else {
		pkglogger.Global().Info("附件识别：MinerU 4.1 单文件解析成功",
			zap.String("fileName", safeAttachmentFileName(name)),
			zap.Int("contentChars", len([]rune(content))))
		base.Content = content
	}
	return base
}

// parseMinerU4 执行 MinerU 4.1 异步任务提交流程并提取结果包中的 Markdown 文本。
func (s *AttachmentRecognitionService) parseMinerU4(ctx context.Context, cfg *RecognitionConfig, name string, raw []byte) (string, error) {
	if cfg == nil || strings.TrimSpace(cfg.MinerU4Endpoint) == "" {
		return "", fmt.Errorf("MinerU 4.1 服务地址未配置")
	}
	if len(raw) == 0 {
		return "", fmt.Errorf("文件内容为空")
	}

	safeName := sanitizeMinerU4FileName(name)
	tier, mode := normalizeMinerU4Options(cfg, safeName)

	timeout := defaultMinerUHTTPTimeout
	if s.httpClient != nil && s.httpClient.Timeout > 0 {
		timeout = s.httpClient.Timeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	sum := sha256.Sum256(raw)
	sha256Hex := hex.EncodeToString(sum[:])
	root := strings.TrimRight(strings.TrimSpace(cfg.MinerU4Endpoint), "/")

	pkglogger.Global().Info("附件识别：提交 MinerU 4.1 解析任务",
		zap.String("fileName", safeName),
		zap.Int("bytes", len(raw)),
		zap.String("tier", tier),
		zap.String("ocrMode", mode))

	var task minerU4Task
	createReq := minerU4CreateTaskRequest{
		Files: []minerU4TaskFileSpec{{
			FileName:  safeName,
			Bytes:     int64(len(raw)),
			SHA256Sum: sha256Hex,
		}},
		Tier:          tier,
		OCRMode:       mode,
		OutputFormats: []string{"default"},
	}
	if err := s.minerU4JSON(ctx, cfg, http.MethodPost, root+"/v1/tasks", createReq, &task); err != nil {
		return "", fmt.Errorf("创建解析任务失败: %w", err)
	}
	taskID := strings.TrimSpace(task.TaskID)
	if taskID == "" {
		return "", fmt.Errorf("创建解析任务响应缺少 task_id")
	}

	fileItem, err := inspectMinerU4TaskState(task, safeName)
	if err != nil {
		return "", err
	}

	needUpload := fileItem.Status == "waiting-file"
	pkglogger.Global().Info("附件识别：MinerU 4.1 任务已创建",
		zap.String("fileName", safeName),
		zap.String("taskID", taskID),
		zap.String("taskStatus", task.Status),
		zap.String("fileStatus", fileItem.Status),
		zap.Bool("needUpload", needUpload))

	uploadAccepted := false
	if needUpload {
		if err := s.uploadMinerU4File(ctx, cfg, fileItem.Upload, raw); err != nil {
			return "", err
		}
		uploadAccepted = true
	}

	resultURL := strings.TrimSpace(fileItem.ResultURL)
	if fileItem.Status != "completed" {
		resultURL, err = s.pollMinerU4TaskResult(ctx, cfg, root, taskID, safeName, raw, &uploadAccepted)
		if err != nil {
			return "", err
		}
	}
	if resultURL == "" {
		return "", fmt.Errorf("MinerU 4.1 任务完成但缺少结果下载地址")
	}

	zipBytes, err := s.minerU4Request(ctx, cfg, http.MethodGet, resultURL, nil, nil)
	if err != nil {
		return "", fmt.Errorf("下载解析结果失败: %w", err)
	}
	markdown, err := extractMinerU4Markdown(zipBytes)
	if err != nil {
		return "", err
	}
	return markdown, nil
}

func normalizeMinerU4Options(cfg *RecognitionConfig, name string) (string, string) {
	tier := strings.TrimSpace(cfg.MinerU4Tier)
	switch tier {
	case "flash", "basic", "standard", "advanced":
	default:
		tier = "standard"
	}
	mode := strings.TrimSpace(cfg.MinerU4OCRMode)
	switch mode {
	case "auto", "txt", "ocr":
	default:
		mode = "auto"
	}
	// 非 PDF 的办公文档与 OFD 走 MinerU 4.1 flash 原生直转档位。
	if ext := extractFileExt(name); isType(documentParserTypes, ext) && ext != "pdf" {
		tier = "flash"
	}
	return tier, mode
}

// sanitizeMinerU4FileName 将文件名规范化为满足 MinerU 4.1 校验规则的纯文件名。
func sanitizeMinerU4FileName(name string) string {
	base := safeAttachmentFileName(name)
	var b strings.Builder
	b.Grow(len(base))
	for _, r := range base {
		switch {
		case r == '/' || r == '\\' || r == ':' || unicode.IsControl(r):
			b.WriteByte('_')
		default:
			b.WriteRune(r)
		}
	}
	cleaned := strings.TrimSpace(b.String())
	if cleaned == "" || cleaned == "." || cleaned == ".." {
		return "attachment"
	}
	for strings.Contains(cleaned, "..") {
		cleaned = strings.ReplaceAll(cleaned, "..", "_")
	}
	// MinerU 4.1 限制 file_name 最长 1024 字符，截断时尽量保留扩展名。
	runes := []rune(cleaned)
	if len(runes) > 255 {
		ext := path.Ext(cleaned)
		extRunes := []rune(ext)
		if len(extRunes) > 0 && len(extRunes) < 32 {
			stemLen := 255 - len(extRunes)
			cleaned = string(runes[:stemLen]) + ext
		} else {
			cleaned = string(runes[:255])
		}
	}
	return cleaned
}

func (s *AttachmentRecognitionService) uploadMinerU4File(ctx context.Context, cfg *RecognitionConfig, upload *minerU4TaskUpload, raw []byte) error {
	if upload == nil || strings.TrimSpace(upload.URL) == "" {
		return fmt.Errorf("MinerU 4.1 未返回文件上传地址")
	}
	if _, err := s.minerU4Request(ctx, cfg, http.MethodPut, upload.URL, bytes.NewReader(raw), upload.Headers); err != nil {
		return fmt.Errorf("上传文件失败: %w", err)
	}
	return nil
}

func (s *AttachmentRecognitionService) pollMinerU4TaskResult(ctx context.Context, cfg *RecognitionConfig, root, taskID, safeName string, raw []byte, uploadAccepted *bool) (string, error) {
	pollURL := root + "/v1/tasks/" + url.PathEscape(taskID)
	for attempt := 0; attempt < minerU4MaxPollAttempts; attempt++ {
		if err := sleepMinerU4Poll(ctx, minerU4PollInterval); err != nil {
			return "", err
		}
		var task minerU4Task
		if err := s.minerU4JSON(ctx, cfg, http.MethodGet, pollURL, nil, &task); err != nil {
			return "", fmt.Errorf("查询任务状态失败: %w", err)
		}
		if task.TaskID != "" && task.TaskID != taskID {
			return "", fmt.Errorf("查询返回的任务 ID 不匹配")
		}
		fileItem, err := inspectMinerU4TaskState(task, safeName)
		if err != nil {
			return "", err
		}
		switch fileItem.Status {
		case "completed":
			return strings.TrimSpace(fileItem.ResultURL), nil
		case "waiting-file":
			if !*uploadAccepted && fileItem.Upload != nil && strings.TrimSpace(fileItem.Upload.URL) != "" {
				if err := s.uploadMinerU4File(ctx, cfg, fileItem.Upload, raw); err != nil {
					return "", err
				}
				*uploadAccepted = true
			}
		case "pending", "running":
		default:
			return "", fmt.Errorf("未知文件任务状态: %s", fileItem.Status)
		}
	}
	return "", fmt.Errorf("任务轮询超过最大次数限制")
}

func inspectMinerU4TaskState(task minerU4Task, safeName string) (*minerU4TaskFile, error) {
	if task.Status == "failed" || task.Status == "canceled" {
		if len(task.Files) > 0 && task.Files[0].Error != nil {
			return nil, fmt.Errorf("解析任务失败: %s", formatMinerU4TaskError(task.Files[0].Error))
		}
		return nil, fmt.Errorf("解析任务%s: %s", minerU4TerminalLabel(task.Status), formatMinerU4TaskError(task.Error))
	}
	if len(task.Files) == 0 {
		return nil, fmt.Errorf("MinerU 4.1 响应缺少文件状态")
	}
	selected := &task.Files[0]
	for i := range task.Files {
		if task.Files[i].FileName == safeName {
			selected = &task.Files[i]
			break
		}
	}
	if selected.Status == "failed" || selected.Status == "canceled" {
		errObj := selected.Error
		if errObj == nil {
			errObj = task.Error
		}
		return nil, fmt.Errorf("文件解析%s: %s", minerU4TerminalLabel(selected.Status), formatMinerU4TaskError(errObj))
	}
	return selected, nil
}

func minerU4TerminalLabel(status string) string {
	if status == "canceled" {
		return "已取消"
	}
	return "失败"
}

func formatMinerU4TaskError(e *minerU4TaskError) string {
	if e == nil {
		return "未返回详细错误信息"
	}
	msg := strings.TrimSpace(e.Message)
	code := strings.TrimSpace(e.Code)
	param := strings.TrimSpace(e.Param)
	switch {
	case code != "" && msg != "" && param != "":
		return fmt.Sprintf("[%s] %s (param=%s)", code, msg, param)
	case code != "" && msg != "":
		return fmt.Sprintf("[%s] %s", code, msg)
	case msg != "":
		return msg
	case code != "":
		return code
	default:
		return "未返回详细错误信息"
	}
}

func sleepMinerU4Poll(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// extractMinerU4Markdown 从 MinerU 4.1 结果 ZIP 压缩包中读取 markdown.md（或首个 .md 文件）。
func extractMinerU4Markdown(zipBytes []byte) (string, error) {
	if len(zipBytes) == 0 {
		return "", fmt.Errorf("解析结果压缩包为空")
	}
	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return "", fmt.Errorf("解析结果压缩包失败: %w", err)
	}

	var target *zip.File
	var fallback *zip.File
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		normalized := strings.ReplaceAll(f.Name, "\\", "/")
		if strings.HasPrefix(normalized, "__MACOSX/") {
			continue
		}
		base := strings.ToLower(path.Base(normalized))
		if base == "markdown.md" {
			target = f
			break
		}
		if fallback == nil && strings.HasSuffix(base, ".md") {
			fallback = f
		}
	}
	if target == nil {
		target = fallback
	}
	if target == nil {
		return "", fmt.Errorf("结果压缩包中缺少 markdown.md")
	}

	rc, err := target.Open()
	if err != nil {
		return "", fmt.Errorf("读取结果文件失败: %w", err)
	}
	defer func() { _ = rc.Close() }()

	rawMD, err := readBoundedResponse(rc, maxCompatibilityJSONBytes)
	if err != nil {
		return "", fmt.Errorf("读取 Markdown 结果失败: %w", err)
	}
	content := strings.TrimSpace(strings.ToValidUTF8(string(rawMD), ""))
	if content == "" {
		return "", fmt.Errorf("MinerU 4.1 未返回有效内容")
	}
	return content, nil
}

func (s *AttachmentRecognitionService) minerU4JSON(ctx context.Context, cfg *RecognitionConfig, method, target string, payload, result any) error {
	var reader io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("序列化请求失败: %w", err)
		}
		reader = bytes.NewReader(raw)
	}
	body, err := s.minerU4Request(ctx, cfg, method, target, reader, map[string]string{"Content-Type": "application/json"})
	if err != nil {
		return err
	}
	if result != nil {
		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("服务返回无效 JSON")
		}
	}
	return nil
}

func minerU4SameOrigin(a, b *url.URL) bool {
	if a == nil || b == nil {
		return false
	}
	return strings.EqualFold(a.Scheme, b.Scheme) && strings.EqualFold(a.Host, b.Host)
}

// minerU4Request 保留部署路径前缀；API 凭证仅发往同源，重定向时逐跳移除跨源凭证。
func (s *AttachmentRecognitionService) minerU4Request(ctx context.Context, cfg *RecognitionConfig, method, target string, body io.Reader, headers map[string]string) ([]byte, error) {
	baseURL, err := url.Parse(strings.TrimRight(strings.TrimSpace(cfg.MinerU4Endpoint), "/") + "/")
	if err != nil || !baseURL.IsAbs() || (baseURL.Scheme != "http" && baseURL.Scheme != "https") {
		return nil, fmt.Errorf("MinerU 4.1 服务地址无效")
	}
	parsedTarget, err := url.Parse(strings.TrimSpace(target))
	if err != nil {
		return nil, fmt.Errorf("服务返回的地址无效")
	}
	resolved := baseURL.ResolveReference(parsedTarget)
	if !resolved.IsAbs() || (resolved.Scheme != "http" && resolved.Scheme != "https") {
		return nil, fmt.Errorf("服务返回的地址无效")
	}

	req, err := http.NewRequestWithContext(ctx, method, resolved.String(), body)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	for k, v := range headers {
		if strings.EqualFold(k, "Host") || strings.EqualFold(k, "Content-Length") {
			continue
		}
		req.Header.Set(k, v)
	}
	if minerU4SameOrigin(baseURL, resolved) && cfg.MinerU4APIKey != "" && req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", "Bearer "+cfg.MinerU4APIKey)
	}

	client := http.Client{Timeout: defaultMinerUHTTPTimeout}
	if s.httpClient != nil {
		client.Transport = s.httpClient.Transport
		if s.httpClient.Timeout > 0 {
			client.Timeout = s.httpClient.Timeout
		}
	}
	client.CheckRedirect = func(next *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("重定向次数过多")
		}
		prev := resolved
		if len(via) > 0 {
			prev = via[len(via)-1].URL
		}
		if !minerU4SameOrigin(baseURL, next.URL) || !minerU4SameOrigin(resolved, next.URL) || !minerU4SameOrigin(prev, next.URL) {
			next.Header.Del("Authorization")
		}
		return nil
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 MinerU 4.1 服务失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := readBoundedResponse(resp.Body, int64(maxCompatibilityJSONBytes))
	if err != nil {
		return nil, fmt.Errorf("读取服务响应失败: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, formatMinerU4HTTPError(resp.StatusCode, data)
	}
	return data, nil
}

func formatMinerU4HTTPError(statusCode int, data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return fmt.Errorf("服务返回 HTTP %d", statusCode)
	}
	var errEnv struct {
		Error   *minerU4TaskError `json:"error"`
		Detail  any               `json:"detail"`
		MsgCode any               `json:"msgCode"`
		Msg     string            `json:"msg"`
	}
	if json.Unmarshal(data, &errEnv) == nil {
		if errEnv.Error != nil && (errEnv.Error.Code != "" || errEnv.Error.Message != "") {
			return fmt.Errorf("服务返回 HTTP %d: %s", statusCode, formatMinerU4TaskError(errEnv.Error))
		}
		if detailStr, ok := errEnv.Detail.(string); ok && strings.TrimSpace(detailStr) != "" {
			return fmt.Errorf("服务返回 HTTP %d: %s", statusCode, truncateText(strings.TrimSpace(detailStr), 200))
		}
		if strings.TrimSpace(errEnv.Msg) != "" {
			return fmt.Errorf("服务返回 HTTP %d: %s", statusCode, truncateText(strings.TrimSpace(errEnv.Msg), 200))
		}
	}
	return fmt.Errorf("服务返回 HTTP %d: %s", statusCode, truncateText(trimmed, 200))
}

func truncateText(s string, maxRunes int) string {
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return string(r[:maxRunes]) + "..."
}

// testMinerU4Connection 检查 MinerU 4.1 健康状态端点是否可用，并在识别到 4.0 版本时给出明确升级提示。
func (s *AttachmentRecognitionService) testMinerU4Connection(ctx context.Context, cfg *RecognitionConfig) error {
	if cfg == nil || strings.TrimSpace(cfg.MinerU4Endpoint) == "" {
		return newServiceError(errcode.ErrInvalidParam, "MinerU 4.1 服务地址未配置")
	}
	var health struct {
		Status  string `json:"status"`
		Version string `json:"version"`
	}
	target := strings.TrimRight(strings.TrimSpace(cfg.MinerU4Endpoint), "/") + "/v1/health"
	if err := s.minerU4JSON(ctx, cfg, http.MethodGet, target, nil, &health); err != nil {
		return newServiceError(errcode.ErrExternal, "连接 MinerU 4.1 失败: "+err.Error())
	}
	if health.Status != "ok" {
		return newServiceError(errcode.ErrExternal, "MinerU 4.1 健康状态异常")
	}
	if v := strings.TrimSpace(health.Version); v != "" && (v == "4.0" || strings.HasPrefix(v, "4.0.")) {
		return newServiceError(errcode.ErrExternal, fmt.Sprintf("检测到 MinerU 服务版本为 %s，当前已升级为 MinerU 4.1 Tasks 协议，请升级 MinerU 服务端", v))
	}
	return nil
}
