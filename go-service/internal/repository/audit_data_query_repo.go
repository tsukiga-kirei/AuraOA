package repository

import (
	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/apptime"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AuditDataQueryRepo 为查询库及不可变版本提供租户隔离的数据访问。
type AuditDataQueryRepo struct{ *BaseRepo }

func NewAuditDataQueryRepo(db *gorm.DB) *AuditDataQueryRepo {
	return &AuditDataQueryRepo{NewBaseRepo(db)}
}

func (r *AuditDataQueryRepo) List(c *gin.Context, keyword string, page, size int) ([]model.AuditDataQuery, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	var items []model.AuditDataQuery
	var total int64
	q := r.WithTenant(c).WithContext(c.Request.Context()).Model(&model.AuditDataQuery{}).Where("archived = false")
	if keyword != "" {
		q = q.Where("name ILIKE ? OR category ILIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("updated_at DESC, id ASC").Offset((page - 1) * size).Limit(size).Find(&items).Error
	if items == nil {
		items = []model.AuditDataQuery{}
	}
	return items, total, err
}

func (r *AuditDataQueryRepo) Get(c *gin.Context, id uuid.UUID) (*model.AuditDataQuery, error) {
	var item model.AuditDataQuery
	err := r.WithTenant(c).WithContext(c.Request.Context()).First(&item, "id = ?", id).Error
	return &item, err
}

func (r *AuditDataQueryRepo) Save(c *gin.Context, item *model.AuditDataQuery, create bool) error {
	if create {
		return r.WithTenant(c).WithContext(c.Request.Context()).Create(item).Error
	}
	return r.WithTenant(c).WithContext(c.Request.Context()).
		Model(&model.AuditDataQuery{}).Where("id = ?", item.ID).
		Updates(map[string]interface{}{
			"name": item.Name, "category": item.Category, "draft": item.Draft,
			"updated_at": apptime.Now(), "tested_hash": "", "tested_at": nil,
		}).Error
}

func (r *AuditDataQueryRepo) MarkTested(c *gin.Context, id uuid.UUID, draft datatypes.JSON, hash string) error {
	return r.WithTenant(c).WithContext(c.Request.Context()).
		Model(&model.AuditDataQuery{}).Where("id = ? AND draft = ?::jsonb", id, string(draft)).
		Updates(map[string]interface{}{"tested_hash": hash, "tested_at": apptime.Now()}).Error
}

// ErrDataQueryReferenced 表示仍有当前流程配置引用该条目，不能删除。
var ErrDataQueryReferenced = errors.New("请先取消所有流程引用，再删除数据记忆条目")

// Delete 从库中删除条目；保留不可变版本供历史执行快照追溯。
func (r *AuditDataQueryRepo) Delete(c *gin.Context, id uuid.UUID) error {
	return r.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		scoped := NewAuditDataQueryRepo(tx)
		var item model.AuditDataQuery
		if err := scoped.WithTenant(c).Clauses(clause.Locking{Strength: "UPDATE"}).First(&item, "id = ? AND archived = false", id).Error; err != nil {
			return err
		}
		refs, err := scoped.References(c, id)
		if err != nil {
			return err
		}
		// 关闭的规则和关闭的引用同样保留关联，必须明确取消后才能删除。
		if len(refs) > 0 {
			return ErrDataQueryReferenced
		}
		return scoped.WithTenant(c).Model(&item).
			Updates(map[string]interface{}{"archived": true, "updated_at": apptime.Now()}).Error
	})
}

// RemoveReferences 取消某流程全部当前规则对指定条目的引用，保留其他数据记忆。
func (r *AuditDataQueryRepo) RemoveReferences(c *gin.Context, id uuid.UUID, module, processType string) (int64, error) {
	if module != "audit_rules" && module != "archive_rules" {
		return 0, fmt.Errorf("引用模块无效")
	}
	var count int64
	err := r.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		scoped := NewAuditDataQueryRepo(tx)
		var item model.AuditDataQuery
		if err := scoped.WithTenant(c).Clauses(clause.Locking{Strength: "UPDATE"}).First(&item, "id = ? AND archived = false", id).Error; err != nil {
			return err
		}
		// 使用数据库内的当前 JSON 原子移除，避免覆盖并行修改的其他引用。
		result := scoped.WithTenant(c).Table(module).
			Where("process_type = ?", processType).
			Where("data_queries @> ?::jsonb", fmt.Sprintf(`[{"query_id":%q}]`, id.String())).
			Updates(map[string]interface{}{
				"data_queries": gorm.Expr(`COALESCE((SELECT jsonb_agg(entry) FROM jsonb_array_elements(data_queries) AS entry WHERE entry->>'query_id' IS DISTINCT FROM ?), '[]'::jsonb)`, id.String()),
				"updated_at":   apptime.Now(),
			})
		count = result.RowsAffected
		return result.Error
	})
	return count, err
}

func (r *AuditDataQueryRepo) Versions(c *gin.Context, id uuid.UUID) ([]model.AuditDataQueryVersion, error) {
	items := []model.AuditDataQueryVersion{}
	err := r.WithTenant(c).WithContext(c.Request.Context()).Where("query_id = ?", id).Order("version_no DESC").Find(&items).Error
	return items, err
}

func (r *AuditDataQueryRepo) Version(c *gin.Context, id uuid.UUID, no int) (*model.AuditDataQueryVersion, error) {
	var item model.AuditDataQueryVersion
	err := r.WithTenant(c).WithContext(c.Request.Context()).Where("query_id = ? AND version_no = ?", id, no).First(&item).Error
	return &item, err
}

// Publish 通过行锁串行分配版本号，并验证预览的是当前草稿及当前数据源。
func (r *AuditDataQueryRepo) Publish(c *gin.Context, id uuid.UUID, hash string) (*model.AuditDataQueryVersion, error) {
	var version model.AuditDataQueryVersion
	err := r.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		scoped := NewAuditDataQueryRepo(tx)
		var item model.AuditDataQuery
		if err := scoped.WithTenant(c).Clauses(clause.Locking{Strength: "UPDATE"}).First(&item, "id = ? AND archived = false", id).Error; err != nil {
			return err
		}
		if item.TestedHash == "" || item.TestedHash != hash {
			return fmt.Errorf("请先对当前已保存草稿运行成功的预览")
		}
		version = model.AuditDataQueryVersion{
			ID: uuid.New(), TenantID: item.TenantID, QueryID: id,
			VersionNo: item.LatestVersion + 1, Definition: item.Draft, CreatedAt: apptime.Now(),
		}
		if err := tx.Create(&version).Error; err != nil {
			return err
		}
		return scoped.WithTenant(c).Model(&item).Updates(map[string]interface{}{"latest_version": version.VersionNo, "updated_at": apptime.Now()}).Error
	})
	return &version, err
}

func (r *AuditDataQueryRepo) Prompts(c *gin.Context, tenantID uuid.UUID) (*model.AuditDataQueryPrompts, error) {
	var p model.AuditDataQueryPrompts
	err := r.WithTenant(c).WithContext(c.Request.Context()).First(&p, "tenant_id = ?", tenantID).Error
	return &p, err
}

func (r *AuditDataQueryRepo) SavePrompts(c *gin.Context, p *model.AuditDataQueryPrompts) error {
	return r.WithTenant(c).WithContext(c.Request.Context()).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"generate", "explain", "repair", "updated_at"}),
	}).Create(p).Error
}

// References 返回当前配置中的引用；历史执行快照及发布版本不在此列表中。
func (r *AuditDataQueryRepo) References(c *gin.Context, id uuid.UUID) ([]map[string]interface{}, error) {
	out := []map[string]interface{}{}
	for _, table := range []string{"audit_rules", "archive_rules"} {
		var rows []map[string]interface{}
		err := r.WithTenant(c).WithContext(c.Request.Context()).Table(table).
			Select("id, process_type, rule_content").
			Where("data_queries @> ?::jsonb", fmt.Sprintf(`[{"query_id":%q}]`, id.String())).Find(&rows).Error
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			row["module"] = table
			out = append(out, row)
		}
	}
	return out, nil
}
