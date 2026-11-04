package service

import "auraoa/go-service/internal/model"

// 技能“覆盖内置”语义（copy-on-write）：
//
// 平台内置技能（tenant_id IS NULL）对所有租户只读。租户要编辑某个内置技能时，
// 后端写入一条 tenant_id = 当前租户、skill_code 相同的租户技能作为覆盖版本；
// 此后列表、运行时解析、权限判断都以租户版本为准，删除租户版本即回到内置版本。
// 智能体绑定键是 skill:<skill_code>，因此覆盖不会影响既有装配关系。

// resolveSkillOverrides 按标识码解析出最终生效的技能定义：同名时租户覆盖版本优先；
// 覆盖版本即使被停用也保留（表示该租户主动关闭了此技能，不再回退到内置版本）。
func resolveSkillOverrides(skills []model.AgentSkill) map[string]model.AgentSkill {
	resolved := make(map[string]model.AgentSkill, len(skills))
	for _, skill := range skills {
		if existing, ok := resolved[skill.SkillCode]; ok && existing.TenantID != nil && skill.TenantID == nil {
			continue
		}
		resolved[skill.SkillCode] = skill
	}
	return resolved
}

// mergeSkillsByCode 把内置与租户技能合并成一份列表（同名只保留生效版本），
// 并保持标识码首次出现的位置，避免前端出现两条相同标识码。
func mergeSkillsByCode(skills []model.AgentSkill) []model.AgentSkill {
	resolved := resolveSkillOverrides(skills)
	merged := make([]model.AgentSkill, 0, len(resolved))
	seen := make(map[string]bool, len(resolved))
	for _, skill := range skills {
		if seen[skill.SkillCode] {
			continue
		}
		seen[skill.SkillCode] = true
		merged = append(merged, resolved[skill.SkillCode])
	}
	return merged
}

// hasSystemSkillFallback 判断同名平台内置技能是否存在，用于删除租户覆盖后是否保留智能体绑定。
func hasSystemSkillFallback(skills []model.AgentSkill, skillCode string) bool {
	for _, skill := range skills {
		if skill.TenantID == nil && skill.SkillCode == skillCode {
			return true
		}
	}
	return false
}
