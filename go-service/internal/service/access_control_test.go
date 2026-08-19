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
