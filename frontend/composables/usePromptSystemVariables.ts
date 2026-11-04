export interface PromptVariableItem {
  key: string
  desc: string
}

export interface PromptSystemVariableOptions {
  /**
   * 是否暴露智能体模块的两段 Generative UI 卡片协议变量
   * （{{form_confirm_protocol}} / {{chart_protocol}}）。
   * 仅智能体系统提示词需要，审核/归档/总结等提示词编辑器不展示。
   */
  includeAgentUiProtocols?: boolean
}

export function usePromptSystemVariables(options: PromptSystemVariableOptions = {}) {
  const { t } = useI18n()

  const systemPromptVariables = computed<PromptVariableItem[]>(() => {
    const items: PromptVariableItem[] = [
      { key: '{{current_date}}', desc: t('admin.ruleConfig.varCurrentDate') },
      { key: '{{current_time}}', desc: t('admin.ruleConfig.varCurrentTime') },
      { key: '{{current_datetime}}', desc: t('admin.ruleConfig.varCurrentDatetime') },
      { key: '{{weekday}}', desc: t('admin.ruleConfig.varWeekday') },
    ]
    if (options.includeAgentUiProtocols) {
      items.push(
        { key: '{{form_confirm_protocol}}', desc: t('admin.ruleConfig.varFormConfirmProtocol') },
        { key: '{{chart_protocol}}', desc: t('admin.ruleConfig.varChartProtocol') },
      )
    }
    return items
  })

  return { systemPromptVariables }
}
