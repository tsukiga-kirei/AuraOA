package service

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/repository"
)

// StrictnessPresetService 处理审核尺度预设的业务逻辑。
type StrictnessPresetService struct {
	presetRepo *repository.StrictnessPresetRepo
}

// NewStrictnessPresetService 创建一个新的 StrictnessPresetService 实例。
func NewStrictnessPresetService(presetRepo *repository.StrictnessPresetRepo) *StrictnessPresetService {
	return &StrictnessPresetService{presetRepo: presetRepo}
}

// ListByTenant 查询当前租户的审核尺度预设，首次访问时自动初始化三条默认记录。
func (s *StrictnessPresetService) ListByTenant(c *gin.Context) ([]model.StrictnessPreset, error) {
	// 检查是否需要自动初始化
	count, err := s.presetRepo.CountByTenant(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	if count == 0 {
		// 自动初始化 strict/standard/loose 三条预设
		if err := s.initDefaultPresets(c); err != nil {
			return nil, err
		}
	}

	presets, err := s.presetRepo.ListByTenant(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	return presets, nil
}

// UpdateByStrictness 更新指定尺度的预设内容。
func (s *StrictnessPresetService) UpdateByStrictness(c *gin.Context, strictness string, req *dto.UpdateStrictnessPresetRequest) (*model.StrictnessPreset, error) {
	// 校验 strictness 值
	if strictness != "strict" && strictness != "standard" && strictness != "loose" {
		return nil, newServiceError(errcode.ErrParamValidation, "无效的审核尺度值")
	}

	fields := map[string]interface{}{
		"reasoning_instruction":  req.ReasoningInstruction,
		"extraction_instruction": req.ExtractionInstruction,
	}

	if err := s.presetRepo.UpdateByStrictness(c, strictness, fields); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	preset, err := s.presetRepo.GetByStrictness(c, strictness)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	return preset, nil
}

// initDefaultPresets 为租户初始化 strict/standard/loose 三条默认预设。
func (s *StrictnessPresetService) initDefaultPresets(c *gin.Context) error {
	tenantID, err := getTenantUUID(c)
	if err != nil {
		return newServiceError(errcode.ErrParamValidation, "租户ID无效")
	}

	defaults := []model.StrictnessPreset{
		{ID: uuid.New(), TenantID: tenantID, Strictness: "loose"},
		{ID: uuid.New(), TenantID: tenantID, Strictness: "standard"},
		{ID: uuid.New(), TenantID: tenantID, Strictness: "strict"},
	}

	for _, preset := range defaults {
		if err := s.presetRepo.Create(c, &preset); err != nil {
			return newServiceError(errcode.ErrDatabase, "初始化审核尺度预设失败")
		}
	}
	return nil
}
