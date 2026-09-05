package service

import (
	"auraoa/go-service/internal/model"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestMCPInitializesPaginatesAndClosesSession(t *testing.T) {
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			methods = append(methods, "DELETE")
			w.WriteHeader(204)
			return
		}
		var req jsonRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		methods = append(methods, req.Method)
		if req.Method != "initialize" && (r.Header.Get("Mcp-Session-Id") != "test-session" || r.Header.Get("MCP-Protocol-Version") != "2025-06-18") {
			t.Error("初始化后的请求缺少会话或协议版本")
		}
		switch req.Method {
		case "initialize":
			w.Header().Set("Mcp-Session-Id", "test-session")
			json.NewEncoder(w).Encode(map[string]interface{}{"jsonrpc": "2.0", "id": req.ID, "result": map[string]interface{}{"protocolVersion": "2025-06-18"}})
		case "notifications/initialized":
			w.WriteHeader(202)
		case "tools/list":
			result := map[string]interface{}{"tools": []map[string]interface{}{{"name": "query", "inputSchema": map[string]string{"type": "object"}}}}
			params, _ := req.Params.(map[string]interface{})
			if params["cursor"] == nil {
				result["nextCursor"] = "page2"
			}
			// 通知可以先于匹配请求的结果，且 SSE 多行由完整帧解析。
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprint(w, "data: {\"jsonrpc\":\"2.0\",\"method\":\"notifications/progress\"}\n\n")
			raw, _ := json.Marshal(map[string]interface{}{"jsonrpc": "2.0", "id": req.ID, "result": result})
			fmt.Fprintf(w, "data: %s\n\n", raw)
		}
	}))
	defer server.Close()
	svc := NewMCPService(nil)
	tools, err := svc.fetchTools(context.Background(), &model.MCPServer{EndpointURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 2 {
		t.Fatalf("未获取完整分页目录: %d", len(tools))
	}
	if !reflect.DeepEqual(methods, []string{"initialize", "notifications/initialized", "tools/list", "tools/list", "DELETE"}) {
		t.Fatal(methods)
	}
}

func TestMCPRejectsMismatchedResponseAndInvalidEndpoint(t *testing.T) {
	svc := NewMCPService(nil)
	if _, err := svc.connect(context.Background(), &model.MCPServer{EndpointURL: "file:///etc/passwd"}); err == nil {
		t.Fatal("非法协议应拒绝")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":"different","result":{}}`)
	}))
	defer server.Close()
	if _, err := svc.connect(context.Background(), &model.MCPServer{EndpointURL: server.URL}); err == nil {
		t.Fatal("错误请求 ID 应拒绝")
	}
}
