package oa

import "testing"

func TestPickBestModelingDisplayColumn(t *testing.T) {
	tests := []struct {
		name     string
		fields   []map[string]interface{}
		expected string
	}{
		{
			name: "优先匹配含wb的文本列(如用户的fplxwb)",
			fields: []map[string]interface{}{
				{"fieldname": "fplx", "labelname": "发票类型", "fieldhtmltype": "5"},
				{"fieldname": "sfkdk", "labelname": "是否可抵扣", "fieldhtmltype": "5"},
				{"fieldname": "zj", "labelname": "主键", "fieldhtmltype": "1"},
				{"fieldname": "fplxwb", "labelname": "发票类型（文本）", "fieldhtmltype": "1"},
			},
			expected: "fplxwb",
		},
		{
			name: "优先匹配含mc的单行文本列(如fylxmc)",
			fields: []map[string]interface{}{
				{"fieldname": "code", "labelname": "编码", "fieldhtmltype": "1"},
				{"fieldname": "fylxmc", "labelname": "费用类型名称", "fieldhtmltype": "1"},
				{"fieldname": "status", "labelname": "状态", "fieldhtmltype": "5"},
			},
			expected: "fylxmc",
		},
		{
			name: "匹配中文标签包含名称的单行文本列",
			fields: []map[string]interface{}{
				{"fieldname": "c1", "labelname": "项目名称", "fieldhtmltype": "1"},
				{"fieldname": "c2", "labelname": "备注", "fieldhtmltype": "2"},
			},
			expected: "c1",
		},
		{
			name: "无名称特征时兜底取第一个非主键单行文本",
			fields: []map[string]interface{}{
				{"fieldname": "id", "labelname": "ID", "fieldhtmltype": "1"},
				{"fieldname": "zj", "labelname": "主键", "fieldhtmltype": "1"},
				{"fieldname": "custom_code", "labelname": "自定义编号", "fieldhtmltype": "1"},
			},
			expected: "custom_code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pickBestModelingDisplayColumn(tt.fields)
			if got != tt.expected {
				t.Errorf("pickBestModelingDisplayColumn() = %v, want %v", got, tt.expected)
			}
		})
	}
}
