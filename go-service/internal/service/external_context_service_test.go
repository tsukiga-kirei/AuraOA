package service

import (
	"strings"
	"testing"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/oa"
)

func TestFormatExternalContextSourceField(t *testing.T) {
	pd := &oa.ProcessData{
		FieldLabels: map[string]map[string]string{
			"main":                   {"supplier_code": "供应商编码"},
			"formtable_main_151_dt1": {"supplier_code": "明细供应商编码"},
		},
	}

	tests := []struct {
		name string
		ref  string
		want string
	}{
		{name: "主表字段显示中文名", ref: "main:supplier_code", want: "供应商编码（main:supplier_code）"},
		{name: "明细字段显示中文名", ref: "formtable_main_151_dt1:supplier_code", want: "明细供应商编码（formtable_main_151_dt1:supplier_code）"},
		{name: "未配置显示名时保留原字段", ref: "main:unknown_field", want: "main:unknown_field"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatExternalContextSourceField(pd, tt.ref); got != tt.want {
				t.Fatalf("formatExternalContextSourceField() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatModelContextResultUsesFieldLabels(t *testing.T) {
	result := formatModelContextResult(
		"关联建模表",
		"供应商编码（main:supplier_code）",
		&model.ExternalModelContextConfig{Mode: "rows"},
		&oa.ModelContextQueryResult{Mode: "rows", Rows: []map[string]interface{}{{"lcm": "采购申请", "lcid": "1001"}}},
		map[string]string{"lcm": "流程名", "lcid": "流程ID"},
	)
	for _, want := range []string{"流程名=采购申请", "流程ID=1001"} {
		if !strings.Contains(result, want) {
			t.Fatalf("formatModelContextResult() = %q, want to contain %q", result, want)
		}
	}
}
