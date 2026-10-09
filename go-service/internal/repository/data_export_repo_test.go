package repository

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// exportSQLTestDB 仅生成 SQL，验证导出范围而不连接业务数据库。
func exportSQLTestDB(t *testing.T) (*gorm.DB, *auditRankingSQLRecorder, *gin.Context) {
	t.Helper()
	recorder := &auditRankingSQLRecorder{Interface: logger.Default.LogMode(logger.Silent)}
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=127.0.0.1 user=test dbname=test"}), &gorm.Config{
		DryRun: true, DisableAutomaticPing: true, SkipDefaultTransaction: true, Logger: recorder,
	})
	if err != nil {
		t.Fatal(err)
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("tenant_id", "11111111-1111-4111-8111-111111111111")
	return db, recorder, c
}

func TestDataExportQueriesHaveNoPaginationAndKeepFilters(t *testing.T) {
	for _, name := range []string{"audit", "archive", "summary", "cron"} {
		t.Run(name, func(t *testing.T) {
			db, recorder, c := exportSQLTestDB(t)
			var err error
			switch name {
			case "audit":
				_, err = NewAuditProcessSnapshotRepo(db).ListAllWithUser(c, AuditSnapshotFilter{Keyword: "导出回归", Channel: "workbench"})
			case "archive":
				_, err = NewArchiveProcessSnapshotRepo(db).ListAllWithUser(c, ArchiveSnapshotFilter{Keyword: "导出回归", Compliance: "compliant"})
			case "summary":
				_, err = NewProcessSummarySnapshotRepo(db).ListAllWithUser(c, ProcessSummarySnapshotFilter{Keyword: "导出回归", Channel: "workbench"})
			case "cron":
				_, err = NewCronLogRepo(db).ListAllByTenant(uuid.MustParse(c.GetString("tenant_id")), CronLogFilter{Keyword: "导出回归", Status: "success"})
			}
			if err != nil {
				t.Fatal(err)
			}
			query := strings.Join(recorder.statements, "\n")
			for _, forbidden := range []string{" LIMIT ", " OFFSET "} {
				if strings.Contains(query, forbidden) {
					t.Errorf("全量导出仍包含分页限制 %q: %s", forbidden, query)
				}
			}
			for _, required := range []string{c.GetString("tenant_id"), "导出回归", "ORDER BY"} {
				if !strings.Contains(query, required) {
					t.Errorf("导出遗漏租户、筛选或排序 %q: %s", required, query)
				}
			}
		})
	}
}

func TestExportHistoryQueriesUseStoredResultFieldsAndJoinedOperator(t *testing.T) {
	for _, name := range []string{"audit", "archive"} {
		t.Run(name, func(t *testing.T) {
			db, recorder, c := exportSQLTestDB(t)
			var err error
			fields := []string{"audit_logs.recommendation", "audit_logs.score", "audit_logs.oa_operator_name"}
			if name == "audit" {
				_, err = NewAuditLogRepo(db).ListValidByProcessIDsWithUser(c, []string{"request-1"})
			} else {
				_, err = NewArchiveLogRepo(db).ListValidByProcessIDsWithUser(c, []string{"request-1"})
				fields = []string{"archive_logs.compliance", "archive_logs.compliance_score"}
			}
			if err != nil {
				t.Fatal(err)
			}
			query := strings.Join(recorder.statements, "\n")
			fields = append(fields, "LEFT JOIN users", "AS user_name", "users.display_name", c.GetString("tenant_id"), "request-1", "completed", "parse_error", "created_at DESC")
			for _, required := range fields {
				if !strings.Contains(query, required) {
					t.Errorf("历史查询遗漏 %q: %s", required, query)
				}
			}
			for _, forbidden := range []string{"overall_decision", "archive_logs.score", "created_at, user_name,"} {
				if strings.Contains(query, forbidden) {
					t.Errorf("历史查询读取了不存在的字段 %q: %s", forbidden, query)
				}
			}
		})
	}
}
