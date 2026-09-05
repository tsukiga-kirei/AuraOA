package service

import (
	"auraoa/go-service/internal/model"
	"encoding/json"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"testing"
)

func TestPublishedSnapshotOverridesDraftAndPreservesTenant(t *testing.T) {
	tenantID, configID, ruleID := uuid.New(), uuid.New(), uuid.New()
	config := &model.ProcessAuditConfig{ID: configID, TenantID: tenantID, MainTableName: "draft", AIConfig: datatypes.JSON(`{"strictness":"loose"}`)}
	raw := datatypes.JSON(`{"main_table_name":"published","ai_config":{"strictness":"strict"},"rules":[{"id":"` + ruleID.String() + `","enabled":false,"related_flow":false,"rule_content":"published rule"}]}`)
	var rules []model.AuditRule
	if err := decodePublishedConfig(&model.TenantConfigVersion{ConfigSnapshot: raw}, config, &rules); err != nil {
		t.Fatal(err)
	}
	if config.TenantID != tenantID || config.ID != configID || config.MainTableName != "published" || len(rules) != 1 || *rules[0].Enabled {
		t.Fatal("快照解析未保留租户身份或未覆盖草稿", config, rules)
	}
}

func TestSourceSnapshotValidationRejectsOldFlattenedPayload(t *testing.T) {
	if err := validateSourceSnapshot("audit", []byte(`{"main_fields":[],"ai_strictness":"strict"}`)); err == nil {
		t.Fatal("不能接受缺失嵌套配置的旧载荷")
	}
	snapshot := map[string]interface{}{"process_type": "1", "main_table_name": "form1", "main_fields": []string{}, "detail_tables": []string{}, "status": "active", "field_mode": "selected", "kb_mode": "none", "ai_config": map[string]interface{}{}, "user_permissions": map[string]interface{}{}, "rules": []string{}}
	raw, _ := json.Marshal(snapshot)
	if err := validateSourceSnapshot("audit", raw); err != nil {
		t.Fatal(err)
	}
	snapshot["ai_config"] = nil
	raw, _ = json.Marshal(snapshot)
	if err := validateSourceSnapshot("audit", raw); err == nil {
		t.Fatal("JSON null 不能写入必填配置")
	}
}
