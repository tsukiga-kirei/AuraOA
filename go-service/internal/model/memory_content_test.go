package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestQueryContentDefaultsAndJSONOrder(t *testing.T) {
	legacy := []byte(`{"name":"统计","description":"说明","sql":"SELECT 1 AS 数量","dialect":"mysql","tables":null,"parameters":null,"builder":{"table":"form1","columns":[{"field":"amount"}],"filters":null,"source_type":"workflow"}}`)
	current := []byte(`{"mode":"config","name":"统计","description":"说明","sql":"SELECT 1 AS 数量","dialect":"mysql","tables":[],"parameters":[],"max_rows":0,"ai_context":{"oa_module":"workflow","notes":"","field_notes":{}},"builder":{"source_type":"workflow","archived_only":true,"table":"form1","columns":[{"aggregate":"","alias":"","field":"amount"}],"filters":[],"order_alias":"","order_dir":"DESC","sql_mode":false}}`)
	if !EqualQueryDefinitions(legacy, current) {
		t.Fatal("补缺省值或JSON键重排不能形成查询变更")
	}
	for _, tc := range []struct{ name, old, next string }{
		{"归档口径", "\"archived_only\":true", "\"archived_only\":false"},
		{"输出列", "\"alias\":\"\"", "\"alias\":\"金额\""},
		{"SQL原文", "SELECT 1 AS 数量", "SELECT 2 AS 数量"},
		{"行数上限", "\"max_rows\":0", "\"max_rows\":100"},
		{"字符上限", "\"max_rows\":0", "\"max_rows\":0,\"max_chars\":64000"},
		{"用途说明", "\"notes\":\"\"", "\"notes\":\"金额按元统计\""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if EqualQueryDefinitions(legacy, []byte(strings.Replace(string(current), tc.old, tc.next, 1))) {
				t.Fatal("真实内容变化被漏报")
			}
		})
	}
}

func TestQuerySQLLiteralAndNumberPrecision(t *testing.T) {
	a := []byte(`{"mode":"sql","sql":"SELECT 'a  b' AS 文本","tables":[],"parameters":[]}`)
	if EqualQueryDefinitions(a, []byte(strings.Replace(string(a), "a  b", "a b", 1))) {
		t.Fatal("SQL字符串内部空格不能归一化")
	}
	raw := []byte(`{"builder":{"table":"x","extension_number":9007199254740993}}`)
	canonical, err := CanonicalQueryDefinition(raw)
	if err != nil || !strings.Contains(string(canonical), "9007199254740993") {
		t.Fatalf("扩展数值精度丢失: %s %v", canonical, err)
	}
	if EqualQueryDefinitions(raw, []byte(strings.Replace(string(raw), "9007199254740993", "9007199254740992", 1))) {
		t.Fatal("不同数值被视为相同")
	}
	if EqualQueryDefinitions([]byte(`{bad`), a) {
		t.Fatal("损坏配置不能视为相同")
	}
}

func TestFeedbackMemoryContentComparison(t *testing.T) {
	before := []byte(`[{"content":"建议核实金额","source_refs":["a","b"]}]`)
	same := []byte(`[{"source_refs":["b","a"],"condition":null,"content":" 建议核实金额 "}]`)
	if !EqualFeedbackMemoryItems(before, same) {
		t.Fatal("空条件、对象键顺序及来源集合顺序不应产生编辑版本")
	}
	for _, raw := range []string{
		`[{"content":"建议核实金额","condition":"金额超过100万","source_refs":["a","b"]}]`,
		`[{"content":"建议核实发票","source_refs":["a","b"]}]`,
		`[{"content":"建议核实金额","source_refs":["a"]}]`,
		`[]`,
	} {
		if EqualFeedbackMemoryItems(before, []byte(raw)) {
			t.Fatal("真实记忆修改未识别")
		}
	}
	if !EqualFeedbackMemoryItems(nil, []byte(`[]`)) || !EqualFeedbackMemoryItems([]byte(`null`), []byte(`[]`)) {
		t.Fatal("空记忆表示应一致")
	}
	items := []FeedbackMemoryItem{{Content: "甲", SourceRefs: []string{"a"}}, {Content: "乙", SourceRefs: []string{"b"}}}
	left, _ := json.Marshal(items)
	items[0], items[1] = items[1], items[0]
	right, _ := json.Marshal(items)
	if EqualFeedbackMemoryItems(left, right) {
		t.Fatal("条目顺序影响最终正文，应作为真实变更")
	}
}
