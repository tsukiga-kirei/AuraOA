package repository

import (
	"encoding/json"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"auraoa/go-service/internal/dto"
	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/apptime"
)

// ErrFeedbackMemoryStale 表示素材或发布基线发生变化，旧草稿不能覆盖新状态。
var ErrFeedbackMemoryStale = errors.New("反馈素材或已发布记忆已变化，请重新生成")

// ErrFeedbackMemoryLimit 表示同一流程的素材条数超过上限。
var ErrFeedbackMemoryLimit = errors.New("每个流程最多保留200条反馈素材，请先整理或删除不再需要的素材")

// ErrFeedbackMemoryEmpty 表示没有可靠条目的草稿不能发布。
var ErrFeedbackMemoryEmpty = errors.New("草稿没有可靠记忆条目，请补充资料后重新生成")

// FeedbackMemoryRepo 提供有租户边界的素材、版本和发布操作。
type FeedbackMemoryRepo struct{ *BaseRepo }

// NewFeedbackMemoryRepo 创建反馈记忆仓储。
func NewFeedbackMemoryRepo(db *gorm.DB) *FeedbackMemoryRepo {
	return &FeedbackMemoryRepo{NewBaseRepo(db)}
}

// Configs 返回当前租户可选择的流程类型，避免把实例 ID 当作流程类型。
func (r *FeedbackMemoryRepo) Configs(c *gin.Context, module string) ([]map[string]interface{}, error) {
	table := "process_audit_configs"
	if module == "archive" {
		table = "process_archive_configs"
	}
	items := []map[string]interface{}{}
	err := r.WithTenant(c).Table(table).Select("process_type, process_type_label").Order("process_type_label, process_type").Find(&items).Error
	return items, err
}

// ValidateScope 只允许当前租户已配置的流程。
func (r *FeedbackMemoryRepo) ValidateScope(c *gin.Context, scope dto.FeedbackMemoryScope) error {
	table := "process_audit_configs"
	if scope.Module == "archive" {
		table = "process_archive_configs"
	}
	var count int64
	if err := r.WithTenant(c).Table(table).Where("process_type = ?", scope.ProcessType).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Get 按类型读取记忆；未创建时返回空状态。
func (r *FeedbackMemoryRepo) Get(c *gin.Context, scope dto.FeedbackMemoryScope) (*model.FeedbackMemory, error) {
	var m model.FeedbackMemory
	err := r.WithTenant(c).Where("module = ? AND process_type = ?", scope.Module, scope.ProcessType).First(&m).Error
	return &m, err
}

func (r *FeedbackMemoryRepo) ensure(c *gin.Context, scope dto.FeedbackMemoryScope) (*model.FeedbackMemory, error) {
	tid, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil {
		return nil, ErrNoTenantContext
	}
	now := apptime.Now()
	m := model.FeedbackMemory{ID: uuid.New(), TenantID: tid, Module: scope.Module, ProcessType: scope.ProcessType, Enabled: true, CreatedAt: now, UpdatedAt: now}
	if err = r.WithTenant(c).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "tenant_id"}, {Name: "module"}, {Name: "process_type"}}, DoNothing: true}).Create(&m).Error; err != nil {
		return nil, err
	}
	return r.Get(c, scope)
}

// Workspace 在短事务内读取一致的素材和发布基线，AI 调用不占用数据库锁。
func (r *FeedbackMemoryRepo) Workspace(c *gin.Context, scope dto.FeedbackMemoryScope) (*dto.FeedbackMemoryWorkspace, error) {
	out := &dto.FeedbackMemoryWorkspace{Entries: []model.FeedbackMemoryEntry{}}
	m, err := r.Get(c, scope)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	err = r.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		repo := NewFeedbackMemoryRepo(tx)
		if err := repo.WithTenant(c).Clauses(clause.Locking{Strength: "SHARE"}).First(m, "id = ?", m.ID).Error; err != nil {
			return err
		}
		out.Memory = m
		if err := repo.WithTenant(c).Where("memory_id = ? AND deleted_at IS NULL", m.ID).Order("created_at, id").Find(&out.Entries).Error; err != nil {
			return err
		}
		if err := repo.WithTenant(c).Where("memory_id = ? AND deleted_at IS NOT NULL", m.ID).Find(&out.DeletedEntries).Error; err != nil {
			return err
		}
		if m.LastVersion > 0 {
			var v model.FeedbackMemoryVersion
			if err := repo.WithTenant(c).Where("memory_id = ? AND version_no = ?", m.ID, m.LastVersion).First(&v).Error; err != nil {
				return err
			}
			out.Baseline = &v
		}
		if m.PublishedVersionID != nil {
			out.Published, err = repo.Version(c, *m.PublishedVersionID)
			return err
		}
		return nil
	})
	return out, err
}

// Mutate 锁定记忆头后修改素材，并递增修订号以使旧生成草稿失效。
func (r *FeedbackMemoryRepo) Mutate(c *gin.Context, scope dto.FeedbackMemoryScope, fn func(*FeedbackMemoryRepo, *model.FeedbackMemory) error) error {
	m, err := r.ensure(c, scope)
	if err != nil {
		return err
	}
	return r.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		repo := NewFeedbackMemoryRepo(tx)
		if err := repo.WithTenant(c).Clauses(clause.Locking{Strength: "UPDATE"}).First(m, "id = ?", m.ID).Error; err != nil {
			return err
		}
		if err := fn(repo, m); err != nil {
			return err
		}
		return repo.WithTenant(c).Model(m).Updates(map[string]interface{}{"revision": m.Revision + 1, "updated_at": apptime.Now()}).Error
	})
}

// SaveEntry 新增手工条件或更新素材；评论原文只读，可调整适用条件与启停。
func (r *FeedbackMemoryRepo) SaveEntry(c *gin.Context, id uuid.UUID, req dto.FeedbackMemoryEntryRequest) error {
	return r.Mutate(c, req.FeedbackMemoryScope, func(repo *FeedbackMemoryRepo, m *model.FeedbackMemory) error {
		if id == uuid.Nil {
			var count int64
			if err := repo.WithTenant(c).Model(&model.FeedbackMemoryEntry{}).Where("memory_id = ? AND deleted_at IS NULL", m.ID).Count(&count).Error; err != nil {
				return err
			}
			if count >= 200 {
				return ErrFeedbackMemoryLimit
			}
			now := apptime.Now()
			return repo.WithTenant(c).Create(&model.FeedbackMemoryEntry{ID: uuid.New(), TenantID: m.TenantID, MemoryID: m.ID, SourceType: "manual", Content: req.Content, Condition: req.Condition, Enabled: req.Enabled, SourceSnapshot: []byte("{}"), CreatedBy: c.GetString("username"), CreatedAt: now, UpdatedAt: now}).Error
		}
		var row model.FeedbackMemoryEntry
		if err := repo.WithTenant(c).Where("memory_id = ? AND id = ? AND deleted_at IS NULL", m.ID, id).First(&row).Error; err != nil {
			return err
		}
		if row.SourceType == "comment" {
			req.Content = row.Content
		}
		// 已植入素材由版本条目维护，禁止通过旧接口修改或重新参与生成。
		if row.ImplantedVersionID != nil {
			return ErrFeedbackMemoryStale
		}
		return repo.WithTenant(c).Model(&row).Updates(map[string]interface{}{"content": req.Content, "condition": req.Condition, "enabled": req.Enabled, "updated_at": apptime.Now()}).Error
	})
}

// DeleteEntry 删除素材不改写历史快照，也不会自动改变已发布记忆。
func (r *FeedbackMemoryRepo) DeleteEntry(c *gin.Context, scope dto.FeedbackMemoryScope, id uuid.UUID) error {
	return r.Mutate(c, scope, func(repo *FeedbackMemoryRepo, m *model.FeedbackMemory) error {
		result := repo.WithTenant(c).Model(&model.FeedbackMemoryEntry{}).Where("memory_id = ? AND id = ? AND deleted_at IS NULL", m.ID, id).Updates(map[string]interface{}{"deleted_at": apptime.Now(), "updated_at": apptime.Now()})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (r *FeedbackMemoryRepo) commentsQuery(c *gin.Context, scope dto.FeedbackMemoryScope) *gorm.DB {
	comments := r.WithTenant(c).Model(&model.AuditComment{})
	// 归档可以复用同流程类型的审核反馈，不能跨流程或跨租户导入。
	return r.DB.Table("(?) AS m", comments).Joins("JOIN audit_logs a ON a.id=m.audit_log_id AND a.tenant_id=m.tenant_id").Where("a.process_type = ?", scope.ProcessType)
}

const memoryCommentColumns = "m.id, m.audit_log_id, a.process_id, a.title, m.username, m.content, m.feedback, m.updated_at"

// Comments 分页筛选当前流程类型的原始评论。
func (r *FeedbackMemoryRepo) Comments(c *gin.Context, scope dto.FeedbackMemoryScope, q dto.FeedbackMemoryCommentQuery) (*dto.ExperiencePage[dto.FeedbackMemoryComment], error) {
	normalizeExperiencePage(&q.ExperienceQuery)
	out := &dto.ExperiencePage[dto.FeedbackMemoryComment]{Items: []dto.FeedbackMemoryComment{}, Page: q.Page, PageSize: q.PageSize}
	query := r.commentsQuery(c, scope)
	// 已导入评论在分页和总数统计前排除；删除素材后自然回到可导入列表。
	query = query.Where(`NOT EXISTS (SELECT 1 FROM feedback_memory_entries e JOIN feedback_memories fm ON fm.id=e.memory_id AND fm.tenant_id=e.tenant_id WHERE e.tenant_id=m.tenant_id AND e.source_id=m.id AND e.deleted_at IS NULL AND fm.module=? AND fm.process_type=?)`, scope.Module, scope.ProcessType)
	deprecatedSQL := `EXISTS (SELECT 1 FROM feedback_memory_comment_exclusions x WHERE x.tenant_id=m.tenant_id AND x.module=? AND x.process_type=? AND x.comment_id=m.id)`
	if q.Deprecation != "all" {
		filter := "NOT " + deprecatedSQL
		if q.Deprecation == "deprecated" {
			filter = deprecatedSQL
		}
		query = query.Where(filter, scope.Module, scope.ProcessType)
	}
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		query = query.Where("m.content ILIKE ? OR a.title ILIKE ? OR m.username ILIKE ?", like, like, like)
	}
	if q.Feedback == "like" || q.Feedback == "dislike" {
		query = query.Where("m.feedback = ?", q.Feedback)
	}
	if err := query.Session(&gorm.Session{}).Count(&out.Total).Error; err != nil {
		return nil, err
	}
	err := query.Select(memoryCommentColumns+`, EXISTS (SELECT 1 FROM feedback_memory_entries e JOIN feedback_memories fm ON fm.id=e.memory_id AND fm.tenant_id=e.tenant_id WHERE e.tenant_id=m.tenant_id AND e.source_id=m.id AND e.deleted_at IS NULL AND fm.module=? AND fm.process_type=?) AS imported, `+deprecatedSQL+` AS deprecated`, scope.Module, scope.ProcessType, scope.Module, scope.ProcessType).Order("m.updated_at DESC,m.id DESC").Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Scan(&out.Items).Error
	return out, err
}

// ImportComments 服务端读取和验证原评论，并保存当时内容；重复选择不会重复导入。
func (r *FeedbackMemoryRepo) ImportComments(c *gin.Context, req dto.FeedbackMemoryImportRequest) error {
	return r.Mutate(c, req.FeedbackMemoryScope, func(repo *FeedbackMemoryRepo, m *model.FeedbackMemory) error {
		rows := []dto.FeedbackMemoryComment{}
		if err := repo.commentsQuery(c, req.FeedbackMemoryScope).Where("m.id IN ?", req.CommentIDs).Where(`NOT EXISTS (SELECT 1 FROM feedback_memory_comment_exclusions x WHERE x.tenant_id=m.tenant_id AND x.module=? AND x.process_type=? AND x.comment_id=m.id)`, req.Module, req.ProcessType).Select(memoryCommentColumns).Scan(&rows).Error; err != nil {
			return err
		}
		unique := map[uuid.UUID]bool{}
		for _, id := range req.CommentIDs {
			unique[id] = true
		}
		if len(rows) != len(unique) {
			return gorm.ErrRecordNotFound
		}
		var count int64
		if err := repo.WithTenant(c).Model(&model.FeedbackMemoryEntry{}).Where("memory_id = ? AND deleted_at IS NULL", m.ID).Count(&count).Error; err != nil {
			return err
		}
		var imported int64
		if err := repo.WithTenant(c).Model(&model.FeedbackMemoryEntry{}).Where("memory_id = ? AND deleted_at IS NULL AND source_id IN ?", m.ID, req.CommentIDs).Count(&imported).Error; err != nil {
			return err
		}
		if count+int64(len(rows))-imported > 200 {
			return ErrFeedbackMemoryLimit
		}
		for _, row := range rows {
			raw, _ := json.Marshal(row)
			now := apptime.Now()
			id := row.ID
			entry := model.FeedbackMemoryEntry{ID: uuid.New(), TenantID: m.TenantID, MemoryID: m.ID, SourceType: "comment", SourceID: &id, Content: row.Content, Enabled: true, SourceSnapshot: raw, CreatedBy: c.GetString("username"), CreatedAt: now, UpdatedAt: now}
			if err := repo.WithTenant(c).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "memory_id"}, {Name: "source_id"}}, TargetWhere: clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "deleted_at IS NULL"}}}, DoNothing: true}).Create(&entry).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// SetCommentDeprecated 独立设置当前记忆库的建议弃用状态；不改变已导入素材和原评论。
func (r *FeedbackMemoryRepo) SetCommentDeprecated(c *gin.Context, id uuid.UUID, req dto.FeedbackMemoryCommentStateRequest) error {
	m, err := r.ensure(c, req.FeedbackMemoryScope)
	if err != nil {
		return err
	}
	return r.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		repo := NewFeedbackMemoryRepo(tx)
		if err := repo.WithTenant(c).Clauses(clause.Locking{Strength: "UPDATE"}).First(m, "id = ?", m.ID).Error; err != nil {
			return err
		}
		var count int64
		if err := repo.commentsQuery(c, req.FeedbackMemoryScope).Where("m.id = ?", id).Count(&count).Error; err != nil {
			return err
		}
		if count != 1 {
			return gorm.ErrRecordNotFound
		}
		row := model.FeedbackMemoryCommentExclusion{TenantID: m.TenantID, Module: req.Module, ProcessType: req.ProcessType, CommentID: id}
		if req.Deprecated {
			return repo.WithTenant(c).Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error
		}
		return repo.WithTenant(c).Where("module = ? AND process_type = ? AND comment_id = ?", req.Module, req.ProcessType, id).Delete(&row).Error
	})
}

// Version 返回租户内不可变生成快照。
func (r *FeedbackMemoryRepo) Version(c *gin.Context, id uuid.UUID) (*model.FeedbackMemoryVersion, error) {
	var v model.FeedbackMemoryVersion
	err := r.WithTenant(c).First(&v, "id = ?", id).Error
	return &v, err
}

// Versions 分页返回版本索引，完整输入输出通过单条详情加载。
func (r *FeedbackMemoryRepo) Versions(c *gin.Context, scope dto.FeedbackMemoryScope, q dto.ExperienceQuery) (*dto.ExperiencePage[model.FeedbackMemoryVersion], error) {
	normalizeExperiencePage(&q)
	out := &dto.ExperiencePage[model.FeedbackMemoryVersion]{Items: []model.FeedbackMemoryVersion{}, Page: q.Page, PageSize: q.PageSize}
	m, err := r.Get(c, scope)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	query := r.WithTenant(c).Model(&model.FeedbackMemoryVersion{}).Where("memory_id = ?", m.ID)
	if err = query.Session(&gorm.Session{}).Count(&out.Total).Error; err != nil {
		return nil, err
	}
	err = query.Select("id, memory_id, version_no, base_version_id, input_revision, status, created_by, created_at, published_at, edit_revision, edited_by, edited_at").Order("version_no DESC").Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&out.Items).Error
	return out, err
}

// SaveVersion 对同一流程串行分配版本号，保留生成时使用的旧基线用于发布校验。
func (r *FeedbackMemoryRepo) SaveVersion(c *gin.Context, scope dto.FeedbackMemoryScope, v *model.FeedbackMemoryVersion) error {
	m, err := r.ensure(c, scope)
	if err != nil {
		return err
	}
	return r.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		repo := NewFeedbackMemoryRepo(tx)
		if err := repo.WithTenant(c).Clauses(clause.Locking{Strength: "UPDATE"}).First(m, "id = ?", m.ID).Error; err != nil {
			return err
		}
		v.MemoryID = m.ID
		v.TenantID = m.TenantID
		if v.InputRevision != m.Revision {
			return ErrFeedbackMemoryStale
		}
		if m.LastVersion == 0 {
			if v.BaseVersionID != nil {
				return ErrFeedbackMemoryStale
			}
		} else {
			var latest model.FeedbackMemoryVersion
			if err := repo.WithTenant(c).Where("memory_id = ? AND version_no = ?", m.ID, m.LastVersion).First(&latest).Error; err != nil {
				return err
			}
			if v.BaseVersionID == nil || *v.BaseVersionID != latest.ID || v.BaseEditRevision != latest.EditRevision {
				return ErrFeedbackMemoryStale
			}
		}
		v.VersionNo = m.LastVersion + 1
		if len(v.GeneratedItems) == 0 {
			v.GeneratedItems = append([]byte(nil), v.Items...)
		}
		if err := repo.WithTenant(c).Create(v).Error; err != nil {
			return err
		}
		if err := repo.markImplanted(c, v); err != nil {
			return err
		}
		return repo.WithTenant(c).Model(m).Update("last_version", v.VersionNo).Error
	})
}

// Publish 检查素材修订号与发布基线，原子切换当前版本；下线保留所有历史快照。
func (r *FeedbackMemoryRepo) Publish(c *gin.Context, scope dto.FeedbackMemoryScope, id *uuid.UUID, editRevision ...int) error {
	m, err := r.Get(c, scope)
	if err != nil {
		return err
	}
	return r.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		repo := NewFeedbackMemoryRepo(tx)
		if err := repo.WithTenant(c).Clauses(clause.Locking{Strength: "UPDATE"}).First(m, "id = ?", m.ID).Error; err != nil {
			return err
		}
		if id != nil {
			v, err := repo.Version(c, *id)
			if err != nil {
				return err
			}
			if v.MemoryID != m.ID {
				return gorm.ErrRecordNotFound
			}
			if len(editRevision) > 0 && v.EditRevision != editRevision[0] {
				return ErrFeedbackMemoryStale
			}
			if m.PublishedVersionID != nil && *m.PublishedVersionID == v.ID {
				return repo.WithTenant(c).Model(m).Updates(map[string]interface{}{"enabled": true, "updated_at": apptime.Now()}).Error
			}
			// 有旧基线的空草稿可显式发布，用于撤销最后一条记忆；首版空结果不能发布。
			if v.PromptText == "" && v.BaseVersionID == nil {
				return ErrFeedbackMemoryEmpty
			}
			if v.Status != "draft" || v.InputRevision != m.Revision || v.VersionNo != m.LastVersion {
				return ErrFeedbackMemoryStale
			}
			if err := repo.WithTenant(c).Model(v).Updates(map[string]interface{}{"status": "published", "published_at": apptime.Now()}).Error; err != nil {
				return err
			}
		}
		updates := map[string]interface{}{"enabled": id != nil, "updated_at": apptime.Now()}
		if id != nil {
			updates["published_version_id"] = id
		}
		return repo.WithTenant(c).Model(m).Updates(updates).Error
	})
}

func (r *FeedbackMemoryRepo) markImplanted(c *gin.Context, v *model.FeedbackMemoryVersion) error {
	var items []model.FeedbackMemoryItem
	if err := json.Unmarshal(v.Items, &items); err != nil {
		return err
	}
	ids := []uuid.UUID{}
	for _, item := range items {
		for _, ref := range item.SourceRefs {
			if id, err := uuid.Parse(ref); err == nil {
				ids = append(ids, id)
			}
		}
	}
	if len(ids) == 0 {
		return nil
	}
	return r.WithTenant(c).Model(&model.FeedbackMemoryEntry{}).Where("memory_id = ? AND id IN ? AND deleted_at IS NULL AND implanted_version_id IS NULL", v.MemoryID, ids).Update("implanted_version_id", v.ID).Error
}

// SaveDraft 只编辑最新草稿；编辑已发布版时创建新版，历史正文不变。
func (r *FeedbackMemoryRepo) SaveDraft(c *gin.Context, scope dto.FeedbackMemoryScope, original *model.FeedbackMemoryVersion, items []byte, prompt string, editRevision, inputRevision int) (*model.FeedbackMemoryVersion, error) {
	var out *model.FeedbackMemoryVersion
	err := r.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		repo := NewFeedbackMemoryRepo(tx)
		m, err := repo.Get(c, scope)
		if err != nil {
			return err
		}
		if err := repo.WithTenant(c).Clauses(clause.Locking{Strength: "UPDATE"}).First(m, "id = ?", m.ID).Error; err != nil {
			return err
		}
		v, err := repo.Version(c, original.ID)
		if err != nil {
			return err
		}
		if v.MemoryID != m.ID {
			return gorm.ErrRecordNotFound
		}
		if v.VersionNo != m.LastVersion || v.EditRevision != editRevision || inputRevision != m.Revision || (v.Status == "draft" && v.InputRevision != m.Revision) {
			return ErrFeedbackMemoryStale
		}
		now := apptime.Now()
		if v.Status == "published" {
			base := v.ID
			v.ID = uuid.New()
			v.BaseVersionID = &base
			v.VersionNo = m.LastVersion + 1
			v.Status = "draft"
			v.PublishedAt = nil
			v.CreatedAt = now
			v.CreatedBy = c.GetString("username")
			v.InputRevision = m.Revision
			v.EditRevision = 0
			v.Items = items
			v.PromptText = prompt
			v.EditedAt = &now
			v.EditedBy = c.GetString("username")
			if err := repo.WithTenant(c).Create(v).Error; err != nil {
				return err
			}
			if err := repo.WithTenant(c).Model(m).Update("last_version", v.VersionNo).Error; err != nil {
				return err
			}
		} else {
			v.Items = items
			v.PromptText = prompt
			v.EditRevision++
			v.EditedAt = &now
			v.EditedBy = c.GetString("username")
			if err := repo.WithTenant(c).Model(v).Select("items", "prompt_text", "edit_revision", "edited_at", "edited_by").Updates(v).Error; err != nil {
				return err
			}
		}
		if err := repo.markImplanted(c, v); err != nil {
			return err
		}
		out = v
		return nil
	})
	return out, err
}

// Enable 恢复最后发布版本，不需要重新生成，也不改变生成基线。
func (r *FeedbackMemoryRepo) Enable(c *gin.Context, scope dto.FeedbackMemoryScope) error {
	result := r.WithTenant(c).Model(&model.FeedbackMemory{}).Where("module = ? AND process_type = ? AND published_version_id IS NOT NULL", scope.Module, scope.ProcessType).Updates(map[string]interface{}{"enabled": true, "updated_at": apptime.Now()})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
