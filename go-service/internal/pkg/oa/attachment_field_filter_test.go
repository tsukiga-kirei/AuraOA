package oa

import (
	"context"
	"testing"
)

func TestAttachmentFieldAllowed(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		field   string
		allowed bool
	}{
		{name: "未设置过滤时允许全部", ctx: context.Background(), field: "fpfj", allowed: true},
		{name: "命中选中字段", ctx: WithAttachmentFieldFilter(context.Background(), map[string]bool{"FPFJ": true}), field: "fpfj", allowed: true},
		{name: "未命中选中字段", ctx: WithAttachmentFieldFilter(context.Background(), map[string]bool{"htfj": true}), field: "fpfj", allowed: false},
		{name: "空集合拒绝全部", ctx: WithAttachmentFieldFilter(context.Background(), map[string]bool{}), field: "fpfj", allowed: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := attachmentFieldAllowed(tt.ctx, tt.field); got != tt.allowed {
				t.Fatalf("attachmentFieldAllowed() = %v, want %v", got, tt.allowed)
			}
		})
	}

	// 测试多表 FieldSet 过滤器
	fieldSet := map[string]map[string]bool{
		"main":                    {"fj": true},
		"formtable_main_60_dt1":   {"fpfj": true},
		"formtable_main_60_dt2":   {}, // 空集合，该明细表全部拒绝
	}
	ctxSet := WithAttachmentFieldSetFilter(context.Background(), fieldSet)

	if !attachmentFieldAllowedForTable(ctxSet, "main", "FJ") {
		t.Errorf("主表 fj 应该允许")
	}
	if attachmentFieldAllowedForTable(ctxSet, "main", "other") {
		t.Errorf("主表 other 应该被拒绝")
	}
	if !attachmentFieldAllowedForTable(ctxSet, "formtable_main_60_dt1", "FPFJ") {
		t.Errorf("明细表1 fpfj 应该允许")
	}
	if attachmentFieldAllowedForTable(ctxSet, "formtable_main_60_dt1", "other") {
		t.Errorf("明细表1 other 应该被拒绝")
	}
	if attachmentFieldAllowedForTable(ctxSet, "formtable_main_60_dt2", "fpfj") {
		t.Errorf("明细表2 空集合应该拒绝全部")
	}
	if !attachmentFieldAllowedForTable(ctxSet, "formtable_main_60_dt3", "any_field") {
		t.Errorf("未声明的明细表3 应该默认允许")
	}
}
