package repository

import (
	"errors"
	"net/http/httptest"
	"os"
	"testing"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/apptime"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestAuditDataQueryContentLifecycle 在隔离 schema 校验内容幂等、预览校验与发布去重。
func TestAuditDataQueryContentLifecycle(t *testing.T) {
	dsn := os.Getenv("AUDIT_DATA_QUERY_TEST_DSN")
	if dsn == "" {
		t.Skip("未配置隔离测试库 AUDIT_DATA_QUERY_TEST_DSN")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	schema := "memory_version_test_" + uuid.NewString()[:8]
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(db.Exec(`CREATE SCHEMA "` + schema + `"`).Error)
	defer sqlDB.Close()
	defer db.Exec(`DROP SCHEMA "` + schema + `" CASCADE`)
	must(db.Exec(`SET search_path TO "` + schema + `"`).Error)
	must(db.AutoMigrate(&model.AuditDataQuery{}, &model.AuditDataQueryVersion{}))
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/", nil)
	c.Set("tenant_id", uuid.NewString())
	other, _ := gin.CreateTestContext(httptest.NewRecorder())
	other.Request = c.Request
	other.Set("tenant_id", uuid.NewString())
	repo := NewAuditDataQueryRepo(db)
	id, tid := uuid.New(), uuid.MustParse(c.GetString("tenant_id"))
	old := datatypes.JSON(`{"name":"统计","description":"数量","sql":"SELECT 1 AS 数量","dialect":"mysql","tables":null,"parameters":null}`)
	same := datatypes.JSON(`{"parameters":[],"tables":[],"mode":"sql","name":"统计","description":"数量","sql":"SELECT 1 AS 数量","dialect":"mysql","ai_context":{"oa_module":"general","notes":"","field_notes":{}},"max_rows":0}`)
	item := &model.AuditDataQuery{ID: id, TenantID: tid, Name: "统计", Category: "采购", Draft: old, CreatedAt: apptime.Now(), UpdatedAt: apptime.Now()}
	must(repo.Save(c, item, true))
	must(repo.MarkTested(c, id, same, "preview-1"))
	before, err := repo.Get(c, id)
	must(err)
	first, err := repo.Publish(c, id, "preview-1", same, "old-preview-1")
	must(err)
	must(repo.Save(c, &model.AuditDataQuery{ID: id, Name: item.Name, Category: item.Category, Draft: same}, false))
	unchanged, err := repo.Get(c, id)
	must(err)
	if unchanged.TestedHash != before.TestedHash || unchanged.TestedAt == nil || !unchanged.TestedAt.Equal(*before.TestedAt) {
		t.Fatal("原样保存清除了预览状态")
	}
	repeated, err := repo.Publish(c, id, "preview-1", same, "old-preview-1")
	must(err)
	if repeated.ID != first.ID || repeated.VersionNo != 1 {
		t.Fatal("重复发布没有复用已有版本")
	}
	must(repo.Save(c, &model.AuditDataQuery{ID: id, Name: item.Name, Category: "新分类", Draft: same}, false))
	categorized, err := repo.Get(c, id)
	must(err)
	if categorized.TestedHash != "preview-1" || categorized.Category != "新分类" {
		t.Fatal("分类变动不应清除查询定义的验证")
	}
	// 定义相同也不能绕过数据源配置变化导致的预览失效。
	if _, err := repo.Publish(c, id, "changed-source", same, "changed-source-legacy"); err == nil {
		t.Fatal("数据源变化绕过了预览校验")
	}
	changed := datatypes.JSON(`{"name":"统计","description":"数量","sql":"SELECT 2 AS 数量","dialect":"mysql","tables":[],"parameters":[]}`)
	must(repo.Save(c, &model.AuditDataQuery{ID: id, Name: item.Name, Category: "新分类", Draft: changed}, false))
	if err := repo.MarkTested(c, id, same, "obsolete-preview"); !errors.Is(err, ErrDataQueryDraftChanged) {
		t.Fatal("旧预览结果验证了新草稿")
	}
	if _, err := repo.Publish(c, id, "preview-1", same, "old-preview-1"); !errors.Is(err, ErrDataQueryDraftChanged) {
		t.Fatal("发布期间草稿变化未被拦截")
	}
	saved, err := repo.Get(c, id)
	must(err)
	if saved.TestedHash != "" || saved.TestedAt != nil {
		t.Fatal("真实定义变动没有清除旧验证")
	}
	must(repo.MarkTested(c, id, changed, "preview-2"))
	second, err := repo.Publish(c, id, "preview-2", changed, "")
	must(err)
	if second.VersionNo != 2 {
		t.Fatal("真实修改应产生下一发布版本")
	}
	history, err := repo.Versions(c, id)
	must(err)
	if len(history) != 2 || !model.EqualQueryDefinitions(history[1].Definition, old) {
		t.Fatal("发布历史被改写或多生成版本")
	}
	if err := repo.MarkTested(other, id, changed, "cross-tenant"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("预览不能跨租户验证")
	}
	if _, err := repo.Publish(other, id, "preview-2", changed, ""); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("发布不能跨租户")
	}
}
