package repository

import (
	"errors"
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

// TestFeedbackMemoryIntegration 仅使用显式指定的隔离测试库，整个用例回滚，不读取业务配置。
func TestFeedbackMemoryIntegration(t *testing.T) {
	dsn := os.Getenv("FEEDBACK_MEMORY_TEST_DSN")
	if dsn == "" {
		t.Skip("未配置隔离测试库 FEEDBACK_MEMORY_TEST_DSN")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	db = db.Begin()
	t.Cleanup(func() { db.Rollback() })
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(db.Exec(`CREATE TABLE tenants(id uuid PRIMARY KEY);
 CREATE TABLE process_audit_configs(id uuid PRIMARY KEY,tenant_id uuid,process_type text,process_type_label text);
 CREATE TABLE process_archive_configs(LIKE process_audit_configs);
 CREATE TABLE users(id uuid PRIMARY KEY,username text,display_name text);
 CREATE TABLE audit_logs(id uuid PRIMARY KEY,tenant_id uuid,process_id text,title text,process_type text,trigger_source text,trigger_detail text,oa_operator_name text,config_version_no int);
 CREATE TABLE archive_logs(id uuid PRIMARY KEY,tenant_id uuid,config_version_no int);
 CREATE TABLE process_summary_logs(id uuid PRIMARY KEY,tenant_id uuid,trigger_source text,trigger_detail text,oa_operator_name text,config_version_no int);
 CREATE TABLE chat_sessions(id uuid PRIMARY KEY,tenant_id uuid,title text);
 CREATE TABLE ai_model_configs(id uuid PRIMARY KEY,model_name text,display_name text);`).Error)
	must(db.Exec(`CREATE TABLE audit_comments(id uuid PRIMARY KEY,tenant_id uuid,audit_log_id uuid,oa_user_id text,username text,feedback text,content text,created_at timestamptz,updated_at timestamptz)`).Error)
	must(db.AutoMigrate(&model.TenantLLMMessageLog{}, &model.TenantLLMMessagePayload{}))
	migration, err := os.ReadFile("../../../db/migrations/000085_feedback_memory.up.sql")
	must(err)
	must(db.Exec(string(migration)).Error)
	migration, err = os.ReadFile("../../../db/migrations/000088_feedback_memory_lifecycle.up.sql")
	must(err)
	must(db.Exec(string(migration)).Error)
	tid, other, uid := uuid.New(), uuid.New(), uuid.New()
	must(db.Exec("INSERT INTO tenants(id) VALUES (?),(?)", tid, other).Error)
	must(db.Exec("INSERT INTO users VALUES (?,?,?)", uid, "admin", "管理员").Error)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Set("tenant_id", tid.String())
	c.Set("user_id", uid.String())
	c.Set("username", "管理员")
	c2, _ := gin.CreateTestContext(httptest.NewRecorder())
	c2.Request = c.Request
	c2.Set("tenant_id", other.String())
	scope := dto.FeedbackMemoryScope{Module: "audit", ProcessType: "100"}
	repo := NewFeedbackMemoryRepo(db)
	must(db.Exec("INSERT INTO process_audit_configs VALUES (?,?,?,?)", uuid.New(), tid, "100", "报销流程").Error)
	must(repo.ValidateScope(c, scope))
	if !errors.Is(repo.ValidateScope(c2, scope), gorm.ErrRecordNotFound) {
		t.Fatal("跨租户流程必须拒绝")
	}
	for i, tenant := range []uuid.UUID{tid, other, tid} {
		process := "100"
		if i == 2 {
			process = "200"
		}
		auditID := uuid.New()
		must(db.Exec("INSERT INTO audit_logs(id,tenant_id,process_id,title,process_type) VALUES (?,?,?,?,?)", auditID, tenant, "123", "差旅报销", process).Error)
		must(db.Create(&model.AuditComment{ID: uuid.New(), TenantID: tenant, AuditLogID: auditID, Content: "跨月票据应按实际业务时间判断", Username: "反馈人", CreatedAt: apptime.Now(), UpdatedAt: apptime.Now()}).Error)
	}
	comments, err := repo.Comments(c, scope, dto.FeedbackMemoryCommentQuery{ExperienceQuery: dto.ExperienceQuery{Page: -1, PageSize: 999}})
	must(err)
	if comments.Total != 1 || len(comments.Items) != 1 || comments.Page != 1 || comments.PageSize != 20 {
		t.Fatalf("评论隔离或分页错误: %+v", comments)
	}
	commentID := comments.Items[0].ID
	must(repo.ImportComments(c, dto.FeedbackMemoryImportRequest{FeedbackMemoryScope: scope, CommentIDs: []uuid.UUID{commentID}}))
	firstImport, err := repo.Workspace(c, scope)
	must(err)

	must(repo.ImportComments(c, dto.FeedbackMemoryImportRequest{FeedbackMemoryScope: scope, CommentIDs: []uuid.UUID{commentID}}))
	repeatedImport, err := repo.Workspace(c, scope)
	must(err)
	if repeatedImport.Memory.Revision != firstImport.Memory.Revision {
		t.Fatal("重复导入评论不应改变素材修订号")
	}

	input := dto.FeedbackMemoryEntryRequest{FeedbackMemoryScope: scope, Content: "需要人工复核时明确说明缺少哪些材料", Condition: "材料不足", Enabled: true}
	must(repo.SaveEntry(c, uuid.Nil, input))
	workspace, err := repo.Workspace(c, scope)
	must(err)
	if len(workspace.Entries) != 2 {
		t.Fatal("重复导入产生重复素材")
	}
	if workspace.Entries[0].Content != comments.Items[0].Content {
		t.Fatal("原文快照不一致")
	}
	comments, err = repo.Comments(c, scope, dto.FeedbackMemoryCommentQuery{ExperienceQuery: dto.ExperienceQuery{Keyword: "跨月", Page: 1, PageSize: 10}})
	must(err)
	if len(comments.Items) != 0 || comments.Total != 0 {
		t.Fatal("已导入评论应在关键词筛选、分页与总数统计前排除")
	}
	allComments, err := repo.Comments(c, scope, dto.FeedbackMemoryCommentQuery{Deprecation: "all"})
	must(err)
	if allComments.Total != 0 {
		t.Fatal("全部建议筛选不应重新展示已导入评论")
	}
	for _, entry := range workspace.Entries {
		must(repo.SaveEntry(c, entry.ID, dto.FeedbackMemoryEntryRequest{FeedbackMemoryScope: scope, Content: entry.Content, Condition: entry.Condition, Enabled: entry.Enabled}))
	}
	unchangedWorkspace, err := repo.Workspace(c, scope)
	must(err)
	if unchangedWorkspace.Memory.Revision != workspace.Memory.Revision {
		t.Fatal("原样保存素材不能使生成草稿失效")
	}
	oldRevision := workspace.Memory.Revision
	if !errors.Is(repo.ImportComments(c, dto.FeedbackMemoryImportRequest{FeedbackMemoryScope: scope, CommentIDs: []uuid.UUID{uuid.New()}}), gorm.ErrRecordNotFound) {
		t.Fatal("非法评论应拒绝")
	}
	workspace, err = repo.Workspace(c, scope)
	must(err)
	if workspace.Memory.Revision != oldRevision {
		t.Fatal("失败写入没有回滚")
	}
	makeVersion := func(base *uuid.UUID, rev int) *model.FeedbackMemoryVersion {
		return &model.FeedbackMemoryVersion{ID: uuid.New(), BaseVersionID: base, InputRevision: rev, Status: "draft", Sources: []byte("[]"), Items: []byte(`[{"content":"核实材料","condition":"","source_refs":["` + workspace.Entries[0].ID.String() + `"]}]`), Warnings: []byte("[]"), SystemPrompt: "生成规则", UserPrompt: "原文快照", PromptText: "核实材料", CreatedBy: "管理员", CreatedAt: apptime.Now()}
	}
	v1 := makeVersion(nil, workspace.Memory.Revision)
	must(repo.SaveVersion(c, scope, v1))
	workspace, err = repo.Workspace(c, scope)
	must(err)
	if workspace.Entries[0].ImplantedVersionID == nil || *workspace.Entries[0].ImplantedVersionID != v1.ID {
		t.Fatal("实际引用的素材未标记已植入")
	}
	if !errors.Is(repo.SaveEntry(c, workspace.Entries[0].ID, input), ErrFeedbackMemoryStale) {
		t.Fatal("旧接口可以重新启停已植入素材")
	}
	must(repo.Publish(c, scope, &v1.ID))
	if _, err := repo.Version(c2, v1.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("不能跨租户读取版本")
	}
	secondScope := dto.FeedbackMemoryScope{Module: "archive", ProcessType: "100"}
	must(repo.SaveEntry(c, uuid.Nil, dto.FeedbackMemoryEntryRequest{FeedbackMemoryScope: secondScope, Content: "归档专用记忆", Enabled: true}))
	if !errors.Is(repo.Publish(c, secondScope, &v1.ID), gorm.ErrRecordNotFound) {
		t.Fatal("不能跨模块发布版本")
	}
	v2 := makeVersion(&v1.ID, workspace.Memory.Revision)
	must(repo.SaveVersion(c, scope, v2))
	must(repo.SaveEntry(c, uuid.Nil, input))
	if !errors.Is(repo.Publish(c, scope, &v2.ID), ErrFeedbackMemoryStale) {
		t.Fatal("素材变更后旧草稿必须失效")
	}
	workspace, err = repo.Workspace(c, scope)
	must(err)
	v3 := makeVersion(&v2.ID, workspace.Memory.Revision)
	v4 := makeVersion(&v2.ID, workspace.Memory.Revision)
	must(repo.SaveVersion(c, scope, v3))
	if !errors.Is(repo.SaveVersion(c, scope, v4), ErrFeedbackMemoryStale) {
		t.Fatal("并发生成不能覆盖新基线")
	}
	v4.BaseVersionID = &v3.ID
	must(repo.SaveVersion(c, scope, v4))
	if !errors.Is(repo.Publish(c, scope, &v3.ID), ErrFeedbackMemoryStale) {
		t.Fatal("不能发布被新版替代的草稿")
	}
	must(repo.Publish(c, scope, &v4.ID))
	must(repo.Publish(c, scope, nil))
	workspace, err = repo.Workspace(c, scope)
	must(err)
	if workspace.Published == nil || workspace.Memory.Enabled || workspace.Baseline.ID != v4.ID {
		t.Fatal("停用失败")
	}
	must(repo.Enable(c, scope))
	workspace, err = repo.Workspace(c, scope)
	must(err)
	if !workspace.Memory.Enabled || workspace.Published.ID != v4.ID {
		t.Fatal("停用后无法恢复原发布版本")
	}
	original, err := repo.Version(c, v1.ID)
	must(err)
	if original.PromptText != "核实材料" || original.PublishedAt == nil {
		t.Fatal("历史版本遭修改")
	}
	history, err := repo.Versions(c, scope, dto.ExperienceQuery{Page: 1, PageSize: 1})
	must(err)
	if history.Total != 4 || len(history.Items) != 1 || history.Items[0].VersionNo != 4 {
		t.Fatal("版本号或分页错误")
	}
	unchangedVersion, err := repo.SaveDraft(c, scope, v4, v4.Items, v4.PromptText, 0, workspace.Memory.Revision)
	must(err)
	if unchangedVersion.ID != v4.ID || unchangedVersion.Status != "published" {
		t.Fatal("原样保存发布版不应创建新草稿")
	}
	// 人工编辑发布版另存草稿，草稿并发编辑与发布必须校验用户已审阅的修订。
	editedItems := []byte(`[{"content":"人工修正","condition":"","source_refs":["` + workspace.Entries[0].ID.String() + `"]}]`)
	draft, err := repo.SaveDraft(c, scope, v4, editedItems, "人工修正", 0, workspace.Memory.Revision)
	must(err)
	if draft.ID == v4.ID || draft.Status != "draft" || draft.VersionNo != 5 {
		t.Fatal("编辑发布版没有另存草稿")
	}
	sameDraft, err := repo.SaveDraft(c, scope, draft, draft.Items, draft.PromptText, 0, workspace.Memory.Revision)
	must(err)
	if sameDraft.EditRevision != 0 {
		t.Fatal("原样保存草稿不应增加编辑修订号")
	}
	updatedItems := []byte(`[{"content":"二次修正","condition":"","source_refs":["` + workspace.Entries[0].ID.String() + `"]}]`)
	updated, err := repo.SaveDraft(c, scope, draft, updatedItems, "二次修正", 0, workspace.Memory.Revision)
	must(err)
	if updated.EditRevision != 1 {
		t.Fatal("编辑修订未递增")
	}
	if _, err := repo.SaveDraft(c, scope, draft, draft.Items, "旧页面覆盖", 0, workspace.Memory.Revision); !errors.Is(err, ErrFeedbackMemoryStale) {
		t.Fatal("旧编辑覆盖了新正文")
	}
	if !errors.Is(repo.Publish(c, scope, &draft.ID, 0), ErrFeedbackMemoryStale) {
		t.Fatal("发布了未审阅的编辑修订")
	}
	must(repo.Publish(c, scope, &draft.ID, 1))
	original, err = repo.Version(c, v4.ID)
	must(err)
	if original.PromptText != "核实材料" {
		t.Fatal("编辑污染了已发布正文")
	}
	// 删除保留撤销标记与历史引用，评论回到可导入位置。
	entryID := workspace.Entries[0].ID
	must(repo.DeleteEntry(c, scope, entryID))
	workspace, err = repo.Workspace(c, scope)
	must(err)
	if len(workspace.DeletedEntries) != 1 {
		t.Fatal("删除没有保留撤销标记")
	}
	returned, err := repo.Comments(c, scope, dto.FeedbackMemoryCommentQuery{})
	must(err)
	if returned.Total != 1 || returned.Items[0].Imported {
		t.Fatal("删除评论素材后未恢复可导入状态")
	}
	// 弃用默认隐藏，恢复筛选可找回，归档与租户状态独立且不使当前草稿失效。
	before := workspace.Memory.Revision
	must(repo.SetCommentDeprecated(c, commentID, dto.FeedbackMemoryCommentStateRequest{FeedbackMemoryScope: scope, Deprecated: true}))
	available, err := repo.Comments(c, scope, dto.FeedbackMemoryCommentQuery{})
	must(err)
	if available.Total != 0 {
		t.Fatal("默认没有隐藏弃用建议")
	}
	deprecated, err := repo.Comments(c, scope, dto.FeedbackMemoryCommentQuery{Deprecation: "deprecated"})
	must(err)
	if deprecated.Total != 1 || !deprecated.Items[0].Deprecated {
		t.Fatal("弃用筛选不可恢复")
	}
	archiveComments, err := repo.Comments(c, secondScope, dto.FeedbackMemoryCommentQuery{})
	must(err)
	if archiveComments.Total != 1 {
		t.Fatal("弃用状态泄漏到归档模块")
	}
	if !errors.Is(repo.SetCommentDeprecated(c2, commentID, dto.FeedbackMemoryCommentStateRequest{FeedbackMemoryScope: scope, Deprecated: true}), gorm.ErrRecordNotFound) {
		t.Fatal("跨租户弃用成功")
	}
	if !errors.Is(repo.ImportComments(c, dto.FeedbackMemoryImportRequest{FeedbackMemoryScope: scope, CommentIDs: []uuid.UUID{commentID}}), gorm.ErrRecordNotFound) {
		t.Fatal("可直接导入已弃用建议")
	}
	must(repo.SetCommentDeprecated(c, commentID, dto.FeedbackMemoryCommentStateRequest{FeedbackMemoryScope: scope}))
	workspace, err = repo.Workspace(c, scope)
	must(err)
	if workspace.Memory.Revision != before {
		t.Fatal("弃用原建议不应修改素材修订号")
	}
	// 重新导入获得新的素材身份，并再次从导入列表隐藏。
	must(repo.ImportComments(c, dto.FeedbackMemoryImportRequest{FeedbackMemoryScope: scope, CommentIDs: []uuid.UUID{commentID}}))
	available, err = repo.Comments(c, scope, dto.FeedbackMemoryCommentQuery{Deprecation: "all"})
	must(err)
	if available.Total != 0 {
		t.Fatal("重新导入后未隐藏评论")
	}
	workspace, err = repo.Workspace(c, scope)
	must(err)
	for _, row := range workspace.Entries {
		if row.SourceID != nil && *row.SourceID == commentID && row.ID == entryID {
			t.Fatal("重新导入复用了已撤销的来源身份")
		}
	}
	cleared, err := repo.SaveDraft(c, scope, updated, []byte("[]"), "", updated.EditRevision, workspace.Memory.Revision)
	must(err)
	must(repo.Publish(c, scope, &cleared.ID, cleared.EditRevision))
	workspace, err = repo.Workspace(c, scope)
	must(err)
	if workspace.Published.PromptText != "" {
		t.Fatal("最后一条记忆不能显式清空")
	}
	// 验证 AI 辅助操作无 process_id 时仍可聚合、筛选、读详情；历史规则导入归入辅助场景。
	logs := NewLLMMessageLogRepo(db)
	operation := uuid.New()
	chat := uuid.New()
	rows := []model.TenantLLMMessageLog{
		{RequestType: "feedback_memory", BusinessLogID: &operation}, {RequestType: "feedback_memory", BusinessLogID: &operation},
		{RequestType: "sql_assist"}, {RequestType: "rule_assist"}, {RequestType: "audit", ProcessID: "rule-import:legacy"},
		{RequestType: "audit", ProcessID: "123"}, {RequestType: "archive", ProcessID: "123"}, {RequestType: "chat", BusinessLogID: &chat},
	}
	for i := range rows {
		row := &rows[i]
		row.ID = uuid.New()
		row.TenantID = tid
		row.UserID = &uid
		row.TotalTokens = 10
		row.CallType = "structured"
		row.ProcessTitle = "测试操作"
		row.CreatedAt = apptime.Now()
		must(logs.CreateWithPayload(row, &model.TenantLLMMessagePayload{SystemPrompt: "系统规则", UserPrompt: "来源快照", ResponseContent: "结果"}))
	}
	foreign := rows[0]
	foreign.ID = uuid.New()
	foreign.TenantID = other
	must(logs.Create(&foreign))
	stats, err := logs.CountStats(c)
	must(err)
	if stats.Total != 6 || stats.FeedbackMemoryCount != 1 || stats.RuleAssistCount != 2 || stats.AuditCount != 1 || stats.ArchiveCount != 1 || stats.ChatCount != 1 || stats.SQLAssistCount != 1 {
		t.Fatalf("分类统计错误: %+v", stats)
	}
	for _, kind := range []string{"feedback_memory", "sql_assist", "rule_assist", "chat", "audit"} {
		list, total, err := logs.ListProcessesPaged(c, LLMLogFilter{RequestType: kind}, 1, 20)
		must(err)
		expected := int64(1)
		if kind == "rule_assist" {
			expected = 2
		}
		if total != expected {
			t.Fatalf("%s 筛选错误: %d", kind, total)
		}
		for _, row := range list {
			chain, err := logs.ListCallsByProcessID(c, row.ProcessID)
			must(err)
			if len(chain) == 0 {
				t.Fatal("调用链为空")
			}
			detail, err := logs.GetByIDWithPayload(c, chain[0].ID)
			must(err)
			if detail.SystemPrompt != "系统规则" {
				t.Fatal("丢失提示词")
			}
			if kind == "rule_assist" && chain[0].RequestType != "rule_assist" {
				t.Fatal("历史规则场景未转换")
			}
			if kind != "audit" && kind != "chat" && row.BusinessKind != "operation" {
				t.Fatal("辅助操作分类错误")
			}
			if kind == "feedback_memory" && (row.CallCount != 2 || row.TotalTokens != 20) {
				t.Fatal("同次生成未聚合")
			}
		}
	}
	usage, err := logs.QueryByTimeRange(c, apptime.Now().Add(-time.Hour), apptime.Now().Add(time.Hour), nil)
	must(err)
	if len(usage) != 1 || usage[0].TotalTokens != 80 || usage[0].CallCount != 8 {
		t.Fatalf("新场景未计入用量: %+v", usage)
	}
	// 所有新增表字段必须具有数据库注释。
	var missing int64
	must(db.Raw(`SELECT count(*) FROM pg_attribute a JOIN pg_class c ON c.oid=a.attrelid WHERE c.relname IN ('feedback_memories','feedback_memory_entries','feedback_memory_versions') AND a.attnum>0 AND NOT a.attisdropped AND col_description(c.oid,a.attnum) IS NULL`).Scan(&missing).Error)
	if missing != 0 {
		t.Fatalf("%d 个字段缺少中文注释", missing)
	}
	down, err := os.ReadFile("../../../db/migrations/000085_feedback_memory.down.sql")
	must(err)
	must(db.Exec(string(down)).Error)
	var table *string
	must(db.Raw("SELECT to_regclass('feedback_memories')::text").Scan(&table).Error)
	if table != nil && strings.TrimSpace(*table) != "" {
		t.Fatal("迁移回滚失败")
	}
}

// TestFeedbackMemoryLifecycleMigration 验证旧停用数据恢复指针但保持停用，植入状态回填及迁移可回退重放。
func TestFeedbackMemoryLifecycleMigration(t *testing.T) {
	dsn := os.Getenv("FEEDBACK_MEMORY_TEST_DSN")
	if dsn == "" {
		t.Skip("未配置隔离测试库")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	db = db.Begin()
	defer db.Rollback()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(db.Exec("CREATE TABLE tenants(id uuid PRIMARY KEY)").Error)
	for _, name := range []string{"000085_feedback_memory.up.sql"} {
		raw, err := os.ReadFile("../../../db/migrations/" + name)
		must(err)
		must(db.Exec(string(raw)).Error)
	}
	tid, mid, eid, vid := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	must(db.Exec("INSERT INTO tenants VALUES (?)", tid).Error)
	must(db.Exec("INSERT INTO feedback_memories(id,tenant_id,module,process_type,last_version,created_at,updated_at) VALUES (?,?,'audit','100',1,?,?)", mid, tid, apptime.Now(), apptime.Now()).Error)
	must(db.Exec("INSERT INTO feedback_memory_entries(id,tenant_id,memory_id,source_type,content,created_at,updated_at) VALUES (?,?,?,'manual','旧素材',?,?)", eid, tid, mid, apptime.Now(), apptime.Now()).Error)
	items := `[{"content":"旧记忆","source_refs":["` + eid.String() + `"]}]`
	must(db.Exec("INSERT INTO feedback_memory_versions(id,tenant_id,memory_id,version_no,input_revision,status,items,system_prompt,user_prompt,prompt_text,created_at) VALUES (?,?,?,1,0,'published',?,'系统','输入','旧记忆',?)", vid, tid, mid, items, apptime.Now()).Error)
	up, err := os.ReadFile("../../../db/migrations/000088_feedback_memory_lifecycle.up.sql")
	must(err)
	down, err := os.ReadFile("../../../db/migrations/000088_feedback_memory_lifecycle.down.sql")
	must(err)
	for i := 0; i < 2; i++ {
		must(db.Exec(string(up)).Error)
		var m model.FeedbackMemory
		must(db.First(&m, "id = ?", mid).Error)
		if m.Enabled || m.PublishedVersionID == nil || *m.PublishedVersionID != vid {
			t.Fatal("旧停用记录未恢复来源基线或被意外启用")
		}
		var entry model.FeedbackMemoryEntry
		must(db.First(&entry, "id = ?", eid).Error)
		if entry.ImplantedVersionID == nil || *entry.ImplantedVersionID != vid {
			t.Fatal("旧素材植入状态未回填")
		}
		must(db.Exec(string(down)).Error)
	}
}
