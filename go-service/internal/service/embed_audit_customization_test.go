package service

import (
	"auraoa/go-service/internal/model"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"testing"
)

func TestEmbedEffectiveAuditCustomization(t *testing.T) {
	on, off, mandatory := uuid.New(), uuid.New(), uuid.New()
	rules := []model.AuditRule{
		{ID: on, RuleContent: "默认开启", RuleScope: "default_on", Enabled: boolPointer(true)},
		{ID: off, RuleContent: "默认关闭", RuleScope: "default_off", Enabled: boolPointer(false)},
		{ID: mandatory, RuleContent: "强制规则", RuleScope: "mandatory", Enabled: boolPointer(true)},
	}
	cases := []struct {
		name       string
		detail     model.AuditDetailItem
		restricted bool
		want       bool
	}{
		{name: "没有个人配置"},
		{name: "只有历史版本号", detail: model.AuditDetailItem{PersonalVersion: 8}},
		{name: "开关均与管理员一致", detail: model.AuditDetailItem{RuleConfig: model.RuleConfig{RuleToggleOverrides: []model.RuleToggleOverride{{RuleID: on.String(), Enabled: true}, {RuleID: off.String(), Enabled: false}}}}},
		{name: "关闭的自定义规则", detail: model.AuditDetailItem{RuleConfig: model.RuleConfig{CustomRules: []model.CustomRule{{Content: "个人规则", Enabled: false}}}}},
		{name: "失效规则和强制规则覆盖", detail: model.AuditDetailItem{RuleConfig: model.RuleConfig{RuleToggleOverrides: []model.RuleToggleOverride{{RuleID: uuid.New().String(), Enabled: true}, {RuleID: mandatory.String(), Enabled: false}}}}},
		{name: "已选及失效字段", detail: model.AuditDetailItem{FieldConfig: model.FieldConfig{FieldOverrides: []string{"main:amount", "main:deleted"}}}},
		{name: "相同审核尺度", detail: model.AuditDetailItem{AIConfig: model.UserAIConfig{StrictnessOverride: "standard"}}},
		{name: "不同规则开关", detail: model.AuditDetailItem{RuleConfig: model.RuleConfig{RuleToggleOverrides: []model.RuleToggleOverride{{RuleID: on.String(), Enabled: false}}}}, want: true},
		{name: "启用自定义规则", detail: model.AuditDetailItem{RuleConfig: model.RuleConfig{CustomRules: []model.CustomRule{{Content: "个人规则", Enabled: true}}}}, want: true},
		{name: "有效新增字段", detail: model.AuditDetailItem{FieldConfig: model.FieldConfig{FieldOverrides: []string{"main:note"}}}, want: true},
		{name: "管理员禁用的个人权限", restricted: true, detail: model.AuditDetailItem{FieldConfig: model.FieldConfig{FieldOverrides: []string{"main:note"}}, RuleConfig: model.RuleConfig{CustomRules: []model.CustomRule{{Content: "个人规则", Enabled: true}}}, AIConfig: model.UserAIConfig{StrictnessOverride: "strict"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config := &model.ProcessAuditConfig{FieldMode: "selected", MainFields: datatypes.JSON(`[{"field_key":"amount","selected":true},{"field_key":"note","selected":false}]`), DetailTables: datatypes.JSON(`[]`), AIConfig: datatypes.JSON(`{"audit_strictness":"standard"}`), UserPermissions: datatypes.JSON(`{"allow_custom_fields":true,"allow_custom_rules":true,"allow_modify_strictness":true}`)}
			if tc.restricted {
				config.UserPermissions = datatypes.JSON(`{}`)
			}
			s := &AuditExecuteService{}
			got, err := s.hasEffectiveAuditCustomization(config, rules, &tc.detail)
			if err != nil || got != tc.want {
				t.Fatalf("个人入口=%v, 期望=%v, err=%v", got, tc.want, err)
			}
		})
	}
}
