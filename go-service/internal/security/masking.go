package security

import (
	"context"
	"oa-smart-audit/internal/oa"
)

type MaskingRule struct {
	FieldPattern string `json:"field_pattern"`
	ValuePattern string `json:"value_pattern"`
	ReplaceWith  string `json:"replace_with"`
}

// DataMasker handles sensitive data masking before sending to AI service.
type DataMasker interface {
	MaskFormData(ctx context.Context, data oa.ProcessFormData) (oa.ProcessFormData, error)
	LoadMaskingRules(ctx context.Context, tenantID string) ([]MaskingRule, error)
}
