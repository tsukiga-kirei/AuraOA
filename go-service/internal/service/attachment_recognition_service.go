package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"oa-smart-audit/go-service/internal/pkg/errcode"
	pkglogger "oa-smart-audit/go-service/internal/pkg/logger"
	"oa-smart-audit/go-service/internal/pkg/oa"
	"oa-smart-audit/go-service/internal/repository"
)

// AttachmentRecognitionService 附件识别服务，负责：
//  1. 调用 OA 系统的自建 REST 接口获取附件文件流（base64）
//  2. 调用 MinerU 服务解析文档为文本
//
// 详见 docs/oa-configurations/01-attachment-recognition.md。
type AttachmentRecognitionService struct {
	configRepo *repository.SystemConfigRepo
	httpClient *http.Client
}

// NewAttachmentRecognitionService 创建附件识别服务实例。
func NewAttachmentRecognitionService(configRepo *repository.SystemConfigRepo) *AttachmentRecognitionService {
	return &AttachmentRecognitionService{
		configRepo: configRepo,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// RecognitionConfig 附件识别运行时配置（来自 system_configs 表的 attachment.* key）。
type RecognitionConfig struct {
	Enabled        bool
	MaxFileSizeMB  int
	SupportedTypes []string

	// MinerU
	MinerUEndpoint      string
	MinerUAPIKey        string
	MinerUBackend       string // pipeline / vlm-* / hybrid-*
	MinerUEnableFormula bool
	MinerUEnableTable   bool
	MinerUEnableOCR     bool
	MinerULanguage      string
}

// LoadConfig 从系统配置加载附件识别配置（兼容旧版只配 endpoint 的最小配置）。
func (s *AttachmentRecognitionService) LoadConfig() (*RecognitionConfig, error) {
	cfg := &RecognitionConfig{
		Enabled:             false,
		MaxFileSizeMB:       10,
		SupportedTypes:      []string{"pdf", "png", "jpg", "jpeg", "bmp", "gif", "tiff", "webp", "docx", "xlsx", "txt"},
		MinerUBackend:       "pipeline",
		MinerUEnableFormula: true,
		MinerUEnableTable:   true,
		MinerUEnableOCR:     true,
		MinerULanguage:      "ch",
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

	cfg.MinerUEndpoint = read("attachment.mineru_endpoint")
	cfg.MinerUAPIKey = read("attachment.mineru_api_key")
	if v := read("attachment.mineru_backend"); v != "" {
		cfg.MinerUBackend = v
	}
	cfg.MinerUEnableFormula = readBool("attachment.mineru_enable_formula", true)
	cfg.MinerUEnableTable = readBool("attachment.mineru_enable_table", true)
	cfg.MinerUEnableOCR = readBool("attachment.mineru_enable_ocr", true)
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

	return cfg, nil
}

// RecognizeAttachments 解析 OA 适配器传入的附件 base64 内容。
// 注意：该服务只负责 MinerU 解析，不再负责从 OA 侧拉取附件文件流。
func (s *AttachmentRecognitionService) RecognizeAttachments(
	ctx context.Context,
	files []oa.AttachmentFilePayload,
	fieldKey string,
	fieldName string,
) ([]oa.AttachmentInfo, error) {
	if len(files) == 0 {
		return []oa.AttachmentInfo{}, nil
	}

	cfg, err := s.LoadConfig()
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "加载附件识别配置失败")
	}
	if !cfg.Enabled {
		return []oa.AttachmentInfo{}, nil
	}

	// 过滤不支持的文件类型与超大文件（结果中保留为 Error 标记，让 prompt 里也能看到原因）。
	maxBytes := int64(cfg.MaxFileSizeMB) * 1024 * 1024
	supported := make(map[string]struct{}, len(cfg.SupportedTypes))
	for _, t := range cfg.SupportedTypes {
		supported[t] = struct{}{}
	}

	now := time.Now().Format(time.RFC3339)
	results := make([]oa.AttachmentInfo, 0, len(files))
	parseable := make([]oa.AttachmentFilePayload, 0, len(files))

	for _, f := range files {
		ext := extractFileExt(f.FileName)
		base := oa.AttachmentInfo{
			DocID:       f.DocID,
			FileName:    f.FileName,
			FileType:    ext,
			FileSize:    f.FileSize,
			FieldKey:    fieldKey,
			FieldName:   fieldName,
			ExtractedAt: now,
		}
		if _, ok := supported[ext]; !ok {
			base.Error = fmt.Sprintf("文件类型 %q 不在 supported_types 列表中，已跳过", ext)
			results = append(results, base)
			continue
		}
		if maxBytes > 0 && f.FileSize > maxBytes {
			base.Error = fmt.Sprintf("文件大小 %d 字节超过限制 %d MB，已跳过", f.FileSize, cfg.MaxFileSizeMB)
			results = append(results, base)
			continue
		}
		parseable = append(parseable, f)
	}

	parsed, err := s.recognizeViaMinerU(ctx, cfg, parseable, fieldKey, fieldName)
	if err != nil {
		return nil, err
	}
	results = append(results, parsed...)
	return results, nil
}

// recognizeViaMinerU 把文件 base64 上传到 MinerU 并取回解析后的文本。
//
// MinerU 请求体字段由 mineru_backend / mineru_enable_* / mineru_language 控制，
// 与 MinerU 自建端点 /api/v1/parse 的契约保持一致（详见 docs/oa-configurations/01-attachment-recognition.md）。
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

	parseURL := strings.TrimRight(cfg.MinerUEndpoint, "/") + "/api/v1/parse"
	now := time.Now().Format(time.RFC3339)
	out := make([]oa.AttachmentInfo, 0, len(files))

	for _, file := range files {
		fileType := extractFileExt(file.FileName)
		base := oa.AttachmentInfo{
			DocID:       file.DocID,
			FileName:    file.FileName,
			FileType:    fileType,
			FileSize:    file.FileSize,
			FieldKey:    fieldKey,
			FieldName:   fieldName,
			ExtractedAt: now,
		}

		body := map[string]interface{}{
			"fileName":       file.FileName,
			"fileData":       file.FileData,
			"fileType":       fileType,
			"backend":        cfg.MinerUBackend,
			"enable_formula": cfg.MinerUEnableFormula,
			"enable_table":   cfg.MinerUEnableTable,
			"enable_ocr":     cfg.MinerUEnableOCR,
			"language":       cfg.MinerULanguage,
		}
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			base.Error = "构建请求失败: " + err.Error()
			out = append(out, base)
			continue
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, parseURL, bytes.NewReader(bodyBytes))
		if err != nil {
			base.Error = "创建请求失败: " + err.Error()
			out = append(out, base)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		if cfg.MinerUAPIKey != "" {
			req.Header.Set("Authorization", "Bearer "+cfg.MinerUAPIKey)
		}

		resp, err := s.httpClient.Do(req)
		if err != nil {
			pkglogger.Global().Error("调用 MinerU 服务失败",
				zap.Error(err),
				zap.String("fileName", file.FileName))
			base.Error = "调用 MinerU 失败: " + err.Error()
			out = append(out, base)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			pkglogger.Global().Error("MinerU 服务返回错误",
				zap.Int("status", resp.StatusCode),
				zap.String("body", string(respBody)),
				zap.String("fileName", file.FileName))
			base.Error = fmt.Sprintf("MinerU 返回 HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 200))
			out = append(out, base)
			continue
		}

		var result struct {
			Code    int    `json:"code"`
			Content string `json:"content"`
			Message string `json:"message"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			base.Error = "解析响应失败: " + err.Error()
			out = append(out, base)
			continue
		}
		resp.Body.Close()

		if result.Code != 0 {
			base.Error = "MinerU 返回错误: " + result.Message
			out = append(out, base)
			continue
		}
		base.Content = result.Content
		out = append(out, base)
	}

	pkglogger.Global().Info("通过 MinerU 识别附件完成",
		zap.Int("total", len(files)),
		zap.Int("count", len(out)))
	return out, nil
}

// TestConnection 仅探测 MinerU /health；不做真实解析（与产品决策一致）。
func (s *AttachmentRecognitionService) TestConnection(ctx context.Context) error {
	cfg, err := s.LoadConfig()
	if err != nil {
		return newServiceError(errcode.ErrDatabase, "加载附件识别配置失败")
	}
	return s.TestConnectionWithConfig(ctx, cfg)
}

// TestConnectionWithConfig 使用指定配置探测 MinerU /health；
// 用于前端“未保存先测试”场景。
func (s *AttachmentRecognitionService) TestConnectionWithConfig(ctx context.Context, cfg *RecognitionConfig) error {
	if cfg == nil {
		return newServiceError(errcode.ErrInvalidParam, "测试配置为空")
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
