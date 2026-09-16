package oa

import (
	"context"
	"strings"
)

type attachmentFieldFilterContextKey struct{}
type attachmentFieldSetFilterContextKey struct{}

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

// WithAttachmentFieldSetFilter 限定本次 OA 数据拉取允许识别的主表与明细表附件字段集合。
// key 为表名（"main" 或明细表名，大小写不敏感），value 为该表选中的字段集合。
// value 为 nil 表示该表全选；空 map 表示该表不选任何字段。
func WithAttachmentFieldSetFilter(ctx context.Context, fieldSet map[string]map[string]bool) context.Context {
	if fieldSet == nil {
		return ctx
	}
	normalized := make(map[string]map[string]bool, len(fieldSet))
	for tbl, fields := range fieldSet {
		tblKey := strings.ToLower(strings.TrimSpace(tbl))
		if fields == nil {
			normalized[tblKey] = nil
			continue
		}
		fieldMap := make(map[string]bool, len(fields))
		for k, v := range fields {
			if v {
				fieldMap[strings.ToLower(strings.TrimSpace(k))] = true
			}
		}
		normalized[tblKey] = fieldMap
	}
	return context.WithValue(ctx, attachmentFieldSetFilterContextKey{}, normalized)
}

// attachmentFieldAllowed 判断主表附件字段是否在本次数据拉取的识别范围内。保持向后兼容。
func attachmentFieldAllowed(ctx context.Context, fieldKey string) bool {
	return attachmentFieldAllowedForTable(ctx, "main", fieldKey)
}

// attachmentFieldAllowedForTable 判断指定表（主表为 "main" 或明细表名）中的附件字段是否允许识别。
func attachmentFieldAllowedForTable(ctx context.Context, tableKey, fieldKey string) bool {
	normalField := strings.ToLower(strings.TrimSpace(fieldKey))
	normalTable := strings.ToLower(strings.TrimSpace(tableKey))
	if normalTable == "" || normalTable == "0" || normalTable == "主表" {
		normalTable = "main"
	}

	// 优先检查多表 FieldSet 过滤器
	if fieldSet, exists := ctx.Value(attachmentFieldSetFilterContextKey{}).(map[string]map[string]bool); exists {
		allowedFields, ok := fieldSet[normalTable]
		if !ok {
			// 尝试模糊匹配（如去除 mainTable 前缀后的 dt1）
			for tKey, fMap := range fieldSet {
				if strings.EqualFold(tKey, normalTable) {
					allowedFields = fMap
					ok = true
					break
				}
			}
		}
		if !ok {
			// 该表未在选择集中指定，默认全选允许
			return true
		}
		if allowedFields == nil {
			// 该表全选
			return true
		}
		return allowedFields[normalField]
	}

	// 降级检查传统单表过滤器（仅在未设置 FieldSet 时生效）
	if allowed, exists := ctx.Value(attachmentFieldFilterContextKey{}).(map[string]bool); exists {
		return allowed[normalField]
	}

	return true
}
