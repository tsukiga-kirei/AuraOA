package service

import (
	"testing"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"auraoa/go-service/internal/model"
)

func TestAuditConfigSourceFingerprintTracksExecutionChanges(t *testing.T) {
	config := &model.ProcessAuditConfig{
		ProcessType: "采购审批", MainTableName: "formtable_main_1",
		MainFields:   datatypes.JSON(`[{"field_key":"amount"}]`),
		DetailTables: datatypes.JSON(`[]`), FieldMode: "selected", KBMode: "rules_only",
		AIConfig: datatypes.JSON(`{"audit_strictness":"standard"}`), Status: "active",
		UserPermissions: datatypes.JSON(`{"allow_modify_strictness":false}`),
		AccessControl:   datatypes.JSON(`{"allow_all":true}`),
	}
	ruleA := model.AuditRule{ID: uuid.New(), RuleContent: "金额必须大于零", RuleScope: "mandatory", Enabled: boolPointer(true)}
	ruleB := model.AuditRule{ID: uuid.New(), RuleContent: "说明必须填写", RuleScope: "default_on", Enabled: boolPointer(true)}

	base := auditConfigSourceFingerprint(config, []model.AuditRule{ruleA, ruleB})
	reordered := auditConfigSourceFingerprint(config, []model.AuditRule{ruleB, ruleA})
	if base != reordered {
		t.Fatal("规则读取顺序变化不应改变来源配置指纹")
	}

	config.AccessControl = datatypes.JSON(`{"allow_all":false}`)
	if got := auditConfigSourceFingerprint(config, []model.AuditRule{ruleA, ruleB}); got != base {
		t.Fatal("访问权限不参与 AI 执行内容，不应改变来源配置指纹")
	}

	config.UserPermissions = datatypes.JSON(`{"allow_modify_strictness":true}`)
	if got := auditConfigSourceFingerprint(config, []model.AuditRule{ruleA, ruleB}); got == base {
		t.Fatal("个人尺度权限会改变最终执行配置，必须形成新的租户基础版本")
	}
	config.UserPermissions = datatypes.JSON(`{"allow_modify_strictness":false}`)

	config.AIConfig = datatypes.JSON(`{"audit_strictness":"loose"}`)
	if got := auditConfigSourceFingerprint(config, []model.AuditRule{ruleA, ruleB}); got == base {
		t.Fatal("审核尺度变化必须改变来源配置指纹")
	}
}

func TestAuditConfigSourceFingerprintTracksRuleChanges(t *testing.T) {
	config := &model.ProcessAuditConfig{
		ProcessType: "采购审批", MainFields: datatypes.JSON(`[]`), DetailTables: datatypes.JSON(`[]`),
		AIConfig: datatypes.JSON(`{}`), FieldMode: "all", KBMode: "rules_only", Status: "active",
	}
	rule := model.AuditRule{ID: uuid.New(), RuleContent: "原规则", RuleScope: "default_on", Enabled: boolPointer(true)}
	base := auditConfigSourceFingerprint(config, []model.AuditRule{rule})
	rule.RuleContent = "修改后的规则"
	if got := auditConfigSourceFingerprint(config, []model.AuditRule{rule}); got == base {
		t.Fatal("规则内容变化必须改变来源配置指纹")
	}
}

func TestSummaryConfigSourceFingerprintTracksBlockChanges(t *testing.T) {
	config := &model.ProcessSummaryConfig{
		ProcessType: "采购审批", MainFields: datatypes.JSON(`[]`), DetailTables: datatypes.JSON(`[]`),
		SummaryBlocks: datatypes.JSON(`[{"id":"overview","user_prompt":"总结流程","enabled":true}]`),
		Status:        "active",
	}
	base := summaryConfigSourceFingerprint(config)
	config.SummaryBlocks = datatypes.JSON(`[{"id":"overview","user_prompt":"重点总结风险","enabled":true}]`)
	if got := summaryConfigSourceFingerprint(config); got == base {
		t.Fatal("总结提示词变化必须改变来源配置指纹")
	}
}

func TestResolveEffectiveAuditRulesUsesTenantRulesAsBase(t *testing.T) {
	mandatoryID := uuid.New()
	optionalID := uuid.New()
	rules := []model.AuditRule{
		{ID: mandatoryID, RuleScope: "mandatory", Enabled: boolPointer(false)},
		{ID: optionalID, RuleScope: "default_on", Enabled: boolPointer(true)},
	}
	detail := &model.AuditDetailItem{RuleConfig: model.RuleConfig{RuleToggleOverrides: []model.RuleToggleOverride{
		{RuleID: mandatoryID.String(), Enabled: false},
		{RuleID: optionalID.String(), Enabled: false},
		{RuleID: uuid.NewString(), Enabled: true},
	}}}

	effective := resolveEffectiveAuditRules(rules, detail)
	if len(effective) != len(rules) {
		t.Fatalf("有效规则必须保持租户规则边界，got=%d want=%d", len(effective), len(rules))
	}
	if !isRuleEnabled(&effective[0]) {
		t.Fatal("mandatory 租户规则不能被个人配置关闭")
	}
	if isRuleEnabled(&effective[1]) {
		t.Fatal("非 mandatory 租户规则应允许个人开关覆盖")
	}
	if !isRuleEnabled(&rules[1]) {
		t.Fatal("合并个人配置不能修改租户规则原对象")
	}
}

func TestResolveEffectiveArchiveRulesUsesTenantRulesAsBase(t *testing.T) {
	mandatoryID := uuid.New()
	optionalID := uuid.New()
	rules := []model.ArchiveRule{
		{ID: mandatoryID, RuleScope: "mandatory", Enabled: boolPointer(false)},
		{ID: optionalID, RuleScope: "default_off", Enabled: boolPointer(false)},
	}
	detail := &model.ArchiveDetailItem{RuleConfig: model.RuleConfig{RuleToggleOverrides: []model.RuleToggleOverride{
		{RuleID: mandatoryID.String(), Enabled: false},
		{RuleID: optionalID.String(), Enabled: true},
	}}}

	effective := resolveEffectiveArchiveRules(rules, detail)
	if len(effective) != len(rules) {
		t.Fatalf("有效规则必须保持租户规则边界，got=%d want=%d", len(effective), len(rules))
	}
	if effective[0].Enabled == nil || !*effective[0].Enabled {
		t.Fatal("mandatory 租户归档规则不能被个人配置关闭")
	}
	if effective[1].Enabled == nil || !*effective[1].Enabled {
		t.Fatal("非 mandatory 租户归档规则应允许个人开关覆盖")
	}
	if rules[1].Enabled == nil || *rules[1].Enabled {
		t.Fatal("合并个人配置不能修改租户归档规则原对象")
	}
}
