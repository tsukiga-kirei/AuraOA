import type { QueryFieldOption, QueryTrigger } from '~/types/audit-data-query'

/** OA 类型不完整时按文本起步，用户可选择实际比较类型。 */
export function queryTriggerType(field?: QueryFieldOption): QueryTrigger['value_type'] {
  const type = (field?.field_type || '').toLowerCase()
  if (/number|decimal|numeric|float|double|int|money/.test(type)) return 'number'
  if (/date|timestamp/.test(type)) return 'date'
  if (/bool|checkbox/.test(type)) return 'boolean'
  if (/select|choice|browser|list/.test(type)) return 'choice'
  if (/file|attachment/.test(type)) return 'file'
  return 'text'
}

export function queryTriggerOperators(type: QueryTrigger['value_type']): QueryTrigger['operator'][] {
  const empty: QueryTrigger['operator'][] = ['is_empty', 'not_empty']
  if (type === 'file') return empty
  if (type === 'choice') return ['contains', 'not_contains', ...empty]
  if (type === 'number' || type === 'date') return ['eq', 'ne', 'gt', 'gte', 'lt', 'lte', ...empty]
  if (type === 'boolean') return ['eq', 'ne', ...empty]
  return ['eq', 'ne', 'contains', 'not_contains', ...empty]
}

export const queryTriggerNeedsValue = (trigger: QueryTrigger) =>
  !['is_empty', 'not_empty'].includes(trigger.operator)

export function queryTriggerOperatorKey(trigger: QueryTrigger, operator = trigger.operator) {
  const special = trigger.value_type === 'date' && ['gt', 'gte', 'lt', 'lte'].includes(operator)
    ? 'date.' : trigger.value_type === 'choice' && ['contains', 'not_contains'].includes(operator)
      ? 'choice.' : ''
  return `query.triggerOperator.${special}${operator}`
}

/** 保存规则与流程预览使用同一基础校验，完整类型校验由服务端执行。 */
export function queryTriggerError(trigger?: QueryTrigger): string | undefined {
  if (!trigger) return
  if (!trigger.field) return 'query.triggerFieldRequired'
  if (!queryTriggerOperators(trigger.value_type).includes(trigger.operator)) return 'query.triggerOperatorRequired'
  if (queryTriggerNeedsValue(trigger) && !String(trigger.value ?? '').trim()) return 'query.triggerValueRequired'
}
