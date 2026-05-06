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

// AttachmentRecognitionService 附件识别服务，支持多种识别方式。
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

// RecognitionConfig 附件识别配置。
type RecognitionConfig struct {
	Enabled        bool
	MinerUEndpoint string
	MinerUAPIKey   string
	OAAPIEndpoint  string
	MaxFileSizeMB  int
	SupportedTypes []string
}

// LoadConfig 从系统配置加载附件识别配置。
func (s *AttachmentRecognitionService) LoadConfig() (*RecognitionConfig, error) {
	config := &RecognitionConfig{
		Enabled:        false,
		MaxFileSizeMB:  10,
		SupportedTypes: []string{"pdf", "png", "jpg", "jpeg", "bmp", "gif", "tiff", "webp", "docx", "xlsx", "txt"},
	}

	// 读取配置项
	if val, err := s.configRepo.FindByKey("attachment.recognition_enabled"); err == nil && val != "" {
		config.Enabled = val == "true" || val == "1"
	}

	if val, err := s.configRepo.FindByKey("attachment.mineru_endpoint"); err == nil && val != "" {
		config.MinerUEndpoint = val
	}

	if val, err := s.configRepo.FindByKey("attachment.mineru_api_key"); err == nil && val != "" {
		config.MinerUAPIKey = val
	}

	if val, err := s.configRepo.FindByKey("attachment.oa_api_endpoint"); err == nil && val != "" {
		config.OAAPIEndpoint = val
	}

	if val, err := s.configRepo.FindByKey("attachment.max_file_size_mb"); err == nil && val != "" {
		if size, parseErr := strconv.Atoi(val); parseErr == nil && size > 0 {
			config.MaxFileSizeMB = size
		}
	}

	if val, err := s.configRepo.FindByKey("attachment.supported_types"); err == nil && val != "" {
		types := strings.Split(val, ",")
		config.SupportedTypes = make([]string, 0, len(types))
		for _, t := range types {
			if trimmed := strings.TrimSpace(t); trimmed != "" {
				config.SupportedTypes = append(config.SupportedTypes, strings.ToLower(trimmed))
			}
		}
	}

	return config, nil
}

// RecognizeAttachmentsByDocIDs 根据 docIds 识别附件内容。
// docIds: 逗号分隔的附件 ID 列表
// fieldKey: 字段标识
// fieldName: 字段名称
func (s *AttachmentRecognitionService) RecognizeAttachmentsByDocIDs(
	ctx context.Context,
	docIds string,
	fieldKey string,
	fieldName string,
) ([]oa.AttachmentInfo, error) {
	config, err := s.LoadConfig()
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "加载附件识别配置失败")
	}

	// 如果未启用附件识别，返回空列表
	if !config.Enabled {
		return []oa.AttachmentInfo{}, nil
	}

	// 1. 从 OA 获取附件文件流
	files, err := s.fetchFilesFromOA(ctx, config, docIds)
	if err != nil {
		return nil, err
	}

	// 2. 调用 MinerU 识别文档内容
	return s.recognizeViaMinerU(ctx, config, files, fieldKey, fieldName)
}

// OAFileData OA 返回的文件数据
type OAFileData struct {
	DocID    string `json:"docId"`
	FileName string `json:"fileName"`
	FileSize int64  `json:"fileSize"`
	FileData string `json:"fileData"` // Base64 编码
}

// fetchFilesFromOA 从 OA 系统获取附件文件流。
func (s *AttachmentRecognitionService) fetchFilesFromOA(
	ctx context.Context,
	config *RecognitionConfig,
	docIds string,
) ([]OAFileData, error) {
	if config.OAAPIEndpoint == "" {
		return nil, newServiceError(errcode.ErrInvalidParam, "OA 附件接口地址未配置")
	}

	// 构建请求 URL
	url := fmt.Sprintf("%s?docIds=%s", config.OAAPIEndpoint, docIds)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, newServiceError(errcode.ErrExternal, "创建 HTTP 请求失败")
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		pkglogger.Global().Error("调用 OA 附件接口失败", zap.Error(err), zap.String("url", url))
		return nil, newServiceError(errcode.ErrExternal, "调用 OA 附件接口失败")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		pkglogger.Global().Error("OA 附件接口返回错误",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(body)))
		return nil, newServiceError(errcode.ErrExternal, "OA 附件接口返回错误")
	}

	// 解析响应
	var result struct {
		Code int          `json:"code"`
		Data []OAFileData `json:"data"`
		Msg  string       `json:"msg"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, newServiceError(errcode.ErrExternal, "解析 OA 附件接口响应失败")
	}

	if result.Code != 0 {
		return nil, newServiceError(errcode.ErrExternal, "OA 附件接口返回错误: "+result.Msg)
	}

	pkglogger.Global().Info("从 OA 获取附件文件流成功",
		zap.String("docIds", docIds),
		zap.Int("count", len(result.Data)))

	return result.Data, nil
}

// recognizeViaMinerU 通过 MinerU 服务识别附件。
func (s *AttachmentRecognitionService) recognizeViaMinerU(
	ctx context.Context,
	config *RecognitionConfig,
	files []OAFileData,
	fieldKey string,
	fieldName string,
) ([]oa.AttachmentInfo, error) {
	if config.MinerUEndpoint == "" {
		return nil, newServiceError(errcode.ErrInvalidParam, "MinerU 服务地址未配置")
	}

	attachments := make([]oa.AttachmentInfo, 0, len(files))
	now := time.Now().Format(time.RFC3339)

	for _, file := range files {
		// 提取文件类型
		fileType := ""
		if idx := strings.LastIndex(file.FileName, "."); idx >= 0 && idx < len(file.FileName)-1 {
			fileType = strings.ToLower(file.FileName[idx+1:])
		}

		// 构建请求
		reqBody := map[string]interface{}{
			"fileName": file.FileName,
			"fileData": file.FileData, // Base64 编码的文件内容
			"fileType": fileType,
		}

		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			attachments = append(attachments, oa.AttachmentInfo{
				DocID:     file.DocID,
				FileName:  file.FileName,
				FileType:  fileType,
				FileSize:  file.FileSize,
				FieldKey:  fieldKey,
				FieldName: fieldName,
				Error:     "构建请求失败: " + err.Error(),
			})
			continue
		}

		url := config.MinerUEndpoint + "/api/v1/parse"
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
		if err != nil {
			attachments = append(attachments, oa.AttachmentInfo{
				DocID:     file.DocID,
				FileName:  file.FileName,
				FileType:  fileType,
				FileSize:  file.FileSize,
				FieldKey:  fieldKey,
				FieldName: fieldName,
				Error:     "创建请求失败: " + err.Error(),
			})
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		if config.MinerUAPIKey != "" {
			req.Header.Set("Authorization", "Bearer "+config.MinerUAPIKey)
		}

		resp, err := s.httpClient.Do(req)
		if err != nil {
			pkglogger.Global().Error("调用 MinerU 服务失败",
				zap.Error(err),
				zap.String("fileName", file.FileName))
			attachments = append(attachments, oa.AttachmentInfo{
				DocID:     file.DocID,
				FileName:  file.FileName,
				FileType:  fileType,
				FileSize:  file.FileSize,
				FieldKey:  fieldKey,
				FieldName: fieldName,
				Error:     "调用 MinerU 失败: " + err.Error(),
			})
			continue
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			pkglogger.Global().Error("MinerU 服务返回错误",
				zap.Int("status", resp.StatusCode),
				zap.String("body", string(body)),
				zap.String("fileName", file.FileName))
			attachments = append(attachments, oa.AttachmentInfo{
				DocID:     file.DocID,
				FileName:  file.FileName,
				FileType:  fileType,
				FileSize:  file.FileSize,
				FieldKey:  fieldKey,
				FieldName: fieldName,
				Error:     fmt.Sprintf("MinerU 返回错误: %d", resp.StatusCode),
			})
			continue
		}

		// 解析响应
		var result struct {
			Code    int    `json:"code"`
			Content string `json:"content"`
			Message string `json:"message"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			attachments = append(attachments, oa.AttachmentInfo{
				DocID:     file.DocID,
				FileName:  file.FileName,
				FileType:  fileType,
				FileSize:  file.FileSize,
				FieldKey:  fieldKey,
				FieldName: fieldName,
				Error:     "解析响应失败: " + err.Error(),
			})
			continue
		}
		resp.Body.Close()

		if result.Code != 0 {
			attachments = append(attachments, oa.AttachmentInfo{
				DocID:     file.DocID,
				FileName:  file.FileName,
				FileType:  fileType,
				FileSize:  file.FileSize,
				FieldKey:  fieldKey,
				FieldName: fieldName,
				Error:     "MinerU 返回错误: " + result.Message,
			})
			continue
		}

		// 成功识别
		attachments = append(attachments, oa.AttachmentInfo{
			DocID:       file.DocID,
			FileName:    file.FileName,
			FileType:    fileType,
			FileSize:    file.FileSize,
			FieldKey:    fieldKey,
			FieldName:   fieldName,
			Content:     result.Content,
			ExtractedAt: now,
		})
	}

	pkglogger.Global().Info("通过 MinerU 识别附件完成",
		zap.Int("total", len(files)),
		zap.Int("success", len(attachments)))

	return attachments, nil
}

// TestConnection 测试附件识别服务连接。
func (s *AttachmentRecognitionService) TestConnection(ctx context.Context) error {
	config, err := s.LoadConfig()
	if err != nil {
		return newServiceError(errcode.ErrDatabase, "加载附件识别配置失败")
	}

	if !config.Enabled {
		return newServiceError(errcode.ErrInvalidParam, "附件识别未启用")
	}

	if config.MinerUEndpoint == "" {
		return newServiceError(errcode.ErrInvalidParam, "MinerU 服务地址未配置")
	}

	// 健康检查
	url := config.MinerUEndpoint + "/health"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return newServiceError(errcode.ErrExternal, "创建测试请求失败")
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return newServiceError(errcode.ErrExternal, "连接 MinerU 服务失败")
	}
	defer resp.Body.Close()
	return nil
}
