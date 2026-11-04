package service

import (
	"testing"

	"github.com/google/uuid"

	"auraoa/go-service/internal/model"
)

func systemSkill(code, name string, enabled bool) model.AgentSkill {
	return model.AgentSkill{ID: uuid.New(), SkillCode: code, Name: name, Enabled: enabled}
}

func tenantSkill(tenantID uuid.UUID, code, name string, enabled bool) model.AgentSkill {
	return model.AgentSkill{ID: uuid.New(), TenantID: &tenantID, SkillCode: code, Name: name, Enabled: enabled}
}

func TestResolveSkillOverridesPrefersTenantCopy(t *testing.T) {
	tenantID := uuid.New()
	skills := []model.AgentSkill{
		systemSkill("oa_workflow_stats", "内置查数指南", true),
		tenantSkill(tenantID, "oa_workflow_stats", "租户改过的查数指南", true),
		systemSkill("create_supplies_request", "内置填单指南", true),
	}

	resolved := resolveSkillOverrides(skills)

	if got := resolved["oa_workflow_stats"].Name; got != "租户改过的查数指南" {
		t.Fatalf("expected tenant override to win, got %q", got)
	}
	if resolved["oa_workflow_stats"].TenantID == nil {
		t.Fatal("expected override to keep tenant ownership")
	}
	if resolved["create_supplies_request"].TenantID != nil {
		t.Fatal("expected untouched builtin to stay system-owned")
	}
}

func TestResolveSkillOverridesKeepsDisabledOverride(t *testing.T) {
	tenantID := uuid.New()
	skills := []model.AgentSkill{
		systemSkill("oa_workflow_stats", "内置查数指南", true),
		tenantSkill(tenantID, "oa_workflow_stats", "租户停用的查数指南", false),
	}

	resolved := resolveSkillOverrides(skills)
	if resolved["oa_workflow_stats"].Enabled {
		t.Fatal("expected disabled override to shadow the builtin instead of falling back")
	}
}

func TestResolveSkillOverridesIgnoresLaterSystemCopy(t *testing.T) {
	tenantID := uuid.New()
	// 平台种子晚于租户覆盖写入的场景：系统版本也不能反向覆盖租户版本
	skills := []model.AgentSkill{
		tenantSkill(tenantID, "oa_workflow_stats", "租户版本", true),
		systemSkill("oa_workflow_stats", "后写入的内置版本", true),
	}

	resolved := resolveSkillOverrides(skills)
	if got := resolved["oa_workflow_stats"].Name; got != "租户版本" {
		t.Fatalf("expected tenant override to stay, got %q", got)
	}
}

func TestMergeSkillsByCodeDedupesAndKeepsFirstPosition(t *testing.T) {
	tenantID := uuid.New()
	skills := []model.AgentSkill{
		systemSkill("oa_workflow_stats", "内置", true),
		systemSkill("create_supplies_request", "内置填单", true),
		tenantSkill(tenantID, "oa_workflow_stats", "覆盖", true),
	}

	merged := mergeSkillsByCode(skills)

	if len(merged) != 2 {
		t.Fatalf("expected 2 merged skills, got %d", len(merged))
	}
	if merged[0].SkillCode != "oa_workflow_stats" || merged[0].Name != "覆盖" {
		t.Fatalf("expected override to occupy original position, got %+v", merged[0])
	}
	if merged[1].SkillCode != "create_supplies_request" {
		t.Fatalf("unexpected order: %+v", merged)
	}
}

func TestHasSystemSkillFallback(t *testing.T) {
	tenantID := uuid.New()
	skills := []model.AgentSkill{
		systemSkill("oa_workflow_stats", "内置", true),
		tenantSkill(tenantID, "oa_workflow_stats", "覆盖", true),
		tenantSkill(tenantID, "custom_only", "自定义", true),
	}

	if !hasSystemSkillFallback(skills, "oa_workflow_stats") {
		t.Fatal("expected builtin fallback for overridden code")
	}
	if hasSystemSkillFallback(skills, "custom_only") {
		t.Fatal("expected no builtin fallback for tenant-only code")
	}
}
