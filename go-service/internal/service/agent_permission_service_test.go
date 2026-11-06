package service

import (
	"context"
	"fmt"
	"testing"

	"auraoa/go-service/internal/repository"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestAgentPermissionsUnifiedAccess 验证旧角色白名单不影响使用范围，页面、租户及智能体权限边界仍然生效。
func TestAgentPermissionsUnifiedAccess(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, stmt := range []string{
		`CREATE TABLE tenants (id TEXT PRIMARY KEY, chat_enabled BOOLEAN)`,
		`CREATE TABLE tenant_chat_allocations (id TEXT PRIMARY KEY, tenant_id TEXT)`,
		`CREATE TABLE org_members (id TEXT PRIMARY KEY, tenant_id TEXT, user_id TEXT, department_id TEXT)`,
		`CREATE TABLE departments (id TEXT PRIMARY KEY)`,
		`CREATE TABLE org_roles (id TEXT PRIMARY KEY, page_permissions TEXT)`,
		`CREATE TABLE org_member_roles (org_member_id TEXT, org_role_id TEXT)`,
		`CREATE TABLE org_role_agent_grants (role_id TEXT, agent_code TEXT)`,
		`CREATE TABLE agent_definitions (id TEXT PRIMARY KEY, tenant_id TEXT, agent_code TEXT, enabled BOOLEAN, access_control TEXT, is_system BOOLEAN, created_at DATETIME)`,
		`CREATE TABLE agent_tool_bindings (id TEXT PRIMARY KEY, agent_id TEXT, tool_code TEXT)`,
	} {
		must(db.Exec(stmt).Error)
	}
	tenantID, userID, memberID, roleID, deptID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	must(db.Exec(`INSERT INTO tenants VALUES (?, true)`, tenantID).Error)
	must(db.Exec(`INSERT INTO tenant_chat_allocations (tenant_id) VALUES (?)`, tenantID).Error)
	must(db.Exec(`INSERT INTO org_members VALUES (?, ?, ?, ?)`, memberID, tenantID, userID, deptID).Error)
	must(db.Exec(`INSERT INTO departments VALUES (?)`, deptID).Error)
	must(db.Exec(`INSERT INTO org_roles VALUES (?, '["/chat"]')`, roleID).Error)
	must(db.Exec(`INSERT INTO org_member_roles VALUES (?, ?)`, memberID, roleID).Error)
	must(db.Exec(`INSERT INTO org_role_agent_grants VALUES (?, 'oa_assist')`, roleID).Error)
	agentIDs := map[string]uuid.UUID{}
	for _, a := range []struct {
		code, access string
		enabled      bool
		tenant       uuid.UUID
	}{
		{"oa_assist", `{"allow_all":true}`, true, tenantID},
		{"oa_creator", `{"allow_all":true}`, true, tenantID},
		{"oa_analyst", fmt.Sprintf(`{"allowed_roles":["%s"]}`, roleID), true, tenantID},
		{"member_only", fmt.Sprintf(`{"allowed_members":["%s"]}`, memberID), true, tenantID},
		{"dept_only", fmt.Sprintf(`{"allowed_departments":["%s"]}`, deptID), true, tenantID},
		{"oa_query", `{"allow_all":true}`, false, tenantID},
		{"denied", fmt.Sprintf(`{"allowed_roles":["%s"]}`, uuid.New()), true, tenantID},
		{"other_tenant", `{"allow_all":true}`, true, uuid.New()},
	} {
		id := uuid.New()
		agentIDs[a.code] = id
		must(db.Exec(`INSERT INTO agent_definitions (id,tenant_id,agent_code,enabled,access_control) VALUES (?, ?, ?, ?, ?)`, id, a.tenant, a.code, a.enabled, a.access).Error)
		must(db.Exec(`INSERT INTO agent_tool_bindings VALUES (?, ?, ?)`, uuid.New(), id, "tool_"+a.code).Error)
	}
	repo := repository.NewAgentRepo(db)
	svc := NewAgentPermissionService(repo, repository.NewTenantRepo(db), repository.NewOrgRepo(db))
	check := func(want int) {
		t.Helper()
		perms, err := svc.CalculateEffectivePermissions(context.Background(), tenantID, userID)
		must(err)
		if len(perms.Agents) != want {
			t.Fatalf("有效智能体数量=%d，期望=%d", len(perms.Agents), want)
		}
		for _, a := range perms.Agents {
			if a.AgentCode == "oa_query" || a.AgentCode == "denied" || a.AgentCode == "other_tenant" {
				t.Fatalf("权限边界失效: %s", a.AgentCode)
			}
		}
		if perms.Tools["tool_oa_query"] || perms.Tools["tool_denied"] || perms.Tools["tool_other_tenant"] {
			t.Fatal("未授权工具泄漏")
		}
	}
	check(5)
	creator, err := repo.GetAgentByID(agentIDs["oa_creator"])
	must(err)
	tools, err := svc.CalculateEffectiveToolsForAgent(context.Background(), tenantID, userID, creator)
	must(err)
	if !tools["tool_oa_creator"] || len(tools) != 1 {
		t.Fatalf("新智能体工具授权异常: %v", tools)
	}
	denied, err := repo.GetAgentByID(agentIDs["denied"])
	must(err)
	if _, err = svc.CalculateEffectiveToolsForAgent(context.Background(), tenantID, userID, denied); err == nil {
		t.Fatal("未授权智能体允许执行")
	}
	must(db.Exec(`DELETE FROM org_role_agent_grants`).Error)
	check(5)
	must(db.Exec(`UPDATE org_roles SET page_permissions='[]'`).Error)
	check(0)
	must(db.Exec(`UPDATE org_roles SET page_permissions='["/chat"]'`).Error)
	must(db.Exec(`UPDATE tenants SET chat_enabled=false`).Error)
	check(0)
	must(db.Exec(`UPDATE tenants SET chat_enabled=true`).Error)
	must(db.Exec(`DELETE FROM org_members`).Error)
	check(0)
}
