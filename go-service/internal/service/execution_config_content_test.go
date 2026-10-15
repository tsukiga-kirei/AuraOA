package service

import (
	"encoding/json"
	"testing"
	"time"

	"auraoa/go-service/internal/model"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

func TestExecutionConfigVersionChangedUsesPublishedNumber(t *testing.T) {
	for _, tc := range []struct {
		name            string
		used, available int
		want            bool
	}{
		{"同版本内容指纹变化不提示", 2, 2, false},
		{"不同版本即使内容相同也提示", 2, 3, true},
		{"当前发布版本回退也提示", 3, 2, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := &model.ExecutionConfigVersion{VersionNo: 99, BaseVersionNo: &tc.used, Fingerprint: "different-content"}
			if got := executionConfigVersionChanged(v, &model.TenantConfigVersion{VersionNo: tc.available}); got != tc.want {
				t.Fatalf("版本比较结果=%v，预期=%v", got, tc.want)
			}
		})
	}
	legacy := &model.ExecutionConfigVersion{VersionNo: 99, ConfigSnapshot: datatypes.JSON(`{"base_config_version_no":2}`)}
	if executionConfigVersionChanged(legacy, &model.TenantConfigVersion{VersionNo: 2}) {
		t.Fatal("旧快照应读取基础版本号，不使用执行快照内部编号")
	}
	if executionConfigVersionChanged(nil, &model.TenantConfigVersion{VersionNo: 2}) {
		t.Fatal("未执行流程不应提示升级")
	}
}

func TestSourceContentIgnoresJSONRepresentationAndMetadata(t *testing.T) {
	id := uuid.New()
	for _, module := range []string{"audit", "archive", "summary"} {
		t.Run(module, func(t *testing.T) {
			legacy := datatypes.JSON(`{"process_type":"100","main_table_name":"form1","main_fields":null,"detail_tables":[],"field_mode":"all","kb_mode":"rules_only","ai_config":{"user_reasoning_prompt":"检查","audit_strictness":"standard"},"user_permissions":{},"status":"active","rules":[{"id":"` + id.String() + `","rule_content":"检查金额","rule_scope":"default_on"}],"summary_blocks":[{"id":"b1","enabled":true,"user_prompt":"总结"}]}`)
			var values map[string]interface{}
			if err := json.Unmarshal(legacy, &values); err != nil {
				t.Fatal(err)
			}
			values["main_fields"] = []interface{}{}
			values["ai_config"] = map[string]interface{}{"audit_strictness": "standard", "user_reasoning_prompt": "检查", "enable_thinking": false}
			rules := values["rules"].([]interface{})
			rule := rules[0].(map[string]interface{})
			rule["data_queries"] = []interface{}{}
			rule["context_mounts"] = []interface{}{}
			rule["applicability_condition"] = ""
			rule["enabled"] = true
			rule["updated_at"] = "2026-10-15T01:00:00Z"
			blocks := values["summary_blocks"].([]interface{})
			block := blocks[0].(map[string]interface{})
			block["selected_fields"] = []interface{}{}
			block["enabled_data_variables"] = []interface{}{}
			block["context_mounts"] = []interface{}{}
			raw, _ := json.Marshal(values)
			fingerprint := sourceFingerprintForTest(t, module, raw)
			svc := &ExecutionConfigSourceService{}
			if !svc.matchesActiveSnapshot(module, legacy, fingerprint) {
				t.Fatal("补默认值、调整JSON键顺序及更新时间不应产生草稿变更")
			}
			if module == "summary" {
				block["user_prompt"] = "修改后的总结要求"
			} else {
				rule["data_queries"] = []interface{}{map[string]interface{}{"query_id": id.String(), "version_no": 2}}
			}
			changed, _ := json.Marshal(values)
			if svc.matchesActiveSnapshot(module, legacy, sourceFingerprintForTest(t, module, changed)) {
				t.Fatal("真实配置变化必须提示草稿已修改")
			}
		})
	}
}

func sourceFingerprintForTest(t *testing.T, module string, raw []byte) string {
	t.Helper()
	var value interface{}
	switch module {
	case "audit":
		var cfg model.ProcessAuditConfig
		var rules struct {
			Rules []model.AuditRule `json:"rules"`
		}
		if err := json.Unmarshal(raw, &cfg); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, &rules); err != nil {
			t.Fatal(err)
		}
		value = auditConfigSourceSnapshot(&cfg, rules.Rules)
	case "archive":
		var cfg model.ProcessArchiveConfig
		var rules struct {
			Rules []model.ArchiveRule `json:"rules"`
		}
		if err := json.Unmarshal(raw, &cfg); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, &rules); err != nil {
			t.Fatal(err)
		}
		value = archiveConfigSourceSnapshot(&cfg, rules.Rules)
	case "summary":
		var cfg model.ProcessSummaryConfig
		if err := json.Unmarshal(raw, &cfg); err != nil {
			t.Fatal(err)
		}
		value = summaryConfigSourceSnapshot(&cfg)
	}
	return sourceContentFingerprint(value)
}

func TestPublishedRulesDoNotInheritDraftFields(t *testing.T) {
	tenantID, configID, ruleID := uuid.New(), uuid.New(), uuid.New()
	raw := datatypes.JSON(`{"tenant_id":"` + uuid.NewString() + `","id":"` + uuid.NewString() + `","main_fields":[],"detail_tables":[],"rules":[{"id":"` + ruleID.String() + `","rule_content":"已发布规则","enabled":true}]}`)
	published := &model.TenantConfigVersion{ConfigSnapshot: raw}
	now := time.Now()
	for _, module := range []string{"audit", "archive"} {
		t.Run(module, func(t *testing.T) {
			if module == "audit" {
				config := &model.ProcessAuditConfig{TenantID: tenantID, ID: configID, AccessControl: datatypes.JSON(`{"allow_all":true}`), EmbedEnabled: true}
				rules := []model.AuditRule{{ID: ruleID, ApplicabilityCondition: "未发布前提", DataQueries: datatypes.JSON(`[{"query_id":"draft"}]`), UpdatedAt: now}}
				if err := decodePublishedConfig(published, config, &rules); err != nil {
					t.Fatal(err)
				}
				if config.TenantID != tenantID || config.ID != configID || !config.EmbedEnabled || string(config.AccessControl) != `{"allow_all":true}` {
					t.Fatal("发布内容不能修改租户身份或运行权限")
				}
				if rules[0].ApplicabilityCondition != "" || !rules[0].UpdatedAt.IsZero() || string(rules[0].DataQueries) != "[]" {
					t.Fatal("发布规则继承了草稿前提、查询或更新时间")
				}
				if rules[0].TenantID != tenantID || *rules[0].ConfigID != configID {
					t.Fatal("规则身份必须来自当前租户配置")
				}
			} else {
				config := &model.ProcessArchiveConfig{TenantID: tenantID, ID: configID}
				rules := []model.ArchiveRule{{ID: ruleID, ApplicabilityCondition: "未发布前提", ContextMounts: datatypes.JSON(`[{}]`), UpdatedAt: now}}
				if err := decodePublishedConfig(published, config, &rules); err != nil {
					t.Fatal(err)
				}
				if rules[0].ApplicabilityCondition != "" || !rules[0].UpdatedAt.IsZero() || string(rules[0].ContextMounts) != "[]" {
					t.Fatal("归档发布规则继承了草稿字段")
				}
			}
		})
	}
}

func TestContentNormalizationPreservesFixedQueryNumber(t *testing.T) {
	raw := datatypes.JSON(`[{"bindings":{"id":{"fixed_value":9007199254740993}}}]`)
	normalized := normalizeFieldArrayJSON(raw)
	encoded, err := json.Marshal(normalized)
	if err != nil || string(encoded) != string(raw) {
		t.Fatalf("固定查询参数归一化丢失精度: %s, %v", encoded, err)
	}
}
