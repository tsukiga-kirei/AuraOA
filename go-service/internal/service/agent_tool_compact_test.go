package service

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGetToolModelName(t *testing.T) {
	if name := getToolModelName("list_my_todos"); name != "list_my_todos" {
		t.Fatalf("内置工具名应保持原样: got %s", name)
	}
	if name := getToolModelName("skill:check_budget"); !strings.HasPrefix(name, "tool_") {
		t.Fatalf("包含冒号的技能键应转换为 tool_ 开头的别名: got %s", name)
	}
	if name := getToolModelName("mcp:feishu:search"); !strings.HasPrefix(name, "tool_") {
		t.Fatalf("MCP 键应转换为 tool_ 开头的别名: got %s", name)
	}
}

func TestCompactToolPayload_Todos(t *testing.T) {
	todosPayload := map[string]interface{}{
		"total": 10,
		"items": []interface{}{
			map[string]interface{}{
				"request_id":   "101",
				"request_name": "张三的差旅报销",
				"applicant":    "张三",
				"amount":       "3500.00",
				"status":       "审批中",
				"extra_ui_css": "long-css-styling-rules-should-be-omitted",
			},
			map[string]interface{}{
				"request_id":   "102",
				"request_name": "李四的请假申请",
				"applicant":    "李四",
				"status":       "待处理",
			},
		},
	}

	compacted := CompactToolPayload("list_my_todos", todosPayload)
	if !strings.Contains(compacted, "张三的差旅报销") || !strings.Contains(compacted, "3500.00") {
		t.Fatalf("核心字段未保留: %s", compacted)
	}
	if strings.Contains(compacted, "extra_ui_css") {
		t.Fatalf("冗余展示配置未被剔除: %s", compacted)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(compacted), &parsed); err != nil {
		t.Fatalf("精简结果应为有效 JSON: %v", err)
	}
}

func TestCompactToolPayload_Process(t *testing.T) {
	procPayload := map[string]interface{}{
		"process_id":   "8848",
		"request_name": "采购服务器",
		"creator_name": "王五",
		"status":       "审批中",
		"form_data": map[string]interface{}{
			"amount":    "58000.00",
			"vendor":    "某某科技",
			"empty_val": "",
		},
		"attachments": []interface{}{
			map[string]interface{}{"name": "报价单.pdf", "url": "http://oa/files/1"},
		},
	}

	compacted := CompactToolPayload("get_process", procPayload)
	if !strings.Contains(compacted, "58000.00") || !strings.Contains(compacted, "报价单.pdf") {
		t.Fatalf("表单关键字段或附件名丢失: %s", compacted)
	}
	if strings.Contains(compacted, "empty_val") {
		t.Fatalf("空值字段未被清除: %s", compacted)
	}
}

func TestCompactToolPayload_GenericMCP(t *testing.T) {
	mcpPayload := map[string]interface{}{
		"status": "ok",
		"result": strings.Repeat("A", 1000), // 超长文本
	}

	compacted := CompactToolPayload("mcp:custom:call", mcpPayload)
	if len([]rune(compacted)) > 1500 {
		t.Fatalf("超长内容未被截断: len=%d", len([]rune(compacted)))
	}
}

// TestCompactToolPayloadRealContracts 使用系统工具实际返回的字段和附件形态，避免模拟另一套接口掩盖信息丢失。
func TestCompactToolPayloadRealContracts(t *testing.T) {
	for _, tc := range []struct {
		tool    string
		payload map[string]interface{}
		want    []string
	}{
		{"get_process", map[string]interface{}{"process_id": "p1", "title": "采购申请", "applicant": "申请人甲", "main_fields": []map[string]interface{}{{"field": "amount", "value": "58000"}}, "attachments": []string{"报价单.pdf"}}, []string{"采购申请", "申请人甲", "58000", "报价单.pdf"}},
		{"get_latest_audit", map[string]interface{}{"process_id": "p1", "has_audit": true, "status": "completed", "recommendation": "return", "score": 40, "audit_result": map[string]interface{}{"risk_points": []string{"金额不一致"}}}, []string{"return", "40", "金额不一致", "has_audit"}},
		{"get_latest_summary", map[string]interface{}{"process_id": "p1", "has_summary": true, "status": "completed", "summary_result": map[string]interface{}{"blocks": []map[string]interface{}{{"content": "采购金额58000元"}}}}, []string{"采购金额58000元", "has_summary"}},
		{"get_approval_flow", map[string]interface{}{"process_id": "p1", "nodes": []map[string]interface{}{{"node_name": "财务", "operated_at": "2026-10-26 10:00:00"}}}, []string{"财务", "2026-10-26 10:00:00"}},
	} {
		t.Run(tc.tool, func(t *testing.T) {
			compact := CompactToolPayload(tc.tool, tc.payload)
			if !json.Valid([]byte(compact)) {
				t.Fatalf("工具结果结构损坏：%s", compact)
			}
			for _, value := range tc.want {
				if !strings.Contains(compact, value) {
					t.Errorf("实际工具字段丢失 %q：%s", value, compact)
				}
			}
		})
	}
}
