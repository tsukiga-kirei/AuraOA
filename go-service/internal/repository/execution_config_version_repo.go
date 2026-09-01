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
	"auraoa/go-service/internal/pkg/apptime"
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

// GetActiveBaseVersion 获取当前租户生效/启用的基础配置版本。
func (r *ExecutionConfigVersionRepo) GetActiveBaseVersion(
	ctx context.Context,
	tenantID uuid.UUID,
	module string,
	sourceConfigID uuid.UUID,
) (*model.TenantConfigVersion, error) {
	var version model.TenantConfigVersion
	// 优先查询 is_active = true 的版本
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND module = ? AND source_config_id = ? AND is_active = ?", tenantID, module, sourceConfigID, true).
		First(&version).Error
	if err == nil {
		return &version, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// 若未显式标记 active（老数据兼容），查询最新版本
	return r.GetLatestBaseVersion(ctx, tenantID, module, sourceConfigID)
}

// SetActiveBaseVersion 切换指定版本为当前可用版本（Active Version）。
func (r *ExecutionConfigVersionRepo) SetActiveBaseVersion(
	ctx context.Context,
	tenantID uuid.UUID,
	module string,
	sourceConfigID uuid.UUID,
	versionNo int,
) (*model.TenantConfigVersion, error) {
	var target model.TenantConfigVersion
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		lockKey := "base:" + tenantID.String() + ":" + module + ":" + sourceConfigID.String()
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", lockKey).Error; err != nil {
			return err
		}

		if err := tx.Where(
			"tenant_id = ? AND module = ? AND source_config_id = ? AND version_no = ?",
			tenantID, module, sourceConfigID, versionNo,
		).First(&target).Error; err != nil {
			return err
		}

		// 将其他版本 active 置为 false
		if err := tx.Model(&model.TenantConfigVersion{}).
			Where("tenant_id = ? AND module = ? AND source_config_id = ?", tenantID, module, sourceConfigID).
			Update("is_active", false).Error; err != nil {
			return err
		}

		// 激活当前版本
		target.IsActive = true
		target.UpdatedAt = apptime.Now()
		return tx.Model(&model.TenantConfigVersion{}).
			Where("id = ?", target.ID).
			Updates(map[string]interface{}{
				"is_active":  true,
				"updated_at": target.UpdatedAt,
			}).Error
	})
	if err != nil {
		return nil, err
	}
	return &target, nil
}

// UpdateBaseVersionSnapshot 更新指定版本的快照内容与指纹。
func (r *ExecutionConfigVersionRepo) UpdateBaseVersionSnapshot(
	ctx context.Context,
	tenantID uuid.UUID,
	module string,
	sourceConfigID uuid.UUID,
	versionNo int,
	fingerprint string,
	snapshot interface{},
) (*model.TenantConfigVersion, error) {
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return nil, fmt.Errorf("序列化租户基础配置快照失败: %w", err)
	}

	var target model.TenantConfigVersion
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		lockKey := "base:" + tenantID.String() + ":" + module + ":" + sourceConfigID.String()
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", lockKey).Error; err != nil {
			return err
		}

		if err := tx.Where(
			"tenant_id = ? AND module = ? AND source_config_id = ? AND version_no = ?",
			tenantID, module, sourceConfigID, versionNo,
		).First(&target).Error; err != nil {
			return err
		}

		target.Fingerprint = fingerprint
		target.ConfigSnapshot = datatypes.JSON(raw)
		target.UpdatedAt = apptime.Now()

		return tx.Model(&model.TenantConfigVersion{}).
			Where("id = ?", target.ID).
			Updates(map[string]interface{}{
				"fingerprint":     target.Fingerprint,
				"config_snapshot": target.ConfigSnapshot,
				"updated_at":      target.UpdatedAt,
			}).Error
	})
	if err != nil {
		return nil, err
	}
	return &target, nil
}

// GetLatestBaseVersion 获取租户最新发布的基础配置版本。
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

// ListBaseVersions 获取租户某项配置发布的所有历史基础版本列表。
func (r *ExecutionConfigVersionRepo) ListBaseVersions(
	ctx context.Context,
	tenantID uuid.UUID,
	module string,
	sourceConfigID uuid.UUID,
) ([]model.TenantConfigVersion, error) {
	var list []model.TenantConfigVersion
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND module = ? AND source_config_id = ?", tenantID, module, sourceConfigID).
		Order("version_no DESC").
		Find(&list).Error
	return list, err
}

// GetBaseVersionByNo 按版本号获取特定的租户基础版本。
func (r *ExecutionConfigVersionRepo) GetBaseVersionByNo(
	ctx context.Context,
	tenantID uuid.UUID,
	module string,
	sourceConfigID uuid.UUID,
	versionNo int,
) (*model.TenantConfigVersion, error) {
	var version model.TenantConfigVersion
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND module = ? AND source_config_id = ? AND version_no = ?", tenantID, module, sourceConfigID, versionNo).
		First(&version).Error
	if err != nil {
		return nil, err
	}
	return &version, nil
}

// PublishBaseVersion 显式发布新基础版本，并自动设为当前启用版本。
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
			// 指纹一致，直接激活最新版本
			_ = tx.Model(&model.TenantConfigVersion{}).
				Where("tenant_id = ? AND module = ? AND source_config_id = ?", tenantID, module, sourceConfigID).
				Update("is_active", false)
			_ = tx.Model(&model.TenantConfigVersion{}).Where("id = ?", latest.ID).Update("is_active", true)
			latest.IsActive = true
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

		// 将先前版本置为非 active
		_ = tx.Model(&model.TenantConfigVersion{}).
			Where("tenant_id = ? AND module = ? AND source_config_id = ?", tenantID, module, sourceConfigID).
			Update("is_active", false)

		now := apptime.Now()
		result = model.TenantConfigVersion{
			ID: uuid.New(), TenantID: tenantID, Module: module, SourceConfigID: sourceConfigID,
			VersionNo: maxVersion + 1, Fingerprint: fingerprint, ConfigSnapshot: datatypes.JSON(raw),
			IsActive:  true,
			CreatedBy: &userID,
			CreatedAt: now,
			UpdatedAt: now,
		}
		return tx.Create(&result).Error
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetOrCreateLatestBaseVersion 获取启用或最新基础版本；若尚无任何版本（首次使用），则创建初始版本 V1。
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

		// 优先取 is_active = true
		err := tx.Where(
			"tenant_id = ? AND module = ? AND source_config_id = ? AND is_active = ?",
			tenantID, module, sourceConfigID, true,
		).First(&result).Error
		if err == nil {
			return nil
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}

		// 若无 active 记录，取最新版
		err = tx.Where(
			"tenant_id = ? AND module = ? AND source_config_id = ?",
			tenantID, module, sourceConfigID,
		).Order("version_no DESC").First(&result).Error
		if err == nil {
			_ = tx.Model(&model.TenantConfigVersion{}).Where("id = ?", result.ID).Update("is_active", true)
			result.IsActive = true
			return nil
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}

		raw, err := json.Marshal(snapshot)
		if err != nil {
			return fmt.Errorf("序列化租户基础配置快照失败: %w", err)
		}
		now := apptime.Now()
		result = model.TenantConfigVersion{
			ID: uuid.New(), TenantID: tenantID, Module: module, SourceConfigID: sourceConfigID,
			VersionNo: 1, Fingerprint: fingerprint, ConfigSnapshot: datatypes.JSON(raw),
			IsActive:  true,
			CreatedBy: &userID,
			CreatedAt: now,
			UpdatedAt: now,
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
