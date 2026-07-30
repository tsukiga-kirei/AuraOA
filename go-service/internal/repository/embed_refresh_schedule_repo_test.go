package repository

import (
	"testing"

	"github.com/google/uuid"

	"auraoa/go-service/internal/model"
)

func TestEmbedRefreshScheduleValuesPreservesInactive(t *testing.T) {
	schedule := &model.EmbedRefreshSchedule{
		TenantID:        uuid.New(),
		Module:          "audit",
		ConfigID:        uuid.New(),
		ProcessType:     "项目立项申请流程",
		IsActive:        false,
		LookbackDays:    3,
		IntervalMinutes: 5,
		CronExpression:  "0 */5 * * * *",
	}

	values := embedRefreshScheduleValues(schedule)
	active, ok := values["is_active"].(bool)
	if !ok {
		t.Fatalf("is_active 应显式写入布尔值，实际类型: %T", values["is_active"])
	}
	if active {
		t.Fatal("关闭的定时检查不应被写成启用")
	}
	if schedule.ID == uuid.Nil {
		t.Fatal("调度写入前应生成明确 ID")
	}
}
