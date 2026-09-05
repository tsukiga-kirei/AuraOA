package service

import (
	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/repository"
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// loadPublishedConfig 只覆盖快照字段，保留当前租户隔离、访问控制及嵌入开关。
func loadPublishedConfig(ctx context.Context, versions *repository.ExecutionConfigVersionRepo, tenantID uuid.UUID, module string, configID uuid.UUID, target interface{}, rules interface{}) error {
	if versions == nil {
		return nil
	}
	version, err := versions.GetActiveBaseVersion(ctx, tenantID, module, configID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return decodePublishedConfig(version, target, rules)
}

func decodePublishedConfig(version *model.TenantConfigVersion, target interface{}, rules interface{}) error {
	if err := json.Unmarshal(version.ConfigSnapshot, target); err != nil {
		return err
	}
	if rules != nil {
		var snapshot struct {
			Rules json.RawMessage `json:"rules"`
		}
		if err := json.Unmarshal(version.ConfigSnapshot, &snapshot); err != nil {
			return err
		}
		if len(snapshot.Rules) > 0 {
			return json.Unmarshal(snapshot.Rules, rules)
		}
	}
	return nil
}
