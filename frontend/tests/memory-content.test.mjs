import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import ts from 'typescript'
const source = fs.readFileSync(new URL('../utils/memoryContent.ts', import.meta.url), 'utf8')
const js = ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ESNext } }).outputText
const { queryDefinitionContentKey, feedbackMemoryContentKey } = await import(`data:text/javascript;base64,${Buffer.from(js).toString('base64')}`)

test('query content ignores key order, legacy defaults and null collections', () => {
  const old = { name: '统计', description: '说明', sql: 'SELECT 1 AS 数量', dialect: 'mysql', tables: null, parameters: null,
    builder: { table: 'form1', source_type: 'workflow', columns: [{ field: 'amount' }], filters: null } }
  const current = { ...old, mode: 'config', tables: [], parameters: [], max_rows: 0,
    ai_context: { oa_module: 'workflow', notes: '', field_notes: {} },
    builder: { columns: [{ aggregate: '', alias: '', field: 'amount' }], filters: [], table: 'form1', source_type: 'workflow', archived_only: true, order_alias: '', order_dir: 'DESC' } }
  assert.equal(queryDefinitionContentKey(old), queryDefinitionContentKey(current))
  assert.notEqual(queryDefinitionContentKey(old), queryDefinitionContentKey({ ...current, builder: { ...current.builder, archived_only: false } }))
  assert.notEqual(queryDefinitionContentKey(old), queryDefinitionContentKey({ ...current, sql: 'SELECT 2 AS 数量' }))
})

test('query content retains SQL literal spaces and binding changes', () => {
  const original = { mode: 'sql', name: '统计', sql: "SELECT 'a  b' AS 文本", parameters: [{ name: 'p', label: '条件', type: 'string', default_binding: { source: 'constant', value: '甲' } }] }
  assert.notEqual(queryDefinitionContentKey(original), queryDefinitionContentKey({ ...original, sql: "SELECT 'a b' AS 文本" }))
  assert.notEqual(queryDefinitionContentKey(original), queryDefinitionContentKey({ ...original, parameters: [{ ...original.parameters[0], default_binding: { source: 'constant', value: '乙' } }] }))
})

test('memory edits compare real body, conditions and provenance while ignoring reference order', () => {
  const original = [{ content: '核实金额', source_refs: ['a', 'b'] }]
  assert.equal(feedbackMemoryContentKey(original), feedbackMemoryContentKey([{ condition: null, content: ' 核实金额 ', source_refs: ['b', 'a'] }]))
  assert.notEqual(feedbackMemoryContentKey(original), feedbackMemoryContentKey([{ ...original[0], condition: '金额超过100万' }]))
  assert.notEqual(feedbackMemoryContentKey(original), feedbackMemoryContentKey([{ ...original[0], source_refs: ['a'] }]))
  assert.equal(feedbackMemoryContentKey(null), feedbackMemoryContentKey([]))
})
