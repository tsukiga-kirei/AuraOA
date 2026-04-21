/**
 * scoreColor.ts — 评分提取与颜色映射工具函数。
 *
 * 从通知 body 文本中提取评分数值，并根据分数区间
 * 返回对应的语义色 CSS 变量名，用于前端颜色标注。
 */

/** 从通知 body 文本中提取评分数值 */
export function extractScore(body: string): number | null {
  const match = body.match(/评分\s*(\d+)/)
  return match ? Number(match[1]) : null
}

/** 根据评分返回语义色 CSS 变量名 */
export function scoreColor(score: number): string {
  if (score >= 80) return 'var(--color-success)'
  if (score >= 60) return 'var(--color-warning)'
  return 'var(--color-danger)'
}
