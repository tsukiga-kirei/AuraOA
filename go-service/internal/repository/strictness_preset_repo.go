package repository

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

// StrictnessPresetRepo 提供审核尺度预设的数据访问方法，按租户隔离。
type StrictnessPresetRepo struct {
	*BaseRepo
}

// NewStrictnessPresetRepo 创建一个新的 StrictnessPresetRepo 实例。
func NewStrictnessPresetRepo(db *gorm.DB) *StrictnessPresetRepo {
	return &StrictnessPresetRepo{BaseRepo: NewBaseRepo(db)}
}

// ListByTenant 查询当前租户的所有审核尺度预设（应为 strict/standard/loose 三条）。
func (r *StrictnessPresetRepo) ListByTenant(c *gin.Context) ([]model.StrictnessPreset, error) {
	var presets []model.StrictnessPreset
	if err := r.WithTenant(c).Order("strictness ASC").Find(&presets).Error; err != nil {
		return nil, err
	}
	return presets, nil
}

// GetByStrictness 查询当前租户指定尺度的预设。
func (r *StrictnessPresetRepo) GetByStrictness(c *gin.Context, strictness string) (*model.StrictnessPreset, error) {
	var preset model.StrictnessPreset
	if err := r.WithTenant(c).Where("strictness = ?", strictness).First(&preset).Error; err != nil {
		return nil, err
	}
	return &preset, nil
}

// UpdateByStrictness 更新当前租户指定尺度的预设内容。
func (r *StrictnessPresetRepo) UpdateByStrictness(c *gin.Context, strictness string, fields map[string]interface{}) error {
	return r.WithTenant(c).Model(&model.StrictnessPreset{}).Where("strictness = ?", strictness).Updates(fields).Error
}

// Create 创建审核尺度预设记录。
func (r *StrictnessPresetRepo) Create(c *gin.Context, preset *model.StrictnessPreset) error {
	return r.WithTenant(c).Create(preset).Error
}

// CountByTenant 统计当前租户的预设数量。
func (r *StrictnessPresetRepo) CountByTenant(c *gin.Context) (int64, error) {
	var count int64
	if err := r.WithTenant(c).Model(&model.StrictnessPreset{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
