import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import ts from 'typescript'

const source = fs.readFileSync(new URL('../utils/configExportImportHelper.ts', import.meta.url), 'utf8')
const js = ts.transpileModule(source, {
  compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ESNext },
}).outputText

const mod = await import(`data:text/javascript;base64,${Buffer.from(js).toString('base64')}`)
const {
  buildExportBundle,
  validateImportBundle,
  applyRuleConflictStrategy,
  extractPlaceholderVariables,
} = mod

test('buildExportBundle creates standardized DSL format', () => {
  const bundle = buildExportBundle({
    module: 'audit',
    processType: 'BX01',
    processTypeLabel: '费用报销',
    mainTableName: 'formtable_main_10',
    contents: {
      rules: [
        { rule_content: '金额合规', rule_scope: 'mandatory' },
      ],
      ai: {
        audit_strictness: 'standard',
        enable_thinking: true,
      },
    },
  })

  assert.equal(bundle.schema_version, '1.0')
  assert.equal(bundle.app, 'AuraOA')
  assert.equal(bundle.module, 'audit')
  assert.equal(bundle.source_meta.process_type, 'BX01')
  assert.equal(bundle.contents.rules.length, 1)
  assert.equal(bundle.contents.ai.audit_strictness, 'standard')
})

test('extractPlaceholderVariables finds {{xxx}} variables', () => {
  const text = '你好，今天是 {{current_date}}，时间是 {{current_time}}，流程 {{workflow_id}}。'
  const vars = extractPlaceholderVariables(text)
  assert.deepEqual(vars, ['{{current_date}}', '{{current_time}}', '{{workflow_id}}'])
})

test('validateImportBundle validates valid audit bundle and detects duplicates', () => {
  const existingRules = [
    { rule_content: '发票必须有效' },
  ]

  const raw = JSON.stringify({
    schema_version: '1.0',
    module: 'audit',
    contents: {
      rules: [
        { rule_content: '发票必须有效', rule_scope: 'mandatory' },
        { rule_content: '金额大于100需审批', rule_scope: 'default_on' },
      ],
      ai: {
        audit_strictness: 'strict',
        user_reasoning_prompt: '请在 {{current_date}} 进行严格审核',
      },
    },
  })

  const report = validateImportBundle(raw, 'audit', existingRules)
  assert.equal(report.valid, true)
  assert.equal(report.has_rules, true)
  assert.equal(report.has_ai, true)
  assert.equal(report.rule_stats.total, 2)
  assert.equal(report.rule_stats.valid, 2)
  assert.equal(report.rule_stats.duplicate, 1)
  assert.equal(report.rule_stats.new_rules, 1)
  assert.equal(report.ai_stats.strictness, 'strict')
})

test('validateImportBundle handles raw rule arrays and invalid scopes', () => {
  const raw = JSON.stringify([
    { rule_content: '规则1', rule_scope: 'invalid_scope' },
    { rule_content: '', rule_scope: 'mandatory' },
  ])

  const report = validateImportBundle(raw, 'audit', [])
  assert.equal(report.valid, true)
  assert.equal(report.has_rules, true)
  assert.equal(report.rule_stats.total, 2)
  assert.equal(report.rule_stats.valid, 1)
  assert.equal(report.rule_stats.invalid, 1)
  assert.equal(report.parsed_data.rules[0].rule_scope, 'default_on') // 自动回退修正
})

test('validateImportBundle recognizes system default prompt variables without warnings', () => {
  const raw = JSON.stringify({
    schema_version: '1.0',
    module: 'audit',
    contents: {
      ai: {
        audit_strictness: 'standard',
        user_reasoning_prompt: '{{main_table}} {{detail_tables}} {{attachments}} {{rules}} {{flow_history}} {{flow_graph}} {{current_node}}',
        user_extraction_prompt: '{{reasoning_result}} {{rules}}',
      },
    },
  })

  const report = validateImportBundle(raw, 'audit', [])
  assert.equal(report.valid, true)
  assert.equal(report.ai_stats.unknown_variables.length, 0)
  assert.equal(report.warnings.some(w => w.includes('系统未预设的变量')), false)
})

test('validateImportBundle warns about unrecognized prompt variables', () => {
  const raw = JSON.stringify({
    schema_version: '1.0',
    module: 'audit',
    contents: {
      ai: {
        audit_strictness: 'standard',
        user_reasoning_prompt: '请核查 {{unknown_var}} 与 {{current_date}} 及 {{main_table}}',
      },
    },
  })

  const report = validateImportBundle(raw, 'audit', [])
  assert.equal(report.valid, true)
  assert.deepEqual(report.ai_stats.unknown_variables, ['{{unknown_var}}'])
  assert.equal(report.warnings.some(w => w.includes('{{unknown_var}}')), true)
  assert.equal(report.warnings.some(w => w.includes('{{main_table}}')), false)
})

test('applyRuleConflictStrategy works for skip, overwrite, and append', () => {
  const existing = [
    { id: '1', rule_content: '规则A', rule_scope: 'default_on', enabled: true },
  ]
  const imported = [
    { rule_content: '规则A', rule_scope: 'mandatory', enabled: false },
    { rule_content: '规则B', rule_scope: 'default_off', enabled: true },
  ]

  let idCounter = 0
  const createId = () => `draft-${++idCounter}`

  // 1. Skip
  const skipped = applyRuleConflictStrategy(existing, imported, 'skip', createId)
  assert.equal(skipped.length, 2)
  assert.equal(skipped[0].rule_scope, 'default_on') // 规则A保持不变
  assert.equal(skipped[1].rule_content, '规则B')

  // 2. Overwrite
  const overwritten = applyRuleConflictStrategy(existing, imported, 'overwrite', createId)
  assert.equal(overwritten.length, 2)
  assert.equal(overwritten[0].rule_scope, 'mandatory') // 规则A被覆盖
  assert.equal(overwritten[0].enabled, false)

  // 3. Append
  const appended = applyRuleConflictStrategy(existing, imported, 'append', createId)
  assert.equal(appended.length, 3) // 规则A出现两次
})
