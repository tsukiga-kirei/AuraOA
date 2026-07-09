export interface PromptVariableItem {
  key: string
  desc: string
}

export function usePromptSystemVariables() {
  const { t } = useI18n()

  const systemPromptVariables = computed<PromptVariableItem[]>(() => [
    { key: '{{current_date}}', desc: t('admin.ruleConfig.varCurrentDate') },
    { key: '{{current_time}}', desc: t('admin.ruleConfig.varCurrentTime') },
    { key: '{{current_datetime}}', desc: t('admin.ruleConfig.varCurrentDatetime') },
    { key: '{{weekday}}', desc: t('admin.ruleConfig.varWeekday') },
  ])

  return { systemPromptVariables }
}
