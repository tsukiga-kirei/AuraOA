package service

import (
	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/repository"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var errConfigNotPublished = errors.New("配置尚处于草稿状态，请先发布首个版本")

// loadPublishedConfig 只覆盖发布内容，保留当前租户身份、访问控制及嵌入开关。
func loadPublishedConfig(ctx context.Context, versions *repository.ExecutionConfigVersionRepo, tenantID uuid.UUID, module string, configID uuid.UUID, target interface{}, rules interface{}) error {
	_, err := loadPublishedConfigVersion(ctx, versions, tenantID, module, configID, target, rules)
	return err
}

// loadPublishedConfigVersion 同时返回读取内容对应的发布版本，避免并发切换导致内容与版本号错配。
func loadPublishedConfigVersion(ctx context.Context, versions *repository.ExecutionConfigVersionRepo, tenantID uuid.UUID, module string, configID uuid.UUID, target interface{}, rules interface{}) (*model.TenantConfigVersion, error) {
	if versions == nil {
		return nil, nil
	}
	version, err := versions.GetActiveBaseVersion(ctx, tenantID, module, configID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errConfigNotPublished
	}
	if err != nil {
		return nil, err
	}
	if err := decodePublishedConfig(version, target, rules); err != nil {
		return nil, err
	}
	return version, nil
}

func decodePublishedConfig(version *model.TenantConfigVersion, target interface{}, rules interface{}) error {
	// 在全新对象中解析后仅复制发布字段，缺省字段不能继承未发布草稿的旧值。
	var identity struct {
		TenantID    uuid.UUID
		ConfigID    uuid.UUID
		ProcessType string
	}
	switch config := target.(type) {
	case *model.ProcessAuditConfig:
		var published model.ProcessAuditConfig
		if err := json.Unmarshal(version.ConfigSnapshot, &published); err != nil {
			return err
		}
		if published.ProcessType != "" {
			config.ProcessType = published.ProcessType
		}
		config.MainTableName, config.MainFields, config.DetailTables = published.MainTableName, publishedJSON(published.MainFields, "[]"), publishedJSON(published.DetailTables, "[]")
		config.FieldMode, config.KBMode = published.FieldMode, published.KBMode
		config.AIConfig, config.UserPermissions, config.Status = publishedJSON(published.AIConfig, "{}"), publishedJSON(published.UserPermissions, "{}"), published.Status
		identity.TenantID, identity.ConfigID, identity.ProcessType = config.TenantID, config.ID, config.ProcessType
	case *model.ProcessArchiveConfig:
		var published model.ProcessArchiveConfig
		if err := json.Unmarshal(version.ConfigSnapshot, &published); err != nil {
			return err
		}
		if published.ProcessType != "" {
			config.ProcessType = published.ProcessType
		}
		config.MainTableName, config.MainFields, config.DetailTables = published.MainTableName, publishedJSON(published.MainFields, "[]"), publishedJSON(published.DetailTables, "[]")
		config.FieldMode, config.KBMode = published.FieldMode, published.KBMode
		config.AIConfig, config.UserPermissions, config.Status = publishedJSON(published.AIConfig, "{}"), publishedJSON(published.UserPermissions, "{}"), published.Status
		identity.TenantID, identity.ConfigID, identity.ProcessType = config.TenantID, config.ID, config.ProcessType
	case *model.ProcessSummaryConfig:
		var published model.ProcessSummaryConfig
		if err := json.Unmarshal(version.ConfigSnapshot, &published); err != nil {
			return err
		}
		if published.ProcessType != "" {
			config.ProcessType = published.ProcessType
		}
		config.MainTableName, config.MainFields, config.DetailTables = published.MainTableName, publishedJSON(published.MainFields, "[]"), publishedJSON(published.DetailTables, "[]")
		config.SummaryBlocks, config.Status = publishedJSON(published.SummaryBlocks, "[]"), published.Status
	default:
		return fmt.Errorf("不支持的发布配置类型")
	}
	if rules == nil {
		return nil
	}
	var snapshot struct {
		Rules json.RawMessage `json:"rules"`
	}
	if err := json.Unmarshal(version.ConfigSnapshot, &snapshot); err != nil {
		return err
	}
	raw := snapshot.Rules
	if len(raw) == 0 || string(raw) == "null" {
		raw = json.RawMessage(`[]`)
	}
	switch target := rules.(type) {
	case *[]model.AuditRule:
		var published []model.AuditRule
		if err := json.Unmarshal(raw, &published); err != nil {
			return err
		}
		for i := range published {
			published[i].TenantID, published[i].ConfigID, published[i].ProcessType = identity.TenantID, &identity.ConfigID, identity.ProcessType
			published[i].DataQueries = publishedJSON(published[i].DataQueries, "[]")
			published[i].ContextMounts = publishedJSON(published[i].ContextMounts, "[]")
		}
		*target = published
	case *[]model.ArchiveRule:
		var published []model.ArchiveRule
		if err := json.Unmarshal(raw, &published); err != nil {
			return err
		}
		for i := range published {
			published[i].TenantID, published[i].ConfigID, published[i].ProcessType = identity.TenantID, &identity.ConfigID, identity.ProcessType
			published[i].DataQueries = publishedJSON(published[i].DataQueries, "[]")
			published[i].ContextMounts = publishedJSON(published[i].ContextMounts, "[]")
		}
		*target = published
	default:
		return fmt.Errorf("不支持的发布规则类型")
	}
	return nil
}

// applyPublishedConfig 使用调用方已读取的发布版本；未传入时兼容原有读取入口。
func applyPublishedConfig(ctx context.Context, versions *repository.ExecutionConfigVersionRepo, tenantID uuid.UUID, module string, configID uuid.UUID, target interface{}, rules interface{}, published ...*model.TenantConfigVersion) error {
	if len(published) > 0 && published[0] != nil {
		return decodePublishedConfig(published[0], target, rules)
	}
	return loadPublishedConfig(ctx, versions, tenantID, module, configID, target, rules)
}

// publishedJSON 将旧发布快照的缺省和 null 集合统一为空集合，不继承草稿值。
func publishedJSON(raw datatypes.JSON, fallback string) datatypes.JSON {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return datatypes.JSON(fallback)
	}
	return raw
}
