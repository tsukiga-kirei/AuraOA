package service

import (
	"encoding/json"
	"time"

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

// UserPersonalConfigService 处理用户个人配置的业务逻辑。
type UserPersonalConfigService struct {
	userConfigRepo *repository.UserPersonalConfigRepo
	configRepo     *repository.ProcessAuditConfigRepo
	tenantRepo     *repository.TenantRepo
	oaConnRepo     *repository.OAConnectionRepo
}

// NewUserPersonalConfigService 创建一个新的 UserPersonalConfigService 实例。
func NewUserPersonalConfigService(
	userConfigRepo *repository.UserPersonalConfigRepo,
	configRepo *repository.ProcessAuditConfigRepo,
	tenantRepo *repository.TenantRepo,
	oaConnRepo *repository.OAConnectionRepo,
) *UserPersonalConfigService {
	return &UserPersonalConfigService{
		userConfigRepo: userConfigRepo,
		configRepo:     configRepo,
		tenantRepo:     tenantRepo,
		oaConnRepo:     oaConnRepo,
	}
}

// GetProcessList 获取用户可见的流程列表（双重校验：OA 权限 + 租户配置存在）。
func (s *UserPersonalConfigService) GetProcessList(c *gin.Context, userID uuid.UUID) ([]dto.ProcessListItem, error) {
	// 获取租户的所有流程审核配置
	configs, err := s.configRepo.ListByTenant(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	if len(configs) == 0 {
		return []dto.ProcessListItem{}, nil
	}

	// 尝试获取 OA 适配器进行权限校验
	adapter, adapterErr := s.getOAAdapter(c)

	var result []dto.ProcessListItem
	for _, cfg := range configs {
		// 如果有 OA 适配器，校验用户权限
		if adapterErr == nil && adapter != nil {
			hasPermission, err := adapter.CheckUserPermission(c.Request.Context(), userID.String(), cfg.ProcessType)
			if err != nil || !hasPermission {
				continue
			}
		}
		// 配置存在且用户有权限（或无 OA 适配器时默认放行）
		result = append(result, dto.ProcessListItem{
			ProcessType:      cfg.ProcessType,
			ProcessTypeLabel: cfg.ProcessTypeLabel,
			ConfigID:         cfg.ID.String(),
		})
	}

	if result == nil {
		result = []dto.ProcessListItem{}
	}
	return result, nil
}

// GetByProcessType 获取用户对指定流程的个性化配置详情。
func (s *UserPersonalConfigService) GetByProcessType(c *gin.Context, userID uuid.UUID, processType string) (*model.AuditDetailItem, error) {
	tenantID, err := getTenantUUID(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "租户ID无效")
	}

	userCfg, err := s.userConfigRepo.GetByTenantAndUser(c, tenantID, userID)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	if userCfg == nil {
		// 用户尚无配置，返回空的默认配置
		return &model.AuditDetailItem{ProcessType: processType}, nil
	}

	// 从 audit_details JSON 中查找对应流程的配置
	var auditDetails []model.AuditDetailItem
	if err := json.Unmarshal(userCfg.AuditDetails, &auditDetails); err != nil {
		return &model.AuditDetailItem{ProcessType: processType}, nil
	}

	for _, detail := range auditDetails {
		if detail.ProcessType == processType {
			return &detail, nil
		}
	}

	return &model.AuditDetailItem{ProcessType: processType}, nil
}

// UpdateByProcessType 更新用户对指定流程的个性化配置，校验权限锁定。
func (s *UserPersonalConfigService) UpdateByProcessType(c *gin.Context, userID uuid.UUID, processType string, req *dto.UpdateUserProcessConfigRequest) error {
	tenantID, err := getTenantUUID(c)
	if err != nil {
		return newServiceError(errcode.ErrParamValidation, "租户ID无效")
	}

	// 获取流程审核配置，检查权限锁定
	processCfg, err := s.configRepo.GetByProcessType(c, processType)
	if err != nil {
		return newServiceError(errcode.ErrConfigNotFound, "流程审核配置不存在")
	}

	// 解析 user_permissions
	var perms model.UserPermissionsData
	if err := json.Unmarshal(processCfg.UserPermissions, &perms); err != nil {
		// 解析失败时使用默认值（全部允许）
		perms = model.UserPermissionsData{
			AllowCustomFields:     true,
			AllowCustomRules:      true,
			AllowModifyStrictness: true,
		}
	}

	// 校验权限锁定
	if !perms.AllowCustomFields && (req.FieldOverrides != nil || req.FieldMode != "") {
		return newServiceError(errcode.ErrPermissionDenied, "字段自定义功能已被锁定")
	}
	if !perms.AllowCustomRules && req.CustomRules != nil {
		return newServiceError(errcode.ErrPermissionDenied, "自定义规则功能已被锁定")
	}
	if !perms.AllowModifyStrictness && req.StrictnessOverride != "" {
		return newServiceError(errcode.ErrPermissionDenied, "审核尺度修改功能已被锁定")
	}

	// 获取或创建用户配置
	userCfg, err := s.userConfigRepo.GetByTenantAndUser(c, tenantID, userID)
	if err != nil {
		return newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	var auditDetails []model.AuditDetailItem
	if userCfg != nil {
		_ = json.Unmarshal(userCfg.AuditDetails, &auditDetails)
	}

	// 构建新的 AuditDetailItem
	newDetail := model.AuditDetailItem{
		ProcessType:        processType,
		FieldMode:          req.FieldMode,
		StrictnessOverride: req.StrictnessOverride,
	}

	// 转换 DTO 到 model
	if req.FieldOverrides != nil {
		newDetail.FieldOverrides = req.FieldOverrides
	}
	if req.CustomRules != nil {
		customRules := make([]model.CustomRule, len(req.CustomRules))
		for i, r := range req.CustomRules {
			customRules[i] = model.CustomRule{ID: r.ID, Content: r.Content, Enabled: r.Enabled}
		}
		newDetail.CustomRules = customRules
	}
	if req.RuleToggleOverrides != nil {
		toggles := make([]model.RuleToggleOverride, len(req.RuleToggleOverrides))
		for i, t := range req.RuleToggleOverrides {
			toggles[i] = model.RuleToggleOverride{RuleID: t.RuleID, Enabled: t.Enabled}
		}
		newDetail.RuleToggleOverrides = toggles
	}

	// 更新或追加到 auditDetails
	found := false
	for i, detail := range auditDetails {
		if detail.ProcessType == processType {
			auditDetails[i] = newDetail
			found = true
			break
		}
	}
	if !found {
		auditDetails = append(auditDetails, newDetail)
	}

	auditDetailsJSON, _ := json.Marshal(auditDetails)

	// Upsert 用户配置
	cfg := &model.UserPersonalConfig{
		ID:           uuid.New(),
		TenantID:     tenantID,
		UserID:       userID,
		AuditDetails: datatypes.JSON(auditDetailsJSON),
		CronDetails:  datatypes.JSON([]byte("[]")),
		ArchiveDetails: datatypes.JSON([]byte("[]")),
		UpdatedAt:    time.Now(),
	}

	if userCfg != nil {
		cfg.ID = userCfg.ID
		cfg.CronDetails = userCfg.CronDetails
		cfg.ArchiveDetails = userCfg.ArchiveDetails
	}

	if err := s.userConfigRepo.Upsert(cfg); err != nil {
		return newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	return nil
}

// getOAAdapter 获取当前租户的 OA 适配器实例。
func (s *UserPersonalConfigService) getOAAdapter(c *gin.Context) (oa.OAAdapter, error) {
	tenantID, err := getTenantUUID(c)
	if err != nil {
		return nil, err
	}

	tenant, err := s.tenantRepo.FindByID(tenantID)
	if err != nil {
		return nil, err
	}

	if tenant.OADBConnectionID == nil {
		return nil, newServiceError(errcode.ErrOAConnectionFailed, "租户未配置OA数据库连接")
	}

	conn, err := s.oaConnRepo.FindByID(*tenant.OADBConnectionID)
	if err != nil {
		return nil, err
	}

	// 解密密码（数据库中存储的是加密密文）
	password, decErr := crypto.Decrypt(conn.Password)
	if decErr != nil {
		return nil, newServiceError(errcode.ErrOAConnectionFailed, "OA数据库密码解密失败")
	}
	conn.Password = password

	return oa.NewOAAdapter(conn.OAType, conn)
}
