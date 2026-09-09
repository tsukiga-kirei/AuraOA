package service

import (
	"encoding/json"

	"gorm.io/datatypes"

	"auraoa/go-service/internal/model"
)

// accessControlAllows 按角色、成员、部门任一命中判断访问权限。
// 未配置或配置损坏时默认拒绝，避免权限遗漏或解析异常导致数据意外公开。
func accessControlAllows(raw datatypes.JSON, member *model.OrgMember) bool {
	if member == nil {
		return false
	}
	var ac model.AccessControlData
	if err := json.Unmarshal(raw, &ac); err != nil {
		return false
	}
	return accessControlDataAllows(ac, member)
}

// agentAccessControlAllows 用于智能体访问控制判断。
// 智能体未配置、为空或解析异常时默认全员可用（allow_all: true）。
func agentAccessControlAllows(raw datatypes.JSON, member *model.OrgMember) bool {
	if member == nil {
		return false
	}
	if len(raw) == 0 || string(raw) == "{}" || string(raw) == "null" {
		return true
	}
	var ac model.AccessControlData
	if err := json.Unmarshal(raw, &ac); err != nil {
		return true
	}
	if ac.AllowAll {
		return true
	}
	return accessControlDataAllows(ac, member)
}

func accessControlDataAllows(ac model.AccessControlData, member *model.OrgMember) bool {
	if member == nil {
		return false
	}
	if ac.AllowAll {
		return true
	}
	if len(ac.AllowedRoles) == 0 && len(ac.AllowedMembers) == 0 && len(ac.AllowedDepartments) == 0 {
		return false
	}
	if sliceContains(ac.AllowedMembers, member.ID.String()) {
		return true
	}
	if sliceContains(ac.AllowedDepartments, member.DepartmentID.String()) {
		return true
	}
	for _, role := range member.Roles {
		if sliceContains(ac.AllowedRoles, role.ID.String()) {
			return true
		}
	}
	return false
}
