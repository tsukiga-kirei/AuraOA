package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/ai"
	"auraoa/go-service/internal/pkg/apptime"
	"auraoa/go-service/internal/pkg/errcode"
	"auraoa/go-service/internal/pkg/sanitize"
	"auraoa/go-service/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// RetryRecheck 校验原任务权限后创建独立复核任务，保留原初审和绑定版本，不重新读取 OA 或附件。
func (s *AuditExecuteService) RetryRecheck(c *gin.Context, id uuid.UUID) (*AuditExecuteResponse, error) {
	source, err := s.getAccessibleAuditLog(c, id)
	if err != nil {
		return nil, err
	}
	var previous model.AIRecheckRecord
	if json.Unmarshal(source.AIRecheck, &previous) != nil || previous.Status != "failed" || (source.Status != model.JobStatusCompleted && source.Status != model.JobStatusFailed) || source.ConfigVersionID == nil {
		return nil, newServiceError(errcode.ErrParamValidation, "当前任务没有可重新执行的复核")
	}
	version, err := s.executionVersions.GetVersionByID(c.Request.Context(), source.TenantID, *source.ConfigVersionID)
	if err != nil {
		return nil, err
	}
	snapshot, err := decodeExecutionSnapshot[AuditExecutionConfigSnapshot](version)
	if err != nil {
		return nil, err
	}
	var cfg model.AIConfigData
	if json.Unmarshal(snapshot.AIConfig, &cfg) != nil || cfg.Recheck == nil || !cfg.Recheck.Enabled {
		return nil, newServiceError(errcode.ErrParamValidation, "原任务绑定版本未开启复核")
	}
	initial, err := ParseAuditResult(string(previous.OriginalResult))
	if err != nil || validateAuditRuleCoverage(initial, snapshot.EffectiveRules) != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "原任务没有完整初审结果，请重新审核")
	}
	rootID := source.ID
	if previous.SourceLogID != "" {
		rootID, err = uuid.Parse(previous.SourceLogID)
		if err != nil {
			return nil, newServiceError(errcode.ErrParamValidation, "原审核记录无效")
		}
	}
	// 自定义模板引用原始材料时，必须仍有当次调用快照，不能用当前 OA 材料替代。
	if recheckNeedsOriginalRequest(cfg.Recheck) {
		if _, err := repository.NewLLMMessageLogRepo(s.db).GetReasoningPayload(c, rootID, "audit"); err != nil {
			return nil, newServiceError(errcode.ErrParamValidation, "原审核输入快照不可用，请重新审核")
		}
	}
	release, acquired, err := acquireEmbedCreateLock(c.Request.Context(), s.rdb, embedRefreshModuleAudit, source.TenantID, source.ProcessID)
	if err != nil {
		return nil, newServiceError(errcode.ErrRedisConn, "复核任务创建失败，请稍后重试")
	}
	if !acquired {
		return nil, newServiceError(errcode.ErrResourceConflict, "该流程正在创建任务，请稍后重试")
	}
	defer release()
	var running *model.AuditLog
	if source.TriggerDetail == "personal_embed_manual" || !model.IsEmbedTrigger(source.TriggerSource) {
		running, err = s.auditLogRepo.GetRunningByProcessIDAndUser(c, source.ProcessID, source.UserID)
	} else {
		running, err = s.auditLogRepo.GetRunningByProcessIDForTriggers(c, source.ProcessID, model.EmbedTriggerSources())
	}
	if err != nil {
		return nil, err
	}
	if running != nil {
		return nil, newServiceError(errcode.ErrResourceConflict, "该流程已有任务正在执行")
	}
	job := newAuditRecheckJob(source, previous.OriginalResult, rootID)
	if err := s.auditLogRepo.Create(job); err != nil {
		return nil, err
	}
	if _, err := EnqueueAuditJob(c.Request.Context(), s.rdb, job.ID, job.TenantID, job.UserID, job.QueueKind); err != nil {
		_ = s.markAuditFailedDB(job.TenantID, job.ID, "复核任务入队失败，请重新复核")
		return nil, newServiceError(errcode.ErrRedisConn, "复核任务入队失败，请稍后重试")
	}
	return auditExecuteResponseFromLog(job), nil
}

func recheckNeedsOriginalRequest(cfg *model.AIRecheckConfig) bool {
	template := cfg.SystemPrompt + cfg.UserPrompt
	return strings.Contains(template, "{{audit_context}}") || strings.Contains(template, "{{original_system_prompt}}")
}

func newAuditRecheckJob(source *model.AuditLog, initial json.RawMessage, rootID uuid.UUID) *model.AuditLog {
	job := *source
	job.ID = uuid.New()
	job.Status = model.JobStatusPending
	job.CreatedAt, job.UpdatedAt = apptime.Now(), apptime.Now()
	job.DurationMs, job.ErrorMessage, job.ParseError, job.RawContent = 0, "", "", ""
	job.AttemptFingerprint = ""
	job.ScheduleConfigID = nil
	if model.IsEmbedTrigger(job.TriggerSource) || job.TriggerDetail == "personal_embed_manual" {
		job.QueueKind = model.JobQueueKindInteractive
	}
	job.AuditResult = datatypes.JSON(initial)
	job.AIRecheck = datatypes.JSON(persistRecheckJSON(&model.AIRecheckRecord{RetryOnly: true, SourceLogID: rootID.String(), Status: "running", OriginalResult: initial}))
	return &job
}

// processAuditRecheckJob 复用原初审结果，只执行复核并按原渠道保存快照。
func (s *AuditExecuteService) processAuditRecheckJob(c *gin.Context, log *model.AuditLog, snapshot AuditExecutionConfigSnapshot, cfg model.AIConfigData, primary, fallback *model.AIModelConfig, maxTokens int) error {
	var previous model.AIRecheckRecord
	if err := json.Unmarshal(log.AIRecheck, &previous); err != nil {
		return err
	}
	if cfg.Recheck == nil || !cfg.Recheck.Enabled {
		return fmt.Errorf("原任务绑定版本未开启复核")
	}
	request := &ai.ChatRequest{RequestType: "audit", ProcessID: log.ProcessID, ProcessTitle: log.Title, BusinessLogID: &log.ID, MaxTokens: maxTokens, ModelConfig: primary}
	if recheckNeedsOriginalRequest(cfg.Recheck) {
		rootID, err := uuid.Parse(previous.SourceLogID)
		if err != nil {
			return err
		}
		payload, err := repository.NewLLMMessageLogRepo(s.db).GetReasoningPayload(c, rootID, "audit")
		if err != nil {
			return fmt.Errorf("原审核输入快照不可用，请重新审核")
		}
		request.SystemPrompt, request.UserPrompt = payload.SystemPrompt, payload.UserPrompt
	}
	scrub := s.sysFlags != nil && s.sysFlags.DataEncryptionEnabled()
	chunk, reset := newReviewStageStream(s.rdb, "audit", log.ID, "recheck", scrub)
	started := time.Now()
	final, record, err := runAIRecheck(c, s.aiCaller, log.TenantID, log.UserID, primary, fallback, cfg.Recheck, request, log.AIReasoning, previous.OriginalResult, scrub, func(record *model.AIRecheckRecord) error {
		record.RetryOnly, record.SourceLogID = true, previous.SourceLogID
		rows, err := s.updateAuditLogIfNotCancelled(log.TenantID, log.ID, map[string]interface{}{"status": model.JobStatusRechecking, "ai_recheck": datatypes.JSON(persistRecheckJSON(record)), "updated_at": apptime.Now()})
		if err == nil && rows == 0 {
			return context.Canceled
		}
		return err
	}, AIRecheckInput{Rules: snapshot.MergedRules, StreamChunkFunc: chunk, StreamResetFunc: reset})
	if err != nil {
		return err
	}
	parsed, err := ParseAuditResult(string(final))
	if err == nil {
		err = validateAuditRuleCoverage(parsed, snapshot.EffectiveRules)
	}
	if err != nil {
		return err
	}
	return s.completeAuditRecheck(c, log, parsed, record, int(time.Since(started).Milliseconds()))
}

// completeAuditRecheck 仅在任务仍活动时提交结果，取消后的迟到响应不能覆盖终态或共享快照。
func (s *AuditExecuteService) completeAuditRecheck(c *gin.Context, log *model.AuditLog, parsed *model.AuditResultJSON, record *model.AIRecheckRecord, duration int) error {
	result, _ := json.Marshal(parsed)
	rows, err := s.updateAuditLogIfNotCancelled(log.TenantID, log.ID, map[string]interface{}{
		"status": model.JobStatusCompleted, "audit_result": datatypes.JSON(result), "recommendation": parsed.Recommendation, "score": parsed.OverallScore, "confidence": parsed.Confidence,
		"ai_recheck": datatypes.JSON(persistRecheckJSON(record)), "raw_content": string(result), "duration_ms": duration, "error_message": "", "parse_error": "", "updated_at": apptime.Now(),
	})
	if err != nil || rows == 0 {
		return err
	}
	if log.TriggerDetail != "personal_embed_manual" {
		if err := s.auditSnapshotRepo.UpsertAppendValid(c, log.TenantID, log.ProcessID, model.AuditSnapshotChannelFromTrigger(log.TriggerSource), log.ID, log.Title, log.ProcessType, parsed.Recommendation, parsed.OverallScore, parsed.Confidence); err != nil {
			return err
		}
	}
	if s.invalidator != nil {
		ctx := context.Background()
		_ = s.invalidator.InvalidateUserTodoCache(ctx, log.TenantID, log.UserID)
		_ = s.invalidator.InvalidateSnapshotCache(ctx, log.TenantID, "audit")
		_ = s.invalidator.InvalidateStatsCache(ctx, log.TenantID, "audit")
		_ = s.invalidator.InvalidateOADataCache(ctx, log.TenantID, "audit")
	}
	return nil
}

// recoverAuditRecheckFailure 对已留存有效初审的复核错误回退，覆盖模型配置、输入快照及阶段超时等失败。
func (s *AuditExecuteService) recoverAuditRecheckFailure(tenantID, id uuid.UUID, cause error) bool {
	c := s.workerGinContext(context.Background(), tenantID, uuid.Nil)
	log, err := s.auditLogRepo.GetByID(c, id)
	if err != nil {
		return false
	}
	var record model.AIRecheckRecord
	if json.Unmarshal(log.AIRecheck, &record) != nil || len(record.OriginalResult) == 0 {
		return false
	}
	parsed, err := ParseAuditResult(string(record.OriginalResult))
	if err != nil {
		return false
	}
	record.Status, record.Changed, record.ReuseOriginal, record.FinalResult = "failed", false, true, nil
	record.Reason, record.Error = "", "AI复核未完成："+cause.Error()
	if s.sysFlags != nil && s.sysFlags.DataEncryptionEnabled() {
		record.Error = sanitize.SanitizeText(record.Error)
	}
	return s.completeAuditRecheck(c, log, parsed, &record, int(time.Since(log.CreatedAt).Milliseconds())) == nil
}
