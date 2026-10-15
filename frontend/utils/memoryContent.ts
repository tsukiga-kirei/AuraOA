import type { QueryDefinition } from '~/types/audit-data-query'
import type { FeedbackMemoryItem } from '~/types/feedback-memory'

// 内容比较固定对象键顺序，保留数组顺序和 SQL 原文，不加入版本号与更新时间。
function stableContent(value: unknown): string {
  const normalize = (entry: any): any => {
    if (Array.isArray(entry)) return entry.map(normalize)
    if (entry && typeof entry === 'object') {
      return Object.fromEntries(Object.keys(entry).sort().filter(key => entry[key] !== undefined)
        .map(key => [key, normalize(entry[key])]))
    }
    return entry
  }
  return JSON.stringify(normalize(value))
}

/** 查询内容补齐旧版缺省值，分类在工作区单独比较。 */
export function queryDefinitionContentKey(value: QueryDefinition): string {
  const content = JSON.parse(JSON.stringify(value))
  content.mode ||= content.builder && !content.builder.sql_mode ? 'config' : 'sql'
  if (content.mode === 'sql') delete content.builder
  else if (content.builder) {
    const builder = content.builder
    delete builder.sql_mode
    builder.source_type ??= 'table'
    builder.table ??= ''
    builder.columns ??= []
    builder.filters ??= []
    builder.order_alias ??= ''
    builder.order_dir ??= 'DESC'
    builder.archived_only ??= builder.source_type === 'workflow'
    builder.columns = builder.columns.map((column: any) => ({
      ...column, field: column.field ?? '', alias: column.alias ?? '', aggregate: column.aggregate ?? '',
    }))
  }
  for (const key of ['name', 'description', 'sql', 'dialect']) content[key] ??= ''
  content.tables ??= []
  content.parameters ??= []
  content.max_rows ??= 0
  // 服务端省略为 0 的字符上限，避免旧定义与显式 0 产生虚假修改。
  if (!content.max_chars) delete content.max_chars
  content.ai_context ??= {
    oa_module: content.builder?.source_type === 'workflow' ? 'workflow' : 'general', notes: '',
  }
  content.ai_context.notes ??= ''
  // 空字段说明与未配置没有内容差异。
  if (!Object.keys(content.ai_context.field_notes || {}).length) delete content.ai_context.field_notes
  content.parameters = content.parameters.map((parameter: any) => {
    const normalized = { ...parameter }
    if (normalized.default_binding == null) delete normalized.default_binding
    else {
      for (const key of ['field', 'value']) if (!normalized.default_binding[key]) delete normalized.default_binding[key]
      if (!normalized.default_binding.days_offset) delete normalized.default_binding.days_offset
    }
    return normalized
  })
  return stableContent(content)
}

/** 正文与适用条件按实际内容比较，来源引用按集合比较，保留记忆条目顺序。 */
export function feedbackMemoryContentKey(items?: FeedbackMemoryItem[] | null): string {
  return stableContent((items || []).map(item => ({
    condition: (item.condition || '').trim(),
    content: (item.content || '').trim(),
    source_refs: [...(item.source_refs || [])].sort(),
  })))
}
