// utils/configExportImportHelper.ts — 流程配置导出导入辅助工具与逐项校验引擎

import type {
  ConfigExportBundle,
  ConfigExportContents,
  ConfigExportModule,
  ExportedAIConfig,
  ExportedRuleItem,
  ImportValidationReport,
  RuleConflictStrategy,
} from '~/types/config-export-import'

/** 浏览器端触发下载 JSON 文件 */
export function downloadJsonFile(data: any, fileName: string) {
  const jsonStr = JSON.stringify(data, null, 2)
  const blob = new Blob([jsonStr], { type: 'application/json;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = fileName.endsWith('.json') ? fileName : `${fileName}.json`
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

/** 格式化日期用于文件名，例如 20260916_1700 */
export function formatExportTimestamp(date = new Date()): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  const y = date.getFullYear()
  const m = pad(date.getMonth() + 1)
  const d = pad(date.getDate())
  const hh = pad(date.getHours())
  const mm = pad(date.getMinutes())
  return `${y}${m}${d}_${hh}${mm}`
}

/** 组装标准配置导出包 */
export function buildExportBundle(params: {
  module: ConfigExportModule
  processType: string
  processTypeLabel?: string
  mainTableName?: string
  contents: ConfigExportContents
}): ConfigExportBundle {
  return {
    schema_version: '1.0',
    app: 'AuraOA',
    module: params.module,
    exported_at: new Date().toISOString(),
    source_meta: {
      process_type: params.processType,
      process_type_label: params.processTypeLabel || '',
      main_table_name: params.mainTableName || '',
    },
    contents: params.contents,
  }
}

/** 系统通用的时间占位符白名单 */
export const KNOWN_SYSTEM_TIME_VARIABLES = new Set([
  '{{current_date}}',
  '{{current_time}}',
  '{{current_datetime}}',
  '{{weekday}}',
])

/** 审核/归档 AI 提示词支持的系统预设变量白名单 */
export const KNOWN_AUDIT_AI_VARIABLES = new Set([
  '{{process_type}}',
  '{{main_table}}',
  '{{fields}}',
  '{{detail_tables}}',
  '{{attachments}}',
  '{{rules}}',
  '{{current_node}}',
  '{{flow_history}}',
  '{{flow_graph}}',
  '{{external_context}}',
  '{{reasoning_result}}',
  ...KNOWN_SYSTEM_TIME_VARIABLES,
])

/** 保持历史命名兼容 */
export const KNOWN_SYSTEM_VARIABLES = KNOWN_AUDIT_AI_VARIABLES

/** 流程总结特有的数据占位符白名单 */
export const KNOWN_SUMMARY_DATA_VARIABLES = new Set([
  '{{process_meta}}',
  '{{main_table}}',
  '{{detail_tables}}',
  '{{attachments}}',
  '{{flow_history}}',
  '{{flow_graph}}',
  ...KNOWN_SYSTEM_TIME_VARIABLES,
])

/** 提取文本中所有的占位符变量形如 {{xxx}} */
export function extractPlaceholderVariables(text?: string): string[] {
  if (!text) return []
  const matches = text.match(/\{\{[^{}]+\}\}/g)
  return matches ? Array.from(new Set(matches)) : []
}

/** 校验导入 JSON 内容的合规性与排重（Dry-Run） */
export function validateImportBundle(
  rawContent: string,
  targetModule: ConfigExportModule,
  existingRules: Array<{ rule_content: string }> = [],
  existingSummaryBlocks: Array<{ title: string }> = []
): ImportValidationReport {
  const report: ImportValidationReport = {
    valid: false,
    detected_module: 'unknown',
    target_module: targetModule,
    module_mismatch: false,
    errors: [],
    warnings: [],
    has_rules: false,
    has_ai: false,
    has_summary_blocks: false,
  }

  let parsed: any
  try {
    parsed = JSON.parse(rawContent)
  } catch (err: any) {
    report.errors.push(`JSON 解析失败: ${err.message || '文件格式不是合法的 JSON'}`)
    return report
  }

  // 1. 结构探测：是标准 Bundle 还是裸数据？
  let detectedModule: ConfigExportModule | 'unknown' = 'unknown'
  let contents: ConfigExportContents = {}

  if (parsed && typeof parsed === 'object' && !Array.isArray(parsed) && parsed.contents && typeof parsed.contents === 'object') {
    // 标准 Bundle 格式
    if (parsed.module === 'audit' || parsed.module === 'archive' || parsed.module === 'summary') {
      detectedModule = parsed.module
    }
    contents = parsed.contents
  } else if (Array.isArray(parsed)) {
    // 裸规则数组或裸总结块数组
    if (parsed.length > 0 && typeof parsed[0] === 'object') {
      if ('rule_content' in parsed[0]) {
        contents = { rules: parsed }
        detectedModule = targetModule === 'summary' ? 'audit' : targetModule
      } else if ('user_prompt' in parsed[0] || 'title' in parsed[0]) {
        contents = { summary_blocks: parsed }
        detectedModule = 'summary'
      }
    }
  } else if (parsed && typeof parsed === 'object') {
    // 裸 AI 配置或混装对象
    if ('audit_strictness' in parsed || 'system_reasoning_prompt' in parsed || 'user_reasoning_prompt' in parsed) {
      contents = { ai: parsed }
      detectedModule = targetModule === 'summary' ? 'audit' : targetModule
    } else if ('rules' in parsed || 'ai' in parsed || 'summary_blocks' in parsed) {
      contents = parsed
      if (parsed.module) detectedModule = parsed.module
    }
  }

  report.detected_module = detectedModule
  report.parsed_data = contents

  // 模块匹配性检测
  if (detectedModule !== 'unknown' && detectedModule !== targetModule) {
    report.module_mismatch = true
    if ((detectedModule === 'audit' && targetModule === 'archive') || (detectedModule === 'archive' && targetModule === 'audit')) {
      report.warnings.push(`导入文件来源于【${detectedModule === 'audit' ? '审核工作台' : '归档复盘'}】，规则与提示词结构完全兼容，可继续导入。`)
    } else {
      report.warnings.push(`导入文件模块【${detectedModule}】与当前目标模块【${targetModule}】不一致，请谨慎确认内容。`)
    }
  }

  // 2. 校验规则部分
  if (contents.rules && Array.isArray(contents.rules) && contents.rules.length > 0) {
    report.has_rules = true
    const total = contents.rules.length
    let validCount = 0
    let invalidCount = 0
    let duplicateCount = 0

    // 建立现有规则的文本指纹集合（忽略两端空格）
    const existingRuleSet = new Set(
      existingRules.map(r => (r.rule_content || '').trim()).filter(Boolean)
    )

    const validScopes = new Set(['mandatory', 'default_on', 'default_off'])

    contents.rules.forEach((r, idx) => {
      const content = (r.rule_content || '').trim()
      if (!content) {
        invalidCount++
        report.warnings.push(`第 ${idx + 1} 条规则内容为空，导入时将自动忽略。`)
        return
      }

      if (r.rule_scope && !validScopes.has(r.rule_scope)) {
        report.warnings.push(`第 ${idx + 1} 条规则 "${content.slice(0, 15)}..." 的作用域 "${r.rule_scope}" 不合规，将默认修正为 default_on。`)
        r.rule_scope = 'default_on'
      }

      validCount++
      if (existingRuleSet.has(content)) {
        duplicateCount++
      }
    })

    report.rule_stats = {
      total,
      valid: validCount,
      invalid: invalidCount,
      duplicate: duplicateCount,
      new_rules: validCount - duplicateCount,
    }
  }

  // 3. 校验 AI 提示词部分
  if (contents.ai && typeof contents.ai === 'object' && Object.keys(contents.ai).length > 0) {
    report.has_ai = true
    const ai = contents.ai
    const unknownVars: string[] = []

    // 检查尺度
    if (ai.audit_strictness && !['relaxed', 'standard', 'strict'].includes(ai.audit_strictness)) {
      report.warnings.push(`AI 审核尺度 "${ai.audit_strictness}" 未知，将默认回退为 standard。`)
      ai.audit_strictness = 'standard'
    }

    // 扫描提示词中的占位符
    const allPrompts = [
      ai.system_reasoning_prompt,
      ai.system_extraction_prompt,
      ai.user_reasoning_prompt,
      ai.user_extraction_prompt,
    ].filter(Boolean).join('\n')

    const vars = extractPlaceholderVariables(allPrompts)
    vars.forEach(v => {
      if (!KNOWN_AUDIT_AI_VARIABLES.has(v)) {
        unknownVars.push(v)
      }
    })

    if (unknownVars.length > 0) {
      report.warnings.push(`AI 提示词中包含系统未预设的变量: ${unknownVars.join(', ')}，请确认是否为业务自定义字段。`)
    }

    report.ai_stats = {
      strictness: ai.audit_strictness,
      enable_thinking: ai.enable_thinking,
      unknown_variables: unknownVars,
    }
  }

  // 4. 校验流程总结块部分
  if (contents.summary_blocks && Array.isArray(contents.summary_blocks) && contents.summary_blocks.length > 0) {
    report.has_summary_blocks = true
    const blockTitles: string[] = []
    const unknownVars: string[] = []

    contents.summary_blocks.forEach((blk, idx) => {
      const title = (blk.title || '').trim()
      if (!title) {
        report.warnings.push(`第 ${idx + 1} 个总结块缺少标题，将命名为“未命名总结块 ${idx + 1}”。`)
        blk.title = `未命名总结块 ${idx + 1}`
      }
      blockTitles.push(blk.title)

      const promptVars = extractPlaceholderVariables(blk.user_prompt)
      promptVars.forEach(v => {
        if (!KNOWN_SUMMARY_DATA_VARIABLES.has(v)) {
          unknownVars.push(v)
        }
      })
    })

    if (unknownVars.length > 0) {
      report.warnings.push(`总结块提示词中包含未识别的数据变量: ${Array.from(new Set(unknownVars)).join(', ')}。`)
    }

    report.summary_stats = {
      total_blocks: contents.summary_blocks.length,
      block_titles: blockTitles,
      unknown_variables: Array.from(new Set(unknownVars)),
    }
  }

  // 是否有任何有效内容
  if (!report.has_rules && !report.has_ai && !report.has_summary_blocks) {
    report.errors.push('未在导入文件中检测到任何可用的规则、AI 提示词或总结块配置。')
    report.valid = false
    return report
  }

  report.valid = report.errors.length === 0
  return report
}

/** 应用规则冲突策略并返回合并后的规则数组 */
export function applyRuleConflictStrategy<T extends { id?: string; rule_content: string; [key: string]: any }>(
  existingRules: T[],
  importedRules: ExportedRuleItem[],
  strategy: RuleConflictStrategy,
  createDraftIdFn: () => string
): T[] {
  const result: T[] = existingRules.map(r => ({ ...r }))
  const existingMap = new Map<string, number>()
  result.forEach((r, idx) => {
    const key = (r.rule_content || '').trim()
    if (key) existingMap.set(key, idx)
  })

  importedRules.forEach(imp => {
    const content = (imp.rule_content || '').trim()
    if (!content) return

    const existingIdx = existingMap.get(content)
    if (existingIdx !== undefined) {
      // 存在重复项
      if (strategy === 'skip') {
        // 跳过，不做任何改动
        return
      }
      if (strategy === 'overwrite') {
        // 覆盖已有项属性
        result[existingIdx] = {
          ...result[existingIdx],
          rule_scope: imp.rule_scope || result[existingIdx].rule_scope,
          enabled: imp.enabled !== undefined ? imp.enabled : result[existingIdx].enabled,
          related_flow: imp.related_flow !== undefined ? imp.related_flow : result[existingIdx].related_flow,
          context_enabled: imp.context_enabled !== undefined ? imp.context_enabled : result[existingIdx].context_enabled,
          context_mounts: imp.context_mounts || result[existingIdx].context_mounts || [],
        }
        return
      }
    }

    // 新增或者 append 策略（全部追加为新规则）
    const newRule: any = {
      id: createDraftIdFn(),
      rule_content: content,
      rule_scope: imp.rule_scope || 'default_on',
      enabled: imp.enabled !== undefined ? imp.enabled : true,
      source: imp.source || 'file_import',
      related_flow: !!imp.related_flow,
      context_enabled: !!imp.context_enabled,
      context_mounts: imp.context_mounts || [],
    }
    result.push(newRule)
    existingMap.set(content, result.length - 1)
  })

  return result
}
