package service

import (
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/repository"
)

// SystemConfigService 处理系统配置的查询与批量更新业务逻辑。
type SystemConfigService struct {
	repo *repository.SystemConfigRepo
}

// NewSystemConfigService 创建 SystemConfigService，注入系统配置仓储。
func NewSystemConfigService(repo *repository.SystemConfigRepo) *SystemConfigService {
	return &SystemConfigService{repo: repo}
}

// ConfigItem 返回给前端的键值对配置项。
type ConfigItem struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Remark string `json:"remark"`
}

// GetAll 返回所有系统配置项（键值对格式）。
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

// UpdateConfigs 批量更新系统配置值，按 key 逐条更新。
func (s *SystemConfigService) UpdateConfigs(updates map[string]string) error {
	for key, value := range updates {
		if err := s.repo.UpdateByKey(key, value); err != nil {
			return newServiceError(errcode.ErrDatabase, "数据库错误")
		}
	}
	return nil
}
