/** 系统级文件识别导入能力。 */
export interface RuleImportCapability {
  enabled: boolean
  max_file_size_mb: number
  supported_types: string[]
  reason?: string
}

export type RuleImportSource = 'file_import' | 'paste_import'

/** AI 从制度文件中提取、等待管理员确认的规则草稿。 */
export interface RuleImportDraft {
  rule_content: string
  rule_scope: 'mandatory' | 'default_on' | 'default_off'
  related_flow: boolean
  context_enabled: boolean
  confidence: number
  reasoning: string
}

export interface SelectableRuleImportDraft extends RuleImportDraft {
  selected: boolean
}

/** MinerU 识别与 AI 结构化后的规则预览。 */
export interface RuleImportPreview {
  file_name: string
  rules: RuleImportDraft[]
  warnings: string[]
}
