package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"

	"auraoa/go-service/internal/cache"
	"auraoa/go-service/internal/dto"
	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/crypto"
	"auraoa/go-service/internal/pkg/errcode"
	"auraoa/go-service/internal/pkg/oa"
	"auraoa/go-service/internal/repository"
)

// ProcessSummaryConfigService 处理流程总结配置。
type ProcessSummaryConfigService struct {
	configRepo    *repository.ProcessSummaryConfigRepo
	tenantRepo    *repository.TenantRepo
	oaConnRepo    *repository.OAConnectionRepo
	invalidator   *cache.InvalidationManager
	scheduleMgr   EmbedRefreshScheduleManager
	oaConnections *oa.ConnectionManager
}

// SetEmbedRefreshScheduleManager 注入流程级嵌入刷新调度管理器。
func (s *ProcessSummaryConfigService) SetEmbedRefreshScheduleManager(manager EmbedRefreshScheduleManager) {
	s.scheduleMgr = manager
}

var defaultSummaryIncludeMeta = true

func NewProcessSummaryConfigService(
	configRepo *repository.ProcessSummaryConfigRepo,
	tenantRepo *repository.TenantRepo,
	oaConnRepo *repository.OAConnectionRepo,
	invalidator *cache.InvalidationManager,
	oaConnections *oa.ConnectionManager,
) *ProcessSummaryConfigService {
	return &ProcessSummaryConfigService{
		configRepo:    configRepo,
		tenantRepo:    tenantRepo,
		oaConnRepo:    oaConnRepo,
		invalidator:   invalidator,
		oaConnections: oaConnections,
	}
}

func (s *ProcessSummaryConfigService) Create(c *gin.Context, req *dto.CreateProcessSummaryConfigRequest) (*model.ProcessSummaryConfig, error) {
	exists, err := s.configRepo.ExistsByProcessType(c, req.ProcessType)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	if exists {
		return nil, newServiceError(errcode.ErrDuplicateProcessType, "该流程类型已存在总结配置")
	}
	tenantID, err := getTenantUUID(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "租户ID无效")
	}
	cfg := &model.ProcessSummaryConfig{
		ID:               uuid.New(),
		TenantID:         tenantID,
		ProcessType:      req.ProcessType,
		ProcessTypeLabel: req.ProcessTypeLabel,
		MainTableName:    req.MainTableName,
		MainFields:       defaultJSON(req.MainFields, "[]"),
		DetailTables:     defaultJSON(req.DetailTables, "[]"),
		SummaryBlocks:    normalizeSummaryBlocksJSON(req.SummaryBlocks),
		EmbedEnabled:     boolPtrValue(req.EmbedEnabled, true),
		EmbedConfig: normalizeSummaryEmbedConfigJSON(defaultJSON(req.EmbedConfig,
			`{"auto_summary_on_open":true,"auto_summary_on_data_change":true,"auto_summary_on_return_resubmit":true,"auto_summary_on_flow_change":false,"scheduled_refresh_enabled":false,"scheduled_refresh_lookback_days":3,"scheduled_refresh_interval_minutes":5}`)),
		Status: defaultStr(req.Status, "active"),
	}
	if err := s.configRepo.Create(c, cfg); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	if s.scheduleMgr != nil {
		if err := s.scheduleMgr.SyncSummaryConfig(c.Request.Context(), cfg); err != nil {
			return nil, newServiceError(errcode.ErrDatabase, "总结配置已保存，但定时检查调度同步失败")
		}
	}
	s.invalidate(tenantID)
	return cfg, nil
}

func (s *ProcessSummaryConfigService) List(c *gin.Context) ([]model.ProcessSummaryConfig, error) {
	configs, err := s.configRepo.ListByTenant(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	return configs, nil
}

func (s *ProcessSummaryConfigService) GetByID(c *gin.Context, id uuid.UUID) (*model.ProcessSummaryConfig, error) {
	cfg, err := s.configRepo.GetByID(c, id)
	if err != nil {
		return nil, newServiceError(errcode.ErrConfigNotFound, "流程总结配置不存在")
	}
	return cfg, nil
}

func (s *ProcessSummaryConfigService) Update(c *gin.Context, id uuid.UUID, req *dto.UpdateProcessSummaryConfigRequest) (*model.ProcessSummaryConfig, error) {
	if _, err := s.configRepo.GetByID(c, id); err != nil {
		return nil, newServiceError(errcode.ErrConfigNotFound, "流程总结配置不存在")
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
	if req.SummaryBlocks != nil {
		fields["summary_blocks"] = normalizeSummaryBlocksJSON(req.SummaryBlocks)
	}
	if req.EmbedEnabled != nil {
		fields["embed_enabled"] = *req.EmbedEnabled
	}
	if req.EmbedConfig != nil {
		fields["embed_config"] = normalizeSummaryEmbedConfigJSON(req.EmbedConfig)
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
	if s.scheduleMgr != nil {
		if err := s.scheduleMgr.SyncSummaryConfig(c.Request.Context(), cfg); err != nil {
			return nil, newServiceError(errcode.ErrDatabase, "总结配置已保存，但定时检查调度同步失败")
		}
	}
	if tenantID, tErr := getTenantUUID(c); tErr == nil {
		s.invalidate(tenantID)
	}
	return cfg, nil
}

func normalizeSummaryEmbedConfigJSON(raw datatypes.JSON) datatypes.JSON {
	cfg := parseSummaryEmbedConfig(raw)
	b, _ := json.Marshal(cfg)
	return datatypes.JSON(b)
}

func (s *ProcessSummaryConfigService) Delete(c *gin.Context, id uuid.UUID) error {
	if _, err := s.configRepo.GetByID(c, id); err != nil {
		return newServiceError(errcode.ErrConfigNotFound, "流程总结配置不存在")
	}
	if err := s.configRepo.Delete(c, id); err != nil {
		return newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	if s.scheduleMgr != nil {
		if err := s.scheduleMgr.DeleteConfig(c.Request.Context(), embedRefreshModuleSummary, id); err != nil {
			return newServiceError(errcode.ErrDatabase, "总结配置已删除，但定时检查调度清理失败")
		}
	}
	if tenantID, tErr := getTenantUUID(c); tErr == nil {
		s.invalidate(tenantID)
	}
	return nil
}

func (s *ProcessSummaryConfigService) TestConnection(c *gin.Context, req *dto.TestConnectionRequest) (*oa.ProcessInfo, error) {
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

func (s *ProcessSummaryConfigService) FetchFields(c *gin.Context, id uuid.UUID) (*oa.ProcessFields, error) {
	cfg, err := s.configRepo.GetByID(c, id)
	if err != nil {
		return nil, newServiceError(errcode.ErrConfigNotFound, "流程总结配置不存在")
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

func (s *ProcessSummaryConfigService) getOAAdapter(c *gin.Context) (oa.OAAdapter, error) {
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
	adapter, err := s.oaConnections.GetAdapter(c.Request.Context(), conn.OAType, conn)
	if err != nil {
		return nil, newServiceError(errcode.ErrOATypeUnsupported, err.Error())
	}
	return adapter, nil
}

func (s *ProcessSummaryConfigService) invalidate(tenantID uuid.UUID) {
	if s.invalidator != nil {
		_ = s.invalidator.InvalidateConfigCache(context.Background(), tenantID, "summary")
	}
}

func boolPtrValue(v *bool, def bool) bool {
	if v == nil {
		return def
	}
	return *v
}

func boolPtr(v bool) *bool {
	return &v
}

func normalizeSummaryBlocksJSON(raw datatypes.JSON) datatypes.JSON {
	if len(raw) == 0 || string(raw) == "null" || string(raw) == "{}" {
		b, _ := json.Marshal(defaultSummaryBlocks())
		return datatypes.JSON(b)
	}
	var blocks []model.SummaryBlockConfig
	if err := json.Unmarshal(raw, &blocks); err != nil || len(blocks) == 0 {
		b, _ := json.Marshal(defaultSummaryBlocks())
		return datatypes.JSON(b)
	}
	for i := range blocks {
		blocks[i] = normalizeSummaryBlock(blocks[i], i)
	}
	ensureOneSummaryBlockEnabled(blocks)
	b, _ := json.Marshal(blocks)
	return datatypes.JSON(b)
}

func defaultSummaryBlocks() []model.SummaryBlockConfig {
	return []model.SummaryBlockConfig{
		{
			ID:          "overall",
			Title:       "流程摘要",
			UserPrompt:  "请概括流程背景、关键申请内容、金额/日期/对象等核心信息，并列出审批人最需要关注的要点。",
			IncludeMeta: boolPtr(defaultSummaryIncludeMeta),
			FieldMode:   "all",
			Enabled:     true,
			SortOrder:   1,
		},
	}
}

func normalizeSummaryBlock(block model.SummaryBlockConfig, idx int) model.SummaryBlockConfig {
	if strings.TrimSpace(block.ID) == "" {
		block.ID = uuid.NewString()
	}
	if strings.TrimSpace(block.Title) == "" {
		block.Title = "总结块"
	}
	if strings.TrimSpace(block.FieldMode) == "" {
		block.FieldMode = "all"
	}
	if block.FieldMode != "all" && block.FieldMode != "selected" {
		block.FieldMode = "selected"
	}
	if block.IncludeMeta == nil {
		block.IncludeMeta = boolPtr(defaultSummaryIncludeMeta)
	}
	if !summaryBlockIncludeAllData(block) && len(block.EnabledDataVariables) == 0 {
		block.EnabledDataVariables = []string{
			"{{main_table}}",
			"{{detail_tables}}",
			"{{attachments}}",
			"{{flow_history}}",
			"{{flow_graph}}",
		}
	}
	if block.SortOrder == 0 {
		block.SortOrder = idx + 1
	}
	return block
}

func ensureOneSummaryBlockEnabled(blocks []model.SummaryBlockConfig) {
	for _, block := range blocks {
		if block.Enabled {
			return
		}
	}
	if len(blocks) > 0 {
		blocks[0].Enabled = true
	}
}
