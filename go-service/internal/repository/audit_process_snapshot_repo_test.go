package repository

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type auditRankingSQLRecorder struct {
	logger.Interface
	statements []string
}

func (r *auditRankingSQLRecorder) Trace(_ context.Context, _ time.Time, sql func() (string, int64), _ error) {
	query, _ := sql()
	r.statements = append(r.statements, query)
}

// TestCountCombinedUserRankingUsesOAOperatorSnapshot 校验仪表盘排名不会把 OA 通用嵌入归到内部承载用户。
func TestCountCombinedUserRankingUsesOAOperatorSnapshot(t *testing.T) {
	recorder := &auditRankingSQLRecorder{Interface: logger.Default.LogMode(logger.Silent)}
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  "host=127.0.0.1 user=test dbname=test",
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		DryRun:                 true,
		DisableAutomaticPing:   true,
		SkipDefaultTransaction: true,
		Logger:                 recorder,
	})
	if err != nil {
		t.Fatal(err)
	}

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("tenant_id", uuid.MustParse("11111111-1111-4111-8111-111111111111").String())
	_, _ = NewAuditProcessSnapshotRepo(db).CountCombinedUserRanking(c, time.Now().AddDate(0, 0, -30), 10)

	query := strings.Join(recorder.statements, "\n")
	for _, fragment := range []string{
		"al.oa_operator_id",
		"al.oa_operator_name",
		"al.oa_operator_dept",
		"COALESCE(al.trigger_detail, '') != 'personal_embed_manual'",
		"psl.trigger_source IN ('summary_embed_auto', 'summary_embed_manual')",
		"'OA 嵌入用户/未识别'",
		"LEFT JOIN org_members om ON NOT a.is_oa_embed",
		"GROUP BY identity_key, username, display_name, department",
		"aps.updated_at >=",
		"psl.updated_at >=",
		"NULLIF(TRIM(d.name), '') = NULLIF(TRIM(a.oa_operator_dept), '')",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("用户活跃排名 SQL 缺少 OA 操作人归并规则 %q: %s", fragment, query)
		}
	}
}
