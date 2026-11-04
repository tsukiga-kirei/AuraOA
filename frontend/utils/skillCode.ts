import type { AgentSkillItem, SaveSkillRequest } from '~/types/chat'

/**
 * 技能（Skill）管理端的纯逻辑：复制标识码生成 + 编辑草稿构建。
 * 抽成纯函数便于单测；页面只负责把结果塞进抽屉。
 */

export type SkillEditMode = 'create' | 'edit' | 'copy' | 'override'

export interface SkillEditDraft {
  mode: SkillEditMode
  form: SaveSkillRequest
}

/** 生成一个在本租户内不冲突的技能标识码：base_copy、base_copy2 ... */
export function nextSkillCopyCode(baseCode: string, existingCodes: string[] = []): string {
  const used = new Set(existingCodes.filter(Boolean))
  // 复制“副本”时回到原标识码，避免出现 xxx_copy_copy
  const base = (baseCode || 'skill').trim().replace(/_copy\d*$/i, '') || 'skill'
  const prefix = base.length > 52 ? base.slice(0, 52) : base

  let candidate = `${prefix}_copy`
  let index = 2
  while (used.has(candidate)) {
    candidate = `${prefix}_copy${index}`
    index += 1
  }
  return candidate
}

/**
 * 编辑既有技能的草稿：
 * - 平台内置技能对租户只读，编辑动作转成“同标识码的租户覆盖版本”（无 id，保存时走创建）
 * - 租户技能走常规更新（带 id，标识码锁定）
 */
export function buildSkillEditDraft(item: AgentSkillItem): SkillEditDraft {
  if (item.is_system) {
    return {
      mode: 'override',
      form: {
        skill_code: item.skill_code,
        name: item.name,
        description: item.description,
        content: item.content,
        enabled: item.enabled,
        agent_codes: [...(item.agent_codes || [])],
      },
    }
  }
  return {
    mode: 'edit',
    form: {
      ...item,
      agent_codes: [...(item.agent_codes || [])],
    },
  }
}

/** 复制任意技能为新技能：新标识码、无 id，保存后原件与副本并存 */
export function buildSkillCopyDraft(item: AgentSkillItem, existingCodes: string[] = []): SkillEditDraft {
  return {
    mode: 'copy',
    form: {
      skill_code: nextSkillCopyCode(item.skill_code, existingCodes),
      name: `${item.name} 副本`,
      description: item.description,
      content: item.content,
      enabled: item.enabled,
      agent_codes: [...(item.agent_codes || [])],
    },
  }
}
