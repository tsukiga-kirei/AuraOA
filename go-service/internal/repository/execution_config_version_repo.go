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
	return bindingVersionResult(version, err)
}

// bindingVersionResult 保证“未找到流程绑定”不会伪装成一个全零配置版本。
// 调用方依赖 nil 判断是否需要创建首次绑定，因此错误场景不能返回零值对象。
func bindingVersionResult(version model.ExecutionConfigVersion, err error) (*model.ExecutionConfigVersion, error) {
	if err != nil {
		return nil, err
	}
	return &version, nil
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

// GetLatestBaseVersion 获取租户当前最新发布的基础配置版本。
func (r *ExecutionConfigVersionRepo) GetLatestBaseVersion(
	ctx context.Context,
	tenantID uuid.UUID,
	module string,
	sourceConfigID uuid.UUID,
) (*model.TenantConfigVersion, error) {
	var version model.TenantConfigVersion
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND module = ? AND source_config_id = ?", tenantID, module, sourceConfigID).
		Order("version_no DESC").
		First(&version).Error
	if err != nil {
		return nil, err
	}
	return &version, nil
}

// PublishBaseVersion 显式发布新基础版本。
// 如果当前最新版本与当前快照指纹一致，则直接返回最新版本（不重复升版）；
// 否则分配新版本号（maxVersion + 1）并写入快照。
func (r *ExecutionConfigVersionRepo) PublishBaseVersion(
	ctx context.Context,
	tenantID, userID uuid.UUID,
	module string,
	sourceConfigID uuid.UUID,
	fingerprint string,
	snapshot interface{},
) (*model.TenantConfigVersion, error) {
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return nil, fmt.Errorf("序列化租户基础配置快照失败: %w", err)
	}
	var result model.TenantConfigVersion
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		lockKey := "base:" + tenantID.String() + ":" + module + ":" + sourceConfigID.String()
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", lockKey).Error; err != nil {
			return err
		}

		var latest model.TenantConfigVersion
		err := tx.Where(
			"tenant_id = ? AND module = ? AND source_config_id = ?",
			tenantID, module, sourceConfigID,
		).Order("version_no DESC").First(&latest).Error

		if err == nil && latest.Fingerprint == fingerprint {
			result = latest
			return nil
		}
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}

		var maxVersion int
		if err := tx.Model(&model.TenantConfigVersion{}).
			Where("tenant_id = ? AND module = ? AND source_config_id = ?", tenantID, module, sourceConfigID).
			Select("COALESCE(MAX(version_no), 0)").Scan(&maxVersion).Error; err != nil {
			return err
		}

		result = model.TenantConfigVersion{
			ID: uuid.New(), TenantID: tenantID, Module: module, SourceConfigID: sourceConfigID,
			VersionNo: maxVersion + 1, Fingerprint: fingerprint, ConfigSnapshot: datatypes.JSON(raw),
			CreatedBy: &userID,
		}
		return tx.Create(&result).Error
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetOrCreateLatestBaseVersion 获取最新基础版本；若尚无任何版本（首次使用），则创建初始版本 V1。
func (r *ExecutionConfigVersionRepo) GetOrCreateLatestBaseVersion(
	ctx context.Context,
	tenantID, userID uuid.UUID,
	module string,
	sourceConfigID uuid.UUID,
	fingerprint string,
	snapshot interface{},
) (*model.TenantConfigVersion, error) {
	var result model.TenantConfigVersion
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		lockKey := "base:" + tenantID.String() + ":" + module + ":" + sourceConfigID.String()
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", lockKey).Error; err != nil {
			return err
		}

		err := tx.Where(
			"tenant_id = ? AND module = ? AND source_config_id = ?",
			tenantID, module, sourceConfigID,
		).Order("version_no DESC").First(&result).Error
		if err == nil {
			return nil
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}

		raw, err := json.Marshal(snapshot)
		if err != nil {
			return fmt.Errorf("序列化租户基础配置快照失败: %w", err)
		}
		result = model.TenantConfigVersion{
			ID: uuid.New(), TenantID: tenantID, Module: module, SourceConfigID: sourceConfigID,
			VersionNo: 1, Fingerprint: fingerprint, ConfigSnapshot: datatypes.JSON(raw),
			CreatedBy: &userID,
		}
		return tx.Create(&result).Error
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// EnsureBaseVersion 将当前管理员配置固化为不可变基础版本；相同内容直接复用。
func (r *ExecutionConfigVersionRepo) EnsureBaseVersion(
	ctx context.Context,
	tenantID, userID uuid.UUID,
	module string,
	sourceConfigID uuid.UUID,
	fingerprint string,
	snapshot interface{},
) (*model.TenantConfigVersion, error) {
	return r.PublishBaseVersion(ctx, tenantID, userID, module, sourceConfigID, fingerprint, snapshot)
}

// BindSnapshot 将流程绑定到内容寻址的不可变配置版本。
// force=false 时保留既有绑定；force=true 仅用于用户明确选择“按最新配置重新执行”。
func (r *ExecutionConfigVersionRepo) BindSnapshot(
	ctx context.Context,
	tenantID, userID uuid.UUID,
	module, processID, processType string,
	sourceConfigID uuid.UUID,
	baseConfigVersionID uuid.UUID,
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

		// 普通重审必须先复用流程已有绑定，不能因为当前租户配置已变化而创建一个实际未使用的新版本。
		if !force {
			err := tx.Table("execution_config_versions AS v").
				Select("v.*").
				Joins("JOIN process_execution_config_bindings AS b ON b.config_version_id = v.id").
				Where("b.tenant_id = ? AND b.module = ? AND b.process_id = ?", tenantID, module, processID).
				First(&result).Error
			if err == nil {
				return nil
			}
			if err != gorm.ErrRecordNotFound {
				return err
			}
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
				ID: uuid.New(), TenantID: tenantID, Module: module, SourceConfigID: sourceConfigID,
				BaseConfigVersionID: &baseConfigVersionID, VersionNo: maxVersion + 1,
				Fingerprint: fingerprint, ConfigSnapshot: datatypes.JSON(raw), CreatedBy: &userID,
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
