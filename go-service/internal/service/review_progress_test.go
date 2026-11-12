package service

import (
	"testing"

	"gorm.io/datatypes"

	"auraoa/go-service/internal/model"
)

func failedStepIndex(steps []map[string]interface{}) int {
	for i, step := range steps {
		if failed, ok := step["failed"].(bool); ok && failed {
			return i
		}
	}
	return -1
}

func TestAuditProgressStepsMarksActualFailureStage(t *testing.T) {
	recheckPlanned := datatypes.JSON(`{"enabled":true}`)

	cases := []struct {
		name string
		log  *model.AuditLog
		want int
	}{
		{
			name: "推理阶段失败不能标成复核",
			log: &model.AuditLog{
				Status:          model.JobStatusFailed,
				AIRecheck:       recheckPlanned,
				OAContextAnchor: datatypes.JSON(`{"execution_fingerprint":"x"}`),
			},
			want: 2,
		},
		{
			name: "提取阶段失败",
			log: &model.AuditLog{
				Status:          model.JobStatusFailed,
				AIRecheck:       recheckPlanned,
				AIReasoning:     "分析结果",
				OAContextAnchor: datatypes.JSON(`{"execution_fingerprint":"x"}`),
			},
			want: 3,
		},
		{
			name: "复核阶段失败",
			log: &model.AuditLog{
				Status:    model.JobStatusFailed,
				AIRecheck: datatypes.JSON(`{"original_result":{"recommendation":"approve"}}`),
			},
			want: 4,
		},
		{
			name: "组装阶段失败",
			log: &model.AuditLog{
				Status:    model.JobStatusFailed,
				AIRecheck: recheckPlanned,
			},
			want: 1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			steps := auditProgressSteps(tc.log.Status, len(tc.log.AIRecheck) > 2, auditFailureStage(tc.log))
			if len(steps) != 5 {
				t.Fatalf("启用复核计划时应有 5 个阶段，got=%d", len(steps))
			}
			if got := failedStepIndex(steps); got != tc.want {
				t.Fatalf("失败阶段索引=%d，期望=%d", got, tc.want)
			}
		})
	}
}

func TestArchiveProgressStepsMarksActualFailureStage(t *testing.T) {
	logEntry := &model.ArchiveLog{
		Status:          model.JobStatusFailed,
		AIRecheck:       datatypes.JSON(`{"enabled":true}`),
		ProcessSnapshot: datatypes.JSON(`{"process_id":"x"}`),
	}
	steps := archiveProgressSteps(logEntry.Status, true, archiveFailureStage(logEntry))
	if got := failedStepIndex(steps); got != 2 {
		t.Fatalf("归档推理阶段失败索引=%d，期望=2", got)
	}
}
