package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/apptime"
	"auraoa/go-service/internal/pkg/errcode"
	"auraoa/go-service/internal/pkg/oa"
)

// EmbedExecuteRequest 嵌入页审核执行请求。
type EmbedExecuteRequest struct {
	ProcessID        string     `json:"process_id" binding:"required"`
	ProcessType      string     `json:"process_type"`
	Title            string     `json:"title"`
	TriggerSource    string     `json:"trigger_source"`
	TriggerDetail    string     `json:"trigger_detail"`
	ScheduleConfigID *uuid.UUID `json:"-"`
	UseLatestConfig  bool       `json:"use_latest_config,omitempty"`
	Perspective      string     `json:"perspective,omitempty"` // "personal" | "standard"
	OAUserID         string     `json:"oa_user_id,omitempty"`
}

// EmbedPersonalView 描述当前 OA 人员在 AuraOA 中的个人定制视角状态
type EmbedPersonalView struct {
	Available    bool                   `json:"available"` // 是否可用（当前 OA 用户在租户内存在且已配置该流程的个性化规则）
	UserID       string                 `json:"user_id,omitempty"`
	Username     string                 `json:"username,omitempty"`
	DisplayName  string                 `json:"display_name,omitempty"`
	HasAudit     bool                   `json:"has_audit"` // 是否已有个人视角专属的有效审核结果
	LastAuditAt  string                 `json:"last_audit_at,omitempty"`
	RunningJobID string                 `json:"running_job_id,omitempty"`
	AuditResult  map[string]interface{} `json:"audit_result,omitempty"`
}

// EmbedContextResponse 嵌入页上下文。
type EmbedContextResponse struct {
	Supported              bool                      `json:"supported"`
	Reason                 string                    `json:"reason,omitempty"`
	Message                string                    `json:"message,omitempty"`
	Process                *oa.ProcessRequestSummary `json:"process,omitempty"`
	EmbedEnabled           bool                      `json:"embed_enabled"`
	HasAudit               bool                      `json:"has_audit"`
	Stale                  bool                      `json:"stale"`
	ShouldAutoAudit        bool                      `json:"should_auto_audit"`
	AutoRetryBlocked       bool                      `json:"auto_retry_blocked"`
	LastAuditAt            string                    `json:"last_audit_at,omitempty"`
	RunningJobID           string                    `json:"running_job_id,omitempty"`
	AuditResult            map[string]interface{}    `json:"audit_result,omitempty"`
	ConfigVersionNo        *int                      `json:"config_version_no,omitempty"`
	ConfigUpgradeAvailable bool                      `json:"config_upgrade_available"`
	CurrentFingerprint     string                    `json:"-"`
	PersonalView           *EmbedPersonalView        `json:"personal_view,omitempty"`
	DefaultPerspective     string                    `json:"default_perspective,omitempty"` // "personal" | "standard"
}

// GetEmbedContext 嵌入页：按 requestid 拉取流程上下文、有效结论与过期状态。
func (s *AuditExecuteService) GetEmbedContext(c *gin.Context, processID string) (*EmbedContextResponse, error) {
	if processID == "" {
		return nil, newServiceError(errcode.ErrParamValidation, "process_id 不能为空")
	}

	tenantID, userID, err := s.extractIDs(c)
	if err != nil {
		return nil, err
	}

	adapter, err := s.getOAAdapter(c.Request.Context(), tenantID)
	if err != nil {
		return nil, err
	}

	summary, err := adapter.FetchProcessRequestSummary(c.Request.Context(), processID)
	if err != nil {
		return &EmbedContextResponse{
			Supported: false,
			Reason:    "not_found_in_oa",
			Message:   "未在 OA 中找到该流程，请确认 requestid 是否正确",
		}, nil
	}

	config, err := s.configRepo.GetByProcessType(c, summary.ProcessType)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &EmbedContextResponse{
				Supported: false,
				Reason:    "no_config",
				Message:   fmt.Sprintf("流程「%s」尚未配置 AI 审核规则", summary.ProcessType),
				Process:   summary,
			}, nil
		}
		return nil, newServiceError(errcode.ErrDatabase, "查询流程配置失败")
	}
	if config.Status != "active" {
		return &EmbedContextResponse{
			Supported: false,
			Reason:    "config_inactive",
			Message:   fmt.Sprintf("流程「%s」的 AI 审核配置已停用", summary.ProcessType),
			Process:   summary,
		}, nil
	}
	if !config.EmbedEnabled {
		return &EmbedContextResponse{
			Supported:    false,
			Reason:       "embed_disabled",
			Message:      fmt.Sprintf("流程「%s」未启用 OA 嵌入审核", summary.ProcessType),
			Process:      summary,
			EmbedEnabled: false,
		}, nil
	}
	rules, err := s.ruleRepo.ListByConfigID(c, config.ID)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "查询审核规则失败")
	}
	fieldSet, mergedRulesText, effectiveRules, effectiveAIConfig, personalVersion, err := s.resolveUserConfig(c, userID, config, rules, summary.ProcessType)
	if err != nil {
		return nil, newServiceError(errcode.ErrNoProcessConfig, "合并个人审核尺度失败: "+err.Error())
	}
	baseSnapshot := auditConfigSourceSnapshot(config, rules)
	baseVersion, err := s.executionVersions.GetOrCreateLatestBaseVersion(
		c.Request.Context(), tenantID, userID, model.ExecutionConfigModuleAudit,
		config.ID, stableJSONFingerprint(baseSnapshot), baseSnapshot,
	)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "保存审核基础配置版本失败")
	}
	currentConfigSnapshot := AuditExecutionConfigSnapshot{
		AIConfig:                effectiveAIConfig,
		FieldSet:                fieldSet,
		MergedRules:             mergedRulesText,
		EffectiveRules:          effectiveRules,
		BaseConfigVersionNo:     baseVersion.VersionNo,
		PersonalConfigVersionNo: personalVersion,
	}
	currentConfigFingerprint := stableJSONFingerprint(currentConfigSnapshot)
	executionFingerprint := currentConfigFingerprint
	var bindingVersion *model.ExecutionConfigVersion
	bindingVersion, err = s.executionVersions.GetBindingVersion(
		c.Request.Context(), tenantID, model.ExecutionConfigModuleAudit, processID,
	)
	if err == nil {
		pinned, decodeErr := decodeExecutionSnapshot[AuditExecutionConfigSnapshot](bindingVersion)
		if decodeErr != nil {
			return nil, newServiceError(errcode.ErrDatabase, "读取流程绑定的审核配置版本失败")
		}
		fieldSet = pinned.FieldSet
		mergedRulesText = pinned.MergedRules
		executionFingerprint = bindingVersion.Fingerprint
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, newServiceError(errcode.ErrDatabase, "查询流程审核配置版本失败")
	}

	embedCfg := parseEmbedConfig(config.EmbedConfig)
	resp := &EmbedContextResponse{
		Supported:              true,
		Process:                summary,
		EmbedEnabled:           true,
		ConfigVersionNo:        executionVersionNumber(bindingVersion),
		ConfigUpgradeAvailable: bindingVersion != nil && bindingVersion.Fingerprint != currentConfigFingerprint,
	}

	running, _ := s.auditLogRepo.GetRunningByProcessIDForTriggers(c, processID, model.EmbedTriggerSources())
	if running != nil {
		resp.RunningJobID = running.ID.String()
		resp.AuditResult = buildAuditResultFromLog(running)
		resp.HasAudit = true
		return resp, nil
	}

	snap, err := s.auditSnapshotRepo.GetByProcessIDAndChannel(c, processID, model.AuditSnapshotChannelEmbed)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "查询审核快照失败")
	}

	var storedAnchor oa.OAContextAnchor
	var latestLog *model.AuditLog
	if snap != nil {
		latestLog, err = s.auditLogRepo.GetByID(c, snap.LatestValidLogID)
		if err == nil && latestLog != nil {
			storedAnchor = parseOAContextAnchor(latestLog.OAContextAnchor)
			resp.HasAudit = true
			resp.LastAuditAt = apptime.FormatRFC3339(latestLog.UpdatedAt)
			resp.AuditResult = buildAuditResultFromLog(latestLog)
		}
	}

	latestAttempt, err := s.auditLogRepo.GetLatestByProcessIDForTriggers(
		c,
		processID,
		model.EmbedTriggerSources(),
	)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "查询最近审核记录失败")
	}
	if resp.HasAudit && strings.EqualFold(c.Query("prefer_cached"), "true") {
		if latestAttempt != nil &&
			latestAttempt.Status == model.JobStatusFailed &&
			(latestLog == nil || latestAttempt.CreatedAt.After(latestLog.CreatedAt)) {
			resp.AutoRetryBlocked = latestAttempt.AttemptFingerprint != ""
			resp.LastAuditAt = apptime.FormatRFC3339(latestAttempt.UpdatedAt)
			resp.AuditResult = buildAuditResultFromLog(latestAttempt)
		}
		return resp, nil
	}

	currentAnchor, err := s.fetchCurrentOAAnchor(c, tenantID, processID, fieldSet, executionFingerprint)
	if err != nil {
		return nil, err
	}
	resp.CurrentFingerprint = stableJSONFingerprint(currentAnchor)
	changes := oa.CompareContextAnchors(storedAnchor, currentAnchor)
	if resp.HasAudit && bindingVersion == nil && changes.ExecutionConfigChanged {
		// 历史结果没有可恢复的最终配置快照。配置变化时宁可保持原结论并提示手动升级，
		// 也不能把字段范围变化误判成 OA 业务数据变化后自动调用 AI。
		changes.DataChanged = false
		changes.AttachmentChanged = false
	}
	resp.Stale = auditRefreshRequired(changes, embedCfg)

	if !resp.HasAudit {
		resp.ShouldAutoAudit = embedCfg.AutoAuditOnOpen
	} else if resp.Stale {
		resp.ShouldAutoAudit = true
	}

	if latestAttempt != nil &&
		latestAttempt.Status == model.JobStatusFailed &&
		latestAttempt.AttemptFingerprint != "" &&
		latestAttempt.AttemptFingerprint == resp.CurrentFingerprint &&
		(latestLog == nil || latestAttempt.CreatedAt.After(latestLog.CreatedAt)) {
		resp.HasAudit = true
		resp.ShouldAutoAudit = false
		resp.AutoRetryBlocked = true
		resp.LastAuditAt = apptime.FormatRFC3339(latestAttempt.UpdatedAt)
		resp.AuditResult = buildAuditResultFromLog(latestAttempt)
	}

	// 检查当前 OA 访问人员在 AuraOA 中是否有个人定制规则与专属审核记录
	oaUserID := strings.TrimSpace(c.GetHeader("X-Embed-OA-User-ID"))
	if oaUserID == "" {
		oaUserID = strings.TrimSpace(c.Query("oa_user_id"))
	}
	if oaUserID == "" {
		oaUserID = strings.TrimSpace(c.Query("oa_current_user_id"))
	}

	var personalView *EmbedPersonalView
	if oaUserID != "" {
		personalUser, _ := s.resolveOAUser(c.Request.Context(), tenantID, adapter, oaUserID)
		if personalUser != nil {
			hasCustom := s.hasUserCustomizedAudit(c, tenantID, personalUser.ID, config.ID, summary.ProcessType)
			if hasCustom {
				personalView = &EmbedPersonalView{
					Available:   true,
					UserID:      personalUser.ID.String(),
					Username:    personalUser.Username,
					DisplayName: personalUser.DisplayName,
				}
				// 检查该用户是否有正在运行的任务
				personalRunning, _ := s.auditLogRepo.GetRunningByProcessIDAndUser(c, processID, personalUser.ID)
				if personalRunning != nil {
					personalView.RunningJobID = personalRunning.ID.String()
					personalView.AuditResult = buildAuditResultFromLog(personalRunning)
					personalView.HasAudit = true
				} else {
					personalLog, _ := s.auditLogRepo.GetLatestValidByProcessIDAndUser(c, processID, personalUser.ID)
					if personalLog != nil {
						personalView.HasAudit = true
						personalView.LastAuditAt = apptime.FormatRFC3339(personalLog.UpdatedAt)
						personalView.AuditResult = buildAuditResultFromLog(personalLog)
					}
				}
			}
		}
	}

	resp.PersonalView = personalView
	if personalView != nil && personalView.HasAudit {
		resp.DefaultPerspective = "personal"
	} else {
		resp.DefaultPerspective = "standard"
	}

	return resp, nil
}

// resolveOAUser 尝试根据 OA 人员 ID（如泛微 Ecology 的 hrmresource.id）反查租户内的对应用户。
func (s *AuditExecuteService) resolveOAUser(ctx context.Context, tenantID uuid.UUID, adapter oa.OAAdapter, oaUserID string) (*model.User, error) {
	oaUserID = strings.TrimSpace(oaUserID)
	if oaUserID == "" {
		return nil, nil
	}
	username, err := adapter.ResolveUsernameByOAUserID(ctx, oaUserID)
	if err != nil || username == "" {
		return nil, err
	}
	var user model.User
	err = s.db.WithContext(ctx).
		Table("users").
		Joins("JOIN org_members ON org_members.user_id = users.id").
		Where("org_members.tenant_id = ? AND users.username = ? AND users.status = 'active'", tenantID, username).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// hasUserCustomizedAudit 检查指定用户是否对当前流程配置了个性化规则、字段或严格度。
func (s *AuditExecuteService) hasUserCustomizedAudit(c *gin.Context, tenantID, userID, configID uuid.UUID, processType string) bool {
	userCfg, err := s.userConfigRepo.GetByTenantAndUser(c, tenantID, userID)
	if err != nil || userCfg == nil {
		return false
	}
	var items []model.AuditDetailItem
	if err := json.Unmarshal(userCfg.AuditDetails, &items); err != nil {
		return false
	}
	for _, item := range items {
		if item.ProcessType == processType || (item.ConfigID != uuid.Nil && item.ConfigID == configID) {
			if len(item.RuleConfig.CustomRules) > 0 ||
				len(item.RuleConfig.RuleToggleOverrides) > 0 ||
				len(item.FieldConfig.FieldOverrides) > 0 ||
				item.AIConfig.StrictnessOverride != "" {
				return true
			}
		}
	}
	return false
}

// ExecuteEmbed 嵌入页发起审核（自动或手动重新审核）。
func (s *AuditExecuteService) ExecuteEmbed(c *gin.Context, req *EmbedExecuteRequest) (*AuditExecuteResponse, error) {
	trigger := normalizeTriggerSource(req.TriggerSource, model.AuditTriggerEmbedManual)
	if trigger != model.AuditTriggerEmbedAuto && trigger != model.AuditTriggerEmbedManual {
		return nil, newServiceError(errcode.ErrParamValidation, "嵌入页 trigger_source 无效")
	}
	triggerDetail, queueKind := normalizeAuditTriggerDetail(trigger, req.TriggerDetail)
	tenantID, userID, err := s.extractIDs(c)
	if err != nil {
		return nil, err
	}
	release, acquired, lockErr := acquireEmbedCreateLock(
		c.Request.Context(),
		s.rdb,
		embedRefreshModuleAudit,
		tenantID,
		req.ProcessID,
	)
	if lockErr != nil {
		return nil, newServiceError(errcode.ErrRedisConn, "审核任务去重锁获取失败")
	}
	if !acquired {
		return nil, newServiceError(errcode.ErrResourceConflict, "该流程正在创建审核任务")
	}
	defer release()

	isPersonalPerspective := req.Perspective == "personal"
	if isPersonalPerspective {
		oaUserID := strings.TrimSpace(req.OAUserID)
		if oaUserID == "" {
			oaUserID = strings.TrimSpace(c.GetHeader("X-Embed-OA-User-ID"))
		}
		if oaUserID == "" {
			oaUserID = strings.TrimSpace(c.Query("oa_user_id"))
		}
		if oaUserID == "" {
			oaUserID = strings.TrimSpace(c.Query("oa_current_user_id"))
		}
		if oaUserID == "" {
			return nil, newServiceError(errcode.ErrParamValidation, "无法识别当前 OA 操作人，无法发起个人定制审核")
		}
		adapter, aErr := s.getOAAdapter(c.Request.Context(), tenantID)
		if aErr != nil {
			return nil, aErr
		}
		personalUser, pErr := s.resolveOAUser(c.Request.Context(), tenantID, adapter, oaUserID)
		if pErr != nil || personalUser == nil {
			return nil, newServiceError(errcode.ErrResourceNotFound, "当前 OA 操作人尚未绑定或未加入系统，无法发起个人定制审核")
		}
		userID = personalUser.ID
		c.Set("embed_user_id", personalUser.ID)
		trigger = model.AuditTriggerWorkbenchManual
		triggerDetail, queueKind = normalizeAuditTriggerDetail(trigger, "personal_embed_manual")

		// 检查该用户是否已有正在运行的任务
		if running, rErr := s.auditLogRepo.GetRunningByProcessIDAndUser(c, req.ProcessID, personalUser.ID); rErr != nil {
			return nil, newServiceError(errcode.ErrDatabase, "查询个人进行中的审核任务失败")
		} else if running != nil {
			return auditExecuteResponseFromLog(running), nil
		}
	} else {
		if running, runningErr := s.auditLogRepo.GetRunningByProcessIDForTriggers(
			c,
			req.ProcessID,
			model.EmbedTriggerSources(),
		); runningErr != nil {
			return nil, newServiceError(errcode.ErrDatabase, "查询进行中的审核任务失败")
		} else if running != nil {
			promoteToInteractive := queueKind == model.JobQueueKindInteractive &&
				running.Status == model.JobStatusPending &&
				running.QueueKind != model.JobQueueKindInteractive
			relabelAsManual := trigger == model.AuditTriggerEmbedManual &&
				running.Status == model.JobStatusPending &&
				running.TriggerDetail != model.SummaryTriggerDetailManual
			if promoteToInteractive || relabelAsManual {
				if err := s.auditLogRepo.UpdateFields(c, running.ID, map[string]interface{}{
					"user_id":            userID,
					"trigger_source":     trigger,
					"trigger_detail":     triggerDetail,
					"queue_kind":         queueKind,
					"schedule_config_id": nil,
					"updated_at":         apptime.Now(),
				}); err != nil {
					return nil, newServiceError(errcode.ErrDatabase, "切换审核任务队列失败")
				}
				if promoteToInteractive {
					if _, err := EnqueueAuditJob(
						c.Request.Context(),
						s.rdb,
						running.ID,
						tenantID,
						userID,
						queueKind,
					); err != nil {
						return nil, newServiceError(errcode.ErrRedisConn, "审核交互任务入队失败: "+err.Error())
					}
				}
				running.UserID = userID
				running.TriggerSource = trigger
				running.TriggerDetail = triggerDetail
				running.QueueKind = queueKind
				running.ScheduleConfigID = nil
			}
			return auditExecuteResponseFromLog(running), nil
		}
	}

	ctxResp, err := s.GetEmbedContext(c, req.ProcessID)
	if err != nil {
		return nil, err
	}
	if !ctxResp.Supported || !ctxResp.EmbedEnabled {
		return nil, newServiceError(errcode.ErrNoProcessConfig, ctxResp.Message)
	}
	if trigger == model.AuditTriggerEmbedAuto && !ctxResp.ShouldAutoAudit {
		if ctxResp.AutoRetryBlocked {
			return nil, newServiceError(errcode.ErrResourceConflict, "相同数据和规则的自动审核已经失败，请手动重新审核")
		}
		return nil, newServiceError(errcode.ErrResourceConflict, "未检测到需要自动刷新的审核内容")
	}

	processType := req.ProcessType
	title := req.Title
	if ctxResp.Process != nil {
		if processType == "" {
			processType = ctxResp.Process.ProcessType
		}
		if title == "" {
			title = ctxResp.Process.Title
		}
	}
	if processType == "" {
		return nil, newServiceError(errcode.ErrParamValidation, "无法识别流程类型")
	}

	useLatest := req.UseLatestConfig
	if isPersonalPerspective {
		// 个人视角执行时，必须拉取并绑定该用户的最新个性化规则
		useLatest = true
	}

	execReq := &AuditExecuteRequest{
		ProcessID:          req.ProcessID,
		ProcessType:        processType,
		Title:              title,
		TriggerSource:      trigger,
		TriggerDetail:      triggerDetail,
		AttemptFingerprint: ctxResp.CurrentFingerprint,
		ScheduleConfigID:   req.ScheduleConfigID,
		UseLatestConfig:    useLatest,
	}
	return s.Execute(c, execReq)
}

func (s *AuditExecuteService) fetchCurrentOAAnchor(
	c *gin.Context,
	tenantID uuid.UUID,
	processID string,
	fieldSet SelectedFieldSet,
	executionFingerprint string,
) (oa.OAContextAnchor, error) {
	tenant, err := s.tenantRepo.FindByID(tenantID)
	if err != nil {
		return oa.OAContextAnchor{}, newServiceError(errcode.ErrDatabase, "获取租户信息失败")
	}
	pd, err := s.fetchOAData(c, tenant, processID, false, fieldSet)
	if err != nil {
		return oa.OAContextAnchor{}, err
	}
	return s.fetchOAAnchorWithData(c, tenantID, processID, pd, fieldSet, executionFingerprint)
}

func (s *AuditExecuteService) fetchOAAnchorWithData(
	c *gin.Context,
	tenantID uuid.UUID,
	processID string,
	pd *oa.ProcessData,
	fieldSet SelectedFieldSet,
	executionFingerprint string,
) (oa.OAContextAnchor, error) {
	adapter, err := s.getOAAdapter(c.Request.Context(), tenantID)
	if err != nil {
		return oa.OAContextAnchor{}, err
	}
	anchor, err := adapter.FetchProcessContextAnchor(c.Request.Context(), processID, pd)
	if err != nil {
		return oa.OAContextAnchor{}, newServiceError(errcode.ErrOAQueryFailed, err.Error())
	}
	anchor.ContentFingerprint = computeSelectedProcessFingerprint(pd, fieldSet, true, true)
	anchor.AttachmentFingerprint = attachmentFingerprintForFieldSet(*anchor, fieldSet)
	anchor.ExecutionFingerprint = executionFingerprint
	return *anchor, nil
}

func (s *AuditExecuteService) buildOAContextAnchorForJob(
	c *gin.Context,
	tenant *model.Tenant,
	processID string,
	pd *oa.ProcessData,
	fieldSet SelectedFieldSet,
	executionFingerprint string,
) datatypes.JSON {
	anchor, err := s.fetchOAAnchorWithData(c, tenant.ID, processID, pd, fieldSet, executionFingerprint)
	if err != nil {
		return datatypes.JSON([]byte("{}"))
	}
	b, _ := json.Marshal(anchor)
	return datatypes.JSON(b)
}

func parseEmbedConfig(raw datatypes.JSON) model.EmbedConfigData {
	cfg := model.EmbedConfigData{
		AutoAuditOnOpen:           true,
		AutoAuditOnDataChange:     true,
		AutoAuditOnReturnResubmit: true,
		AutoAuditOnFlowChange:     false,
		ScheduledLookbackDays:     3,
		ScheduledIntervalMinutes:  5,
	}
	if len(raw) == 0 {
		return cfg
	}
	_ = json.Unmarshal(raw, &cfg)
	normalizeScheduledRefreshConfig(&cfg.ScheduledLookbackDays, &cfg.ScheduledIntervalMinutes)
	return cfg
}

func parseOAContextAnchor(raw datatypes.JSON) oa.OAContextAnchor {
	var anchor oa.OAContextAnchor
	if len(raw) == 0 {
		return anchor
	}
	_ = json.Unmarshal(raw, &anchor)
	return anchor
}

func normalizeTriggerSource(source, fallback string) string {
	switch source {
	case model.AuditTriggerWorkbenchManual,
		model.AuditTriggerWorkbenchBatch,
		model.AuditTriggerEmbedAuto,
		model.AuditTriggerEmbedManual,
		model.AuditTriggerCronScheduled:
		return source
	default:
		return fallback
	}
}

func normalizeAuditTriggerDetail(trigger, detail string) (string, string) {
	if trigger == model.AuditTriggerEmbedManual {
		return model.SummaryTriggerDetailManual, model.JobQueueKindInteractive
	}
	if trigger != model.AuditTriggerEmbedAuto {
		return strings.TrimSpace(detail), model.JobQueueKindWorkbench
	}
	switch strings.TrimSpace(detail) {
	case model.SummaryTriggerDetailVisibleOpen:
		return model.SummaryTriggerDetailVisibleOpen, model.JobQueueKindInteractive
	case model.SummaryTriggerDetailScheduled:
		return model.SummaryTriggerDetailScheduled, model.JobQueueKindScheduled
	case model.SummaryTriggerDetailSaveRequested:
		return model.SummaryTriggerDetailSaveRequested, model.JobQueueKindBackground
	case model.SummaryTriggerDetailSubmitRequested:
		return model.SummaryTriggerDetailSubmitRequested, model.JobQueueKindBackground
	default:
		// 兼容旧版可见嵌入页：未传详细来源的自动请求按前台打开处理。
		return model.SummaryTriggerDetailVisibleOpen, model.JobQueueKindInteractive
	}
}

// validateEmbedTrigger 嵌入触发时校验 embed_enabled。
func (s *AuditExecuteService) validateEmbedTrigger(c *gin.Context, processType, trigger string) error {
	if trigger != model.AuditTriggerEmbedAuto && trigger != model.AuditTriggerEmbedManual {
		return nil
	}
	config, err := s.configRepo.GetByProcessType(c, processType)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return newServiceError(errcode.ErrNoProcessConfig, fmt.Sprintf("流程 '%s' 的审核配置不存在", processType))
		}
		return newServiceError(errcode.ErrDatabase, "查询流程配置失败")
	}
	if !config.EmbedEnabled {
		return newServiceError(errcode.ErrPermissionDenied, fmt.Sprintf("流程 '%s' 未启用 OA 嵌入审核", processType))
	}
	return nil
}
