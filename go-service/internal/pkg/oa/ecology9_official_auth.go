package oa

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"auraoa/go-service/internal/pkg/apptime"
	pkglogger "auraoa/go-service/internal/pkg/logger"
)

var (
	defaultClientRSAKey *rsa.PrivateKey
	defaultClientCPK    string
	initRSAOnce         sync.Once
)

func initClientRSA() {
	initRSAOnce.Do(func() {
		key, err := rsa.GenerateKey(rand.Reader, 1024)
		if err != nil {
			pkglogger.Global().Error("初始化 Ecology RSA 客户端密钥失败", zap.Error(err))
			return
		}
		defaultClientRSAKey = key

		derBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
		if err != nil {
			pkglogger.Global().Error("序列化 Ecology RSA 客户端公钥失败", zap.Error(err))
			return
		}
		defaultClientCPK = base64.StdEncoding.EncodeToString(derBytes)
	})
}

// EcologyCachedToken 缓存的 Ecology Token 与服务端 RSA 公钥
type EcologyCachedToken struct {
	Token     string
	SPK       string
	SPKKey    *rsa.PublicKey
	Secret    string
	ExpiresAt time.Time
}

// EcologyOfficialAuthService 泛微 Ecology 9 官方 OpenAPI 鉴权与调用通用服务
type EcologyOfficialAuthService struct {
	httpClient *http.Client
	cacheMu    sync.RWMutex
	tokenCache map[string]*EcologyCachedToken
}

var (
	globalEcologyAuthService *EcologyOfficialAuthService
	authServiceOnce          sync.Once
)

// GetEcologyOfficialAuthService 获取单例服务
func GetEcologyOfficialAuthService() *EcologyOfficialAuthService {
	authServiceOnce.Do(func() {
		initClientRSA()
		globalEcologyAuthService = &EcologyOfficialAuthService{
			httpClient: &http.Client{
				Timeout: 30 * time.Second,
			},
			tokenCache: make(map[string]*EcologyCachedToken),
		}
	})
	return globalEcologyAuthService
}

func (s *EcologyOfficialAuthService) cacheKey(apiURL, appID string) string {
	cleanURL := strings.TrimRight(strings.TrimSpace(apiURL), "/")
	return fmt.Sprintf("%s|%s", cleanURL, strings.TrimSpace(appID))
}

// ParseRSAPublicKey 解析泛微返回的服务端公钥（兼容 PKIX、PKCS#1、PEM 以及裸 Base64 格式）
func ParseRSAPublicKey(spkStr string) (*rsa.PublicKey, error) {
	clean := strings.TrimSpace(spkStr)
	if clean == "" {
		return nil, errors.New("服务端公钥为空")
	}

	var der []byte
	if strings.Contains(clean, "-----BEGIN") {
		block, _ := pem.Decode([]byte(clean))
		if block == nil {
			return nil, errors.New("无法解析 PEM 格式的服务端公钥")
		}
		der = block.Bytes
	} else {
		// 尝试去除换行后直接 Base64 解码
		normalized := strings.ReplaceAll(clean, "\r", "")
		normalized = strings.ReplaceAll(normalized, "\n", "")
		normalized = strings.ReplaceAll(normalized, " ", "")
		decoded, err := base64.StdEncoding.DecodeString(normalized)
		if err != nil {
			return nil, fmt.Errorf("Base64 解码服务端公钥失败: %w", err)
		}
		der = decoded
	}

	// 1. 尝试 PKIX (X.509)
	pubInterface, err := x509.ParsePKIXPublicKey(der)
	if err == nil {
		if rsaKey, ok := pubInterface.(*rsa.PublicKey); ok {
			return rsaKey, nil
		}
	}

	// 2. 尝试 PKCS#1
	rsaKey, errPKCS1 := x509.ParsePKCS1PublicKey(der)
	if errPKCS1 == nil {
		return rsaKey, nil
	}

	return nil, fmt.Errorf("解析 RSA 公钥失败（PKIX: %v, PKCS#1: %v）", err, errPKCS1)
}

// EncryptWithSPK 使用泛微服务端 RSA 公钥进行 PKCS#1 v1.5 加密并输出单行 Base64
func EncryptWithSPK(spkKey *rsa.PublicKey, plainText string) (string, error) {
	if spkKey == nil {
		return "", errors.New("服务端公钥为空")
	}
	cipherBytes, err := rsa.EncryptPKCS1v15(rand.Reader, spkKey, []byte(plainText))
	if err != nil {
		return "", fmt.Errorf("RSA 加密失败: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(cipherBytes)
	encoded = strings.ReplaceAll(encoded, "\r", "")
	encoded = strings.ReplaceAll(encoded, "\n", "")
	return encoded, nil
}

// RegistAndApplyToken 调用官方 /api/ec/dev/auth/regist 与 /api/ec/dev/auth/applytoken 获取可用 Token
func (s *EcologyOfficialAuthService) RegistAndApplyToken(
	ctx context.Context,
	apiURL, appID, configuredSecret string,
) (*EcologyCachedToken, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(apiURL), "/")
	if baseURL == "" {
		return nil, errors.New("泛微 API 地址 (weaver_api_url) 为空")
	}
	if strings.TrimSpace(appID) == "" {
		return nil, errors.New("泛微 appid (weaver_appid) 为空")
	}

	initClientRSA()
	cpk := defaultClientCPK

	// 阶段一：调用 regist 接口获取公钥 spk 和注册密钥 secret
	registURL := baseURL + "/api/ec/dev/auth/regist"
	reqRegist, err := http.NewRequestWithContext(ctx, http.MethodPost, registURL, nil)
	if err != nil {
		return nil, fmt.Errorf("构建 regist 请求失败: %w", err)
	}
	reqRegist.Header.Set("appid", appID)
	reqRegist.Header.Set("cpk", cpk)

	respRegist, err := s.httpClient.Do(reqRegist)
	if err != nil {
		return nil, fmt.Errorf("请求泛微 regist 接口失败: %w", err)
	}
	defer respRegist.Body.Close()

	bodyBytes, err := io.ReadAll(respRegist.Body)
	if err != nil {
		return nil, fmt.Errorf("读取 regist 响应失败: %w", err)
	}

	var registResult struct {
		Status  interface{} `json:"status"`
		Code    interface{} `json:"code"`
		SPK     string      `json:"spk"`
		Secret  string      `json:"secret"`
		Secrit  string      `json:"secrit"`
		Msg     string      `json:"msg"`
		Message string      `json:"message"`
	}
	if err := json.Unmarshal(bodyBytes, &registResult); err != nil {
		return nil, fmt.Errorf("解析 regist 响应 JSON 失败 (%s): %w", string(bodyBytes), err)
	}

	// 判断注册接口是否成功
	isSuccess := false
	if b, ok := registResult.Status.(bool); ok && b {
		isSuccess = true
	} else if str, ok := registResult.Status.(string); ok && (strings.EqualFold(str, "true") || str == "0") {
		isSuccess = true
	} else if c, ok := registResult.Code.(string); ok && (c == "0" || strings.EqualFold(c, "SUCCESS")) {
		isSuccess = true
	} else if registResult.SPK != "" {
		// 部分 E9 实例返回体直接包含 spk
		isSuccess = true
	}

	if !isSuccess && registResult.SPK == "" {
		errMsg := registResult.Msg
		if errMsg == "" {
			errMsg = registResult.Message
		}
		if errMsg == "" {
			errMsg = string(bodyBytes)
		}
		return nil, fmt.Errorf("泛微 regist 注册失败: %s", errMsg)
	}

	spkKey, err := ParseRSAPublicKey(registResult.SPK)
	if err != nil {
		return nil, fmt.Errorf("解析泛微服务端公钥 spk 失败: %w", err)
	}

	// 提取密钥（优先响应体中的 secret / secrit，若无则使用配置的 weaver_app_secret）
	secret := strings.TrimSpace(registResult.Secret)
	if secret == "" {
		secret = strings.TrimSpace(registResult.Secrit)
	}
	if secret == "" {
		secret = strings.TrimSpace(configuredSecret)
	}
	if secret == "" {
		return nil, errors.New("泛微 regist 未返回密钥且未配置 weaver_app_secret")
	}

	// 阶段二：用 spk 对 secret 进行 RSA 加密，换取 Token
	encryptedSecret, err := EncryptWithSPK(spkKey, secret)
	if err != nil {
		return nil, fmt.Errorf("加密 secret 凭证失败: %w", err)
	}

	applyTokenURL := baseURL + "/api/ec/dev/auth/applytoken"
	reqToken, err := http.NewRequestWithContext(ctx, http.MethodPost, applyTokenURL, nil)
	if err != nil {
		return nil, fmt.Errorf("构建 applytoken 请求失败: %w", err)
	}
	reqToken.Header.Set("appid", appID)
	reqToken.Header.Set("secret", encryptedSecret)
	reqToken.Header.Set("time", "1800")

	respToken, err := s.httpClient.Do(reqToken)
	if err != nil {
		return nil, fmt.Errorf("请求泛微 applytoken 接口失败: %w", err)
	}
	defer respToken.Body.Close()

	tokenBody, err := io.ReadAll(respToken.Body)
	if err != nil {
		return nil, fmt.Errorf("读取 applytoken 响应失败: %w", err)
	}

	var tokenResult struct {
		Status  interface{} `json:"status"`
		Code    interface{} `json:"code"`
		Token   string      `json:"token"`
		Msg     string      `json:"msg"`
		Message string      `json:"message"`
	}
	if err := json.Unmarshal(tokenBody, &tokenResult); err != nil {
		return nil, fmt.Errorf("解析 applytoken 响应 JSON 失败 (%s): %w", string(tokenBody), err)
	}

	token := strings.TrimSpace(tokenResult.Token)
	if token == "" {
		errMsg := tokenResult.Msg
		if errMsg == "" {
			errMsg = tokenResult.Message
		}
		if errMsg == "" {
			errMsg = string(tokenBody)
		}
		return nil, fmt.Errorf("泛微 applytoken 获取失败: %s", errMsg)
	}

	cached := &EcologyCachedToken{
		Token:     token,
		SPK:       registResult.SPK,
		SPKKey:    spkKey,
		Secret:    secret,
		ExpiresAt: apptime.Now().Add(25 * time.Minute), // 官方 30 分钟，提前 5 分钟换新
	}

	s.cacheMu.Lock()
	s.tokenCache[s.cacheKey(apiURL, appID)] = cached
	s.cacheMu.Unlock()

	pkglogger.Global().Info("泛微官方 OpenAPI Token 申请成功",
		zap.String("apiURL", baseURL),
		zap.String("appID", appID))

	return cached, nil
}

// GetValidToken 获取有效的 Token 和 SPK 公钥，若快过期自动握手刷新
func (s *EcologyOfficialAuthService) GetValidToken(
	ctx context.Context,
	apiURL, appID, configuredSecret string,
) (*EcologyCachedToken, error) {
	key := s.cacheKey(apiURL, appID)

	s.cacheMu.RLock()
	cached, ok := s.tokenCache[key]
	s.cacheMu.RUnlock()

	// 缓存存在且距离过期超过 5 分钟则直接复用
	if ok && cached != nil && cached.Token != "" && cached.SPKKey != nil {
		if apptime.Now().Before(cached.ExpiresAt) {
			return cached, nil
		}
	}

	// 重新握手申请
	return s.RegistAndApplyToken(ctx, apiURL, appID, configuredSecret)
}

// CreateRequestInput 泛微 doCreateRequest 报文参数
type CreateRequestInput struct {
	WorkflowID     string
	RequestName    string
	RequestLevel   string // "0": 正常
	IsNextFlow     string // 固定 "0" 为草稿
	MainDataJSON   string
	DetailDataJSON string
}

// DoCreateRequest 调用泛微官方 doCreateRequest 接口创建草稿。
// oaUserID 必须是 hrmresource.id 的十进制字符串，调用方先按当前用户登录名查表，再传入。
func (s *EcologyOfficialAuthService) DoCreateRequest(
	ctx context.Context,
	apiURL, appID, configuredSecret, oaUserID string,
	input *CreateRequestInput,
) (string, error) {
	cached, err := s.GetValidToken(ctx, apiURL, appID, configuredSecret)
	if err != nil {
		return "", fmt.Errorf("获取泛微鉴权 Token 失败: %w", err)
	}

	// 官方接口的 userid 是人员数字 ID 经服务端公钥加密后的值，不能直接传登录名。
	encryptedUserID, err := EncryptWithSPK(cached.SPKKey, oaUserID)
	if err != nil {
		return "", fmt.Errorf("加密泛微人员 ID 失败: %w", err)
	}

	baseURL := strings.TrimRight(strings.TrimSpace(apiURL), "/")
	createURL := baseURL + "/api/workflow/paService/doCreateRequest"

	formData := url.Values{}
	formData.Set("workflowId", input.WorkflowID)
	formData.Set("requestName", input.RequestName)
	if input.RequestLevel == "" {
		formData.Set("requestLevel", "0")
	} else {
		formData.Set("requestLevel", input.RequestLevel)
	}
	formData.Set("isnextflow", "0") // 永远固定为 0 草稿模式
	formData.Set("mainData", input.MainDataJSON)
	if input.DetailDataJSON == "" {
		formData.Set("detailData", "[]")
	} else {
		formData.Set("detailData", input.DetailDataJSON)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, createURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return "", fmt.Errorf("构建 doCreateRequest 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	req.Header.Set("appid", appID)
	req.Header.Set("token", cached.Token)
	req.Header.Set("userid", encryptedUserID)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("调用 doCreateRequest 接口失败: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取 doCreateRequest 响应失败: %w", err)
	}

	var result struct {
		Code   interface{} `json:"code"`
		Status interface{} `json:"status"`
		Msg    string      `json:"msg"`
		ErrMsg string      `json:"errMsg"`
		Data   struct {
			RequestID interface{} `json:"requestid"`
		} `json:"data"`
		RequestID interface{} `json:"requestid"`
	}

	if err := json.Unmarshal(respBytes, &result); err != nil {
		return "", fmt.Errorf("解析 doCreateRequest 响应失败 (%s): %w", string(respBytes), err)
	}

	// 检查返回结果
	var reqIDStr string
	if result.Data.RequestID != nil {
		reqIDStr = fmt.Sprint(result.Data.RequestID)
	} else if result.RequestID != nil {
		reqIDStr = fmt.Sprint(result.RequestID)
	}

	// 判断是否成功
	codeStr := strings.ToUpper(fmt.Sprint(result.Code))
	isSuccess := codeStr == "SUCCESS" || codeStr == "0" || codeStr == "200"
	if !isSuccess && reqIDStr != "" && reqIDStr != "0" {
		isSuccess = true
	}

	if !isSuccess || reqIDStr == "" || reqIDStr == "0" {
		errMsg := result.ErrMsg
		if errMsg == "" {
			errMsg = result.Msg
		}
		if errMsg == "" {
			errMsg = string(respBytes)
		}
		return "", fmt.Errorf("泛微流程草稿创建失败: %s", errMsg)
	}

	return reqIDStr, nil
}

// GetCreateWorkflowList 查询指定人员可创建的流程。
// oaUserID 是 hrmresource.id。conditionsJSON 为空时不附加筛选，返回该人员的全部可创建流程。
func (s *EcologyOfficialAuthService) GetCreateWorkflowList(
	ctx context.Context,
	apiURL, appID, configuredSecret, oaUserID, conditionsJSON string,
) ([]CreatableWorkflow, error) {
	cached, err := s.GetValidToken(ctx, apiURL, appID, configuredSecret)
	if err != nil {
		return nil, fmt.Errorf("获取泛微鉴权 Token 失败: %w", err)
	}
	encryptedUserID, err := EncryptWithSPK(cached.SPKKey, strings.TrimSpace(oaUserID))
	if err != nil {
		return nil, fmt.Errorf("加密泛微人员 ID 失败: %w", err)
	}

	formData := url.Values{}
	if strings.TrimSpace(conditionsJSON) != "" {
		formData.Set("conditions", conditionsJSON)
	}

	baseURL := strings.TrimRight(strings.TrimSpace(apiURL), "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/workflow/paService/getCreateWorkflowList", strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("构建可创建流程列表请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	req.Header.Set("appid", appID)
	req.Header.Set("token", cached.Token)
	req.Header.Set("userid", encryptedUserID)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("调用可创建流程列表接口失败: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取可创建流程列表响应失败: %w", err)
	}
	return parseCreatableWorkflowList(respBytes)
}

func parseCreatableWorkflowList(body []byte) ([]CreatableWorkflow, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil, errors.New("可创建流程列表响应为空")
	}
	if strings.HasPrefix(trimmed, "[") {
		return decodeCreatableWorkflowRows([]byte(trimmed))
	}

	var wrapped struct {
		Code    interface{}     `json:"code"`
		Status  interface{}     `json:"status"`
		Msg     string          `json:"msg"`
		ErrMsg  string          `json:"errMsg"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("解析可创建流程列表失败 (%s): %w", truncateOABody(trimmed), err)
	}
	if len(wrapped.Data) > 0 && string(wrapped.Data) != "null" {
		return decodeCreatableWorkflowRows(wrapped.Data)
	}
	errMsg := wrapped.ErrMsg
	if errMsg == "" {
		errMsg = wrapped.Msg
	}
	if errMsg == "" {
		errMsg = wrapped.Message
	}
	if errMsg == "" {
		errMsg = truncateOABody(trimmed)
	}
	return nil, fmt.Errorf("查询可创建流程失败: %s", errMsg)
}

func decodeCreatableWorkflowRows(raw []byte) ([]CreatableWorkflow, error) {
	var rows []map[string]interface{}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("解析可创建流程列表失败: %w", err)
	}
	items := make([]CreatableWorkflow, 0, len(rows))
	for _, row := range rows {
		item := CreatableWorkflow{
			FormID:           oaMapString(row, "formId"),
			WorkflowID:       oaMapString(row, "workflowId"),
			WorkflowName:     oaMapString(row, "workflowName"),
			WorkflowTypeID:   oaMapString(row, "workflowTypeId"),
			WorkflowTypeName: oaMapString(row, "workflowTypeName"),
		}
		if item.WorkflowID == "" {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func oaMapString(row map[string]interface{}, key string) string {
	value, ok := row[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func truncateOABody(body string) string {
	if len(body) > 300 {
		return body[:300] + "..."
	}
	return body
}
