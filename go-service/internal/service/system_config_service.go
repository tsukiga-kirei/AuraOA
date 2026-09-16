package service

import (
	"strings"

	"go.uber.org/zap"

	"auraoa/go-service/internal/pkg/crypto"
	"auraoa/go-service/internal/pkg/errcode"
	pkglogger "auraoa/go-service/internal/pkg/logger"
	"auraoa/go-service/internal/repository"
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
// 敏感凭证（如阿里云 AccessKey Secret）不回显，并生成 companion configured 标志；AccessKey ID 自动解密回显。
func (s *SystemConfigService) GetAll() ([]ConfigItem, error) {
	configs, err := s.repo.ListAll()
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	result := make([]ConfigItem, 0, len(configs)+1)
	for _, c := range configs {
		item := ConfigItem{Key: c.Key, Value: c.Value, Remark: c.Remark}
		switch c.Key {
		case "attachment.aliyun_ocr_access_key_secret":
			configured := "false"
			if strings.TrimSpace(c.Value) != "" {
				configured = "true"
			}
			item.Value = "" // 不向前端回显 Secret 明文或密文
			result = append(result, item)
			result = append(result, ConfigItem{
				Key:    "attachment.aliyun_ocr_access_key_secret_configured",
				Value:  configured,
				Remark: "阿里云 AccessKey Secret 是否已配置",
			})
			continue
		case "attachment.aliyun_ocr_access_key_id":
			if strings.TrimSpace(c.Value) != "" {
				if decrypted, decErr := crypto.Decrypt(c.Value); decErr == nil {
					item.Value = decrypted
				}
			}
		}
		result = append(result, item)
	}
	return result, nil
}

// UpdateConfigs 批量更新系统配置值，按 key 逐条更新。
// 对敏感凭证使用 AES 加密存储；Secret 留空时保持原值不覆盖。
func (s *SystemConfigService) UpdateConfigs(updates map[string]string) error {
	for key, value := range updates {
		// 忽略虚拟状态标记键
		if key == "attachment.aliyun_ocr_access_key_secret_configured" {
			continue
		}

		// 阿里云 AccessKey Secret：若前端传空（留空保持），则保留现有数据库值不修改
		if key == "attachment.aliyun_ocr_access_key_secret" {
			trimmed := strings.TrimSpace(value)
			if trimmed == "" {
				continue
			}
			encrypted, encErr := crypto.Encrypt(trimmed)
			if encErr != nil {
				return newServiceError(errcode.ErrInternalServer, "加密 AccessKey Secret 失败")
			}
			value = encrypted
		}

		// 阿里云 AccessKey ID：若有值则使用 AES 加密存储
		if key == "attachment.aliyun_ocr_access_key_id" {
			trimmed := strings.TrimSpace(value)
			if trimmed != "" {
				encrypted, encErr := crypto.Encrypt(trimmed)
				if encErr != nil {
					return newServiceError(errcode.ErrInternalServer, "加密 AccessKey ID 失败")
				}
				value = encrypted
			} else {
				value = ""
			}
		}

		if err := s.repo.UpdateByKey(key, value); err != nil {
			return newServiceError(errcode.ErrDatabase, "数据库错误")
		}
	}
	keyList := make([]string, 0, len(updates))
	for k := range updates {
		keyList = append(keyList, k)
	}
	pkglogger.Global().Info("系统配置更新成功", zap.Strings("keys", keyList))
	return nil
}
