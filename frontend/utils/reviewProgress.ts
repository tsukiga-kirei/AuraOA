type ReviewProgress = {
  id?: string
  status?: string
  ai_reasoning?: string
  deep_thinking?: string
}

/** 终态以后忽略迟到的流事件，由轮询返回的持久化结果决定正文。 */
export function isReviewRunning(status?: string): boolean {
  return !!status && ['pending', 'assembling', 'reasoning', 'extracting', 'rechecking'].includes(status)
}

/** 轮询尚无正文时保留流式展示；已有正文及终态结果直接采用，不按长度选取。 */
export function mergeReviewProgress<T extends ReviewProgress>(current: T | null | undefined, incoming: T): T {
  const merged = { ...current, ...incoming }
  if (current?.id && incoming.id && current.id !== incoming.id) return incoming
  if (isReviewRunning(incoming.status)) {
    for (const field of ['ai_reasoning', 'deep_thinking'] as const) {
      if (incoming[field] === undefined || (!incoming[field] && ['pending', 'assembling', 'reasoning'].includes(incoming.status!))) {
        merged[field] = current?.[field]
      }
    }
  }
  return merged
}
