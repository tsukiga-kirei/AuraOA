import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import ts from 'typescript'

function loadTs(relativePath) {
  const module = { exports: {} }
  const code = ts.transpileModule(fs.readFileSync(new URL(relativePath, import.meta.url), 'utf8'), {
    compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS },
  }).outputText
  new Function('module', 'exports', code)(module, module.exports)
  return module.exports
}

const { nextSkillCopyCode, buildSkillEditDraft, buildSkillCopyDraft } = loadTs('../utils/skillCode.ts')

function skill(overrides = {}) {
  return {
    id: 'skill-id',
    tenant_id: 'tenant-id',
    skill_code: 'oa_workflow_stats',
    name: 'OA 流程与人员统计查数指南',
    description: 'desc',
    content: 'content',
    enabled: true,
    is_system: false,
    agent_codes: ['oa_analyst'],
    created_at: '2026-11-01T00:00:00Z',
    updated_at: '2026-11-01T00:00:00Z',
    ...overrides,
  }
}

test('appends _copy to a builtin skill code', () => {
  assert.equal(nextSkillCopyCode('oa_workflow_stats', []), 'oa_workflow_stats_copy')
})

test('avoids collisions by appending an index', () => {
  assert.equal(
    nextSkillCopyCode('oa_workflow_stats', ['oa_workflow_stats_copy']),
    'oa_workflow_stats_copy2',
  )
  assert.equal(
    nextSkillCopyCode('oa_workflow_stats', ['oa_workflow_stats_copy', 'oa_workflow_stats_copy2']),
    'oa_workflow_stats_copy3',
  )
})

test('copying a copy returns the original base instead of _copy_copy', () => {
  assert.equal(nextSkillCopyCode('oa_workflow_stats_copy', []), 'oa_workflow_stats_copy')
  assert.equal(
    nextSkillCopyCode('oa_workflow_stats_copy', ['oa_workflow_stats_copy']),
    'oa_workflow_stats_copy2',
  )
})

test('keeps the generated code within the 64-char skill_code limit', () => {
  const result = nextSkillCopyCode('x'.repeat(80), [])
  assert.ok(result.length <= 64, `expected <=64 chars, got ${result.length}`)
})

test('editing a builtin skill produces a same-code tenant override without id', () => {
  const draft = buildSkillEditDraft(skill({ is_system: true, tenant_id: null }))
  assert.equal(draft.mode, 'override')
  assert.equal(draft.form.skill_code, 'oa_workflow_stats')
  assert.equal(draft.form.id, undefined)
  assert.equal(draft.form.content, 'content')
  assert.deepEqual(draft.form.agent_codes, ['oa_analyst'])
})

test('editing a tenant skill keeps its id so it updates in place', () => {
  const draft = buildSkillEditDraft(skill())
  assert.equal(draft.mode, 'edit')
  assert.equal(draft.form.id, 'skill-id')
  assert.equal(draft.form.skill_code, 'oa_workflow_stats')
})

test('copying a skill creates a new code and no id, leaving the original untouched', () => {
  const draft = buildSkillCopyDraft(skill({ is_system: true, tenant_id: null }), ['oa_workflow_stats'])
  assert.equal(draft.mode, 'copy')
  assert.equal(draft.form.id, undefined)
  assert.equal(draft.form.skill_code, 'oa_workflow_stats_copy')
  assert.match(draft.form.name, /副本$/)
  assert.equal(draft.form.content, 'content')
})
