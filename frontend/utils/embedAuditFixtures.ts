import { ref } from 'vue'
import type { AuditResult } from '~/types/audit'
import type { EmbedContextResponse } from '~/types/embed'
import type { AuditComment, Feedback } from '~/types/experience'
import type { EmbedAuditPreview } from '~/utils/embedAuditPreview'

export const embedAuditPreviewScenarios = ['return', 'approve', 'review', 'long', 'commentsEmpty', 'commentsPaged', 'readonly', 'stale', 'contentChanged', 'bothChanged', 'empty', 'running', 'failed', 'parseError', 'unsupported'] as const
export type EmbedAuditPreviewScenario = typeof embedAuditPreviewScenarios[number]

const reasoning = `## 审核结论
建议补充材料后重新提交。当前申请的主要问题集中在**项目启动时间**与**立项报告完整性**，其他基础信息及预算勾稽关系符合要求。

### 1. 项目启动时间与审批顺序
申请于 2026 年 10 月 16 日提交，计划启动日期为 10 月 17 日，仅预留 1 个自然日。按照项目管理要求，应在审批完成后再启动，并建议预留至少 7 天准备时间。

> 时间依据来自本次申请表的“提交日期”和“计划启动日期”。这里的风险提示用于辅助审批，应由审批人结合实际情况确认。

### 2. 立项报告的必备章节
已提交《协同平台升级项目立项报告（草案）》和《预算测算明细》。报告包含项目背景、目标和实施范围，但尚未完整说明以下内容：

- **组织分工**：补充项目负责人、业务接口人与验收责任人。
- **风险及应对措施**：说明系统切换、历史数据迁移和服务连续性安排。
- **投资估算依据**：列明供应商报价、工作量测算口径与预备费依据。

### 3. 预算核对
| 核对项目 | 申请金额 | 明细合计 | 结果 |
| --- | ---: | ---: | --- |
| 软件与实施 | 180,000 元 | 180,000 元 | 一致 |
| 数据迁移 | 45,000 元 | 45,000 元 | 一致 |
| 运维与培训 | 25,000 元 | 25,000 元 | 一致 |
| 合计 | 250,000 元 | 250,000 元 | 一致 |

### 4. 审批建议
建议先补齐报告缺失章节，调整启动日期，再提交项目负责人复核。预算本身无需重复修改，但应补充测算依据，便于后续采购与验收追溯。

### 5. 资料范围与结论边界
本次分析依据申请表、预算测算明细及立项报告草案。未获得供应商正式报价，也未核实线下审批结果；如已有补充材料，请由审批人结合实际情况判断。`

const thinking = `### 核对思路
先按规则逐项对照申请表与附件，再合并重复问题。将缺失信息与明确不符合规则的情况分开处理，不把“尚未提供”推断为“已经违反”。

1. 对照申请人、部门、项目名称与申请金额，确认主表和附件对应同一项目。
2. 按申请日期、计划启动日期和审批节点核对先后顺序。
3. 从附件中定位实施组织、风险措施、投资估算和验收计划。
4. 对预算明细进行加总，检查总额是否一致。
5. 汇总可追溯的问题，形成需要人工确认的事项。

### 日期与材料判断
申请日期是 10 月 16 日，计划启动是 10 月 17 日。审批仍处于部门负责人节点，因此先标记审批前启动的可能性，由审批人确认。

报告已经说明了建设背景和目标，不应把整个附件判断为缺失。未找到的组织分工与风险应对章节单独列为待补充项。

### 综合评价
两项规则未通过，四项规则通过。评分反映本次材料的完整性与规则符合情况，不能替代审批人的业务判断。`

const baseResult: AuditResult = {
  id: 'preview-standard', trace_id: 'preview-trace', process_id: 'MOCK-20261017-001', status: 'completed',
  recommendation: 'return', overall_score: 40, confidence: 95, duration_ms: 57200,
  rule_results: [
    { rule_content: '项目启动日期应晚于申请日期，并预留至少 7 天准备时间', passed: false, reason: '计划启动日期为 2026-10-17，仅晚于申请日期 1 天，且当前审批尚未完成，存在流程倒置风险。' },
    { rule_content: '立项报告应覆盖实施组织、风险应对措施与投资估算依据', passed: false, reason: '报告草案已包含项目背景和建设目标，但缺少组织分工、风险应对和投资估算依据，建议补充后再提交。' },
    { rule_content: '申请信息完整，主表与附件的项目名称一致', passed: true, reason: '申请人、部门、项目名称和建设范围均已填写，主表与附件保持一致。' },
    { rule_content: '预算明细合计应与申请总金额一致', passed: true, reason: '软件与实施 18 万元、数据迁移 4.5 万元、运维与培训 2.5 万元，合计 25 万元，与申请总额一致。' },
    { rule_content: '应明确建设目标与预期成果', passed: true, reason: '已列明流程整合、统一身份认证和移动审批优化三个建设目标。' },
    { rule_content: '应提供项目实施及验收计划', passed: true, reason: '已附阶段计划及验收清单，覆盖试运行、培训与最终验收。' },
  ],
  risk_points: ['审批尚未完成，计划启动时间过近，可能导致先实施后审批。', '组织分工与风险措施缺失，影响项目执行和后续验收追溯。'],
  suggestions: ['补齐立项报告中的组织分工、风险措施和估算依据。', '将启动日期调整至审批完成后，并预留合理的准备时间。'],
  ai_reasoning: reasoning, deep_thinking: thinking,
  created_at: '2026-10-17T10:28:00+08:00', config_version_no: 3,
}

const copy = <T>(value: T): T => JSON.parse(JSON.stringify(value))

/** 所有数据与写入均留在当前预览实例的内存中，不读取 OA、不调用 AI 或业务接口。 */
export function createEmbedAuditPreview(scenario: EmbedAuditPreviewScenario, labels: { failure: string; parseError: string; unsupported: string; stages: string[] }) {
  const result = copy(baseResult)
  if (scenario === 'approve') {
    result.recommendation = 'approve'; result.overall_score = 96; result.confidence = 98
    result.rule_results.forEach(rule => { rule.passed = true; rule.reason = '申请表与补充材料内容一致，已满足该项审核要求。' })
    result.risk_points = []; result.suggestions = ['建议按已批准的阶段计划推进，并保留验收记录。']
    result.ai_reasoning = '## 审核结论\n材料已补充完整，六项规则均通过，建议同意。\n\n### 补充材料核对\n计划启动日期已调整至 2026 年 10 月 26 日；组织职责、风险措施、投资估算依据与验收计划齐全。\n\n### 预算核对\n预算合计 250,000 元，与申请总额一致。'
    result.deep_thinking = '### 核对过程\n逐项核对日期、预算与立项报告。补充材料已覆盖此前缺失的章节，未发现需要退回的问题。'
  }
  if (scenario === 'review') {
    result.recommendation = 'review'; result.overall_score = 76; result.confidence = 88
    result.rule_results[0] = { rule_content: result.rule_results[0]!.rule_content, passed: true, reason: '启动日期已调整，满足准备时间要求。' }
    result.rule_results[1] = { rule_content: result.rule_results[1]!.rule_content, passed: false, reason: '实施组织与风险措施已补充，但投资估算缺少正式报价，需项目负责人进一步确认。' }
    result.risk_points = ['投资估算依据仍需项目负责人进一步确认。']
    result.suggestions = ['请人工复核报价与工作量口径，并附上确认记录。']
    result.ai_reasoning = '## 审核结论\n建议人工复核投资估算依据。\n\n### 需要确认的事项\n计划启动日期已调整，基础信息、预算勾稽和实施计划符合要求。立项报告提供了初步估算，但未包含正式报价，建议项目负责人补充确认记录后审批。'
    result.deep_thinking = '### 核对过程\n将预算合计正确与估算依据充分分开判断。金额一致不代表报价已核实，因此保留人工复核建议。'
  }
  if (scenario === 'long') {
    result.ai_reasoning += '\n\n' + Array.from({ length: 10 }, (_, i) => `### 补充分析 ${i + 1}：实施阶段核对\n从业务连续性、迁移校验、权限继承与人员培训四个方面检查项目计划。应明确负责人、完成日期和可核对的交付物，避免只描述原则而缺少执行安排。\n\n- 确认切换窗口与回退方案。\n- 保留历史数据核对记录。\n- 在验收前完成业务人员培训。`).join('\n\n')
    result.deep_thinking += '\n\n' + thinking
  }
  if (scenario === 'failed') { result.status = 'failed'; result.error_message = labels.failure }
  if (scenario === 'parseError') result.parse_error = labels.parseError
  if (scenario === 'running') result.status = 'reasoning'

  const context = ref<EmbedContextResponse>({
    supported: scenario !== 'unsupported', message: labels.unsupported, has_audit: scenario !== 'empty',
    process: {
      process_id: result.process_id, title: scenario === 'long' ? '跨区域业务协同与统一身份认证平台建设项目立项申请流程－一期基础能力升级及历史数据迁移专项' : '协同平台升级项目立项申请－周予安－2026-10-16',
      applicant: scenario === 'long' ? '周予安（项目联合负责人）' : '周予安',
      department: scenario === 'long' ? '企业数字化与业务协同建设中心 · 平台产品及技术管理部' : '企业数字化部',
      process_type: 'project', process_type_label: '项目管理', current_node: scenario === 'long' ? '部门负责人及分管领导联合审批（技术与预算复核）' : '部门负责人审批', submit_time: '2026-10-16T09:15:00+08:00',
    },
    audit_result: scenario === 'empty' ? null : result, last_audit_at: scenario === 'empty' ? undefined : '2026-10-17T10:28:00+08:00',
    config_version_no: 3, stale: scenario === 'contentChanged' || scenario === 'bothChanged', config_upgrade_available: scenario === 'stale' || scenario === 'bothChanged', allow_latest_config_rerun: scenario === 'stale' || scenario === 'bothChanged',
    personal_view: { available: true, has_audit: true, last_audit_at: '2026-10-17T10:29:00+08:00', audit_result: { ...copy(baseResult), id: 'preview-personal', overall_score: 68, recommendation: 'review', confidence: 91, ai_reasoning: '## 我的定制审核\n结合项目执行视角，重点关注预算估算、交付责任与实施风险。\n\n' + reasoning, result_source: 'personal' } },
  })

  const comments = new Map<string, AuditComment[]>()
  let nextId = 100
  function itemsFor(id: string) {
    if (!comments.has(id)) {
      const total = scenario === 'commentsEmpty' ? 0 : scenario === 'commentsPaged' ? 23 : 2
      comments.set(id, Array.from({ length: total }, (_, index) => ({
        id: `mock-comment-${index}`, audit_log_id: id, oa_user_id: index % 2 === 0 ? 'mock-owner' : 'mock-colleague',
        username: index % 2 === 0 ? '周予安' : '林知远',
        content: index === 0 ? '时间顺序的提醒很有帮助。我会先补全组织分工与风险措施，再调整启动日期重新提交。' : index === 1 ? '预算合计没有问题，但投资估算的依据还需要补充，建议把供应商报价一并附上。' : `第 ${index + 1} 条虚拟评论：已阅读本次审核意见，后续会结合补充材料进行复核。`,
        feedback: index % 3 === 0 ? 'like' : index % 3 === 1 ? null : 'dislike',
        created_at: '2026-10-17T10:35:00+08:00', updated_at: '2026-10-17T10:35:00+08:00',
        can_manage: scenario !== 'readonly' && index % 2 === 0,
      })))
    }
    return comments.get(id)!
  }
  const pause = (ms: number) => new Promise<void>(resolve => setTimeout(resolve, ms))
  let disposed = false
  const preview: EmbedAuditPreview = {
    context,
    async execute(progress, personal, latest) {
      const completed = copy(personal ? context.value.personal_view!.audit_result! : context.value.audit_result || baseResult)
      completed.status = 'completed'; delete completed.error_message; delete completed.parse_error
      completed.id = `preview-${personal ? 'personal' : 'standard'}-${++nextId}`
      if (latest) { context.value.config_version_no = 4; completed.config_version_no = 4; context.value.config_upgrade_available = false; context.value.stale = false }
      const stages = ['assembling', 'reasoning', 'extracting'] as const
      for (let i = 0; i < stages.length; i++) {
        if (disposed) return completed
        progress({ status: stages[i], progress_steps: stages.map((key, n) => ({ key, label: labels.stages[n] || key, done: n < i, current: n === i })) })
        await pause(650)
        if (i === 1) {
          for (let chunk = 1; chunk <= 8; chunk++) {
            if (disposed) return completed
            progress({ ai_reasoning: completed.ai_reasoning.slice(0, Math.ceil(completed.ai_reasoning.length * chunk / 8)), deep_thinking: (completed.deep_thinking || '').slice(0, Math.ceil((completed.deep_thinking || '').length * chunk / 8)) })
            await pause(230)
          }
        }
      }
      if (personal) context.value.personal_view!.audit_result = completed
      else { context.value.audit_result = completed; context.value.has_audit = true }
      return completed
    },
    feedbackApi: {
      async interactions(id, page = 1) {
        const all = itemsFor(id)
        return { items: copy(all.slice((page - 1) * 20, page * 20)), total: all.length, page, page_size: 20, can_interact: scenario !== 'readonly', like_count: all.filter(c => c.feedback === 'like').length, dislike_count: all.filter(c => c.feedback === 'dislike').length }
      },
      async comment(id, content, feedback) {
        await pause(320)
        const item: AuditComment = { id: `mock-new-${++nextId}`, audit_log_id: id, oa_user_id: 'mock-owner', username: '周予安', content, feedback, created_at: '2026-10-17T11:00:00+08:00', updated_at: '2026-10-17T11:00:00+08:00', can_manage: true }
        itemsFor(id).push(item)
        return copy(item)
      },
      async updateComment(id, commentId, content, feedback: Feedback | null) {
        await pause(320)
        const item = itemsFor(id).find(c => c.id === commentId && c.can_manage)
        if (item) { item.content = content; item.feedback = feedback }
        return {}
      },
      async deleteComment(id, commentId) {
        await pause(320)
        comments.set(id, itemsFor(id).filter(c => c.id !== commentId || !c.can_manage))
        return {}
      },
    },
  }
  return { preview, dispose: () => { disposed = true } }
}
