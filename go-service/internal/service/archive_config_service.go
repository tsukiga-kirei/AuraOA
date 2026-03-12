package service

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"

	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/crypto"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/pkg/oa"
	"oa-smart-audit/go-service/internal/repository"
)

// ProcessArchiveConfigService 处理归档复盘配置的业务逻辑。
type ProcessArchiveConfigService struct {
	configRepo   *repository.ProcessArchiveConfigRepo
	tenantRepo   *repository.TenantRepo
	oaConnRepo   *repository.OAConnectionRepo
	templateRepo *repository.SystemPromptTemplateRepo
}

// NewProcessArchiveConfigService 创建一个新的 ProcessArchiveConfigService 实例。
func NewProcessArchiveConfigService(
	configRepo *repository.ProcessArchiveConfigRepo,
	tenantRepo *repository.TenantRepo,
	oaConnRepo *repository.OAConnectionRepo,
	templateRepo *repository.SystemPromptTemplateRepo,
) *ProcessArchiveConfigService {
	return &ProcessArchiveConfigService{
		configRepo:   configRepo,
		tenantRepo:   tenantRepo,
		oaConnRepo:   oaConnRepo,
		templateRepo: templateRepo,
	}
}

// Create 创建归档复盘配置。
func (s *ProcessArchiveConfigService) Create(c *gin.Context, req *dto.CreateProcessArchiveConfigRequest) (*model.ProcessArchiveConfig, error) {
	exists, err := s.configRepo.ExistsByProcessType(c, req.ProcessType)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	if exists {
		return nil, newServiceError(errcode.ErrDuplicateProcessType, "该流程类型已存在归档配置")
	}

	tenantID, err := getTenantUUID(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "租户ID无效")
	}

	// 自动初始化 AI 配置（使用标准尺度归档提示词）
	aiConfig := req.AIConfig
	if aiConfig == nil || string(aiConfig) == "{}" || string(aiConfig) == "null" || len(aiConfig) == 0 {
		aiConfig = s.buildDefaultAIConfig("standard")
	} else {
		var parsed model.ArchiveAIConfigData
		if err := json.Unmarshal(aiConfig, &parsed); err == nil {
			if parsed.SystemReasoningPrompt == "" && parsed.UserReasoningPrompt == "" {
				strictness := defaultStr(parsed.AuditStrictness, "standard")
				aiConfig = s.buildDefaultAIConfig(strictness)
			}
		}
	}

	// 初始化 access_control 默认值
	accessControl := req.AccessControl
	if accessControl == nil || string(accessControl) == "null" || len(accessControl) == 0 {
		defaultAC := model.AccessControlData{
			AllowedRoles:       []string{},
			AllowedMembers:     []string{},
			AllowedDepartments: []string{},
		}
		b, _ := json.Marshal(defaultAC)
		accessControl = datatypes.JSON(b)
	}

	cfg := &model.ProcessArchiveConfig{
		ID:               uuid.New(),
		TenantID:         tenantID,
		ProcessType:      req.ProcessType,
		ProcessTypeLabel: req.ProcessTypeLabel,
		MainTableName:    req.MainTableName,
		MainFields:       defaultJSON(req.MainFields, "[]"),
		DetailTables:     defaultJSON(req.DetailTables, "[]"),
		FieldMode:        defaultStr(req.FieldMode, "all"),
		KBMode:           defaultStr(req.KBMode, "rules_only"),
		AIConfig:         defaultJSON(aiConfig, "{}"),
		UserPermissions:  defaultJSON(req.UserPermissions, "{}"),
		AccessControl:    accessControl,
		Status:           defaultStr(req.Status, "active"),
	}

	if err := s.configRepo.Create(c, cfg); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	return cfg, nil
}

// buildDefaultAIConfig 从归档专用（archive_ 前缀）系统提示词模板构建默认 ai_config JSON。
func (s *ProcessArchiveConfigService) buildDefaultAIConfig(strictness string) datatypes.JSON {
	// 查询归档专用模板（prompt_key 以 archive_ 开头，strictness 匹配）
	allTemplates, err := s.templateRepo.ListAll()
	if err != nil {
		fallback, _ := json.Marshal(model.ArchiveAIConfigData{AuditStrictness: strictness})
		return datatypes.JSON(fallback)
	}

	data := model.ArchiveAIConfigData{AuditStrictness: strictness}
	for _, t := range allTemplates {
		// 只处理归档专用模板（archive_ 前缀）且符合目标尺度的
		if t.Strictness == nil || *t.Strictness != strictness {
			continue
		}
		key := t.PromptKey
		if !strings.HasPrefix(key, "archive_") {
			continue
		}
		switch {
		case t.PromptType == "system" && t.Phase == "reasoning":
			data.SystemReasoningPrompt = t.Content
		case t.PromptType == "system" && t.Phase == "extraction":
			data.SystemExtractionPrompt = t.Content
		case t.PromptType == "user" && t.Phase == "reasoning":
			data.UserReasoningPrompt = t.Content
		case t.PromptType == "user" && t.Phase == "extraction":
			data.UserExtractionPrompt = t.Content
		}
	}

	result, _ := json.Marshal(data)
	return datatypes.JSON(result)
}

// ListArchivePromptTemplates 返回归档复盘专用的系统提示词模板（archive_ 前缀）。
func (s *ProcessArchiveConfigService) ListArchivePromptTemplates() ([]model.SystemPromptTemplate, error) {
	all, err := s.templateRepo.ListAll()
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	var filtered []model.SystemPromptTemplate
	for _, t := range all {
		if strings.HasPrefix(t.PromptKey, "archive_") {
			filtered = append(filtered, t)
		}
	}
	return filtered, nil
}

// List 查询当前租户的所有归档复盘配置。
func (s *ProcessArchiveConfigService) List(c *gin.Context) ([]model.ProcessArchiveConfig, error) {
	cfgs, err := s.configRepo.ListByTenant(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	return cfgs, nil
}

// GetByID 通过 ID 查询单个归档复盘配置。
func (s *ProcessArchiveConfigService) GetByID(c *gin.Context, id uuid.UUID) (*model.ProcessArchiveConfig, error) {
	cfg, err := s.configRepo.GetByID(c, id)
	if err != nil {
		return nil, newServiceError(errcode.ErrConfigNotFound, "归档复盘配置不存在")
	}
	return cfg, nil
}

// Update 更新归档复盘配置。
func (s *ProcessArchiveConfigService) Update(c *gin.Context, id uuid.UUID, req *dto.UpdateProcessArchiveConfigRequest) (*model.ProcessArchiveConfig, error) {
	_, err := s.configRepo.GetByID(c, id)
	if err != nil {
		return nil, newServiceError(errcode.ErrConfigNotFound, "归档复盘配置不存在")
	}

	fields := make(map[string]interface{})
	if req.ProcessType != "" {
		fields["process_type"] = req.ProcessType
	}
	if req.ProcessTypeLabel != "" {
		fields["process_type_label"] = req.ProcessTypeLabel
	}
	if req.MainTableName != "" {
		fields["main_table_name"] = req.MainTableName
	}
	if req.MainFields != nil {
		fields["main_fields"] = req.MainFields
	}
	if req.DetailTables != nil {
		fields["detail_tables"] = req.DetailTables
	}
	if req.FieldMode != "" {
		fields["field_mode"] = req.FieldMode
	}
	if req.KBMode != "" {
		fields["kb_mode"] = req.KBMode
	}
	if req.AIConfig != nil {
		fields["ai_config"] = req.AIConfig
	}
	if req.UserPermissions != nil {
		fields["user_permissions"] = req.UserPermissions
	}
	if req.AccessControl != nil {
		fields["access_control"] = req.AccessControl
	}
	if req.Status != "" {
		fields["status"] = req.Status
	}

	if len(fields) > 0 {
		if err := s.configRepo.UpdateFields(c, id, fields); err != nil {
			return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
		}
	}

	cfg, err := s.configRepo.GetByID(c, id)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	return cfg, nil
}

// Delete 删除归档复盘配置。
func (s *ProcessArchiveConfigService) Delete(c *gin.Context, id uuid.UUID) error {
	_, err := s.configRepo.GetByID(c, id)
	if err != nil {
		return newServiceError(errcode.ErrConfigNotFound, "归档复盘配置不存在")
	}
	if err := s.configRepo.Delete(c, id); err != nil {
		return newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	return nil
}

// TestConnection 测试 OA 流程连接（与审核工作台共用 OA 适配器逻辑）。
func (s *ProcessArchiveConfigService) TestConnection(c *gin.Context, req *dto.TestConnectionRequest) (*oa.ProcessInfo, error) {
	adapter, err := s.getOAAdapter(c)
	if err != nil {
		return nil, err
	}

	info, err := adapter.ValidateProcess(c.Request.Context(), req.ProcessType)
	if err != nil {
		return nil, newServiceError(errcode.ErrProcessNotFound, "流程在OA系统中不存在: "+err.Error())
	}

	if req.MainTableName != "" && !strings.EqualFold(req.MainTableName, info.MainTable) {
		info.TableMismatch = true
		info.ExpectedTable = info.MainTable
	}
	if req.ProcessTypeLabel != "" && !strings.EqualFold(req.ProcessTypeLabel, info.ProcessTypeLabel) {
		info.TypeLabelMismatch = true
		info.ExpectedTypeLabel = info.ProcessTypeLabel
	}

	return info, nil
}

// FetchFields 从 OA 系统拉取字段定义并持久化。
func (s *ProcessArchiveConfigService) FetchFields(c *gin.Context, id uuid.UUID) (*oa.ProcessFields, error) {
	cfg, err := s.configRepo.GetByID(c, id)
	if err != nil {
		return nil, newServiceError(errcode.ErrConfigNotFound, "归档复盘配置不存在")
	}

	adapter, err := s.getOAAdapter(c)
	if err != nil {
		return nil, err
	}

	fields, err := adapter.FetchFields(c.Request.Context(), cfg.ProcessType)
	if err != nil {
		return nil, newServiceError(errcode.ErrOAQueryFailed, "OA字段拉取失败: "+err.Error())
	}

	mainFieldsJSON, _ := json.Marshal(fields.MainFields)
	detailTablesJSON, _ := json.Marshal(fields.DetailTables)

	if err := s.configRepo.UpdateFields(c, id, map[string]interface{}{
		"main_fields":   datatypes.JSON(mainFieldsJSON),
		"detail_tables": datatypes.JSON(detailTablesJSON),
	}); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	return fields, nil
}

// getOAAdapter 获取当前租户的 OA 适配器实例（与审核配置共用逻辑）。
func (s *ProcessArchiveConfigService) getOAAdapter(c *gin.Context) (oa.OAAdapter, error) {
	tenantID, err := getTenantUUID(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "租户ID无效")
	}

	tenant, err := s.tenantRepo.FindByID(tenantID)
	if err != nil {
		return nil, newServiceError(errcode.ErrConfigNotFound, "租户不存在")
	}

	if tenant.OADBConnectionID == nil {
		return nil, newServiceError(errcode.ErrOAConnectionFailed, "租户未配置OA数据库连接")
	}

	conn, err := s.oaConnRepo.FindByID(*tenant.OADBConnectionID)
	if err != nil {
		return nil, newServiceError(errcode.ErrOAConnectionFailed, "OA数据库连接不存在")
	}

	password, err := crypto.Decrypt(conn.Password)
	if err != nil {
		return nil, newServiceError(errcode.ErrOAConnectionFailed, "OA数据库密码解密失败")
	}
	conn.Password = password

	adapter, err := oa.NewOAAdapter(conn.OAType, conn)
	if err != nil {
		return nil, newServiceError(errcode.ErrOATypeUnsupported, err.Error())
	}

	return adapter, nil
}

// ====== ArchiveRuleService ======

// ArchiveRuleService 处理归档规则的业务逻辑。
type ArchiveRuleService struct {
	ruleRepo *repository.ArchiveRuleRepo
}

// NewArchiveRuleService 创建一个新的 ArchiveRuleService 实例。
func NewArchiveRuleService(ruleRepo *repository.ArchiveRuleRepo) *ArchiveRuleService {
	return &ArchiveRuleService{ruleRepo: ruleRepo}
}

// Create 创建归档规则。
func (s *ArchiveRuleService) Create(c *gin.Context, req *dto.CreateArchiveRuleRequest) (*model.ArchiveRule, error) {
	tenantID, err := getTenantUUID(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "租户ID无效")
	}

	rule := &model.ArchiveRule{
		ID:          uuid.New(),
		TenantID:    tenantID,
		ProcessType: req.ProcessType,
		RuleContent: req.RuleContent,
		RuleScope:   defaultStr(req.RuleScope, "default_on"),
		Enabled:     true,
		Source:      defaultStr(req.Source, "manual"),
		RelatedFlow: req.RelatedFlow,
	}

	if req.ConfigID != "" {
		configID, err := uuid.Parse(req.ConfigID)
		if err == nil {
			rule.ConfigID = &configID
		}
	}

	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}

	if err := s.ruleRepo.Create(c, rule); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	return rule, nil
}

// Update 更新归档规则。
func (s *ArchiveRuleService) Update(c *gin.Context, id uuid.UUID, req *dto.UpdateArchiveRuleRequest) (*model.ArchiveRule, error) {
	_, err := s.ruleRepo.GetByID(c, id)
	if err != nil {
		return nil, newServiceError(errcode.ErrRuleNotFound, "归档规则不存在")
	}

	fields := make(map[string]interface{})
	if req.RuleContent != "" {
		fields["rule_content"] = req.RuleContent
	}
	if req.RuleScope != "" {
		fields["rule_scope"] = req.RuleScope
	}
	if req.Enabled != nil {
		fields["enabled"] = *req.Enabled
	}
	if req.RelatedFlow != nil {
		fields["related_flow"] = *req.RelatedFlow
	}

	if len(fields) > 0 {
		if err := s.ruleRepo.UpdateFields(c, id, fields); err != nil {
			return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
		}
	}

	rule, err := s.ruleRepo.GetByID(c, id)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	return rule, nil
}

// Delete 删除归档规则：manual 来源硬删除，file_import 来源标记禁用。
func (s *ArchiveRuleService) Delete(c *gin.Context, id uuid.UUID) error {
	rule, err := s.ruleRepo.GetByID(c, id)
	if err != nil {
		return newServiceError(errcode.ErrRuleNotFound, "归档规则不存在")
	}

	if rule.Source == "manual" {
		if err := s.ruleRepo.Delete(c, id); err != nil {
			return newServiceError(errcode.ErrDatabase, "数据库错误")
		}
	} else {
		if err := s.ruleRepo.UpdateFields(c, id, map[string]interface{}{"enabled": false}); err != nil {
			return newServiceError(errcode.ErrDatabase, "数据库错误")
		}
	}
	return nil
}

// ListByConfigIDFilter 按配置 ID 查询关联的归档规则。
func (s *ArchiveRuleService) ListByConfigIDFilter(c *gin.Context, configID uuid.UUID, ruleScope *string, enabled *bool) ([]model.ArchiveRule, error) {
	rules, err := s.ruleRepo.ListByConfigIDFilter(c, configID, ruleScope, enabled)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	return rules, nil
}
