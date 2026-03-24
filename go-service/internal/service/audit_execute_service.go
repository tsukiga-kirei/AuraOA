package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/crypto"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	jwtpkg "oa-smart-audit/go-service/internal/pkg/jwt"
	"oa-smart-audit/go-service/internal/pkg/oa"
	"oa-smart-audit/go-service/internal/repository"
)

const batchAuditMaxLimit = 10

// AuditExecuteService 审核执行业务逻辑：串联 OA 数据 → 提示词构建 → AI 调用 → 结果解析 → 写入日志。
type AuditExecuteService struct {
	auditLogRepo    *repository.AuditLogRepo
	configRepo      *repository.ProcessAuditConfigRepo
	ruleRepo        *repository.AuditRuleRepo
	userConfigRepo  *repository.UserPersonalConfigRepo
	tenantRepo      *repository.TenantRepo
	oaConnRepo      *repository.OAConnectionRepo
	aiModelRepo     *repository.AIModelRepo
	aiCaller        *AIModelCallerService
	db              *gorm.DB
}

func NewAuditExecuteService(
	auditLogRepo *repository.AuditLogRepo,
	configRepo *repository.ProcessAuditConfigRepo,
	ruleRepo *repository.AuditRuleRepo,
	userConfigRepo *repository.UserPersonalConfigRepo,
	tenantRepo *repository.TenantRepo,
	oaConnRepo *repository.OAConnectionRepo,
	aiModelRepo *repository.AIModelRepo,
	aiCaller *AIModelCallerService,
	db *gorm.DB,
) *AuditExecuteService {
	return &AuditExecuteService{
		auditLogRepo:   auditLogRepo,
		configRepo:     configRepo,
		ruleRepo:       ruleRepo,
		userConfigRepo: userConfigRepo,
		tenantRepo:     tenantRepo,
		oaConnRepo:     oaConnRepo,
		aiModelRepo:    aiModelRepo,
		aiCaller:       aiCaller,
		db:             db,
	}
}

// AuditExecuteRequest 审核执行请求
type AuditExecuteRequest struct {
	ProcessID   string `json:"process_id" binding:"required"`
	ProcessType string `json:"process_type" binding:"required"`
	Title       string `json:"title"`
}

// AuditExecuteResponse 审核执行响应
type AuditExecuteResponse struct {
	ID             string                  `json:"id"`
	TraceID        string                  `json:"trace_id"`
	ProcessID      string                  `json:"process_id"`
	Recommendation string                  `json:"recommendation"`
	OverallScore   int                     `json:"overall_score"`
	RuleResults    []model.RuleResultJSON  `json:"rule_results"`
	RiskPoints     []string                `json:"risk_points"`
	Suggestions    []string                `json:"suggestions"`
	Confidence     int                     `json:"confidence"`
	AIReasoning    string                  `json:"ai_reasoning"`
	DurationMs     int                     `json:"duration_ms"`
	CreatedAt      string                  `json:"created_at"`
	ParseError     string                  `json:"parse_error,omitempty"`
	RawContent     string                  `json:"raw_content,omitempty"`
}

// Execute 执行单条审核：OA 数据拉取 → 两阶段 AI 调用 → 解析结果 → 写入 audit_logs。
func (s *AuditExecuteService) Execute(c *gin.Context, req *AuditExecuteRequest) (*AuditExecuteResponse, error) {
	startTime := time.Now()
	tenantID, userID, err := s.extractIDs(c)
	if err != nil {
		return nil, err
	}

	// 1. 获取租户信息和 AI 模型配置
	tenant, err := s.tenantRepo.FindByID(tenantID)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "获取租户信息失败")
	}
	if tenant.PrimaryModelID == nil {
		return nil, newServiceError(errcode.ErrNoAIModelConfig, "租户未配置主用 AI 模型")
	}
	modelCfg, err := s.aiModelRepo.FindByID(*tenant.PrimaryModelID)
	if err != nil {
		return nil, newServiceError(errcode.ErrNoAIModelConfig, "AI 模型配置不存在")
	}

	// 2. 获取流程审核配置
	config, err := s.configRepo.GetByProcessType(c, req.ProcessType)
	if err != nil {
		return nil, newServiceError(errcode.ErrNoProcessConfig, fmt.Sprintf("流程 '%s' 的审核配置不存在", req.ProcessType))
	}

	// 3. 解析 AI 配置
	var aiConfig model.AIConfigData
	if err := json.Unmarshal(config.AIConfig, &aiConfig); err != nil {
		return nil, newServiceError(errcode.ErrInternalServer, "AI 配置解析失败")
	}

	// 4. 获取审核规则（租户级）
	rules, err := s.ruleRepo.ListByConfigID(c, config.ID)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "获取审核规则失败")
	}

	// 5. 获取用户个人配置，合并字段选择 + 规则覆盖
	fieldSet, mergedRulesText := s.resolveUserConfig(c, userID, config, rules, req.ProcessType)

	// 6. 从 OA 拉取流程数据
	processData, err := s.fetchOAData(c, tenant, req.ProcessID)
	if err != nil {
		return nil, err
	}

	// 7. 获取当前节点
	currentNode := "当前节点"

	// 8. 阶段一：推理
	reasoningReq := BuildReasoningPrompt(&aiConfig, req.ProcessType, processData, mergedRulesText, currentNode, fieldSet)
	reasoningReq.Temperature = float64(tenant.Temperature)
	reasoningReq.MaxTokens = tenant.MaxTokensPerRequest
	reasoningReq.ModelConfig = modelCfg

	reasoningResp, err := s.aiCaller.Chat(c, tenantID, userID, modelCfg, reasoningReq)
	if err != nil {
		return nil, err
	}
	aiReasoning := reasoningResp.Content

	// 9. 阶段二：提取
	extractionReq := BuildExtractionPrompt(&aiConfig, aiReasoning, mergedRulesText)
	extractionReq.Temperature = 0.1
	extractionReq.MaxTokens = tenant.MaxTokensPerRequest
	extractionReq.ModelConfig = modelCfg

	extractionResp, err := s.aiCaller.Chat(c, tenantID, userID, modelCfg, extractionReq)
	if err != nil {
		return nil, err
	}

	// 9. 解析 JSON 结果
	totalDuration := int(time.Since(startTime).Milliseconds())
	traceID := fmt.Sprintf("TR-%s-%s", time.Now().Format("20060102150405"), uuid.New().String()[:8])

	parsed, parseErr := ParseAuditResult(extractionResp.Content)

	// 10. 构建审核日志
	logEntry := &model.AuditLog{
		ID:          uuid.New(),
		TenantID:    tenantID,
		UserID:      userID,
		ProcessID:   req.ProcessID,
		Title:       req.Title,
		ProcessType: req.ProcessType,
		DurationMs:  totalDuration,
		AIReasoning: aiReasoning,
		RawContent:  extractionResp.Content,
		CreatedAt:   time.Now(),
	}

	resp := &AuditExecuteResponse{
		ID:          logEntry.ID.String(),
		TraceID:     traceID,
		ProcessID:   req.ProcessID,
		AIReasoning: aiReasoning,
		DurationMs:  totalDuration,
		CreatedAt:   logEntry.CreatedAt.Format(time.RFC3339),
	}

	if parseErr != nil {
		logEntry.Recommendation = "review"
		logEntry.Score = 0
		logEntry.Confidence = 0
		logEntry.ParseError = parseErr.Error()
		logEntry.AuditResult = datatypes.JSON([]byte("{}"))
		resp.Recommendation = "review"
		resp.OverallScore = 0
		resp.Confidence = 0
		resp.ParseError = parseErr.Error()
		resp.RawContent = extractionResp.Content
		resp.RuleResults = []model.RuleResultJSON{}
		resp.RiskPoints = []string{}
		resp.Suggestions = []string{}
	} else {
		resultJSON, _ := json.Marshal(parsed)
		logEntry.Recommendation = parsed.Recommendation
		logEntry.Score = parsed.OverallScore
		logEntry.Confidence = parsed.Confidence
		logEntry.AuditResult = datatypes.JSON(resultJSON)
		resp.Recommendation = parsed.Recommendation
		resp.OverallScore = parsed.OverallScore
		resp.RuleResults = parsed.RuleResults
		resp.RiskPoints = parsed.RiskPoints
		resp.Suggestions = parsed.Suggestions
		resp.Confidence = parsed.Confidence
	}

	// 11. 写入数据库
	if err := s.auditLogRepo.Create(logEntry); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "审核日志写入失败")
	}

	return resp, nil
}

// BatchExecute 批量审核（上限 10 条）。
func (s *AuditExecuteService) BatchExecute(c *gin.Context, items []AuditExecuteRequest) (*BatchAuditResult, error) {
	if len(items) > batchAuditMaxLimit {
		return nil, newServiceError(errcode.ErrBatchLimitExceeded,
			fmt.Sprintf("批量审核上限 %d 条，当前 %d 条", batchAuditMaxLimit, len(items)))
	}

	result := &BatchAuditResult{
		Total: len(items),
	}

	for _, item := range items {
		resp, err := s.Execute(c, &item)
		if err != nil {
			result.Failed++
			result.Results = append(result.Results, AuditExecuteResponse{
				ProcessID:  item.ProcessID,
				ParseError: err.Error(),
			})
			continue
		}
		result.Success++
		result.Results = append(result.Results, *resp)
	}

	return result, nil
}

type BatchAuditResult struct {
	Results []AuditExecuteResponse `json:"results"`
	Total   int                    `json:"total"`
	Success int                    `json:"success"`
	Failed  int                    `json:"failed"`
}

// GetAuditChain 获取审核链：查询租户内该流程所有已完成的审核记录（所有用户）。
func (s *AuditExecuteService) GetAuditChain(c *gin.Context, processID string) ([]model.AuditLog, error) {
	return s.auditLogRepo.ListByProcessID(c, processID)
}

// GetStats 获取审核工作台统计（结合 OA 待办 + 租户配置 + 审核记录）。
func (s *AuditExecuteService) GetStats(c *gin.Context) (map[string]int, error) {
	tenantID, _, err := s.extractIDs(c)
	if err != nil {
		return nil, err
	}
	username := s.extractUsername(c)
	if username == "" {
		return nil, newServiceError(errcode.ErrNoAuthToken, "用户信息缺失")
	}

	adapter, err := s.getOAAdapter(tenantID)
	if err != nil {
		return nil, err
	}

	todoItems, err := adapter.FetchTodoList(c.Request.Context(), username)
	if err != nil {
		return nil, newServiceError(errcode.ErrOAQueryFailed, "获取 OA 待办失败: "+err.Error())
	}

	// 按租户配置的主表名过滤
	allowedTables := s.getAllowedMainTables(c)
	var filtered []oa.TodoItem
	for _, item := range todoItems {
		if allowedTables[strings.ToLower(item.MainTableName)] {
			filtered = append(filtered, item)
		}
	}

	processIDs := make([]string, len(filtered))
	for i, item := range filtered {
		processIDs[i] = item.ProcessID
	}
	auditMap, _ := s.auditLogRepo.GetLatestResultMap(c, processIDs)

	pendingAI, aiDone := 0, 0
	for _, item := range filtered {
		if _, has := auditMap[item.ProcessID]; has {
			aiDone++
		} else {
			pendingAI++
		}
	}

	// completed：有审核记录但不在当前待办中的流程数
	var completedCount int64
	q := s.db.Model(&model.AuditLog{}).Where("tenant_id = ?", tenantID)
	if len(processIDs) > 0 {
		q = q.Where("process_id NOT IN ?", processIDs)
	}
	// 只统计租户配置的流程类型
	configuredTypes := s.getAllowedProcessTypes(c)
	if len(configuredTypes) > 0 {
		q = q.Where("process_type IN ?", configuredTypes)
	}
	q.Select("COUNT(DISTINCT process_id)").Scan(&completedCount)

	return map[string]int{
		"pending_ai_count": pendingAI,
		"ai_done_count":    aiDone,
		"completed_count":  int(completedCount),
	}, nil
}

// ListProcesses 获取审核工作台流程列表（结合 OA 待办 + 租户配置 + AI 审核状态）。
func (s *AuditExecuteService) ListProcesses(c *gin.Context, tab string, username string) ([]map[string]interface{}, error) {
	tenantID, _, err := s.extractIDs(c)
	if err != nil {
		return nil, err
	}

	// 获取 OA 适配器
	adapter, err := s.getOAAdapter(tenantID)
	if err != nil {
		return nil, err
	}

	// 从 OA 拉取用户待办
	todoItems, err := adapter.FetchTodoList(c.Request.Context(), username)
	if err != nil {
		return nil, newServiceError(errcode.ErrOAQueryFailed, "获取 OA 待办失败: "+err.Error())
	}

	// 按租户配置的主表名过滤：只保留租户已配置审核的流程
	allowedTables := s.getAllowedMainTables(c)
	var filteredTodo []oa.TodoItem
	for _, item := range todoItems {
		if allowedTables[strings.ToLower(item.MainTableName)] {
			filteredTodo = append(filteredTodo, item)
		}
	}

	// 获取已有审核记录
	processIDs := make([]string, len(filteredTodo))
	for i, item := range filteredTodo {
		processIDs[i] = item.ProcessID
	}
	auditMap, err := s.auditLogRepo.GetLatestResultMap(c, processIDs)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "查询审核记录失败")
	}

	var results []map[string]interface{}
	for _, item := range filteredTodo {
		record := map[string]interface{}{
			"process_id":         item.ProcessID,
			"title":              item.Title,
			"applicant":          item.Applicant,
			"department":         item.Department,
			"process_type":       item.ProcessType,
			"process_type_label": item.ProcessTypeLabel,
			"current_node":       item.CurrentNode,
			"submit_time":        item.SubmitTime,
			"urgency":            item.Urgency,
			"has_audit":          false,
			"audit_result":       nil,
			"in_todo":            true,
		}

		auditLog, hasAudit := auditMap[item.ProcessID]
		if hasAudit {
			record["has_audit"] = true
			record["audit_result"] = buildAuditResultFromLog(auditLog)
		}

		switch tab {
		case "pending_ai":
			if !hasAudit {
				results = append(results, record)
			}
		case "ai_done":
			if hasAudit {
				results = append(results, record)
			}
		case "completed":
			// completed 在下面单独查询
		}
	}

	// 对 completed tab：查询已有审核记录但不在当前待办的流程
	if tab == "completed" {
		results, err = s.listCompletedProcesses(c, tenantID, username, adapter, processIDs)
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

func (s *AuditExecuteService) listCompletedProcesses(c *gin.Context, tenantID uuid.UUID, username string, adapter oa.OAAdapter, todoProcessIDs []string) ([]map[string]interface{}, error) {
	todoSet := make(map[string]bool)
	for _, id := range todoProcessIDs {
		todoSet[id] = true
	}

	// 查询该租户所有审核日志中不在当前待办里的流程，且属于租户配置的流程类型
	configuredTypes := s.getAllowedProcessTypes(c)
	var logs []model.AuditLog
	query := s.db.Where("tenant_id = ?", tenantID).Order("created_at DESC")
	if len(todoProcessIDs) > 0 {
		query = query.Where("process_id NOT IN ?", todoProcessIDs)
	}
	if len(configuredTypes) > 0 {
		query = query.Where("process_type IN ?", configuredTypes)
	}
	if err := query.Find(&logs).Error; err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "查询已完成审核记录失败")
	}

	seen := make(map[string]bool)
	var results []map[string]interface{}
	for _, log := range logs {
		if seen[log.ProcessID] {
			continue
		}
		seen[log.ProcessID] = true
		results = append(results, map[string]interface{}{
			"process_id":         log.ProcessID,
			"title":              log.Title,
			"applicant":          "",
			"department":         "",
			"process_type":       log.ProcessType,
			"process_type_label": "",
			"current_node":       "已完成",
			"submit_time":        log.CreatedAt.Format("2006-01-02 15:04"),
			"urgency":            "low",
			"has_audit":          true,
			"audit_result":       buildAuditResultFromLog(&log),
			"in_todo":            false,
		})
	}
	return results, nil
}

func buildAuditResultFromLog(log *model.AuditLog) map[string]interface{} {
	result := map[string]interface{}{
		"id":             log.ID.String(),
		"trace_id":       fmt.Sprintf("TR-%s", log.ID.String()[:8]),
		"process_id":     log.ProcessID,
		"recommendation": log.Recommendation,
		"overall_score":  log.Score,
		"confidence":     log.Confidence,
		"ai_reasoning":   log.AIReasoning,
		"duration_ms":    log.DurationMs,
		"created_at":     log.CreatedAt.Format(time.RFC3339),
	}

	if log.ParseError != "" {
		result["parse_error"] = log.ParseError
		result["raw_content"] = log.RawContent
		result["rule_results"] = []interface{}{}
		result["risk_points"] = []string{}
		result["suggestions"] = []string{}
	} else {
		var parsed model.AuditResultJSON
		if err := json.Unmarshal(log.AuditResult, &parsed); err == nil {
			result["rule_results"] = parsed.RuleResults
			result["risk_points"] = parsed.RiskPoints
			result["suggestions"] = parsed.Suggestions
		} else {
			result["rule_results"] = []interface{}{}
			result["risk_points"] = []string{}
			result["suggestions"] = []string{}
		}
	}
	return result
}

// getAllowedMainTables 获取当前租户所有启用的流程审核配置的主表名集合（小写），用于过滤 OA 待办。
func (s *AuditExecuteService) getAllowedMainTables(c *gin.Context) map[string]bool {
	configs, err := s.configRepo.ListByTenant(c)
	if err != nil {
		return map[string]bool{}
	}
	m := make(map[string]bool, len(configs))
	for _, cfg := range configs {
		if cfg.Status == "active" && cfg.MainTableName != "" {
			m[strings.ToLower(cfg.MainTableName)] = true
		}
	}
	return m
}

// getAllowedProcessTypes 获取当前租户所有启用的流程类型名称列表。
func (s *AuditExecuteService) getAllowedProcessTypes(c *gin.Context) []string {
	configs, err := s.configRepo.ListByTenant(c)
	if err != nil {
		return nil
	}
	var types []string
	for _, cfg := range configs {
		if cfg.Status == "active" {
			types = append(types, cfg.ProcessType)
		}
	}
	return types
}

func (s *AuditExecuteService) decryptOAConn(conn *model.OADatabaseConnection) error {
	password, err := crypto.Decrypt(conn.Password)
	if err != nil {
		return newServiceError(errcode.ErrInternalServer, "OA 数据库密码解密失败")
	}
	conn.Password = password
	return nil
}

func (s *AuditExecuteService) getOAAdapter(tenantID uuid.UUID) (oa.OAAdapter, error) {
	tenant, err := s.tenantRepo.FindByID(tenantID)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "获取租户失败")
	}
	if tenant.OADBConnectionID == nil {
		return nil, newServiceError(errcode.ErrOAConnectionFailed, "租户未配置 OA 数据库连接")
	}
	conn, err := s.oaConnRepo.FindByID(*tenant.OADBConnectionID)
	if err != nil {
		return nil, newServiceError(errcode.ErrOAConnectionFailed, "OA 数据库连接配置不存在")
	}
	if err := s.decryptOAConn(conn); err != nil {
		return nil, err
	}
	adapter, err := oa.NewOAAdapter(conn.OAType, conn)
	if err != nil {
		return nil, newServiceError(errcode.ErrOAConnectionFailed, "创建 OA 适配器失败: "+err.Error())
	}
	return adapter, nil
}

func (s *AuditExecuteService) fetchOAData(c *gin.Context, tenant *model.Tenant, processID string) (*oa.ProcessData, error) {
	if tenant.OADBConnectionID == nil {
		return nil, newServiceError(errcode.ErrOAConnectionFailed, "租户未配置 OA 数据库连接")
	}
	conn, err := s.oaConnRepo.FindByID(*tenant.OADBConnectionID)
	if err != nil {
		return nil, newServiceError(errcode.ErrOAConnectionFailed, "OA 数据库连接配置不存在")
	}
	if err := s.decryptOAConn(conn); err != nil {
		return nil, err
	}
	adapter, err := oa.NewOAAdapter(conn.OAType, conn)
	if err != nil {
		return nil, newServiceError(errcode.ErrOAConnectionFailed, "创建 OA 适配器失败: "+err.Error())
	}
	data, err := adapter.FetchProcessData(c.Request.Context(), processID)
	if err != nil {
		return nil, newServiceError(errcode.ErrOAQueryFailed, "拉取 OA 流程数据失败: "+err.Error())
	}
	return data, nil
}

func (s *AuditExecuteService) extractIDs(c *gin.Context) (uuid.UUID, uuid.UUID, error) {
	tidVal, exists := c.Get("tenant_id")
	if !exists {
		return uuid.Nil, uuid.Nil, newServiceError(errcode.ErrNoAuthToken, "租户ID缺失")
	}
	tenantID, err := uuid.Parse(fmt.Sprintf("%v", tidVal))
	if err != nil {
		return uuid.Nil, uuid.Nil, newServiceError(errcode.ErrNoAuthToken, "租户ID格式无效")
	}

	claimsVal, _ := c.Get("jwt_claims")
	claims, ok := claimsVal.(*jwtpkg.JWTClaims)
	if !ok {
		return uuid.Nil, uuid.Nil, newServiceError(errcode.ErrNoAuthToken, "用户认证信息缺失")
	}
	userID, err := uuid.Parse(claims.Sub)
	if err != nil {
		return uuid.Nil, uuid.Nil, newServiceError(errcode.ErrNoAuthToken, "用户ID格式无效")
	}

	return tenantID, userID, nil
}

func (s *AuditExecuteService) extractUsername(c *gin.Context) string {
	claimsVal, _ := c.Get("jwt_claims")
	if claims, ok := claimsVal.(*jwtpkg.JWTClaims); ok {
		return claims.Username
	}
	return ""
}

func isRuleEnabled(r *model.AuditRule) bool {
	return r.Enabled == nil || *r.Enabled
}

func formatRules(rules []model.AuditRule) string {
	if len(rules) == 0 {
		return "（无审核规则）"
	}
	var sb strings.Builder
	for i, r := range rules {
		if !isRuleEnabled(&r) {
			continue
		}
		sb.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, r.RuleScope, r.RuleContent))
	}
	if sb.Len() == 0 {
		return "（无启用的审核规则）"
	}
	return sb.String()
}

// resolveUserConfig 解析租户流程配置 + 用户个人配置，返回最终字段集和规则文本。
//
// 核心原则：以租户配置为权威来源，用户个人配置是"锦上添花"。
//   - 租户已删除的字段/规则 → 自动忽略用户中的对应残留（读时过滤）
//   - 租户关闭 AllowCustomFields → 用户 field_overrides 不生效
//   - 租户关闭 AllowCustomRules → 用户 custom_rules 不生效
//   - mandatory 规则 → 始终强制启用，无论租户 Enabled 或用户 override
func (s *AuditExecuteService) resolveUserConfig(
	c *gin.Context, userID uuid.UUID,
	config *model.ProcessAuditConfig,
	tenantRules []model.AuditRule,
	processType string,
) (SelectedFieldSet, string) {
	// 解析租户权限配置
	var perms model.UserPermissionsData
	if err := json.Unmarshal(config.UserPermissions, &perms); err != nil {
		perms = model.UserPermissionsData{
			AllowCustomFields: true, AllowCustomRules: true, AllowModifyStrictness: true,
		}
	}

	// 获取用户个人配置
	var userDetail *model.AuditDetailItem
	userCfg, _ := s.userConfigRepo.GetByUserID(c, userID)
	if userCfg != nil {
		var items []model.AuditDetailItem
		_ = json.Unmarshal(userCfg.AuditDetails, &items)
		for i := range items {
			if items[i].ProcessType == processType || items[i].ConfigID == config.ID {
				userDetail = &items[i]
				break
			}
		}
	}

	// ── 字段解析 ──
	fieldSet := s.resolveFieldSet(config, userDetail, perms)

	// ── 规则解析 ──
	rulesText := s.resolveRulesText(tenantRules, userDetail, perms)

	return fieldSet, rulesText
}

func (s *AuditExecuteService) resolveFieldSet(
	config *model.ProcessAuditConfig,
	userDetail *model.AuditDetailItem,
	perms model.UserPermissionsData,
) SelectedFieldSet {
	if config.FieldMode == "all" {
		return nil
	}

	type fieldItem struct {
		FieldKey string `json:"field_key"`
		Selected bool   `json:"selected"`
	}
	type detailTableItem struct {
		TableName string      `json:"table_name"`
		Fields    []fieldItem `json:"fields"`
	}

	var mainFields []fieldItem
	var detailTables []detailTableItem
	_ = json.Unmarshal(config.MainFields, &mainFields)
	_ = json.Unmarshal(config.DetailTables, &detailTables)

	// 构建租户字段索引：只有租户当前字段列表中存在的 key 才有资格被选中
	tenantFieldIndex := make(map[string]map[string]bool) // table -> fieldKey -> exists
	tenantFieldIndex["main"] = make(map[string]bool)
	for _, f := range mainFields {
		tenantFieldIndex["main"][f.FieldKey] = true
	}
	for _, dt := range detailTables {
		tenantFieldIndex[dt.TableName] = make(map[string]bool)
		for _, f := range dt.Fields {
			tenantFieldIndex[dt.TableName][f.FieldKey] = true
		}
	}

	// 用户额外字段：仅在权限允许且字段仍存在于租户列表时生效
	userAddedMap := make(map[string]map[string]bool)
	if userDetail != nil && perms.AllowCustomFields {
		for _, key := range userDetail.FieldConfig.FieldOverrides {
			parts := strings.SplitN(key, ":", 2)
			if len(parts) != 2 {
				continue
			}
			table, field := parts[0], parts[1]
			if tenantFieldIndex[table] == nil || !tenantFieldIndex[table][field] {
				continue
			}
			if userAddedMap[table] == nil {
				userAddedMap[table] = make(map[string]bool)
			}
			userAddedMap[table][field] = true
		}
	}

	fieldSet := make(SelectedFieldSet)

	mainSet := make(map[string]bool)
	for _, f := range mainFields {
		if f.Selected || (userAddedMap["main"] != nil && userAddedMap["main"][f.FieldKey]) {
			mainSet[strings.ToLower(f.FieldKey)] = true
		}
	}
	if len(mainSet) > 0 {
		fieldSet["main"] = mainSet
	}

	for _, dt := range detailTables {
		dtSet := make(map[string]bool)
		for _, f := range dt.Fields {
			if f.Selected || (userAddedMap[dt.TableName] != nil && userAddedMap[dt.TableName][f.FieldKey]) {
				dtSet[strings.ToLower(f.FieldKey)] = true
			}
		}
		if len(dtSet) > 0 {
			fieldSet[dt.TableName] = dtSet
		}
	}

	return fieldSet
}

func (s *AuditExecuteService) resolveRulesText(
	tenantRules []model.AuditRule,
	userDetail *model.AuditDetailItem,
	perms model.UserPermissionsData,
) string {
	// 构建用户规则开关覆盖 map（仅引用仍存在于租户规则中的 ID）
	tenantRuleIDs := make(map[string]bool, len(tenantRules))
	for _, r := range tenantRules {
		tenantRuleIDs[r.ID.String()] = true
	}

	toggleMap := make(map[string]bool)
	var customRules []model.CustomRule
	if userDetail != nil {
		for _, t := range userDetail.RuleConfig.RuleToggleOverrides {
			if !tenantRuleIDs[t.RuleID] {
				continue
			}
			toggleMap[t.RuleID] = t.Enabled
		}
		customRules = userDetail.RuleConfig.CustomRules
	}

	var sb strings.Builder
	idx := 1

	for _, r := range tenantRules {
		// mandatory 规则始终强制启用
		if r.RuleScope == "mandatory" {
			sb.WriteString(fmt.Sprintf("%d. [%s] %s\n", idx, r.RuleScope, r.RuleContent))
			idx++
			continue
		}

		// 非 mandatory：先取租户默认 Enabled，再看用户覆盖
		enabled := isRuleEnabled(&r)
		if override, ok := toggleMap[r.ID.String()]; ok {
			enabled = override
		}
		if !enabled {
			continue
		}
		sb.WriteString(fmt.Sprintf("%d. [%s] %s\n", idx, r.RuleScope, r.RuleContent))
		idx++
	}

	// 用户自定义规则：仅在权限允许时追加
	if perms.AllowCustomRules {
		for _, cr := range customRules {
			if !cr.Enabled {
				continue
			}
			sb.WriteString(fmt.Sprintf("%d. [用户自定义] %s\n", idx, cr.Content))
			idx++
		}
	}

	if sb.Len() == 0 {
		return "（无启用的审核规则）"
	}
	return sb.String()
}
