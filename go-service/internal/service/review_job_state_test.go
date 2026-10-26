package service

import (
	"strings"
	"testing"

	"auraoa/go-service/internal/model"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestAuditUpdatesProtectCompletedJobs(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=127.0.0.1 user=test dbname=test"}), &gorm.Config{DryRun: true, DisableAutomaticPing: true, SkipDefaultTransaction: true})
	if err != nil {
		t.Fatal(err)
	}
	var sql string
	db.Callback().Update().After("gorm:update").Register("test:capture", func(tx *gorm.DB) { sql = tx.Statement.SQL.String() })
	s := &AuditExecuteService{db: db}
	if _, err := s.updateAuditLogIfNotCancelled(uuid.New(), uuid.New(), map[string]interface{}{"status": model.JobStatusFailed}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sql, "status IN") || strings.Contains(sql, "status NOT IN") || !strings.Contains(sql, "tenant_id") {
		t.Fatalf("取消与后续阶段必须只更新活跃任务：%s", sql)
	}
	for _, status := range model.ActiveReviewJobStatuses() {
		if status == model.JobStatusCompleted || status == model.JobStatusFailed || status == model.JobStatusCancelled {
			t.Fatalf("终态不能被更新：%s", status)
		}
	}
}
