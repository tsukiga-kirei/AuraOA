import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import ts from 'typescript'

const source = fs.readFileSync(new URL('../utils/fieldTransmission.ts', import.meta.url), 'utf8')
const js = ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ESNext } }).outputText
const { isEmptyFieldSkipped, setEmptyFieldSkipped, setSkipEmptyFields } = await import(`data:text/javascript;base64,${Buffer.from(js).toString('base64')}`)

test('global override retains individual settings and restores them when disabled', () => {
  const config = { audit_strictness: 'standard' }
  const field = { source: 'main', field_key: 'remark' }
  const other = { source: 'form_dt1', field_key: 'remark' }
  setEmptyFieldSkipped(config, field, true)
  setSkipEmptyFields(config, true)
  assert.equal(config.skip_empty_fields, true)
  assert.equal(isEmptyFieldSkipped(config, field), true)
  assert.equal(isEmptyFieldSkipped(config, other), false)
  setSkipEmptyFields(config, false)
  assert.equal(config.skip_empty_fields, undefined)
  assert.equal(isEmptyFieldSkipped(config, field), true)
  assert.equal(isEmptyFieldSkipped(config, other), false)
  assert.equal(config.audit_strictness, 'standard')
})

test('same field names in different tables remain independent and case insensitive', () => {
  const config = {}
  const main = { source: 'main', field_key: 'REMARK' }
  const detail = { source: 'FORM_DT1', field_key: 'remark' }
  setEmptyFieldSkipped(config, main, true)
  setEmptyFieldSkipped(config, detail, true)
  setEmptyFieldSkipped(config, { source: 'MAIN', field_key: 'remark' }, true)
  assert.deepEqual(config.skip_empty_field_keys, ['form_dt1:remark', 'main:remark'])
  setEmptyFieldSkipped(config, main, false)
  assert.equal(isEmptyFieldSkipped(config, main), false)
  assert.equal(isEmptyFieldSkipped(config, detail), true)
})

test('switching all controls on then off restores legacy configuration content', () => {
  const config = { user_reasoning_prompt: '{{main_table}}' }
  const initial = JSON.stringify(config)
  const field = { source: 'main', field_key: 'remark' }
  assert.equal(isEmptyFieldSkipped(config, field), false)
  setSkipEmptyFields(config, true)
  setSkipEmptyFields(config, false)
  setEmptyFieldSkipped(config, field, true)
  setEmptyFieldSkipped(config, field, false)
  assert.equal(JSON.stringify(config), initial)
})
