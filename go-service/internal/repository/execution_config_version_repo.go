package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"auraoa/go-service/internal/model"
)

// ExecutionConfigVersionRepo 管理不可变执行配置版本与流程绑定。
type ExecutionConfigVersionRepo struct {
	db *gorm.DB
}

func NewExecutionConfigVersionRepo(db *gorm.DB) *ExecutionConfigVersionRepo {
	return &ExecutionConfigVersionRepo{db: db}
}

// GetBindingVersion 返回流程当前绑定版本；未绑定时返回 gorm.ErrRecordNotFound。
func (r *ExecutionConfigVersionRepo) GetBindingVersion(
	ctx context.Context,
	tenantID uuid.UUID,
	module, processID string,
) (*model.ExecutionConfigVersion, error) {
	var version model.ExecutionConfigVersion
	err := r.db.WithContext(ctx).
		Table("execution_config_versions AS v").
		Select("v.*").
		Joins("JOIN process_execution_config_bindings AS b ON b.config_version_id = v.id").
		Where("b.tenant_id = ? AND b.module = ? AND b.process_id = ?", tenantID, module, processID).
		First(&version).Error
	return &version, err
}

// GetVersionByID 按租户读取日志实际引用的配置版本。
func (r *ExecutionConfigVersionRepo) GetVersionByID(
	ctx context.Context,
	tenantID, versionID uuid.UUID,
) (*model.ExecutionConfigVersion, error) {
	var version model.ExecutionConfigVersion
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, versionID).
		First(&version).Error
	return &version, err
}

// BindSnapshot 将流程绑定到内容寻址的不可变配置版本。
// force=false 时保留既有绑定；force=true 仅用于用户明确选择“按最新配置重新执行”。
func (r *ExecutionConfigVersionRepo) BindSnapshot(
	ctx context.Context,
	tenantID, userID uuid.UUID,
	module, processID, processType string,
	sourceConfigID uuid.UUID,
	fingerprint string,
	snapshot interface{},
	force bool,
) (*model.ExecutionConfigVersion, error) {
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return nil, fmt.Errorf("序列化执行配置快照失败: %w", err)
	}
	var result model.ExecutionConfigVersion
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 同一租户、模块和源配置串行分配版本号，避免并发首次执行产生重复版本号。
		lockKey := tenantID.String() + ":" + module + ":" + sourceConfigID.String()
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", lockKey).Error; err != nil {
			return err
		}

		var version model.ExecutionConfigVersion
		err := tx.Where(
			"tenant_id = ? AND module = ? AND source_config_id = ? AND fingerprint = ?",
			tenantID, module, sourceConfigID, fingerprint,
		).First(&version).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		if err == gorm.ErrRecordNotFound {
			var maxVersion int
			if err := tx.Model(&model.ExecutionConfigVersion{}).
				Where("tenant_id = ? AND module = ? AND source_config_id = ?", tenantID, module, sourceConfigID).
				Select("COALESCE(MAX(version_no), 0)").Scan(&maxVersion).Error; err != nil {
				return err
			}
			version = model.ExecutionConfigVersion{
				ID:             uuid.New(),
				TenantID:       tenantID,
				Module:         module,
				SourceConfigID: sourceConfigID,
				VersionNo:      maxVersion + 1,
				Fingerprint:    fingerprint,
				ConfigSnapshot: datatypes.JSON(raw),
				CreatedBy:      &userID,
			}
			if err := tx.Create(&version).Error; err != nil {
				return err
			}
		}

		binding := model.ProcessExecutionConfigBinding{
			ID:              uuid.New(),
			TenantID:        tenantID,
			Module:          module,
			ProcessID:       processID,
			ProcessType:     processType,
			ConfigVersionID: version.ID,
			BoundBy:         &userID,
		}
		if force {
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "tenant_id"}, {Name: "module"}, {Name: "process_id"}},
				DoUpdates: clause.Assignments(map[string]interface{}{
					"process_type":      processType,
					"config_version_id": version.ID,
					"bound_by":          userID,
					"updated_at":        gorm.Expr("NOW()"),
				}),
			}).Create(&binding).Error; err != nil {
				return err
			}
			result = version
			return nil
		}

		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "module"}, {Name: "process_id"}},
			DoNothing: true,
		}).Create(&binding).Error; err != nil {
			return err
		}
		return tx.Table("execution_config_versions AS v").
			Select("v.*").
			Joins("JOIN process_execution_config_bindings AS b ON b.config_version_id = v.id").
			Where("b.tenant_id = ? AND b.module = ? AND b.process_id = ?", tenantID, module, processID).
			First(&result).Error
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}
