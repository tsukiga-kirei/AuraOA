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
