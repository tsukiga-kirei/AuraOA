package service

import (
	"testing"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"auraoa/go-service/internal/model"
)

func TestAccessControlDataAllows(t *testing.T) {
	memberID := uuid.New()
	departmentID := uuid.New()
	roleID := uuid.New()
	member := &model.OrgMember{
		ID:           memberID,
		DepartmentID: departmentID,
		Roles:        []model.OrgRole{{ID: roleID}},
	}

	tests := []struct {
		name    string
		ac      model.AccessControlData
		member  *model.OrgMember
		allowed bool
	}{
		{name: "空配置默认拒绝", ac: model.AccessControlData{}, member: member, allowed: false},
		{name: "所有人允许租户成员", ac: model.AccessControlData{AllowAll: true}, member: member, allowed: true},
		{name: "所有人仍拒绝非租户成员", ac: model.AccessControlData{AllowAll: true}, member: nil, allowed: false},
		{name: "非租户成员拒绝", ac: model.AccessControlData{AllowedMembers: []string{memberID.String()}}, member: nil, allowed: false},
		{name: "成员命中", ac: model.AccessControlData{AllowedMembers: []string{memberID.String()}}, member: member, allowed: true},
		{name: "部门命中", ac: model.AccessControlData{AllowedDepartments: []string{departmentID.String()}}, member: member, allowed: true},
		{name: "角色命中", ac: model.AccessControlData{AllowedRoles: []string{roleID.String()}}, member: member, allowed: true},
		{name: "均未命中", ac: model.AccessControlData{AllowedRoles: []string{uuid.NewString()}}, member: member, allowed: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := accessControlDataAllows(tt.ac, tt.member); got != tt.allowed {
				t.Fatalf("accessControlDataAllows() = %v, want %v", got, tt.allowed)
			}
		})
	}
}

func TestAccessControlAllowsJSON(t *testing.T) {
	member := &model.OrgMember{ID: uuid.New(), DepartmentID: uuid.New()}
	if !accessControlAllows(datatypes.JSON(`{"allow_all":true}`), member) {
		t.Fatal("allow_all=true 应允许当前租户成员")
	}
	if accessControlAllows(datatypes.JSON(`{"allow_all":false}`), member) {
		t.Fatal("关闭所有人且没有精确授权时应拒绝访问")
	}
	if accessControlAllows(datatypes.JSON(`{invalid`), member) {
		t.Fatal("无效访问控制 JSON 应拒绝访问")
	}
}

func TestAgentAccessControlAllows(t *testing.T) {
	memberID := uuid.New()
	deptID := uuid.New()
	roleID := uuid.New()
	member := &model.OrgMember{
		ID:           memberID,
		DepartmentID: deptID,
		Roles:        []model.OrgRole{{ID: roleID}},
	}

	// 1. 空配置/未设置时默认所有人可用
	if !agentAccessControlAllows(datatypes.JSON(""), member) {
		t.Fatal("空字符串配置默认应允许")
	}
	if !agentAccessControlAllows(datatypes.JSON("{}"), member) {
		t.Fatal("{} 配置默认应允许")
	}
	if !agentAccessControlAllows(datatypes.JSON("null"), member) {
		t.Fatal("null 配置默认应允许")
	}
	if !agentAccessControlAllows(datatypes.JSON("{invalid"), member) {
		t.Fatal("格式错误配置默认应允许")
	}

	// 2. nil 成员始终拒绝
	if agentAccessControlAllows(datatypes.JSON(`{"allow_all":true}`), nil) {
		t.Fatal("nil 成员应拒绝")
	}

	// 3. allow_all: true 允许
	if !agentAccessControlAllows(datatypes.JSON(`{"allow_all":true}`), member) {
		t.Fatal("allow_all: true 应允许租户成员")
	}

	// 4. allow_all: false，精确匹配
	if agentAccessControlAllows(datatypes.JSON(`{"allow_all":false,"allowed_roles":[],"allowed_members":[],"allowed_departments":[]}`), member) {
		t.Fatal("allow_all: false 且无授权时应拒绝")
	}

	// 角色命中
	if !agentAccessControlAllows(datatypes.JSON(`{"allow_all":false,"allowed_roles":["`+roleID.String()+`"]}`), member) {
		t.Fatal("角色命中应允许")
	}

	// 成员命中
	if !agentAccessControlAllows(datatypes.JSON(`{"allow_all":false,"allowed_members":["`+memberID.String()+`"]}`), member) {
		t.Fatal("成员命中应允许")
	}

	// 部门命中
	if !agentAccessControlAllows(datatypes.JSON(`{"allow_all":false,"allowed_departments":["`+deptID.String()+`"]}`), member) {
		t.Fatal("部门命中应允许")
	}

	// 未命中
	if agentAccessControlAllows(datatypes.JSON(`{"allow_all":false,"allowed_members":["`+uuid.NewString()+`"]}`), member) {
		t.Fatal("未命中授权名单应拒绝")
	}
}
