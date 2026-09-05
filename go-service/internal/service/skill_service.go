package service

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/repository"
)

// SkillService 负责智能体 Skills 指令包的解析与提示词注入
type SkillService struct {
	agentRepo *repository.AgentRepo
}

// NewSkillService 创建 SkillService
func NewSkillService(agentRepo *repository.AgentRepo) *SkillService {
	return &SkillService{agentRepo: agentRepo}
}

// SkillInfo Skill 描述信息
type SkillInfo struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Content     string `json:"content"`
}

// ResolveAgentSkillsOverview 获取智能体绑定的所有已启用 Skill 概览
func (s *SkillService) ResolveAgentSkillsOverview(ctx context.Context, tenantID uuid.UUID, skillCodes []string) ([]SkillInfo, error) {
	if len(skillCodes) == 0 {
		return nil, nil
	}
	allSkills, err := s.agentRepo.ListSkills(tenantID)
	if err != nil {
		return nil, err
	}

	skillMap := make(map[string]model.AgentSkill)
	for _, sk := range allSkills {
		if sk.Enabled {
			skillMap[sk.SkillCode] = sk
		}
	}

	var res []SkillInfo
	for _, code := range skillCodes {
		if sk, ok := skillMap[code]; ok {
			desc := sk.Description
			if desc == "" {
				desc = extractDescriptionFromContent(sk.Content)
			}
			res = append(res, SkillInfo{
				Code:        sk.SkillCode,
				Name:        sk.Name,
				Description: desc,
				Content:     sk.Content,
			})
		}
	}
	return res, nil
}

// BuildSkillsPromptSection 将 Skill 概览构建为系统提示词中的一段指南
func (s *SkillService) BuildSkillsPromptSection(skills []SkillInfo) string {
	if len(skills) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n\n### 专项技能与参考指南 (Skills)，使用前必须调用对应的技能工具读取完整指南:\n")
	for _, sk := range skills {
		sb.WriteString("- 【" + sk.Name + " (" + sk.Code + ")】: " + sk.Description + "\n")

	}
	return sb.String()
}

func extractDescriptionFromContent(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "description:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "description:"))
		}
	}
	return ""
}
