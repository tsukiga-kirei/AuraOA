package service

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"strings"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	ocr_api "github.com/alibabacloud-go/ocr-api-20210707/v3/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"go.uber.org/zap"

	pkglogger "auraoa/go-service/internal/pkg/logger"
	"auraoa/go-service/internal/pkg/oa"
)

// recognizeOneViaAliyunOCR 通过阿里云 OCR 识别单附件并返回 AttachmentInfo。
func (s *AttachmentRecognitionService) recognizeOneViaAliyunOCR(
	ctx context.Context,
	cfg *RecognitionConfig,
	file oa.AttachmentFilePayload,
	raw []byte,
	base oa.AttachmentInfo,
) oa.AttachmentInfo {
	content, err := s.recognizeViaAliyunOCR(ctx, cfg, file.FileName, raw)
	if err != nil {
		base.Error = serviceErrorMessage(err)
		return base
	}
	base.Content = content
	return base
}

// recognizeViaAliyunOCR 调用阿里云统一文字识别（RecognizeAllText 2021-07-07）接口提取图片或文档文本。
func (s *AttachmentRecognitionService) recognizeViaAliyunOCR(
	ctx context.Context,
	cfg *RecognitionConfig,
	fileName string,
	raw []byte,
) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("附件识别配置为空")
	}
	ak := strings.TrimSpace(cfg.AliyunOCRAccessKeyID)
	sk := strings.TrimSpace(cfg.AliyunOCRAccessKeySecret)
	if ak == "" || sk == "" {
		return "", fmt.Errorf("阿里云 OCR 未配置 AccessKey ID 或 AccessKey Secret")
	}

	endpoint := strings.TrimSpace(cfg.AliyunOCREndpoint)
	if endpoint == "" {
		endpoint = "ocr-api.cn-hangzhou.aliyuncs.com"
	}
	ocrType := strings.TrimSpace(cfg.AliyunOCRType)
	if ocrType == "" {
		ocrType = "General"
	}

	config := &openapi.Config{
		AccessKeyId:     tea.String(ak),
		AccessKeySecret: tea.String(sk),
		Endpoint:        tea.String(endpoint),
	}

	client, err := ocr_api.NewClient(config)
	if err != nil {
		pkglogger.Global().Error("初始化阿里云 OCR 客户端失败",
			zap.String("fileName", fileName),
			zap.Error(err))
		return "", fmt.Errorf("初始化阿里云 OCR 客户端失败: %w", err)
	}

	req := &ocr_api.RecognizeAllTextRequest{
		Type: tea.String(ocrType),
		Body: bytes.NewReader(raw),
	}

	runtime := &util.RuntimeOptions{
		ConnectTimeout: tea.Int(10000), // 10秒连接超时
		ReadTimeout:    tea.Int(60000), // 60秒读取超时
	}

	pkglogger.Global().Info("调用阿里云 OCR 接口",
		zap.String("fileName", fileName),
		zap.String("endpoint", endpoint),
		zap.String("type", ocrType),
		zap.Int("fileBytes", len(raw)))

	resp, err := client.RecognizeAllTextWithOptions(req, runtime)
	if err != nil {
		pkglogger.Global().Warn("调用阿里云 OCR 失败",
			zap.String("fileName", fileName),
			zap.Error(err))
		return "", fmt.Errorf("调用阿里云 OCR 失败: %w", err)
	}

	if resp == nil || resp.Body == nil {
		return "", fmt.Errorf("阿里云 OCR 返回空响应")
	}

	if resp.Body.Code != nil && *resp.Body.Code != "200" && *resp.Body.Code != "" {
		msg := ""
		if resp.Body.Message != nil {
			msg = *resp.Body.Message
		}
		return "", fmt.Errorf("阿里云 OCR 返回错误 [%s]: %s", *resp.Body.Code, msg)
	}

	var content string
	if resp.Body.Data != nil && resp.Body.Data.Content != nil {
		content = *resp.Body.Data.Content
	}
	content = strings.TrimSpace(strings.ToValidUTF8(content, ""))
	pkglogger.Global().Info("阿里云 OCR 识别成功",
		zap.String("fileName", fileName),
		zap.Int("contentLength", len(content)))
	return content, nil
}

// TestAliyunOCRConnection 探测已配置的阿里云 OCR 服务是否连通且凭证有效。
func (s *AttachmentRecognitionService) TestAliyunOCRConnection(ctx context.Context) error {
	cfg, err := s.LoadConfig()
	if err != nil {
		return err
	}
	return s.TestAliyunOCRConnectionWithConfig(ctx, cfg)
}

// TestAliyunOCRConnectionWithConfig 探测指定配置的阿里云 OCR 是否可达且鉴权有效。
func (s *AttachmentRecognitionService) TestAliyunOCRConnectionWithConfig(ctx context.Context, cfg *RecognitionConfig) error {
	if cfg == nil {
		return fmt.Errorf("配置不可为空")
	}
	if strings.TrimSpace(cfg.AliyunOCRAccessKeyID) == "" {
		return fmt.Errorf("阿里云 AccessKey ID 尚未填写")
	}
	if strings.TrimSpace(cfg.AliyunOCRAccessKeySecret) == "" {
		return fmt.Errorf("阿里云 AccessKey Secret 尚未填写")
	}

	testPNG, err := createTestPNGImage()
	if err != nil {
		return fmt.Errorf("生成测试图片失败: %w", err)
	}

	_, err = s.recognizeViaAliyunOCR(ctx, cfg, "test_connection.png", testPNG)
	if err != nil {
		return err
	}
	return nil
}

// createTestPNGImage 生成一张 20x20 的合规测试 PNG 图片（阿里云 OCR 要求尺寸长宽大于 15 像素）。
func createTestPNGImage() ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, 20, 20))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
