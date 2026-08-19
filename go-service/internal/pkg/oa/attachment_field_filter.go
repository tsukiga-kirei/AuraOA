package oa

import (
	"context"
	"strings"
)

type attachmentFieldFilterContextKey struct{}

// WithAttachmentFieldFilter 限定本次 OA 数据拉取允许识别的主表附件字段。
// allowed 为 nil 表示全部字段；非 nil 的空集合表示不识别任何附件字段。
func WithAttachmentFieldFilter(ctx context.Context, allowed map[string]bool) context.Context {
	if allowed == nil {
		return ctx
	}
	normalized := make(map[string]bool, len(allowed))
	for key, enabled := range allowed {
		if enabled {
			normalized[strings.ToLower(strings.TrimSpace(key))] = true
		}
	}
	return context.WithValue(ctx, attachmentFieldFilterContextKey{}, normalized)
}

// attachmentFieldAllowed 判断附件字段是否在本次数据拉取的识别范围内。
func attachmentFieldAllowed(ctx context.Context, fieldKey string) bool {
	allowed, exists := ctx.Value(attachmentFieldFilterContextKey{}).(map[string]bool)
	if !exists {
		return true
	}
	return allowed[strings.ToLower(strings.TrimSpace(fieldKey))]
}
