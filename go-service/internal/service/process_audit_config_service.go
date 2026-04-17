package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/cache"
	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/crypto"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/pkg/oa"
	"oa-smart-audit/go-service/internal/repository"
)

// ProcessAuditConfigService 处理流程审核配置的业务逻辑。
type ProcessAuditConfigService struct {
	configRepo   *repository.ProcessAuditConfigRepo
	tenantRepo   *repository.TenantRepo
	oaConnRepo   *repository.OAConnectionRepo
	templateRepo *repository.SystemPromptTemplateRepo
	db           *gorm.DB
	invalidator  *cache.InvalidationManager
}

// NewProcessAuditConfigService 创建一个新的 ProcessAuditConfigService 实例。
func NewProcessAuditConfigService(
	configRepo *repository.ProcessAuditConfigRepo,
	tenantRepo *repository.TenantRepo,
	oaConnRepo *repository.OAConnectionRepo,
	templateRepo *repository.SystemPromptTemplateRepo,
	db *gorm.DB,
	invalidator *cache.InvalidationManager,
) *ProcessAuditConfigService {
	return &ProcessAuditConfigService{
		configRepo:   configRepo,
		tenantRepo:   tenantRepo,
		oaConnRepo:   oaConnRepo,
		templateRepo: templateRepo,
		db:           db,
		invalidator:  invalidator,
	}
}

// getTenantUUID 从 gin.Context 中提取租户 UUID。
func getTenantUUID(c *gin.Context) (uuid.UUID, error) {
	tidVal, exists := c.Get("tenant_id")
	if !exists {
		return uuid.Nil, fmt.Errorf("租户ID缺失")
	}
	return uuid.Parse(tidVal.(string))
}

// getUserUUID 从 gin.Context 中提取当前用户 UUID（JWT 注入的 user_id）。
func getUserUUID(c *gin.Context) (uuid.UUID, error) {
	uidVal, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, fmt.Errorf("用户ID缺失")
	}
	return uuid.Parse(fmt.Sprintf("%v", uidVal))
}

// Create 创建流程审核配置，校验 process_type 租户内唯一性。
// 若 ai_config 为空，自动从系统提示词模板初始化。
func (s *ProcessAuditConfigService) Create(c *gin.Context, req *dto.CreateProcessAuditConfigRequest) (*model.ProcessAuditConfig, error) {
	exists, err := s.configRepo.ExistsByProcessType(c, req.ProcessType)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	if exists {
		return nil, newServiceError(errcode.ErrDuplicateProcessType, "该流程类型已存在")
	}

	tenantID, err := getTenantUUID(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "租户ID无效")
	}

	aiConfig := req.AIConfig
	if aiConfig == nil || string(aiConfig) == "{}" || string(aiConfig) == "null" || len(aiConfig) == 0 {
		aiConfig = s.buildDefaultAIConfig("standard")
	} else {
		var parsed model.AIConfigData
		if err := json.Unmarshal(aiConfig, &parsed); err == nil {
			if parsed.SystemReasoningPrompt == "" && parsed.UserReasoningPrompt == "" {
				strictness := defaultStr(parsed.AuditStrictness, "standard")
				aiConfig = s.buildDefaultAIConfig(strictness)
			}
		}
	}

	cfg := &model.ProcessAuditConfig{
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
		AccessControl:    defaultJSON(req.AccessControl, "{}"),
		Status:           defaultStr(req.Status, "active"),
	}

	if err := s.configRepo.Create(c, cfg); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	// 清除该租户的审核配置缓存
	if s.invalidator != nil {
		if err := s.invalidator.InvalidateConfigCache(context.Background(), tenantID, "audit"); err != nil {
			_ = err
		}
	}

	return cfg, nil
}

// buildDefaultAIConfig 从系统提示词模板构建默认 ai_config JSON。
func (s *ProcessAuditConfigService) buildDefaultAIConfig(strictness string) datatypes.JSON {
	templates, err := s.templateRepo.GetByStrictnessAuditWorkbench(strictness)
	if err != nil || len(templates) == 0 {
		fallback, _ := json.Marshal(model.AIConfigData{AuditStrictness: strictness})
		return datatypes.JSON(fallback)
	}

	data := model.AIConfigData{AuditStrictness: strictness}
	for _, t := range templates {
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

// ListPromptTemplates 返回审核工作台系统提示词模板（prompt_key 以 audit_ 为前缀，与归档 archive_ 区分）。
func (s *ProcessAuditConfigService) ListPromptTemplates() ([]model.SystemPromptTemplate, error) {
	templates, err := s.templateRepo.ListAll()
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	out := templates[:0]
	for _, t := range templates {
		if strings.HasPrefix(t.PromptKey, "audit_") {
			out = append(out, t)
		}
	}
	return out, nil
}

// GetByID 通过 ID 查询单个流程审核配置。
func (s *ProcessAuditConfigService) GetByID(c *gin.Context, id uuid.UUID) (*model.ProcessAuditConfig, error) {
	cfg, err := s.configRepo.GetByID(c, id)
	if err != nil {
		return nil, newServiceError(errcode.ErrConfigNotFound, "流程审核配置不存在")
	}
	return cfg, nil
}

// List 查询当前租户的所有流程审核配置。
func (s *ProcessAuditConfigService) List(c *gin.Context) ([]model.ProcessAuditConfig, error) {
	configs, err := s.configRepo.ListByTenant(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	return configs, nil
}

// Update 更新流程审核配置。
func (s *ProcessAuditConfigService) Update(c *gin.Context, id uuid.UUID, req *dto.UpdateProcessAuditConfigRequest) (*model.ProcessAuditConfig, error) {
	_, err := s.configRepo.GetByID(c, id)
	if err != nil {
		return nil, newServiceError(errcode.ErrConfigNotFound, "流程审核配置不存在")
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

	// 清除该租户的审核配置缓存
	if s.invalidator != nil {
		tenantID, tErr := getTenantUUID(c)
		if tErr == nil {
			if err := s.invalidator.InvalidateConfigCache(context.Background(), tenantID, "audit"); err != nil {
				_ = err
			}
		}
	}

	return cfg, nil
}

// Delete 删除流程审核配置。
func (s *ProcessAuditConfigService) Delete(c *gin.Context, id uuid.UUID) error {
	_, err := s.configRepo.GetByID(c, id)
	if err != nil {
		return newServiceError(errcode.ErrConfigNotFound, "流程审核配置不存在")
	}
	if err := s.configRepo.Delete(c, id); err != nil {
		return newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	// 清除该租户的审核配置缓存
	if s.invalidator != nil {
		tenantID, tErr := getTenantUUID(c)
		if tErr == nil {
			if err := s.invalidator.InvalidateConfigCache(context.Background(), tenantID, "audit"); err != nil {
				_ = err
			}
		}
	}

	return nil
}

// TestConnection 测试 OA 流程连接，验证流程是否存在，并可选校验主表名。
func (s *ProcessAuditConfigService) TestConnection(c *gin.Context, req *dto.TestConnectionRequest) (*oa.ProcessInfo, error) {
	adapter, err := s.getOAAdapter(c)
	if err != nil {
		return nil, err
	}

	info, err := adapter.ValidateProcess(c.Request.Context(), req.ProcessType)
	if err != nil {
		return nil, newServiceError(errcode.ErrProcessNotFound, "流程在OA系统中不存在: "+err.Error())
	}

	// 如果前端传了 main_table_name，校验是否与 OA 实际主表名一致
	if req.MainTableName != "" && !strings.EqualFold(req.MainTableName, info.MainTable) {
		info.TableMismatch = true
		info.ExpectedTable = info.MainTable
	}

	// 如果前端传了 process_type_label，校验是否与 OA 实际流程类型分类一致
	if req.ProcessTypeLabel != "" && !strings.EqualFold(req.ProcessTypeLabel, info.ProcessTypeLabel) {
		info.TypeLabelMismatch = true
		info.ExpectedTypeLabel = info.ProcessTypeLabel
	}

	return info, nil
}

// FetchFields 从 OA 系统拉取字段定义并持久化到配置中。
func (s *ProcessAuditConfigService) FetchFields(c *gin.Context, id uuid.UUID) (*oa.ProcessFields, error) {
	cfg, err := s.configRepo.GetByID(c, id)
	if err != nil {
		return nil, newServiceError(errcode.ErrConfigNotFound, "流程审核配置不存在")
	}

	adapter, err := s.getOAAdapter(c)
	if err != nil {
		return nil, err
	}

	fields, err := adapter.FetchFields(c.Request.Context(), cfg.ProcessType)
	if err != nil {
		return nil, newServiceError(errcode.ErrOAQueryFailed, "OA字段拉取失败: "+err.Error())
	}

	// 持久化字段信息到配置
	mainFieldsJSON, _ := json.Marshal(fields.MainFields)
	detailTablesJSON, _ := json.Marshal(fields.DetailTables)

	updateFields := map[string]interface{}{
		"main_fields":   datatypes.JSON(mainFieldsJSON),
		"detail_tables": datatypes.JSON(detailTablesJSON),
	}
	if err := s.configRepo.UpdateFields(c, id, updateFields); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	return fields, nil
}

// getOAAdapter 获取当前租户的 OA 适配器实例。
func (s *ProcessAuditConfigService) getOAAdapter(c *gin.Context) (oa.OAAdapter, error) {
	tenantID, err := getTenantUUID(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "租户ID无效")
	}

	// 查询租户信息获取 OA 连接 ID
	tenant, err := s.tenantRepo.FindByID(tenantID)
	if err != nil {
		return nil, newServiceError(errcode.ErrConfigNotFound, "租户不存在")
	}

	if tenant.OADBConnectionID == nil {
		return nil, newServiceError(errcode.ErrOAConnectionFailed, "租户未配置OA数据库连接")
	}

	// 查询 OA 连接配置
	conn, err := s.oaConnRepo.FindByID(*tenant.OADBConnectionID)
	if err != nil {
		return nil, newServiceError(errcode.ErrOAConnectionFailed, "OA数据库连接不存在")
	}

	// 解密密码（数据库中存储的是加密密文）
	password, err := crypto.Decrypt(conn.Password)
	if err != nil {
		return nil, newServiceError(errcode.ErrOAConnectionFailed, "OA数据库密码解密失败")
	}
	conn.Password = password

	// 创建 OA 适配器
	adapter, err := oa.NewOAAdapter(conn.OAType, conn)
	if err != nil {
		return nil, newServiceError(errcode.ErrOATypeUnsupported, err.Error())
	}

	return adapter, nil
}

// defaultJSON 返回 JSON 值，如果为 nil 则返回默认值。
func defaultJSON(val datatypes.JSON, defaultVal string) datatypes.JSON {
	if val == nil {
		return datatypes.JSON([]byte(defaultVal))
	}
	return val
}

// defaultStr 返回字符串值，如果为空则返回默认值。
func defaultStr(val, defaultVal string) string {
	if val == "" {
		return defaultVal
	}
	return val
}
