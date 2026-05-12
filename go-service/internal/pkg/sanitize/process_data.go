package sanitize

import (
	"oa-smart-audit/go-service/internal/pkg/oa"
)

// sanitizeJSONValue 递归处理 map / []interface{} / string，其它类型原样返回。
func sanitizeJSONValue(v interface{}) interface{} {
	switch t := v.(type) {
	case string:
		return SanitizeText(t)
	case map[string]interface{}:
		for k, vv := range t {
			t[k] = sanitizeJSONValue(vv)
		}
		return t
	case []interface{}:
		for i, vv := range t {
			t[i] = sanitizeJSONValue(vv)
		}
		return t
	default:
		return v
	}
}

// SanitizeProcessData 对 OA 拉取的主表、明细表、附件文本字段做敏感信息脱敏（就地修改）。
func SanitizeProcessData(pd *oa.ProcessData) {
	if pd == nil {
		return
	}
	for k, v := range pd.MainData {
		pd.MainData[k] = sanitizeJSONValue(v)
	}
	for table, rows := range pd.DetailTables {
		for i := range rows {
			for fk, fv := range rows[i] {
				rows[i][fk] = sanitizeJSONValue(fv)
			}
		}
		pd.DetailTables[table] = rows
	}
	for i := range pd.Attachments {
		a := &pd.Attachments[i]
		a.Content = SanitizeText(a.Content)
		a.FileName = SanitizeText(a.FileName)
		a.Error = SanitizeText(a.Error)
	}
}

// SanitizeFlowSnapshot 对审批流快照中的文本字段脱敏（就地修改）。
func SanitizeFlowSnapshot(fs *oa.ProcessFlowSnapshot) {
	if fs == nil {
		return
	}
	fs.HistoryText = SanitizeText(fs.HistoryText)
	fs.GraphText = SanitizeText(fs.GraphText)
	for i := range fs.Nodes {
		fs.Nodes[i].Opinion = SanitizeText(fs.Nodes[i].Opinion)
		fs.Nodes[i].Approver = SanitizeText(fs.Nodes[i].Approver)
		fs.Nodes[i].NodeName = SanitizeText(fs.Nodes[i].NodeName)
		fs.Nodes[i].Action = SanitizeText(fs.Nodes[i].Action)
	}
}
