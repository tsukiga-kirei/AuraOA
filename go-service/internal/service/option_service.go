package service

import (
	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/repository"
)

// OptionService 提供选项数据查询服务。
type OptionService struct {
	repo *repository.OptionRepo
}

func NewOptionService(repo *repository.OptionRepo) *OptionService {
	return &OptionService{repo: repo}
}

// ListOATypes 返回OA系统类型选项列表。
func (s *OptionService) ListOATypes() ([]dto.OptionItem, error) {
	items, err := s.repo.ListOATypes()
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	result := make([]dto.OptionItem, len(items))
	for i, item := range items {
		result[i] = dto.OptionItem{Code: item.Code, Label: item.Label}
	}
	return result, nil
}

// ListDBDrivers 返回数据库驱动选项列表。
func (s *OptionService) ListDBDrivers() ([]dto.DBDriverOptionItem, error) {
	items, err := s.repo.ListDBDrivers()
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	result := make([]dto.DBDriverOptionItem, len(items))
	for i, item := range items {
		result[i] = dto.DBDriverOptionItem{Code: item.Code, Label: item.Label, DefaultPort: item.DefaultPort}
	}
	return result, nil
}

// ListAIDeployTypes 返回AI部署类型选项列表。
func (s *OptionService) ListAIDeployTypes() ([]dto.OptionItem, error) {
	items, err := s.repo.ListAIDeployTypes()
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	result := make([]dto.OptionItem, len(items))
	for i, item := range items {
		result[i] = dto.OptionItem{Code: item.Code, Label: item.Label}
	}
	return result, nil
}

// ListAIProviders 返回AI服务商选项列表。
func (s *OptionService) ListAIProviders() ([]dto.AIProviderOptionItem, error) {
	items, err := s.repo.ListAIProviders()
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	result := make([]dto.AIProviderOptionItem, len(items))
	for i, item := range items {
		result[i] = dto.AIProviderOptionItem{Code: item.Code, Label: item.Label, DeployType: item.DeployType}
	}
	return result, nil
}
