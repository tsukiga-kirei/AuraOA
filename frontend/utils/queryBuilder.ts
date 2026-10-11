import type {
  QueryBuilder,
  QueryColumn,
  QueryDefinition,
  QueryParameter,
} from '../types/audit-data-query'

/** 简单查询配置编译为统一 SQL；标识符、操作符和聚合函数均限制为固定集合。 */
export function compileQueryBuilder(
  builder: QueryBuilder,
  dialect: QueryDefinition['dialect'],
  schema: QueryColumn[],
) {
  const fail = (code: string): never => {
    throw new Error(code)
  }
  const identifier = (value: string) => {
    if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(value)) fail('identifier')
    return dialect === 'mysql' ? `\`${value}\`` : `"${value.toUpperCase()}"`
  }
  const field = (value: string) => {
    if (!schema.some((column) => column.key.toLowerCase() === value.toLowerCase())) fail('field')
    return `d.${identifier(value)}`
  }
  const alias = (value: string) => {
    if (!/[\u3400-\u9fff]/.test(value) || value.length > 100 || /[\x00-\x1f`"\\]/.test(value))
      fail('alias')
    return dialect === 'mysql' ? `\`${value}\`` : `"${value}"`
  }
  if (!builder.table || !schema.length) fail('schema')
  if (!builder.columns.length || builder.columns.length > 40 || builder.filters.length > 20)
    fail('columns')
  const table = identifier(builder.table.trim())
  const names = new Set<string>()
  const expressions = builder.columns.map((column) => {
    if (names.has(column.alias)) fail('alias')
    names.add(column.alias)
    const expression =
      column.field === '*' && column.aggregate === 'COUNT' ? '*' : field(column.field)
    if (!['', 'SUM', 'COUNT', 'MIN', 'MAX', 'AVG'].includes(column.aggregate)) fail('aggregate')
    return `${column.aggregate ? `${column.aggregate}(${expression})` : expression} AS ${alias(column.alias)}`
  })
  const parameters: QueryParameter[] = []
  const conditions = builder.filters.map((filter) => {
    const expression = field(filter.field)
    if (
      !['=', '<>', '>', '>=', '<', '<=', 'LIKE', 'IN', 'IS NULL', 'IS NOT NULL'].includes(
        filter.operator,
      )
    )
      fail('operator')
    if (filter.operator === 'IS NULL' || filter.operator === 'IS NOT NULL')
      return `${expression} ${filter.operator}`
    if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(filter.parameter) || !/[\u3400-\u9fff]/.test(filter.label))
      fail('parameter')
    if (!['string', 'number', 'date', 'string_list'].includes(filter.type)) fail('parameter')
    if ((filter.operator === 'IN') !== (filter.type === 'string_list')) fail('list')
    if (filter.binding?.source === 'constant' && !filter.binding.value?.trim()) fail('value')
    if (
      filter.binding?.source === 'system' &&
      !['process_id', 'current_date', 'year_start', 'month_start', 'current_year'].includes(
        filter.binding.value || '',
      )
    )
      fail('system')
    const parameter = {
      name: filter.parameter,
      label: filter.label,
      type: filter.type,
      ...(filter.binding ? { default_binding: { ...filter.binding } } : {}),
    }
    const previous = parameters.find((item) => item.name === parameter.name)
    if (previous && JSON.stringify(previous) !== JSON.stringify(parameter)) fail('parameter')
    if (!previous) parameters.push(parameter)
    let placeholder = `:${filter.parameter}`
    const targetColumn = schema.find((c) => c.key.toLowerCase() === filter.field.toLowerCase())
    const isNativeDate = targetColumn && /^(date|timestamp)/i.test(targetColumn.data_type || '')
    if (filter.type === 'date' && dialect !== 'mysql' && isNativeDate)
      placeholder = `TO_DATE(${placeholder}, 'YYYY-MM-DD')`
    if (filter.operator === 'IN') placeholder = `(${placeholder})`
    return `${expression} ${filter.operator} ${placeholder}`
  })
  const tables = [builder.table.trim()]
  // 按开关用流程状态判定归档，requestid 只用于关联；EXISTS 避免重复计数。
  if (builder.source_type === 'workflow' && builder.archived_only !== false) {
    const request = field('requestid')
    tables.push('workflow_requestbase')
    conditions.push(
      `EXISTS (SELECT 1 FROM ${identifier('workflow_requestbase')} r WHERE r.${identifier('requestid')} = ${request} AND r.${identifier('currentnodetype')} = '3')`,
    )
  }
  let sql = `SELECT\n  ${expressions.join(',\n  ')}\nFROM ${table} d`
  if (conditions.length) sql += `\nWHERE ${conditions.join('\n  AND ')}`
  if (builder.columns.some((column) => column.aggregate)) {
    const groups = [
      ...new Set(
        builder.columns.filter((column) => !column.aggregate).map((column) => field(column.field)),
      ),
    ]
    if (groups.length) sql += `\nGROUP BY ${groups.join(', ')}`
  }
  if (builder.order_alias) {
    if (!names.has(builder.order_alias) || !['ASC', 'DESC'].includes(builder.order_dir))
      fail('order')
    sql += `\nORDER BY ${alias(builder.order_alias)} ${builder.order_dir}`
  }
  return { sql, tables, parameters }
}
