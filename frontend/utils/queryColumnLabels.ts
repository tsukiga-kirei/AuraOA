import type { QueryBuilder, QueryColumn, QueryDefinition } from '~/types/audit-data-query'

/** 按物理字段名查找中文标签，兼容 OA 数据库返回的大写字段。 */
export function queryColumnLabel(columns: QueryColumn[], field: string): string {
  const label =
    columns.find((column) => column.key.toLowerCase() === field.toLowerCase())?.label.trim() || ''
  return /[\u3400-\u9fff]/.test(label) ? label : ''
}

/** 读取字段后补全空白用途，保留用户填写的输出别名与筛选说明。 */
export function fillQueryBuilderLabels(builder: QueryBuilder, columns: QueryColumn[]): void {
  for (const column of builder.columns) {
    if (!column.alias?.trim()) column.alias = queryColumnLabel(columns, column.field)
  }
  for (const filter of builder.filters) {
    if (!filter.label?.trim()) filter.label = queryColumnLabel(columns, filter.field)
  }
}

/** 选择输出字段时同时更新中文别名；切换字段可更新自动标签，保留自定义别名。 */
export function selectQueryOutputField(
  builder: QueryBuilder, index: number, field: string, columns: QueryColumn[],
): void {
  const output = builder.columns[index]
  const previousLabel = queryColumnLabel(columns, output.field)
  const useDefault = !output.alias?.trim() || output.alias === previousLabel
  output.field = field
  if (useDefault) output.alias = queryColumnLabel(columns, field)
}

/** AI 上下文只在表结构中发送字段说明，避免与草稿的 field_notes 重复。 */
export function queryAssistFieldContext(
  definition: QueryDefinition, schema: Record<string, QueryColumn[]>,
) {
  const notes = new Map(
    Object.entries(definition.ai_context?.field_notes || {}).map(([key, value]) => [key.toLowerCase(), value]),
  )
  const { field_notes: _fieldNotes, ...context } = definition.ai_context || {}
  return {
    definition: { ...definition, ...(definition.ai_context ? { ai_context: context } : {}) },
    schema: Object.fromEntries(Object.entries(schema).map(([table, columns]) => [
      table,
      columns.map((column) => {
        const description = notes.get(`${table}.${column.key}`.toLowerCase())?.trim()
        return {
          ...column,
          ...(description && description !== column.label.trim() ? { description } : {}),
        }
      }),
    ])),
  }
}

/** 默认用中文标签填充业务说明，大小写不同的已有说明仍优先保留。 */
export function fillQueryFieldNotes(
  notes: Record<string, string>,
  table: string,
  columns: QueryColumn[],
): Record<string, string> {
  const result = { ...notes }
  const keys = new Map(Object.keys(result).map((key) => [key.toLowerCase(), key]))
  for (const column of columns) {
    const key = `${table}.${column.key}`
    const existingKey = keys.get(key.toLowerCase())
    const label = queryColumnLabel(columns, column.key)
    const existing = existingKey ? result[existingKey] : ''
    const value = existing?.trim() ? existing : label
    if (existingKey && existingKey !== key) delete result[existingKey]
    if (value) result[key] = value
  }
  return result
}
