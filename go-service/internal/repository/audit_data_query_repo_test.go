package repository

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestAuditDataQueryTenantSQL 验证各条数据访问实际生成的 SQL 均带租户条件。
func TestAuditDataQueryTenantSQL(t *testing.T) {
	recorder := &experienceSQLRecorder{Interface: logger.Default.LogMode(logger.Silent)}
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=127.0.0.1 user=test dbname=test"}), &gorm.Config{
		DryRun: true, DisableAutomaticPing: true, SkipDefaultTransaction: true, Logger: recorder,
	})
	if err != nil {
		t.Fatal(err)
	}
	repo := NewAuditDataQueryRepo(db)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	tenant := uuid.NewString()
	c.Set("tenant_id", tenant)
	id := uuid.New()
	_, _ = repo.Get(c, id)
	_, _ = repo.Version(c, id, 1)
	_, _ = repo.Versions(c, id)
	_ = repo.MarkTested(c, id, datatypes.JSON(`{"name":"草稿"}`), "hash")
	_ = repo.Archive(c, id)
	_, _ = repo.References(c, id)
	_, _, _ = repo.List(c, "预算", -1, 5000)
	if len(recorder.statements) < 7 {
		t.Fatalf("missing SQL statements: %v", recorder.statements)
	}
	for _, statement := range recorder.statements {
		if !strings.Contains(statement, tenant) || !strings.Contains(statement, "tenant_id") {
			t.Fatalf("tenant condition lost: %s", statement)
		}
		if strings.Contains(statement, "LIMIT 5000") || strings.Contains(statement, "OFFSET -") {
			t.Fatalf("invalid paging was not normalized: %s", statement)
		}
	}
}
