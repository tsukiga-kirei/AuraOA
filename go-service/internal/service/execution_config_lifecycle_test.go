package service

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestExecutionConfigLifecycle 使用独立 schema 验证发布去重、丢弃草稿及租户隔离，不访问业务表。
func TestExecutionConfigLifecycle(t *testing.T) {
	dsn := os.Getenv("EXECUTION_CONFIG_TEST_DSN")
	if dsn == "" {
		t.Skip("未配置隔离测试库 EXECUTION_CONFIG_TEST_DSN")
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
	schema := "execution_version_test_" + uuid.New().String()[:8]
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
	must(db.AutoMigrate(&model.ProcessAuditConfig{}, &model.AuditRule{}, &model.ProcessArchiveConfig{}, &model.ArchiveRule{}, &model.ProcessSummaryConfig{}, &model.TenantConfigVersion{}, &model.ExecutionConfigVersion{}, &model.ProcessExecutionConfigBinding{}))
	tid, uid := uuid.New(), uuid.New()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/", nil)
	c.Set("tenant_id", tid.String())
	c.Set("user_id", uid.String())
	other, _ := gin.CreateTestContext(httptest.NewRecorder())
	other.Request = c.Request
	other.Set("tenant_id", uuid.NewString())
	other.Set("user_id", uid.String())
	svc := NewExecutionConfigSourceService(repository.NewExecutionConfigVersionRepo(db), repository.NewProcessAuditConfigRepo(db), repository.NewAuditRuleRepo(db), repository.NewProcessArchiveConfigRepo(db), repository.NewArchiveRuleRepo(db), repository.NewProcessSummaryConfigRepo(db))
	for _, module := range []string{"audit", "archive", "summary"} {
		t.Run(module, func(t *testing.T) {
			configID, ruleID := uuid.New(), uuid.New()
			switch module {
			case "audit":
				must(db.Create(&model.ProcessAuditConfig{ID: configID, TenantID: tid, ProcessType: "100", MainTableName: "published", MainFields: datatypes.JSON(`[]`), DetailTables: datatypes.JSON(`[]`), AIConfig: datatypes.JSON(`{}`), UserPermissions: datatypes.JSON(`{}`), AccessControl: datatypes.JSON(`{"allow_all":true}`), Status: "active"}).Error)
				must(db.Create(&model.AuditRule{ID: ruleID, TenantID: tid, ConfigID: &configID, ProcessType: "100", RuleContent: "发布规则", RuleScope: "default_on", Enabled: boolPointer(true), DataQueries: datatypes.JSON(`[]`), ContextMounts: datatypes.JSON(`[]`)}).Error)
			case "archive":
				must(db.Create(&model.ProcessArchiveConfig{ID: configID, TenantID: tid, ProcessType: "100", MainTableName: "published", MainFields: datatypes.JSON(`[]`), DetailTables: datatypes.JSON(`[]`), AIConfig: datatypes.JSON(`{}`), UserPermissions: datatypes.JSON(`{}`), AccessControl: datatypes.JSON(`{"allow_all":true}`), Status: "active"}).Error)
				must(db.Create(&model.ArchiveRule{ID: ruleID, TenantID: tid, ConfigID: &configID, ProcessType: "100", RuleContent: "发布规则", RuleScope: "default_on", Enabled: boolPointer(true), DataQueries: datatypes.JSON(`[]`), ContextMounts: datatypes.JSON(`[]`)}).Error)
			case "summary":
				must(db.Create(&model.ProcessSummaryConfig{ID: configID, TenantID: tid, ProcessType: "100", MainTableName: "published", MainFields: datatypes.JSON(`[]`), DetailTables: datatypes.JSON(`[]`), SummaryBlocks: datatypes.JSON(`[{"id":"b1","enabled":true,"title":"摘要","user_prompt":"总结"}]`), Status: "active"}).Error)
			}
			if _, err := svc.DiscardDraft(c, module, configID); err == nil {
				t.Fatal("首次发布前不能丢弃整个配置")
			}
			snapshot, _, err := svc.currentSourceSnapshot(c, module, configID)
			must(err)
			raw, err := json.Marshal(snapshot)
			must(err)
			// 模拟升级前缺少 data_queries 字段且指纹算法不同的已发布快照。
			var legacy map[string]interface{}
			must(json.Unmarshal(raw, &legacy))
			if rules, ok := legacy["rules"].([]interface{}); ok {
				for _, rule := range rules {
					delete(rule.(map[string]interface{}), "data_queries")
				}
			}
			raw, err = json.Marshal(legacy)
			must(err)
			published := model.TenantConfigVersion{ID: uuid.New(), TenantID: tid, Module: module, SourceConfigID: configID, VersionNo: 2, Fingerprint: "legacy-fingerprint", ConfigSnapshot: raw, IsActive: true}
			must(db.Create(&published).Error)
			status, err := svc.GetStatus(c, module, configID)
			must(err)
			if status.HasPendingChanges {
				t.Fatal("升级补默认字段不能产生草稿变更")
			}
			status, err = svc.Publish(c, module, configID)
			must(err)
			if *status.ActiveVersionNo != 2 {
				t.Fatal("内容相同应复用旧发布版本")
			}
			var count int64
			must(db.Model(&model.TenantConfigVersion{}).Where("source_config_id = ?", configID).Count(&count).Error)
			if count != 1 {
				t.Fatal("无实质变化不能创建重复版本")
			}
			execution := model.ExecutionConfigVersion{ID: uuid.New(), TenantID: tid, Module: module, SourceConfigID: configID, BaseConfigVersionID: &published.ID, VersionNo: 1, Fingerprint: "bound", ConfigSnapshot: datatypes.JSON(`{}`)}
			must(db.Create(&execution).Error)
			binding := model.ProcessExecutionConfigBinding{ID: uuid.New(), TenantID: tid, Module: module, ProcessID: "process-1", ConfigVersionID: execution.ID}
			must(db.Create(&binding).Error)
			var draft map[string]interface{}
			must(json.Unmarshal(raw, &draft))
			draft["main_table_name"] = "draft"
			draft["process_type"] = "draft-type"
			status, err = svc.SaveDraft(c, module, configID, draft)
			must(err)
			if !status.HasPendingChanges {
				t.Fatal("真实变更必须形成草稿")
			}
			if _, err := svc.DiscardDraft(other, module, configID); err == nil {
				t.Fatal("跨租户不能恢复配置")
			}
			status, err = svc.DiscardDraft(c, module, configID)
			must(err)
			if status.HasPendingChanges || *status.ActiveVersionNo != 2 {
				t.Fatal("丢弃后应恢复当前发布版本并清除修改状态")
			}
			actual, _, err := svc.currentSourceSnapshot(c, module, configID)
			must(err)
			if actual.(map[string]interface{})["main_table_name"] != "published" || actual.(map[string]interface{})["process_type"] != "100" {
				t.Fatal("草稿内容未恢复")
			}
			var retained model.ProcessExecutionConfigBinding
			must(db.First(&retained, "id = ?", binding.ID).Error)
			if retained.ConfigVersionID != execution.ID {
				t.Fatal("丢弃草稿不能更改执行绑定")
			}
			status, err = svc.SaveVersion(c, module, configID, 2, draft)
			must(err)
			if !status.HasPendingChanges {
				t.Fatal("历史版本兼容编辑应另存为草稿")
			}
			var unchanged model.TenantConfigVersion
			must(db.First(&unchanged, "id = ?", published.ID).Error)
			if string(unchanged.ConfigSnapshot) != string(published.ConfigSnapshot) {
				// JSONB 会调整对象键顺序，比较内容而非原始字节。
				if sourceContentFingerprint(unchanged.ConfigSnapshot) != sourceContentFingerprint(published.ConfigSnapshot) {
					t.Fatal("历史发布内容被覆盖")
				}
			}
			_, err = svc.DiscardDraft(c, module, configID)
			must(err)
		})
	}
}
