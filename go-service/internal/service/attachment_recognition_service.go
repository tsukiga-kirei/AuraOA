package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"auraoa/go-service/internal/pkg/apptime"
	"auraoa/go-service/internal/pkg/crypto"
	"auraoa/go-service/internal/pkg/errcode"
	pkglogger "auraoa/go-service/internal/pkg/logger"
	"auraoa/go-service/internal/pkg/oa"
	"auraoa/go-service/internal/repository"
)

type systemConfigReader interface {
	FindByKey(key string) (string, error)
}

// AttachmentRecognitionService 附件识别服务，负责按文件类型选择内置解析、代码文档解析或 MinerU。
//
// 详见 docs/oa-configurations/01-attachment-recognition.md。
type AttachmentRecognitionService struct {
	configRepo systemConfigReader
	httpClient *http.Client
}

const defaultMinerUHTTPTimeout = 300 * time.Second

const (
	attachmentAIContentLimitBytes     = "bytes"
	attachmentAIContentLimitUnlimited = "unlimited"
	defaultAttachmentAIContentBytes   = 10000
)

// NewAttachmentRecognitionService 创建附件识别服务实例。
func NewAttachmentRecognitionService(configRepo *repository.SystemConfigRepo, timeout ...time.Duration) *AttachmentRecognitionService {
	httpTimeout := defaultMinerUHTTPTimeout
	if len(timeout) > 0 && timeout[0] > 0 {
		httpTimeout = timeout[0]
	}
	return &AttachmentRecognitionService{
		configRepo: configRepo,
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
	}
}

// RecognitionConfig 附件识别运行时配置（来自 system_configs 表的 attachment.* key）。
type RecognitionConfig struct {
	Enabled            bool
	MaxFileSizeMB      int
	SupportedTypes     []string
	AIContentLimitMode string
	AIContentMaxBytes  int

	// 单流程附件数量上限控制（环境变量 + 系统配置）
	EnableCountLimit   bool
	MaxAttachmentCount int

	// 文档内容解析服务（配置键为兼容历史部署保留 compat 前缀）
	CompatEndpoint        string
	CompatAPIKey          string
	DocumentParserTypes   []string
	LegacyOfficeEnabled   bool
	OFDEnabled            bool
	VisualFallbackEnabled bool

	// OCR 提供商（mineru / mineru4 / aliyun）
	OCRProvider string

	// 阿里云 OCR（attachment.aliyun_ocr_*）
	AliyunOCREndpoint        string
	AliyunOCRAccessKeyID     string
	AliyunOCRAccessKeySecret string
	AliyunOCRType            string

	// MinerU 4.1（独立配置，使用统一 Tasks 协议，不沿用旧版 backend 与语言参数）
	MinerU4Endpoint string
	MinerU4APIKey   string
	MinerU4Tier     string
	MinerU4OCRMode  string

	// MinerU
	MinerUEndpoint      string
	MinerUAPIKey        string
	MinerUBackend       string // pipeline / vlm-* / hybrid-*
	MinerUEnableFormula bool
	MinerUEnableTable   bool
	MinerUParseMethod   string // auto / txt / ocr
	MinerULanguage      string
}

var allowedMinerUParseMethods = map[string]struct{}{
	"auto": {},
	"txt":  {},
	"ocr":  {},
}

func normalizeMinerUParseMethod(method string, legacyOCREnabled bool) string {
	method = strings.ToLower(strings.TrimSpace(method))
	if _, ok := allowedMinerUParseMethods[method]; ok {
		return method
	}
	if legacyOCREnabled {
		return "ocr"
	}
	return "txt"
}

// LoadConfig 从系统配置加载附件识别配置（兼容旧版只配 endpoint 的最小配置）。
func (s *AttachmentRecognitionService) LoadConfig() (*RecognitionConfig, error) {
	cfg := &RecognitionConfig{
		Enabled:                  false,
		MaxFileSizeMB:            10,
		SupportedTypes:           []string{"pdf", "png", "jpg", "jpeg", "bmp", "gif", "tiff", "webp", "txt", "csv", "md", "docx", "xlsx", "pptx", "doc", "xls", "ppt", "ofd"},
		AIContentLimitMode:       attachmentAIContentLimitBytes,
		AIContentMaxBytes:        defaultAttachmentAIContentBytes,
		OCRProvider:              "mineru",
		MinerU4Tier:              "standard",
		MinerU4OCRMode:           "auto",
		AliyunOCREndpoint:        "ocr-api.cn-hangzhou.aliyuncs.com",
		AliyunOCRAccessKeyID:     "",
		AliyunOCRAccessKeySecret: "",
		AliyunOCRType:            "General",
		CompatEndpoint:           "http://document-parser:8090",
		DocumentParserTypes:      []string{},
		LegacyOfficeEnabled:      false,
		OFDEnabled:               false,
		VisualFallbackEnabled:    true,
		MinerUBackend:            "pipeline",
		MinerUEnableFormula:      true,
		MinerUEnableTable:        true,
		MinerUParseMethod:        "ocr",
		MinerULanguage:           "ch",
	}

	read := func(key string) string {
		val, err := s.configRepo.FindByKey(key)
		if err != nil {
			return ""
		}
		return val
	}
	readBool := func(key string, def bool) bool {
		val := read(key)
		if val == "" {
			return def
		}
		return val == "true" || val == "1"
	}

	cfg.Enabled = readBool("attachment.recognition_enabled", false)
	if mode := strings.ToLower(strings.TrimSpace(read("attachment.ai_content_limit_mode"))); mode == attachmentAIContentLimitUnlimited {
		cfg.AIContentLimitMode = attachmentAIContentLimitUnlimited
	}
	if v := read("attachment.ai_content_max_bytes"); v != "" {
		if maxBytes, parseErr := strconv.Atoi(v); parseErr == nil && maxBytes > 0 {
			cfg.AIContentMaxBytes = maxBytes
		}
	}

	if v, err := s.configRepo.FindByKey("attachment.compat_endpoint"); err == nil {
		cfg.CompatEndpoint = v
	}
	cfg.CompatAPIKey = read("attachment.compat_api_key")
	cfg.LegacyOfficeEnabled = readBool("attachment.legacy_office_enabled", false)
	cfg.OFDEnabled = readBool("attachment.ofd_enabled", false)
	cfg.VisualFallbackEnabled = readBool("attachment.visual_fallback_enabled", true)
	if value, err := s.configRepo.FindByKey("attachment.document_parser_types"); err == nil {
		cfg.DocumentParserTypes = splitNormalizedTypes(value, documentParserTypes)
	} else {
		// 兼容尚未执行新迁移的部署：沿用旧版 Office / OFD 开关，不改变既有路由。
		if cfg.LegacyOfficeEnabled {
			cfg.DocumentParserTypes = append(cfg.DocumentParserTypes, "doc", "xls", "ppt")
		}
		if cfg.OFDEnabled {
			cfg.DocumentParserTypes = append(cfg.DocumentParserTypes, "ofd")
		}
	}

	if p := strings.ToLower(strings.TrimSpace(read("attachment.ocr_provider"))); p != "" {
		cfg.OCRProvider = p
	}
	if ep := strings.TrimSpace(read("attachment.aliyun_ocr_endpoint")); ep != "" {
		cfg.AliyunOCREndpoint = ep
	}
	akRaw := strings.TrimSpace(read("attachment.aliyun_ocr_access_key_id"))
	if akRaw != "" {
		if decAK, err := crypto.Decrypt(akRaw); err == nil {
			cfg.AliyunOCRAccessKeyID = decAK
		} else {
			cfg.AliyunOCRAccessKeyID = akRaw
		}
	}
	skRaw := strings.TrimSpace(read("attachment.aliyun_ocr_access_key_secret"))
	if skRaw != "" {
		if decSK, err := crypto.Decrypt(skRaw); err == nil {
			cfg.AliyunOCRAccessKeySecret = decSK
		} else {
			cfg.AliyunOCRAccessKeySecret = skRaw
		}
	}
	if t := strings.TrimSpace(read("attachment.aliyun_ocr_type")); t != "" {
		cfg.AliyunOCRType = t
	}

	cfg.MinerU4Endpoint = strings.TrimSpace(read("attachment.mineru4_endpoint"))
	cfg.MinerU4APIKey = read("attachment.mineru4_api_key")
	if v := read("attachment.mineru4_tier"); v != "" {
		cfg.MinerU4Tier = v
	}
	if v := read("attachment.mineru4_ocr_mode"); v != "" {
		cfg.MinerU4OCRMode = v
	}

	cfg.MinerUEndpoint = read("attachment.mineru_endpoint")
	cfg.MinerUAPIKey = read("attachment.mineru_api_key")
	if v := read("attachment.mineru_backend"); v != "" {
		cfg.MinerUBackend = v
	}
	cfg.MinerUEnableFormula = readBool("attachment.mineru_enable_formula", true)
	cfg.MinerUEnableTable = readBool("attachment.mineru_enable_table", true)
	cfg.MinerUParseMethod = normalizeMinerUParseMethod(
		read("attachment.mineru_parse_method"),
		readBool("attachment.mineru_enable_ocr", true),
	)
	if v := read("attachment.mineru_language"); v != "" {
		cfg.MinerULanguage = v
	}

	if v := read("attachment.max_file_size_mb"); v != "" {
		if size, parseErr := strconv.Atoi(v); parseErr == nil && size > 0 {
			cfg.MaxFileSizeMB = size
		}
	}

	if v := read("attachment.supported_types"); v != "" {
		parts := strings.Split(v, ",")
		cfg.SupportedTypes = cfg.SupportedTypes[:0]
		for _, t := range parts {
			if trimmed := strings.TrimSpace(t); trimmed != "" {
				cfg.SupportedTypes = append(cfg.SupportedTypes, strings.ToLower(trimmed))
			}
		}
	}

	// 单流程附件数量上限控制（环境变量 > 系统配置 > 默认值 10）
	cfg.EnableCountLimit = true
	if envVal := os.Getenv("AURAOA_ATTACHMENT_ENABLE_COUNT_LIMIT"); envVal != "" {
		cfg.EnableCountLimit = envVal == "true" || envVal == "1"
	} else if v := read("attachment.enable_count_limit"); v != "" {
		cfg.EnableCountLimit = v == "true" || v == "1"
	}

	cfg.MaxAttachmentCount = 10
	if envVal := os.Getenv("AURAOA_ATTACHMENT_MAX_COUNT"); envVal != "" {
		if count, parseErr := strconv.Atoi(envVal); parseErr == nil {
			cfg.MaxAttachmentCount = count
		}
	} else if v := read("attachment.max_attachment_count"); v != "" {
		if count, parseErr := strconv.Atoi(v); parseErr == nil {
			cfg.MaxAttachmentCount = count
		}
	}

	return cfg, nil
}

func splitNormalizedTypes(value string, allowed map[string]struct{}) []string {
	items := make([]string, 0)
	seen := make(map[string]struct{})
	for _, raw := range strings.Split(value, ",") {
		item := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(raw, ".")))
		if _, ok := allowed[item]; !ok {
			continue
		}
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		items = append(items, item)
	}
	return items
}

// ParseDocumentParserTypes 校正管理端提交的代码文档解析扩展名列表。
func ParseDocumentParserTypes(value string) []string {
	return splitNormalizedTypes(value, documentParserTypes)
}

// RecognizeAttachments 解析 OA 适配器传入的附件 base64 内容。
// 该方法保持附件级容错：单个文件解析失败会写入对应结果的 Error，不阻断其他附件及主业务流程。
func (s *AttachmentRecognitionService) RecognizeAttachments(
	ctx context.Context,
	files []oa.AttachmentFilePayload,
	fieldKey string,
	fieldName string,
) ([]oa.AttachmentInfo, error) {
	if len(files) == 0 {
		pkglogger.Global().Debug("附件识别：无待解析文件",
			zap.String("field", fieldKey))
		return []oa.AttachmentInfo{}, nil
	}

	cfg, err := s.LoadConfig()
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "加载附件识别配置失败")
	}
	if !cfg.Enabled {
		pkglogger.Global().Info("附件识别：功能未启用，跳过附件解析",
			zap.String("field", fieldKey),
			zap.String("fieldName", fieldName),
			zap.Int("fileCount", len(files)))
		return []oa.AttachmentInfo{}, nil
	}

	pkglogger.Global().Info("附件识别：开始按格式路由解析",
		zap.String("field", fieldKey),
		zap.String("fieldName", fieldName),
		zap.Int("fileCount", len(files)))

	// 过滤不支持的文件类型与超大文件（结果中保留为 Error 标记，让 prompt 里也能看到原因）。
	maxBytes := int64(cfg.MaxFileSizeMB) * 1024 * 1024
	supported := make(map[string]struct{}, len(cfg.SupportedTypes))
	for _, t := range cfg.SupportedTypes {
		supported[t] = struct{}{}
	}

	now := apptime.FormatRFC3339(apptime.Now())
	results := make([]oa.AttachmentInfo, 0, len(files))

	for _, f := range files {
		ext := extractFileExt(f.FileName)
		base := oa.AttachmentInfo{
			DocID:            f.DocID,
			FileName:         f.FileName,
			FileType:         ext,
			FileSize:         f.FileSize,
			FieldKey:         fieldKey,
			FieldName:        fieldName,
			ContentLimitMode: cfg.AIContentLimitMode,
			ContentMaxBytes:  cfg.AIContentMaxBytes,
			ExtractedAt:      now,
		}
		if cfg.EnableCountLimit && cfg.MaxAttachmentCount > 0 && len(results) >= cfg.MaxAttachmentCount {
			base.Error = fmt.Sprintf("已达到单流程附件识别上限（%d个），跳过内容解析", cfg.MaxAttachmentCount)
			pkglogger.Global().Info("附件识别：达到单流程数量上限，跳过内容解析",
				zap.String("field", fieldKey),
				zap.String("fileName", f.FileName),
				zap.Int("maxCount", cfg.MaxAttachmentCount))
			results = append(results, base)
			continue
		}
		if ext == "zip" {
			zipInfos := s.recognizeZipArchive(ctx, cfg, f, maxBytes, supported, base, now)
			results = append(results, zipInfos...)
			continue
		}
		if _, ok := supported[ext]; !ok {
			base.Error = fmt.Sprintf("文件类型 %q 不在 supported_types 列表中，已跳过", ext)
			pkglogger.Global().Info("附件识别：跳过不支持的文件类型",
				zap.String("field", fieldKey),
				zap.String("fileName", f.FileName),
				zap.String("fileType", ext))
			results = append(results, base)
			continue
		}
		if maxBytes > 0 && f.FileSize > maxBytes {
			base.Error = fmt.Sprintf("文件大小 %d 字节超过限制 %d MB，已跳过", f.FileSize, cfg.MaxFileSizeMB)
			pkglogger.Global().Info("附件识别：跳过超大文件",
				zap.String("field", fieldKey),
				zap.String("fileName", f.FileName),
				zap.Int64("fileSize", f.FileSize),
				zap.Int("maxSizeMB", cfg.MaxFileSizeMB))
			results = append(results, base)
			continue
		}
		raw, decodeErr := decodeAttachmentData(f.FileData, maxBytes)
		if decodeErr != nil {
			base.Error = decodeErr.Error()
			results = append(results, base)
			continue
		}
		if f.FileSize <= 0 {
			f.FileSize = int64(len(raw))
			base.FileSize = f.FileSize
		}
		results = append(results, s.recognizeAttachment(ctx, cfg, f, raw, base))
	}

	var withContent, withError int
	for _, r := range results {
		if r.Error != "" {
			withError++
		} else if r.Content != "" {
			withContent++
		}
	}
	pkglogger.Global().Info("附件识别：字段解析结束",
		zap.String("field", fieldKey),
		zap.Int("resultCount", len(results)),
		zap.Int("withContent", withContent),
		zap.Int("withError", withError))
	return results, nil
}

// recognizeViaMinerU 把文件 base64 上传到 MinerU 并取回解析后的文本。
//
// MinerU 自建服务使用官方 mineru-api 的同步接口：
// POST /file_parse，multipart/form-data 上传文件。
// 请求字段由 mineru_backend / mineru_enable_* / mineru_language 控制。
func (s *AttachmentRecognitionService) recognizeViaMinerU(
	ctx context.Context,
	cfg *RecognitionConfig,
	files []oa.AttachmentFilePayload,
	fieldKey string,
	fieldName string,
) ([]oa.AttachmentInfo, error) {
	if len(files) == 0 {
		return []oa.AttachmentInfo{}, nil
	}
	if cfg.MinerUEndpoint == "" {
		return nil, newServiceError(errcode.ErrInvalidParam, "MinerU 服务地址未配置 (attachment.mineru_endpoint)")
	}

	parseURL := strings.TrimRight(cfg.MinerUEndpoint, "/") + "/file_parse"
	now := apptime.FormatRFC3339(apptime.Now())
	out := make([]oa.AttachmentInfo, 0, len(files))

	for _, file := range files {
		fileType := extractFileExt(file.FileName)
		base := oa.AttachmentInfo{
			DocID:            file.DocID,
			FileName:         file.FileName,
			FileType:         fileType,
			FileSize:         file.FileSize,
			FieldKey:         fieldKey,
			FieldName:        fieldName,
			ContentLimitMode: cfg.AIContentLimitMode,
			ContentMaxBytes:  cfg.AIContentMaxBytes,
			ExtractedAt:      now,
		}

		rawBytes, err := base64.StdEncoding.DecodeString(file.FileData)
		if err != nil {
			base.Error = "解码附件内容失败: " + err.Error()
			out = append(out, base)
			continue
		}

		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		fields := map[string]string{
			"return_md":           "true",
			"return_images":       "false",
			"table_enable":        fmt.Sprintf("%v", cfg.MinerUEnableTable),
			"formula_enable":      fmt.Sprintf("%v", cfg.MinerUEnableFormula),
			"parse_method":        cfg.MinerUParseMethod,
			"start_page_id":       "0",
			"end_page_id":         "99999",
			"backend":             cfg.MinerUBackend,
			"response_format_zip": "false",
			"return_middle_json":  "false",
			"return_model_output": "false",
			"return_content_list": "false",
		}
		if cfg.MinerULanguage != "" {
			fields["lang_list"] = cfg.MinerULanguage
		}
		formBuildFailed := false
		for k, v := range fields {
			if err := writer.WriteField(k, v); err != nil {
				base.Error = "构建表单字段失败: " + err.Error()
				out = append(out, base)
				formBuildFailed = true
				break
			}
		}
		if formBuildFailed {
			continue
		}
		part, err := writer.CreateFormFile("files", safeAttachmentFileName(file.FileName))
		if err != nil {
			base.Error = "构建文件表单失败: " + err.Error()
			out = append(out, base)
			continue
		}
		if _, err := part.Write(rawBytes); err != nil {
			base.Error = "写入附件内容失败: " + err.Error()
			out = append(out, base)
			continue
		}
		if err := writer.Close(); err != nil {
			base.Error = "关闭表单失败: " + err.Error()
			out = append(out, base)
			continue
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, parseURL, &body)
		if err != nil {
			base.Error = "创建请求失败: " + err.Error()
			out = append(out, base)
			continue
		}
		req.Header.Set("Content-Type", writer.FormDataContentType())
		if cfg.MinerUAPIKey != "" {
			req.Header.Set("Authorization", "Bearer "+cfg.MinerUAPIKey)
		}

		pkglogger.Global().Info("附件识别：请求 MinerU",
			zap.String("field", fieldKey),
			zap.String("fileName", file.FileName),
			zap.String("docId", file.DocID),
			zap.Int64("fileSize", file.FileSize))

		resp, err := s.httpClient.Do(req)
		if err != nil {
			pkglogger.Global().Error("附件识别：调用 MinerU 失败",
				zap.Error(err),
				zap.String("field", fieldKey),
				zap.String("fileName", file.FileName))
			base.Error = "调用 MinerU 失败: " + err.Error()
			out = append(out, base)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			pkglogger.Global().Error("附件识别：MinerU HTTP 错误",
				zap.Int("status", resp.StatusCode),
				zap.Int("responseLength", len(respBody)),
				zap.String("field", fieldKey),
				zap.String("fileName", file.FileName))
			base.Error = fmt.Sprintf("MinerU 返回 HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 200))
			out = append(out, base)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			base.Error = "读取响应失败: " + err.Error()
			out = append(out, base)
			continue
		}

		var result struct {
			TaskID    string          `json:"task_id"`
			Status    string          `json:"status"`
			ResultURL string          `json:"result_url"`
			Results   json.RawMessage `json:"results"`
		}
		if err := json.Unmarshal(respBody, &result); err != nil {
			pkglogger.Global().Error("附件识别：解析 MinerU 响应失败",
				zap.Error(err),
				zap.String("field", fieldKey),
				zap.String("fileName", file.FileName),
				zap.Int("responseLength", len(respBody)))
			base.Error = "解析响应失败: " + err.Error()
			out = append(out, base)
			continue
		}

		content := extractMinerUMarkdown(result.Results)
		if content == "" && result.Status == "completed" && result.ResultURL != "" {
			content, err = s.fetchMinerUTaskResult(ctx, result.ResultURL, cfg.MinerUAPIKey)
			if err != nil {
				pkglogger.Global().Warn("附件识别：拉取 MinerU 任务结果失败",
					zap.Error(err),
					zap.String("field", fieldKey),
					zap.String("fileName", file.FileName),
					zap.String("taskId", result.TaskID),
					zap.String("resultURL", result.ResultURL))
				base.Error = "拉取 MinerU 任务结果失败: " + err.Error()
				out = append(out, base)
				continue
			}
		}
		if content == "" {
			pkglogger.Global().Warn("附件识别：MinerU 响应缺少 Markdown 内容",
				zap.String("field", fieldKey),
				zap.String("fileName", file.FileName),
				zap.String("taskId", result.TaskID),
				zap.Int("responseLength", len(respBody)))
			base.Error = "MinerU 响应缺少 Markdown 内容"
			out = append(out, base)
			continue
		}
		base.Content = content
		pkglogger.Global().Info("附件识别：MinerU 单文件解析成功",
			zap.String("field", fieldKey),
			zap.String("fileName", file.FileName),
			zap.Int("contentLength", len(content)))
		out = append(out, base)
	}

	var successCount int
	for _, item := range out {
		if item.Error == "" && item.Content != "" {
			successCount++
		}
	}
	pkglogger.Global().Info("附件识别：MinerU 批次完成",
		zap.String("field", fieldKey),
		zap.Int("total", len(files)),
		zap.Int("resultCount", len(out)),
		zap.Int("successCount", successCount))
	return out, nil
}

type mineruMarkdownResult struct {
	MDContent string `json:"md_content"`
	Content   string `json:"content"`
	Markdown  string `json:"markdown"`
}

func (r mineruMarkdownResult) text() string {
	if r.MDContent != "" {
		return r.MDContent
	}
	if r.Markdown != "" {
		return r.Markdown
	}
	return r.Content
}

type mineruInlineResults struct {
	mineruMarkdownResult
	Document mineruMarkdownResult `json:"document"`
	Files    mineruMarkdownResult `json:"files"`
}

func extractMinerUMarkdown(results json.RawMessage) string {
	if len(results) == 0 || string(results) == "null" {
		return ""
	}

	var inline mineruInlineResults
	if err := json.Unmarshal(results, &inline); err == nil {
		for _, content := range []string{
			inline.text(),
			inline.Document.text(),
			inline.Files.text(),
		} {
			if content != "" {
				return content
			}
		}
	}

	var byName map[string]mineruMarkdownResult
	if err := json.Unmarshal(results, &byName); err != nil {
		return ""
	}
	keys := make([]string, 0, len(byName))
	for key := range byName {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if content := byName[key].text(); content != "" {
			return content
		}
	}
	return ""
}

func (s *AttachmentRecognitionService) fetchMinerUTaskResult(ctx context.Context, resultURL, apiKey string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, resultURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建任务结果请求失败: %w", err)
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求任务结果失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取任务结果失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("任务结果返回 HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	var payload struct {
		Results map[string]struct {
			MDContent string `json:"md_content"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("解析任务结果失败: %w", err)
	}
	for _, item := range payload.Results {
		if item.MDContent != "" {
			return item.MDContent, nil
		}
	}
	return "", fmt.Errorf("任务结果中缺少 Markdown 内容")
}

// TestConnection 按所选版本探测 MinerU 健康接口，不做真实解析。
func (s *AttachmentRecognitionService) TestConnection(ctx context.Context) error {
	cfg, err := s.LoadConfig()
	if err != nil {
		return newServiceError(errcode.ErrDatabase, "加载附件识别配置失败")
	}
	return s.TestConnectionWithConfig(ctx, cfg)
}

// TestConnectionWithConfig 使用指定配置探测旧版 /health 或 4.0 /v1/health；
// 用于前端“未保存先测试”场景。
func (s *AttachmentRecognitionService) TestConnectionWithConfig(ctx context.Context, cfg *RecognitionConfig) error {
	if cfg == nil {
		return newServiceError(errcode.ErrInvalidParam, "测试配置为空")
	}
	if cfg.OCRProvider == "mineru4" {
		return s.testMinerU4Connection(ctx, cfg)
	}
	if cfg.MinerUEndpoint == "" {
		return newServiceError(errcode.ErrInvalidParam, "MinerU 服务地址未配置")
	}

	healthURL := strings.TrimRight(cfg.MinerUEndpoint, "/") + "/health"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return newServiceError(errcode.ErrExternal, "创建测试请求失败")
	}
	if cfg.MinerUAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.MinerUAPIKey)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return newServiceError(errcode.ErrExternal, "连接 MinerU 服务失败: "+err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return newServiceError(errcode.ErrExternal, fmt.Sprintf("MinerU /health 返回 HTTP %d", resp.StatusCode))
	}
	return nil
}

// extractFileExt 取文件扩展名（小写，不含点号）；无扩展名返回空串。
func extractFileExt(name string) string {
	idx := strings.LastIndex(name, ".")
	if idx < 0 || idx >= len(name)-1 {
		return ""
	}
	return strings.ToLower(name[idx+1:])
}

const (
	defaultMaxZipFileCount  = 50                // 单个压缩包最多提取子文件数
	defaultMaxZipTotalBytes = 100 * 1024 * 1024 // 单个压缩包解压累计最大字节数（100MB），防 Zip 炸弹
)

func (s *AttachmentRecognitionService) recognizeZipArchive(
	ctx context.Context,
	cfg *RecognitionConfig,
	f oa.AttachmentFilePayload,
	maxBytes int64,
	supported map[string]struct{},
	base oa.AttachmentInfo,
	now string,
) []oa.AttachmentInfo {
	// 1. 解码 ZIP 压缩包 Base64 数据
	raw, decodeErr := decodeAttachmentData(f.FileData, maxBytes)
	if decodeErr != nil {
		base.Error = "解码 ZIP 压缩包失败: " + decodeErr.Error()
		return []oa.AttachmentInfo{base}
	}
	if f.FileSize <= 0 {
		base.FileSize = int64(len(raw))
	}

	// 2. 打开 ZIP 归档
	zipReader, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		base.Error = "打开 ZIP 压缩包失败: " + err.Error()
		return []oa.AttachmentInfo{base}
	}

	var results []oa.AttachmentInfo
	var totalUncompressedBytes int64
	extractedCount := 0

	for _, fileEntry := range zipReader.File {
		// 跳过目录
		if fileEntry.FileInfo().IsDir() {
			continue
		}

		cleanPath := filepath.Clean(fileEntry.Name)
		// 防 Zip Slip 目录穿越
		if strings.HasPrefix(cleanPath, "..") || filepath.IsAbs(cleanPath) {
			continue
		}
		// 过滤 macOS 资源分叉和系统元数据文件
		baseName := filepath.Base(cleanPath)
		if strings.HasPrefix(cleanPath, "__MACOSX/") || strings.HasPrefix(baseName, "._") ||
			baseName == ".DS_Store" || baseName == "Thumbs.db" || strings.TrimSpace(baseName) == "" {
			continue
		}

		extractedCount++
		if extractedCount > defaultMaxZipFileCount {
			pkglogger.Global().Warn("附件识别：ZIP 压缩包内文件数超过上限，截断后续文件",
				zap.String("zipFile", f.FileName),
				zap.Int("maxCount", defaultMaxZipFileCount))
			results = append(results, oa.AttachmentInfo{
				DocID:            base.DocID,
				FileName:         cleanPath,
				FileType:         extractFileExt(cleanPath),
				FieldKey:         base.FieldKey,
				FieldName:        base.FieldName,
				DetailTable:      base.DetailTable,
				RowIndex:         base.RowIndex,
				ContentLimitMode: cfg.AIContentLimitMode,
				ContentMaxBytes:  cfg.AIContentMaxBytes,
				ExtractedAt:      now,
				Error:            fmt.Sprintf("压缩包内文件数超出限制（最多%d个），已跳过解析", defaultMaxZipFileCount),
				ArchiveFileName:  base.FileName,
			})
			break
		}

		subExt := extractFileExt(cleanPath)
		subSize := int64(fileEntry.UncompressedSize64)
		child := oa.AttachmentInfo{
			DocID:            base.DocID,
			FileName:         cleanPath,
			FileType:         subExt,
			FileSize:         subSize,
			FieldKey:         base.FieldKey,
			FieldName:        base.FieldName,
			DetailTable:      base.DetailTable,
			RowIndex:         base.RowIndex,
			ContentLimitMode: cfg.AIContentLimitMode,
			ContentMaxBytes:  cfg.AIContentMaxBytes,
			ExtractedAt:      now,
			ArchiveFileName:  base.FileName,
		}

		// 检查嵌套压缩包
		if subExt == "zip" {
			child.Error = "暂不支持嵌套压缩包二次解压"
			results = append(results, child)
			continue
		}

		// 检查是否属于支持的文件类型
		if _, ok := supported[subExt]; !ok {
			child.Error = fmt.Sprintf("文件类型 %q 不在支持列表中，已跳过内容解析", subExt)
			results = append(results, child)
			continue
		}

		// 检查解压单文件大小限制
		if maxBytes > 0 && subSize > maxBytes {
			child.Error = fmt.Sprintf("解压文件大小 %d 字节超过限制 %d MB，已跳过内容解析", subSize, cfg.MaxFileSizeMB)
			results = append(results, child)
			continue
		}

		// 检查累计解包大小（防 Zip 炸弹）
		totalUncompressedBytes += subSize
		if totalUncompressedBytes > defaultMaxZipTotalBytes {
			child.Error = "压缩包累计解压大小超出安全上限（100MB），已跳过内容解析"
			results = append(results, child)
			continue
		}

		// 读取解压流
		rc, err := fileEntry.Open()
		if err != nil {
			child.Error = "打开压缩包内文件失败: " + err.Error()
			results = append(results, child)
			continue
		}

		subLimit := maxBytes
		if subLimit <= 0 {
			subLimit = 50 * 1024 * 1024
		}
		subRaw, readErr := io.ReadAll(io.LimitReader(rc, subLimit+1024))
		rc.Close()
		if readErr != nil {
			child.Error = "读取解压文件失败: " + readErr.Error()
			results = append(results, child)
			continue
		}
		if int64(len(subRaw)) > subLimit {
			child.Error = fmt.Sprintf("解压实际文件大小超过限制 %d MB，已跳过内容解析", cfg.MaxFileSizeMB)
			results = append(results, child)
			continue
		}
		if child.FileSize <= 0 {
			child.FileSize = int64(len(subRaw))
		}

		subPayload := oa.AttachmentFilePayload{
			DocID:    base.DocID,
			FileName: cleanPath,
			FileSize: child.FileSize,
		}
		recognized := s.recognizeAttachment(ctx, cfg, subPayload, subRaw, child)
		recognized.ArchiveFileName = base.FileName
		results = append(results, recognized)
	}

	if len(results) == 0 {
		base.Error = "ZIP 压缩包为空或未包含有效文件"
		return []oa.AttachmentInfo{base}
	}

	return results
}
