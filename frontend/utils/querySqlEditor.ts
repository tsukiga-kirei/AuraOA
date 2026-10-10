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
