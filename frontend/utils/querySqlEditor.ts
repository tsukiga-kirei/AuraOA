import type { QueryDefinition } from '~/types/audit-data-query'

/** SQL 展示辅助仅负责格式化与高亮，发布和执行校验始终由后端负责。 */
const sqlTokens =
  /'(?:''|[^'])*'|"(?:""|[^"])*"|`(?:``|[^`])*`|:[A-Za-z_][A-Za-z0-9_]*|\b(?:SELECT|FROM|WHERE|AS|JOIN|LEFT|RIGHT|INNER|OUTER|ON|AND|OR|GROUP|BY|ORDER|HAVING|WITH|UNION|ALL|DISTINCT|IN|NOT|NULL|IS|CASE|WHEN|THEN|ELSE|END|SUM|COUNT|AVG|MAX|MIN|COALESCE|NVL|TO_DATE|TO_CHAR|LIMIT|FETCH)\b/gi
const escapeHTML = (value: string) =>
  value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
export function highlightQuerySql(sql: string): string {
  let result = ''
  let offset = 0
  for (const match of sql.matchAll(sqlTokens)) {
    const index = match.index ?? 0
    result += escapeHTML(sql.slice(offset, index))
    const kind = match[0].startsWith(':')
      ? 'parameter'
      : /^["'`]/.test(match[0])
        ? 'literal'
        : 'keyword'
    result += `<span class="sql-${kind}">${escapeHTML(match[0])}</span>`
    offset = index + match[0].length
  }
  return result + escapeHTML(sql.slice(offset))
}
export function formatQuerySql(sql: string): string {
  const protectedParts: string[] = []
  const masked = sql.replace(/'(?:''|[^'])*'|"(?:""|[^"])*"|`(?:``|[^`])*`/g, (value) => {
    protectedParts.push(value)
    return `\u0000${protectedParts.length - 1}\u0000`
  })
  return masked
    .replace(/\s+/g, ' ')
    .replace(
      /\b(SELECT|FROM|WHERE|LEFT JOIN|RIGHT JOIN|INNER JOIN|JOIN|GROUP BY|ORDER BY|HAVING|UNION ALL|UNION)\b/gi,
      '\n$1',
    )
    .replace(/\b(AND|OR)\b/gi, '\n  $1')
    .trim()
    .replace(/\u0000(\d+)\u0000/g, (_, index) => protectedParts[Number(index)] || '')
}

/** 展示与执行端一致的行数包装；额外一行仅用于判定截断，不写回原始 SQL。 */
export function previewQuerySql(
  definition: Pick<QueryDefinition, 'sql' | 'dialect' | 'max_rows'>,
): string {
  const sql = definition.sql.trim()
  if (!sql || !Number.isInteger(definition.max_rows) || definition.max_rows <= 0) return sql
  const limit = String(BigInt(definition.max_rows) + BigInt(1))
  const indented = sql
    .split('\n')
    .map((line) => `  ${line}`)
    .join('\n')
  const clause = definition.dialect === 'mysql' ? `LIMIT ${limit}` : `WHERE ROWNUM <= ${limit}`
  return `SELECT *\nFROM (\n${indented}\n) auraoa_query\n${clause}`
}

/** 从 SQL 识别值占位符，忽略注释、字符串、带引号列名和双冒号转换。 */
export function querySqlParameterNames(sql: string): string[] {
  const tokens =
    /--[^\r\n]*|#[^\r\n]*|\/\*[\s\S]*?\*\/|'(?:''|\\.|[^'\\])*'|"(?:""|[^"])*"|`(?:``|[^`])*`|:[A-Za-z_][A-Za-z0-9_]*/g
  const names = new Set<string>()
  for (const match of sql.matchAll(tokens)) {
    if (match[0].startsWith(':') && sql[(match.index ?? 0) - 1] !== ':') {
      names.add(match[0].slice(1))
    }
  }
  return [...names]
}
