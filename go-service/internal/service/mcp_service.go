package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	ID      interface{} `json:"id"`
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
	server, err := s.agentRepo.GetMCPServerByID(tenantID, serverID)
	if err != nil {
		return nil, fmt.Errorf("未找到指定的 MCP 服务器: %w", err)
	}

	tools, err := s.fetchTools(ctx, server)
	if err != nil {
		return nil, err
	}

	// 缓存工具列表并更新时间戳
	toolsJSON, _ := json.Marshal(tools)
	now := time.Now()
	_ = s.agentRepo.UpdateMCPServer(tenantID, serverID, map[string]interface{}{
		"cached_tools":   datatypes.JSON(toolsJSON),
		"last_synced_at": &now,
	})

	return tools, nil
}

// CallTool 调用指定的 MCP 工具
func (s *MCPService) CallTool(ctx context.Context, tenantID uuid.UUID, serverCode, toolName, argumentsJSON string) (interface{}, error) {
	servers, err := s.agentRepo.ListMCPServers(tenantID)
	if err != nil {
		return nil, err
	}

	var targetServer *model.MCPServer
	for i := range servers {
		if servers[i].ServerCode == serverCode {
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
		args = map[string]interface{}{}
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

	var callResult struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(respRaw, &callResult); err == nil && len(callResult.Content) > 0 {
		return map[string]interface{}{
			"content":  callResult.Content[0].Text,
			"is_error": callResult.IsError,
		}, nil
	}

	var genericResult interface{}
	_ = json.Unmarshal(respRaw, &genericResult)
	return genericResult, nil
}

func (s *MCPService) fetchTools(ctx context.Context, server *model.MCPServer) ([]MCPToolItem, error) {
	reqBody := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
	}

	respRaw, err := s.doRequest(ctx, server, reqBody)
	if err != nil {
		return nil, err
	}

	var listResult struct {
		Tools []MCPToolItem `json:"tools"`
	}
	if err := json.Unmarshal(respRaw, &listResult); err != nil {
		return nil, fmt.Errorf("解析 tools/list 响应失败: %w", err)
	}

	return listResult.Tools, nil
}

func (s *MCPService) doRequest(ctx context.Context, server *model.MCPServer, body interface{}) (json.RawMessage, error) {
	jsonBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, server.EndpointURL, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json, text/event-stream")

	// 解密自定义 Headers
	if server.HeadersEncrypted != "" {
		if decrypted, decErr := crypto.Decrypt(server.HeadersEncrypted); decErr == nil {
			var headers map[string]string
			if err := json.Unmarshal([]byte(decrypted), &headers); err == nil {
				for k, v := range headers {
					httpReq.Header.Set(k, v)
				}
			}
		}
	}

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求 MCP 端点失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respData, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("MCP 端点返回非 200 状态码 (%d): %s", resp.StatusCode, string(respData))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取 MCP 响应失败: %w", err)
	}

	var rpcResp jsonRPCResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("解析 JSON-RPC 格式失败: %w", err)
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("MCP RPC 错误 [%d]: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
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
