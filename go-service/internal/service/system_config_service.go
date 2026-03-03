package service

import (
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/repository"
)

// SystemConfigService handles system configuration CRUD.
type SystemConfigService struct {
	repo *repository.SystemConfigRepo
}

func NewSystemConfigService(repo *repository.SystemConfigRepo) *SystemConfigService {
	return &SystemConfigService{repo: repo}
}

// ConfigItem is a key-value pair returned to the frontend.
type ConfigItem struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Remark string `json:"remark"`
}

// GetAll returns all system configs as key-value pairs.
func (s *SystemConfigService) GetAll() ([]ConfigItem, error) {
	configs, err := s.repo.ListAll()
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	result := make([]ConfigItem, len(configs))
	for i, c := range configs {
		result[i] = ConfigItem{Key: c.Key, Value: c.Value, Remark: c.Remark}
	}
	return result, nil
}

// UpdateConfigs batch-updates config values by key.
func (s *SystemConfigService) UpdateConfigs(updates map[string]string) error {
	for key, value := range updates {
		if err := s.repo.UpdateByKey(key, value); err != nil {
			return newServiceError(errcode.ErrDatabase, "数据库错误")
		}
	}
	return nil
}
