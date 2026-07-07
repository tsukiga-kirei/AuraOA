package service

import (
	"encoding/json"
	"errors"
	"fmt"

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
	ProcessID     string `json:"process_id" binding:"required"`
	ProcessType   string `json:"process_type"`
	Title         string `json:"title"`
	TriggerSource string `json:"trigger_source"`
}

// EmbedContextResponse 嵌入页上下文。
type EmbedContextResponse struct {
	Supported       bool                      `json:"supported"`
	Reason          string                    `json:"reason,omitempty"`
	Message         string                    `json:"message,omitempty"`
	Process         *oa.ProcessRequestSummary `json:"process,omitempty"`
	EmbedEnabled    bool                      `json:"embed_enabled"`
	HasAudit        bool                      `json:"has_audit"`
	Stale           bool                      `json:"stale"`
	ShouldAutoAudit bool                      `json:"should_auto_audit"`
	LastAuditAt     string                    `json:"last_audit_at,omitempty"`
	RunningJobID    string                    `json:"running_job_id,omitempty"`
	AuditResult     map[string]interface{}    `json:"audit_result,omitempty"`
}

// GetEmbedContext 嵌入页：按 requestid 拉取流程上下文、有效结论与过期状态。
func (s *AuditExecuteService) GetEmbedContext(c *gin.Context, processID string) (*EmbedContextResponse, error) {
	if processID == "" {
		return nil, newServiceError(errcode.ErrParamValidation, "process_id 不能为空")
	}

	tenantID, _, err := s.extractIDs(c)
	if err != nil {
		return nil, err
	}

	adapter, err := s.getOAAdapter(tenantID)
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

	embedCfg := parseEmbedConfig(config.EmbedConfig)
	resp := &EmbedContextResponse{
		Supported:    true,
		Process:      summary,
		EmbedEnabled: true,
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

	currentAnchor, err := s.fetchCurrentOAAnchor(c, tenantID, processID)
	if err != nil {
		return nil, err
	}
	resp.Stale = oa.IsAnchorStale(storedAnchor, currentAnchor)

	if !resp.HasAudit {
		resp.ShouldAutoAudit = embedCfg.AutoAuditOnOpen
		return resp, nil
	}
	if resp.Stale && embedCfg.AutoAuditOnStale {
		resp.ShouldAutoAudit = true
	}
	return resp, nil
}

// ExecuteEmbed 嵌入页发起审核（自动或手动重新审核）。
func (s *AuditExecuteService) ExecuteEmbed(c *gin.Context, req *EmbedExecuteRequest) (*AuditExecuteResponse, error) {
	trigger := normalizeTriggerSource(req.TriggerSource, model.AuditTriggerEmbedManual)
	if trigger != model.AuditTriggerEmbedAuto && trigger != model.AuditTriggerEmbedManual {
		return nil, newServiceError(errcode.ErrParamValidation, "嵌入页 trigger_source 无效")
	}

	ctxResp, err := s.GetEmbedContext(c, req.ProcessID)
	if err != nil {
		return nil, err
	}
	if !ctxResp.Supported || !ctxResp.EmbedEnabled {
		return nil, newServiceError(errcode.ErrNoProcessConfig, ctxResp.Message)
	}
	if ctxResp.RunningJobID != "" {
		return nil, newServiceError(errcode.ErrResourceConflict, "该流程已有进行中的审核任务")
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

	execReq := &AuditExecuteRequest{
		ProcessID:     req.ProcessID,
		ProcessType:   processType,
		Title:         title,
		TriggerSource: trigger,
	}
	return s.Execute(c, execReq)
}

func (s *AuditExecuteService) fetchCurrentOAAnchor(c *gin.Context, tenantID uuid.UUID, processID string) (oa.OAContextAnchor, error) {
	tenant, err := s.tenantRepo.FindByID(tenantID)
	if err != nil {
		return oa.OAContextAnchor{}, newServiceError(errcode.ErrDatabase, "获取租户信息失败")
	}
	pd, err := s.fetchOAData(c, tenant, processID, false)
	if err != nil {
		return oa.OAContextAnchor{}, err
	}
	adapter, err := s.getOAAdapter(tenantID)
	if err != nil {
		return oa.OAContextAnchor{}, err
	}
	anchor, err := adapter.FetchProcessContextAnchor(c.Request.Context(), processID, pd)
	if err != nil {
		return oa.OAContextAnchor{}, newServiceError(errcode.ErrOAQueryFailed, err.Error())
	}
	return *anchor, nil
}

func (s *AuditExecuteService) buildOAContextAnchorForJob(c *gin.Context, tenant *model.Tenant, processID string) datatypes.JSON {
	anchor, err := s.fetchCurrentOAAnchor(c, tenant.ID, processID)
	if err != nil {
		return datatypes.JSON([]byte("{}"))
	}
	b, _ := json.Marshal(anchor)
	return datatypes.JSON(b)
}

func parseEmbedConfig(raw datatypes.JSON) model.EmbedConfigData {
	cfg := model.EmbedConfigData{AutoAuditOnOpen: true, AutoAuditOnStale: true}
	if len(raw) == 0 {
		return cfg
	}
	_ = json.Unmarshal(raw, &cfg)
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
