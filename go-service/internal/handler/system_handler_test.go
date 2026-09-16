package handler

import (
	"testing"

	"auraoa/go-service/internal/service"
)

func TestAttachmentRecognitionTestRequestAllowsClearingUnsavedValues(t *testing.T) {
	empty := ""
	disabled := false
	cfg := &service.RecognitionConfig{
		Enabled:        true,
		MinerUEndpoint: "http://saved-mineru",
		MinerUAPIKey:   "saved-key",
	}
	req := attachmentRecognitionTestRequest{
		AttachmentRecognitionEnabled: &disabled,
		AttachmentMinerUEndpoint:     &empty,
		AttachmentMinerUAPIKey:       &empty,
	}

	req.apply(cfg)

	if cfg.Enabled {
		t.Fatal("Enabled 应被未保存请求覆盖为 false")
	}
	if cfg.MinerUEndpoint != "" || cfg.MinerUAPIKey != "" {
		t.Fatalf("空字符串应覆盖已保存值: %+v", cfg)
	}
}

func TestAttachmentCompatibilityTestRequestAppliesUnsavedValues(t *testing.T) {
	endpoint := "http://unsaved-parser"
	apiKey := "unsaved-key"
	documentTypes := "pdf,docx,xlsx"
	enabled := true
	disabled := false
	cfg := &service.RecognitionConfig{}
	req := attachmentCompatibilityTestRequest{
		AttachmentCompatEndpoint:        &endpoint,
		AttachmentCompatAPIKey:          &apiKey,
		AttachmentDocumentParserTypes:   &documentTypes,
		AttachmentLegacyOfficeEnabled:   &enabled,
		AttachmentOFDEnabled:            &enabled,
		AttachmentVisualFallbackEnabled: &disabled,
	}

	req.apply(cfg)

	if cfg.CompatEndpoint != endpoint || cfg.CompatAPIKey != apiKey {
		t.Fatalf("兼容解析服务临时地址或密钥未应用: %+v", cfg)
	}
	if len(cfg.DocumentParserTypes) != 3 || cfg.DocumentParserTypes[0] != "pdf" {
		t.Fatalf("代码文档解析类型未应用: %+v", cfg.DocumentParserTypes)
	}
	if !cfg.LegacyOfficeEnabled || !cfg.OFDEnabled || cfg.VisualFallbackEnabled {
		t.Fatalf("兼容解析服务临时开关未应用: %+v", cfg)
	}
}

func TestAttachmentAliyunOCRTestRequestAppliesUnsavedValues(t *testing.T) {
	endpoint := "ocr-api.cn-beijing.aliyuncs.com"
	ak := "test-ak"
	sk := "test-sk"
	ocrType := "Advanced"
	cfg := &service.RecognitionConfig{}
	req := attachmentAliyunOCRTestRequest{
		AttachmentAliyunOCREndpoint:        &endpoint,
		AttachmentAliyunOCRAccessKeyID:     &ak,
		AttachmentAliyunOCRAccessKeySecret: &sk,
		AttachmentAliyunOCRType:            &ocrType,
	}

	req.apply(cfg)

	if cfg.AliyunOCREndpoint != endpoint {
		t.Fatalf("AliyunOCREndpoint = %q, want %q", cfg.AliyunOCREndpoint, endpoint)
	}
	if cfg.AliyunOCRAccessKeyID != ak {
		t.Fatalf("AliyunOCRAccessKeyID = %q, want %q", cfg.AliyunOCRAccessKeyID, ak)
	}
	if cfg.AliyunOCRAccessKeySecret != sk {
		t.Fatalf("AliyunOCRAccessKeySecret = %q, want %q", cfg.AliyunOCRAccessKeySecret, sk)
	}
	if cfg.AliyunOCRType != ocrType {
		t.Fatalf("AliyunOCRType = %q, want %q", cfg.AliyunOCRType, ocrType)
	}
}

