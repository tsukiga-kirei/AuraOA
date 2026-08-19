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
}
