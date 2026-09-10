package repository

import (
	"testing"

	"auraoa/go-service/internal/model"
)

func TestRolesHavePagePermission(t *testing.T) {
	roles := []model.OrgRole{
		{PagePermissions: []byte(`["/overview","/summary"]`)},
		{PagePermissions: []byte(`invalid`)},
	}
	if !rolesHavePagePermission(roles, "/summary") {
		t.Fatal("拥有 /summary 的组织角色应允许访问流程总结接口")
	}
	if rolesHavePagePermission(roles, "/archive") {
		t.Fatal("未分配的页面权限不应被放行")
	}
}
