package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path"
	"strings"
	"unicode/utf8"

	"go.uber.org/zap"

	"auraoa/go-service/internal/pkg/errcode"
	pkglogger "auraoa/go-service/internal/pkg/logger"
	"auraoa/go-service/internal/pkg/oa"
)

const (
	maxCompatibilityJSONBytes = 32 * 1024 * 1024
	maxConvertedPDFBytes      = 64 * 1024 * 1024
	maxExternalErrorBytes     = 64 * 1024
)

var (
	builtinTextTypes = map[string]struct{}{
		"txt": {},
		"csv": {},
		"md":  {},
	}
	minerUTypes = map[string]struct{}{
		"pdf":  {},
		"png":  {},
		"jpg":  {},
		"jpeg": {},
		"bmp":  {},
		"gif":  {},
		"tiff": {},
		"webp": {},
		"docx": {},
		"xlsx": {},
		"pptx": {},
	}
	legacyOfficeTypes = map[string]struct{}{
		"doc": {},
		"xls": {},
		"ppt": {},
	}
)

type compatibilityParseResult struct {
	Parser           string   `json:"parser"`
	FileType         string   `json:"file_type"`
	Content          string   `json:"content"`
	HasTextLayer     bool     `json:"has_text_layer"`
	FallbackRequired bool     `json:"fallback_required"`
	FallbackFormat   string   `json:"fallback_format"`
	Warnings         []string `json:"warnings"`
}

// effectiveSupportedTypes 返回当前已启用且已配置解析器实际可处理的白名单类型。
// MinerU、旧版 Office 与 OFD 类型都会按各自端点和功能开关过滤。
func effectiveSupportedTypes(cfg *RecognitionConfig) []string {
	if cfg == nil {
		return nil
	}
	types := make([]string, 0, len(cfg.SupportedTypes))
	seen := make(map[string]struct{}, len(cfg.SupportedTypes))
	for _, item := range cfg.SupportedTypes {
		fileType := strings.ToLower(strings.TrimSpace(item))
		if fileType == "" {
			continue
		}
		if _, exists := seen[fileType]; exists {
			continue
		}

		enabled := false
		switch {
		case isType(builtinTextTypes, fileType):
			enabled = true
		case isType(minerUTypes, fileType):
			enabled = strings.TrimSpace(cfg.MinerUEndpoint) != ""
		case isType(legacyOfficeTypes, fileType):
			enabled = cfg.LegacyOfficeEnabled && strings.TrimSpace(cfg.CompatEndpoint) != ""
		case fileType == "ofd":
			enabled = cfg.OFDEnabled && strings.TrimSpace(cfg.CompatEndpoint) != ""
		}
		if enabled {
			types = append(types, fileType)
			seen[fileType] = struct{}{}
		}
	}
	return types
}

func (s *AttachmentRecognitionService) recognizeAttachment(
	ctx context.Context,
	cfg *RecognitionConfig,
	file oa.AttachmentFilePayload,
	raw []byte,
	base oa.AttachmentInfo,
) oa.AttachmentInfo {
	fileType := extractFileExt(file.FileName)
	switch {
	case isType(builtinTextTypes, fileType):
		content, err := readBuiltinText(raw)
		if err != nil {
			base.Error = err.Error()
			return base
		}
		base.Content = content
		return base

	case isType(minerUTypes, fileType):
		return s.recognizeOneViaMinerU(ctx, cfg, file, base)

	case isType(legacyOfficeTypes, fileType):
		if !cfg.LegacyOfficeEnabled {
			base.Error = "旧版 Office 解析未启用"
			return base
		}
		return s.recognizeOneViaCompatibility(ctx, cfg, file, raw, base, false)

	case fileType == "ofd":
		if !cfg.OFDEnabled {
			base.Error = "OFD 解析未启用"
			return base
		}
		return s.recognizeOneViaCompatibility(ctx, cfg, file, raw, base, true)

	default:
		base.Error = fmt.Sprintf("文件类型 %q 没有可用的解析器", fileType)
		return base
	}
}

func (s *AttachmentRecognitionService) recognizeOneViaMinerU(
	ctx context.Context,
	cfg *RecognitionConfig,
	file oa.AttachmentFilePayload,
	base oa.AttachmentInfo,
) oa.AttachmentInfo {
	parsed, err := s.recognizeViaMinerU(ctx, cfg, []oa.AttachmentFilePayload{file}, base.FieldKey, base.FieldName)
	if err != nil {
		base.Error = serviceErrorMessage(err)
		return base
	}
	if len(parsed) == 0 {
		base.Error = "MinerU 未返回附件解析结果"
		return base
	}
	return parsed[0]
}

func (s *AttachmentRecognitionService) recognizeOneViaCompatibility(
	ctx context.Context,
	cfg *RecognitionConfig,
	file oa.AttachmentFilePayload,
	raw []byte,
	base oa.AttachmentInfo,
	allowVisualFallback bool,
) oa.AttachmentInfo {
	result, err := s.parseViaCompatibility(ctx, cfg, file.FileName, raw)
	if err != nil {
		base.Error = err.Error()
		return base
	}
	content := strings.TrimSpace(result.Content)
	if len(result.Warnings) > 0 {
		pkglogger.Global().Warn("附件识别：兼容解析服务返回警告",
			zap.String("fileName", file.FileName),
			zap.String("fileType", base.FileType),
			zap.Int("warningCount", len(result.Warnings)))
	}

	if allowVisualFallback && result.FallbackRequired && cfg.VisualFallbackEnabled {
		fallbackFormat := strings.ToLower(strings.TrimSpace(result.FallbackFormat))
		if fallbackFormat != "" && fallbackFormat != "pdf" {
			if content != "" {
				pkglogger.Global().Warn("附件识别：兼容服务请求了不支持的回退格式，保留文字层结果",
					zap.String("fileName", file.FileName),
					zap.String("fallbackFormat", fallbackFormat))
				base.Content = content
				return base
			}
			base.Error = fmt.Sprintf("兼容解析服务请求了不支持的回退格式 %q", fallbackFormat)
			return base
		}

		pdf, convertErr := s.convertCompatibilityToPDF(ctx, cfg, file.FileName, raw)
		if convertErr == nil {
			fallbackFile := file
			fallbackFile.FileName = replaceFileExtension(file.FileName, "pdf")
			fallbackFile.FileSize = int64(len(pdf))
			fallbackFile.FileData = base64.StdEncoding.EncodeToString(pdf)
			fallbackBase := base
			fallbackBase.FileSize = fallbackFile.FileSize
			minerUResult := s.recognizeOneViaMinerU(ctx, cfg, fallbackFile, fallbackBase)
			if minerUResult.Error == "" && strings.TrimSpace(minerUResult.Content) != "" {
				base.Content = minerUResult.Content
				return base
			}
			fallbackMessage := strings.TrimSpace(minerUResult.Error)
			if fallbackMessage == "" {
				fallbackMessage = "MinerU 未返回文本内容"
			}
			convertErr = fmt.Errorf("%s", fallbackMessage)
		}

		if content != "" {
			pkglogger.Global().Warn("附件识别：OFD 视觉回退失败，保留文字层结果",
				zap.String("fileName", file.FileName),
				zap.String("fileType", base.FileType))
			base.Content = content
			return base
		}
		base.Error = "OFD 视觉回退失败: " + convertErr.Error()
		return base
	}

	if content == "" {
		if allowVisualFallback && result.FallbackRequired && !cfg.VisualFallbackEnabled {
			base.Error = "OFD 未提取到文字层，且视觉回退未启用"
		} else {
			base.Error = "兼容解析服务未返回文本内容"
		}
		return base
	}
	base.Content = content
	return base
}

func (s *AttachmentRecognitionService) parseViaCompatibility(
	ctx context.Context,
	cfg *RecognitionConfig,
	fileName string,
	raw []byte,
) (*compatibilityParseResult, error) {
	if strings.TrimSpace(cfg.CompatEndpoint) == "" {
		return nil, fmt.Errorf("兼容格式解析服务地址未配置")
	}
	resp, err := s.postCompatibilityFile(
		ctx,
		strings.TrimRight(cfg.CompatEndpoint, "/")+"/parse",
		cfg.CompatAPIKey,
		fileName,
		raw,
	)
	if err != nil {
		return nil, fmt.Errorf("调用兼容格式解析服务失败: %w", err)
	}
	defer resp.Body.Close()

	body, readErr := readBoundedResponse(resp.Body, maxCompatibilityJSONBytes)
	if readErr != nil {
		return nil, fmt.Errorf("读取兼容格式解析响应失败: %w", readErr)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("兼容格式解析服务返回 HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	var result compatibilityParseResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析兼容格式服务响应失败: %w", err)
	}
	return &result, nil
}

func (s *AttachmentRecognitionService) convertCompatibilityToPDF(
	ctx context.Context,
	cfg *RecognitionConfig,
	fileName string,
	raw []byte,
) ([]byte, error) {
	if strings.TrimSpace(cfg.CompatEndpoint) == "" {
		return nil, fmt.Errorf("兼容格式解析服务地址未配置")
	}
	resp, err := s.postCompatibilityFile(
		ctx,
		strings.TrimRight(cfg.CompatEndpoint, "/")+"/convert/pdf",
		cfg.CompatAPIKey,
		fileName,
		raw,
	)
	if err != nil {
		return nil, fmt.Errorf("请求 OFD 转 PDF 失败: %w", err)
	}
	defer resp.Body.Close()

	pdf, readErr := readBoundedResponse(resp.Body, maxConvertedPDFBytes)
	if readErr != nil {
		return nil, fmt.Errorf("读取转换后 PDF 失败: %w", readErr)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OFD 转 PDF 返回 HTTP %d: %s", resp.StatusCode, truncate(string(pdf), 200))
	}
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if contentType != "" && !strings.Contains(contentType, "application/pdf") {
		return nil, fmt.Errorf("OFD 转换响应类型不是 application/pdf")
	}
	if len(pdf) < 5 || !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		return nil, fmt.Errorf("OFD 转换响应不是有效的 PDF 文件")
	}
	return pdf, nil
}

func (s *AttachmentRecognitionService) postCompatibilityFile(
	ctx context.Context,
	requestURL string,
	apiKey string,
	fileName string,
	raw []byte,
) (*http.Response, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", safeAttachmentFileName(fileName))
	if err != nil {
		return nil, fmt.Errorf("构建附件表单失败: %w", err)
	}
	if _, err := part.Write(raw); err != nil {
		return nil, fmt.Errorf("写入附件表单失败: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("关闭附件表单失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, &body)
	if err != nil {
		return nil, fmt.Errorf("创建兼容解析请求失败: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	return s.httpClient.Do(req)
}

func decodeAttachmentData(encoded string, maxBytes int64) ([]byte, error) {
	reader := base64.NewDecoder(base64.StdEncoding, strings.NewReader(encoded))
	var source io.Reader = reader
	if maxBytes > 0 {
		source = io.LimitReader(reader, maxBytes+1)
	}
	raw, err := io.ReadAll(source)
	if err != nil {
		return nil, fmt.Errorf("解码附件内容失败: %w", err)
	}
	if maxBytes > 0 && int64(len(raw)) > maxBytes {
		return nil, fmt.Errorf("附件实际大小超过限制 %d MB，已跳过", maxBytes/(1024*1024))
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("附件内容为空")
	}
	return raw, nil
}

func readBuiltinText(raw []byte) (string, error) {
	raw = bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF})
	if bytes.IndexByte(raw, 0) >= 0 {
		return "", fmt.Errorf("文本附件包含二进制空字节，已拒绝读取")
	}
	if !utf8.Valid(raw) {
		return "", fmt.Errorf("文本附件不是有效的 UTF-8 编码")
	}
	content := strings.TrimSpace(string(raw))
	if content == "" {
		return "", fmt.Errorf("文本附件内容为空")
	}
	return content, nil
}

func readBoundedResponse(reader io.Reader, limit int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("响应体超过 %d 字节限制", limit)
	}
	return body, nil
}

func safeAttachmentFileName(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	name = path.Base(name)
	if name == "" || name == "." || name == "/" {
		return "attachment"
	}
	return name
}

func replaceFileExtension(name, extension string) string {
	safeName := safeAttachmentFileName(name)
	if idx := strings.LastIndex(safeName, "."); idx > 0 {
		safeName = safeName[:idx]
	}
	return safeName + "." + extension
}

func isType(types map[string]struct{}, fileType string) bool {
	_, ok := types[fileType]
	return ok
}

func serviceErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	if serviceErr, ok := err.(*ServiceError); ok {
		return serviceErr.Message
	}
	return err.Error()
}

// TestCompatibilityConnection 使用已保存配置探测兼容格式解析服务 /ready。
func (s *AttachmentRecognitionService) TestCompatibilityConnection(ctx context.Context) error {
	cfg, err := s.LoadConfig()
	if err != nil {
		return newServiceError(errcode.ErrDatabase, "加载附件识别配置失败")
	}
	return s.TestCompatibilityConnectionWithConfig(ctx, cfg)
}

// TestCompatibilityConnectionWithConfig 使用指定配置探测受鉴权的兼容格式解析服务 /ready。
// 用于系统设置页面“未保存先测试”，同时校验服务连通性与 API Key。
func (s *AttachmentRecognitionService) TestCompatibilityConnectionWithConfig(
	ctx context.Context,
	cfg *RecognitionConfig,
) error {
	if cfg == nil {
		return newServiceError(errcode.ErrInvalidParam, "测试配置为空")
	}
	if strings.TrimSpace(cfg.CompatEndpoint) == "" {
		return newServiceError(errcode.ErrInvalidParam, "兼容格式解析服务地址未配置")
	}

	readyURL := strings.TrimRight(cfg.CompatEndpoint, "/") + "/ready"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, readyURL, nil)
	if err != nil {
		return newServiceError(errcode.ErrExternal, "创建兼容解析服务测试请求失败")
	}
	if cfg.CompatAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.CompatAPIKey)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return newServiceError(errcode.ErrExternal, "连接兼容格式解析服务失败: "+err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := readBoundedResponse(resp.Body, maxExternalErrorBytes)
		return newServiceError(
			errcode.ErrExternal,
			fmt.Sprintf("兼容格式解析服务 /ready 返回 HTTP %d: %s", resp.StatusCode, truncate(string(body), 200)),
		)
	}
	return nil
}
