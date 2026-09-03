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
