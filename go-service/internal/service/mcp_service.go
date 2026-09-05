package service

import (
	"auraoa/go-service/internal/pkg/apptime"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/ai"
	"auraoa/go-service/internal/pkg/crypto"
	"auraoa/go-service/internal/repository"
)

// MCPService 负责外部 MCP (Model Context Protocol) 服务的连接测试、工具探测与执行
type MCPService struct {
	agentRepo *repository.AgentRepo
	client    *http.Client
}

// NewMCPService 创建 MCPService
func NewMCPService(agentRepo *repository.AgentRepo) *MCPService {
	return &MCPService{
		agentRepo: agentRepo,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// JSON-RPC 2.0 基础结构
type jsonRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MCPToolItem MCP 工具定义
type MCPToolItem struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// TestAndRefreshTools 测试连接并刷新可用工具列表
func (s *MCPService) TestAndRefreshTools(ctx context.Context, tenantID, serverID uuid.UUID) ([]MCPToolItem, error) {
	alloc, err := s.agentRepo.GetTenantAllocation(tenantID)
	if err != nil || alloc == nil || !alloc.AllowTenantMCP {
		return nil, fmt.Errorf("当前租户未获得 MCP 权限")
	}
	server, err := s.agentRepo.GetMCPServerByID(tenantID, serverID)
	if err != nil {
		return nil, fmt.Errorf("未找到指定的 MCP 服务器: %w", err)
	}
	if server.TenantID == nil || *server.TenantID != tenantID {
		return nil, fmt.Errorf("无权测试该 MCP 服务")
	}

	tools, err := s.fetchTools(ctx, server)
	if err != nil {
		return nil, err
	}

	// 缓存工具列表并更新时间戳
	toolsJSON, _ := json.Marshal(tools)
	now := apptime.Now()
	if err := s.agentRepo.UpdateMCPServer(tenantID, serverID, map[string]interface{}{
		"cached_tools":   datatypes.JSON(toolsJSON),
		"last_synced_at": &now,
	}); err != nil {
		return nil, err
	}

	return tools, nil
}

// CallTool 调用指定的 MCP 工具
func (s *MCPService) CallTool(ctx context.Context, tenantID uuid.UUID, serverCode, toolName, argumentsJSON string) (interface{}, error) {
	alloc, err := s.agentRepo.GetTenantAllocation(tenantID)
	if err != nil || !alloc.AllowTenantMCP {
		return nil, fmt.Errorf("当前租户未获得 MCP 权限")
	}
	servers, err := s.agentRepo.ListMCPServers(tenantID)
	if err != nil {
		return nil, err
	}

	var targetServer *model.MCPServer
	for i := range servers {
		if servers[i].ServerCode == serverCode && servers[i].TenantID != nil && *servers[i].TenantID == tenantID {
			targetServer = &servers[i]
			break
		}
	}
	if targetServer == nil {
		return nil, fmt.Errorf("未找到编码为 %s 的 MCP 服务", serverCode)
	}
	if !targetServer.Enabled {
		return nil, fmt.Errorf("MCP 服务 %s 已停用", serverCode)
	}

	var args map[string]interface{}
	if err := json.Unmarshal([]byte(argumentsJSON), &args); err != nil {
		return nil, fmt.Errorf("MCP 工具参数必须为 JSON 对象")
	}

	reqBody := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      time.Now().UnixNano(),
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      toolName,
			"arguments": args,
		},
	}

	respRaw, err := s.doRequest(ctx, targetServer, reqBody)
	if err != nil {
		return nil, err
	}

	var result struct {
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(respRaw, &result); err != nil {
		return nil, err
	}
	if result.IsError {
		return nil, fmt.Errorf("MCP 工具返回执行错误")
	}
	var payload interface{}
	if err := json.Unmarshal(respRaw, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (s *MCPService) fetchTools(ctx context.Context, server *model.MCPServer) ([]MCPToolItem, error) {
	conn, err := s.connect(ctx, server)
	if err != nil {
		return nil, err
	}
	defer conn.close()
	tools := []MCPToolItem{}
	cursor := ""
	for page := 0; page < 100; page++ {
		params := map[string]interface{}{}
		if cursor != "" {
			params["cursor"] = cursor
		}
		raw, err := conn.request(ctx, jsonRPCRequest{JSONRPC: "2.0", ID: uuid.NewString(), Method: "tools/list", Params: params})
		if err != nil {
			return nil, err
		}
		var result struct {
			Tools      []MCPToolItem `json:"tools"`
			NextCursor string        `json:"nextCursor"`
		}
		if err := json.Unmarshal(raw, &result); err != nil {
			return nil, err
		}
		tools = append(tools, result.Tools...)
		if result.NextCursor == "" {
			return tools, nil
		}
		if result.NextCursor == cursor {
			return nil, fmt.Errorf("MCP 返回重复分页游标")
		}
		cursor = result.NextCursor
	}
	return nil, fmt.Errorf("MCP 工具目录超过分页上限")
}

func (s *MCPService) doRequest(ctx context.Context, server *model.MCPServer, body jsonRPCRequest) (json.RawMessage, error) {
	conn, err := s.connect(ctx, server)
	if err != nil {
		return nil, err
	}
	defer conn.close()
	return conn.request(ctx, body)
}

type mcpHTTPConnection struct {
	client   *http.Client
	endpoint string
	headers  http.Header
	session  string
}

// connect 完成 Streamable HTTP 初始化与版本协商，每次业务操作结束后释放会话。
func (s *MCPService) connect(ctx context.Context, server *model.MCPServer) (*mcpHTTPConnection, error) {
	endpoint, err := url.Parse(server.EndpointURL)
	if err != nil || endpoint.Host == "" || (endpoint.Scheme != "https" && endpoint.Scheme != "http") || endpoint.User != nil {
		return nil, fmt.Errorf("MCP 地址必须是有效的 HTTP(S) 服务地址")
	}
	conn := &mcpHTTPConnection{client: s.client, endpoint: endpoint.String(), headers: make(http.Header)}
	if server.HeadersEncrypted != "" {
		plain, err := crypto.Decrypt(server.HeadersEncrypted)
		if err != nil {
			return nil, fmt.Errorf("MCP 请求头解密失败")
		}
		var headers map[string]string
		if err := json.Unmarshal([]byte(plain), &headers); err != nil {
			return nil, fmt.Errorf("MCP 请求头格式无效")
		}
		for key, value := range headers {
			conn.headers.Set(key, value)
		}
	}
	conn.headers.Set("Content-Type", "application/json")
	conn.headers.Set("Accept", "application/json, text/event-stream")
	raw, err := conn.request(ctx, jsonRPCRequest{JSONRPC: "2.0", ID: uuid.NewString(), Method: "initialize", Params: map[string]interface{}{"protocolVersion": "2025-06-18", "capabilities": map[string]interface{}{}, "clientInfo": map[string]string{"name": "AuraOA", "version": "1.0"}}})
	if err != nil {
		return nil, err
	}
	var initialized struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	if err := json.Unmarshal(raw, &initialized); err != nil {
		conn.close()
		return nil, err
	}
	if initialized.ProtocolVersion != "2025-06-18" && initialized.ProtocolVersion != "2025-03-26" {
		conn.close()
		return nil, fmt.Errorf("MCP 协议版本不受支持: %s", initialized.ProtocolVersion)
	}
	conn.headers.Set("MCP-Protocol-Version", initialized.ProtocolVersion)
	if _, err := conn.request(ctx, jsonRPCRequest{JSONRPC: "2.0", Method: "notifications/initialized"}); err != nil {
		conn.close()
		return nil, err
	}
	return conn, nil
}

func (conn *mcpHTTPConnection) close() {
	if conn.session == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, conn.endpoint, nil)
	if err != nil {
		return
	}
	req.Header = conn.headers.Clone()
	resp, err := conn.client.Do(req)
	if err == nil {
		resp.Body.Close()
	}
}

func (conn *mcpHTTPConnection) request(ctx context.Context, body jsonRPCRequest) (json.RawMessage, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, conn.endpoint, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header = conn.headers.Clone()
	resp, err := conn.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("MCP 服务连接失败: %w", err)
	}
	defer resp.Body.Close()
	if id := resp.Header.Get("Mcp-Session-Id"); id != "" {
		conn.session = id
		conn.headers.Set("Mcp-Session-Id", id)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("MCP 服务返回 HTTP %d", resp.StatusCode)
	}
	if body.ID == nil {
		return nil, nil
	}
	decode := func(raw []byte) (json.RawMessage, bool, error) {
		var result jsonRPCResponse
		if err := json.Unmarshal(raw, &result); err != nil {
			return nil, false, err
		}
		if fmt.Sprint(result.ID) != fmt.Sprint(body.ID) {
			return nil, false, nil
		}
		if result.Error != nil {
			return nil, true, fmt.Errorf("MCP RPC 错误 [%d]: %s", result.Error.Code, result.Error.Message)
		}
		if len(result.Result) == 0 {
			return nil, true, fmt.Errorf("MCP 响应缺少结果")
		}
		return result.Result, true, nil
	}
	const maxResponse = 4 << 20
	reader := io.LimitReader(resp.Body, maxResponse+1)
	if strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 4096), maxResponse)
		var data []string
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" && len(data) > 0 {
				result, matched, err := decode([]byte(strings.Join(data, "\n")))
				if err != nil || matched {
					return result, err
				}
				data = nil
			}
			if strings.HasPrefix(line, "data:") {
				data = append(data, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("MCP 事件流在返回结果前中断")
	}
	raw, err = io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if len(raw) > maxResponse {
		return nil, fmt.Errorf("MCP 响应超过 4MB 上限")
	}
	result, matched, err := decode(raw)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("MCP 响应 ID 不匹配")
	}
	return result, nil
}

// ConvertMCPToolsToDefinitions 将 MCP 缓存工具列表转换为 AI 统一工具定义
func ConvertMCPToolsToDefinitions(serverCode string, cachedTools datatypes.JSON) []ai.ToolDefinition {
	var tools []MCPToolItem
	if len(cachedTools) == 0 {
		return nil
	}
	if err := json.Unmarshal(cachedTools, &tools); err != nil {
		return nil
	}

	var res []ai.ToolDefinition
	for _, t := range tools {
		authKey := fmt.Sprintf("mcp:%s:%s", serverCode, t.Name)
		res = append(res, ai.ToolDefinition{
			Type: "function",
			Function: ai.FunctionSpec{
				Name:        authKey,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		})
	}
	return res
}
