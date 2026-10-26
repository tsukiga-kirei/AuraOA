package service

import (
	"encoding/json"
	"testing"
	"time"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/apptime"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

func TestAuditRecheckRetryPreservesBoundInputsAndOwnership(t *testing.T) {
	versionID, rootID := uuid.New(), uuid.New()
	versionNo := 3
	for _, detail := range []string{"manual", "personal_embed_manual"} {
		source := &model.AuditLog{ID: rootID, TenantID: uuid.New(), UserID: uuid.New(), ProcessID: "process-1", ProcessType: "project", Title: "立项", Status: model.JobStatusCompleted,
			ConfigVersionID: &versionID, ConfigVersionNo: &versionNo, AIReasoning: "当次分析", DeepThinking: "当次思考", TriggerSource: model.AuditTriggerEmbedManual, TriggerDetail: detail,
			OAContextAnchor: datatypes.JSON(`{"fingerprint":"original"}`), QueryEvidence: datatypes.JSON(`[{"status":"failed"}]`), CreatedAt: apptime.Now().Add(-24 * time.Hour), Score: 95, Recommendation: "approve", Confidence: 95,
			ErrorMessage: "旧错误", ParseError: "旧错误", RawContent: "失败输出", AttemptFingerprint: "旧指纹", QueueKind: model.JobQueueKindBackground}
		job := newAuditRecheckJob(source, json.RawMessage(initialAuditRecheck), rootID)
		if job.ID == source.ID || job.TenantID != source.TenantID || job.UserID != source.UserID || job.ProcessID != source.ProcessID || job.TriggerDetail != detail || job.ConfigVersionID != source.ConfigVersionID || job.ConfigVersionNo != source.ConfigVersionNo || string(job.OAContextAnchor) != string(source.OAContextAnchor) || string(job.QueryEvidence) != string(source.QueryEvidence) || job.AIReasoning != source.AIReasoning || job.DeepThinking != source.DeepThinking {
			t.Fatal("独立复核必须保留原租户、所有者、个人视角、不可变版本和当次证据")
		}
		var record model.AIRecheckRecord
		if json.Unmarshal(job.AIRecheck, &record) != nil || !record.RetryOnly || record.SourceLogID != rootID.String() || string(record.OriginalResult) != initialAuditRecheck || job.Status != model.JobStatusPending || job.QueueKind != model.JobQueueKindInteractive || time.Since(job.CreatedAt) > time.Minute {
			t.Fatal("复核应创建新的可恢复队列任务，从最后一步执行，不继承旧任务超时")
		}
		if job.ErrorMessage != "" || job.ParseError != "" || job.RawContent != "" || job.AttemptFingerprint != "" || source.Status != model.JobStatusCompleted || source.ErrorMessage != "旧错误" {
			t.Fatal("新任务不应继承技术错误，也不能修改已完成的原记录")
		}
	}
}

func TestAuditRecheckRetryOriginalMaterialIsExplicit(t *testing.T) {
	for _, prompt := range []string{"{{rules}} {{initial_result}}", "{{reasoning_result}}"} {
		if recheckNeedsOriginalRequest(&model.AIRecheckConfig{UserPrompt: prompt}) {
			t.Fatal("默认复核不能重新加载原始材料")
		}
	}
	for _, variable := range []string{"{{audit_context}}", "{{original_system_prompt}}"} {
		if !recheckNeedsOriginalRequest(&model.AIRecheckConfig{SystemPrompt: variable}) || !recheckNeedsOriginalRequest(&model.AIRecheckConfig{UserPrompt: variable}) {
			t.Fatal("显式引用时必须读取当次输入快照")
		}
	}
}
