package repository

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"auraoa/go-service/internal/dto"
	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/apptime"
)

type experienceSQLRecorder struct {
	logger.Interface
	statements []string
}

func (r *experienceSQLRecorder) Trace(_ context.Context, _ time.Time, sql func() (string, int64), _ error) {
	query, _ := sql()
	r.statements = append(r.statements, query)
}

// TestExperienceQueries 校验实际生成 SQL 的租户、消息归属、分页和按条追加语义。
func TestExperienceQueries(t *testing.T) {
	recorder := &experienceSQLRecorder{Interface: logger.Default.LogMode(logger.Silent)}
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=127.0.0.1 user=test dbname=test", PreferSimpleProtocol: true}), &gorm.Config{DryRun: true, DisableAutomaticPing: true, SkipDefaultTransaction: true, Logger: recorder})
	if err != nil {
		t.Fatal(err)
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	tenant := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	audit := uuid.MustParse("33333333-3333-4333-8333-333333333333")
	user := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	c.Set("tenant_id", tenant.String())
	repo := NewExperienceRepo(db)
	like, dislike := "like", "dislike"
	// 同一个 OA 人员的两条评论拥有独立 ID，不使用按人冲突覆盖。
	var firstCommentID uuid.UUID
	for index, feedback := range []*string{&like, &dislike} {
		row := &model.AuditComment{ID: uuid.New(), TenantID: tenant, AuditLogID: audit, OAUserID: "23", Username: "测试用户", Content: []string{"很满意，结论清晰", "补充评论：希望提供附件清单"}[index], Feedback: feedback, CreatedAt: apptime.Now()}
		if index == 0 {
			firstCommentID = row.ID
		}
		if err := repo.CreateComment(c, row); err != nil {
			t.Fatal(err)
		}
	}
	for _, sql := range recorder.statements {
		if strings.Contains(sql, "ON CONFLICT") {
			t.Fatal("评论不能按人覆盖")
		}
	}
	recorder.statements = append(recorder.statements, "-- interactions")
	_, _ = repo.Interactions(c, audit, dto.ExperienceQuery{Page: -1, PageSize: 999})
	recorder.statements = append(recorder.statements, "-- audit-list")
	_, _ = repo.ListAudits(c, dto.ExperienceQuery{Keyword: "补充", Feedback: "dislike", Page: 1, PageSize: 10})
	recorder.statements = append(recorder.statements, "-- agent-list")
	_, _ = repo.ListAgents(c, dto.ExperienceQuery{Feedback: "dislike", Page: 1, PageSize: 10})
	recorder.statements = append(recorder.statements, "-- comment-update")
	_, _ = repo.UpdateOwnComment(c, audit, firstCommentID, "23", dto.AuditCommentRequest{Content: "已编辑评论", Feedback: nil})
	recorder.statements = append(recorder.statements, "-- comment-delete")
	_, _ = repo.DeleteOwnComment(c, audit, firstCommentID, "23")
	recorder.statements = append(recorder.statements, "-- chat-update")
	_ = NewChatRepo(db).UpdateMessageFeedback(tenant, user, uuid.MustParse("55555555-5555-4555-8555-555555555555"), &dislike, nil)
	for _, query := range recorder.statements {
		if strings.HasPrefix(query, "UPDATE \"audit_comments\"") || strings.HasPrefix(query, "DELETE FROM \"audit_comments\"") {
			for _, value := range []string{audit.String(), firstCommentID.String(), "oa_user_id = '23'"} {
				if !strings.Contains(query, value) {
					t.Errorf("评论修改缺少归属限制: %s", query)
				}
			}
		}
	}
	combined := strings.Join(recorder.statements, "\n")
	for _, fragment := range []string{"LIMIT 20", "LIMIT 10", "m.tenant_id=audit_logs.tenant_id", "s.tenant_id=m.tenant_id", "role = 'assistant'", "s.user_id = '" + user.String() + "'"} {
		if !strings.Contains(combined, fragment) {
			t.Errorf("SQL 缺少必要约束 %s", fragment)
		}
	}
	for _, sql := range recorder.statements {
		if !strings.HasPrefix(sql, "--") && !strings.Contains(sql, tenant.String()) {
			t.Errorf("SQL 丢失租户范围: %s", sql)
		}
	}
	// 可选导出供隔离 PostgreSQL 引擎验证实际 SQL；不读取业务连接配置。
	if path := os.Getenv("EXPERIENCE_SQL_OUTPUT"); path != "" {
		data, _ := json.Marshal(recorder.statements)
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
}
