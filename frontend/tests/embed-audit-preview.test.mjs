import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import ts from 'typescript'
import * as vue from 'vue'

const module = { exports: {} }
const source = fs.readFileSync(new URL('../utils/embedAuditFixtures.ts', import.meta.url), 'utf8')
const compiled = ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS } }).outputText
new Function('require', 'module', 'exports', compiled)(id => {
  if (id === 'vue') return vue
  throw new Error(`Unexpected dependency: ${id}`)
}, module, module.exports)
const { createEmbedAuditPreview } = module.exports
const labels = { failure: '调用失败', parseError: '结构化失败', unsupported: '未开启', stages: ['排队中', '组装提示词', '推理分析', '结构化提取', 'AI 复核'], recheckErrors: { call: '调用失败', empty: '内容为空', json: 'JSON 转换失败', invalid: '必要字段缺失' } }

async function advance(t, promise) {
  let done = false
  promise.then(() => { done = true }, () => { done = true })
  for (let tick = 0; tick < 400 && !done; tick++) {
    t.mock.timers.tick(10000)
    await Promise.resolve()
    await Promise.resolve()
  }
  assert.ok(done, 'the simulated run should settle')
  return promise
}

for (const scenario of ['recheckCallFailed', 'recheckEmpty', 'recheckJsonFailed', 'recheckInvalid']) {
  test(`${scenario}: retain initial results, and retry only the recheck stage`, async t => {
    t.mock.timers.enable({ apis: ['setTimeout'] })
    const fixture = createEmbedAuditPreview(scenario, labels)
    try {
      const initial = fixture.preview.context.value.audit_result
      assert.equal(initial.status, 'completed')
      assert.equal(initial.overall_score, 96)
      assert.equal(initial.recommendation, 'approve')
      assert.equal(initial.ai_recheck.status, 'failed')
      assert.deepEqual(JSON.parse(JSON.stringify(initial.rule_results)), JSON.parse(JSON.stringify(initial.ai_recheck.original_result.rule_results)))
      const updates = []
      const result = await advance(t, fixture.preview.retryRecheck(update => updates.push(update), initial, false))
      assert.equal(updates[0].status, 'rechecking')
      assert.deepEqual(updates[0].progress_steps.filter(step => step.done).map(step => step.key), ['pending', 'assembling', 'reasoning', 'extracting'])
      assert.equal(updates[0].progress_steps.find(step => step.current).key, 'rechecking')
      assert.ok(updates.slice(1).every(update => Object.keys(update).join() === 'recheck_output'))
      assert.equal(result.id, initial.id)
      assert.equal(result.ai_recheck.status, 'completed')
      assert.equal(result.ai_recheck.reuse_original, true)
      assert.equal(result.overall_score, 96)
      assert.equal(fixture.preview.context.value.audit_result.id, initial.id)
    } finally { fixture.dispose() }
  })
}

test('full playback streams analysis, extraction and recheck before publishing the fallback result', async t => {
  t.mock.timers.enable({ apis: ['setTimeout'] })
  const fixture = createEmbedAuditPreview('recheckJsonFailed', labels, { autoStart: true })
  try {
    assert.equal(fixture.preview.context.value.audit_result, null)
    const updates = []
    const result = await advance(t, fixture.preview.execute(update => {
      assert.equal(fixture.preview.context.value.audit_result, null)
      updates.push(update)
    }, false, false))
    assert.deepEqual(updates.filter(update => update.status).map(update => update.status), ['pending', 'assembling', 'reasoning', 'extracting', 'rechecking'])
    assert.ok(updates.some(update => update.ai_reasoning?.length > 0 && update.ai_reasoning.length < result.ai_reasoning.length))
    assert.ok(updates.findIndex(update => update.deep_thinking?.length > 0) < updates.findIndex(update => update.ai_reasoning?.length > 0))
    assert.ok(updates.some(update => update.structured_output?.length > 0))
    assert.ok(updates.some(update => update.recheck_output?.length > 0))
    assert.equal(result.status, 'completed')
    assert.equal(result.ai_recheck.status, 'failed')
    assert.equal(result.overall_score, 96)
  } finally { fixture.dispose() }
})

test('another failed recheck retains the initial data and failure details', async t => {
  t.mock.timers.enable({ apis: ['setTimeout'] })
  const fixture = createEmbedAuditPreview('retryFailed', labels)
  try {
    const initial = fixture.preview.context.value.audit_result
    const result = await advance(t, fixture.preview.retryRecheck(() => {}, initial, false))
    assert.equal(result.status, 'completed')
    assert.equal(result.ai_recheck.status, 'failed')
    assert.equal(result.ai_recheck.error, labels.recheckErrors.json)
    assert.equal(result.overall_score, initial.overall_score)
    assert.deepEqual(result.rule_results, initial.rule_results)
  } finally { fixture.dispose() }
})

test('disabled recheck uses two stages; changed recheck updates the final data and preserves the initial result', async t => {
  t.mock.timers.enable({ apis: ['setTimeout'] })
  const fixture = createEmbedAuditPreview('noRecheck', labels)
  try {
    const statuses = []
    const result = await advance(t, fixture.preview.execute(update => { if (update.status) statuses.push(update.status) }, false, false))
    assert.equal(result.ai_recheck, undefined)
    assert.ok(!statuses.includes('rechecking'))
  } finally { fixture.dispose() }
  const changed = createEmbedAuditPreview('recheckChanged', labels)
  try {
    const result = changed.preview.context.value.audit_result
    assert.equal(result.ai_recheck.changed, true)
    assert.equal(result.ai_recheck.original_result.overall_score, 96)
    assert.equal(result.overall_score, 76)
    assert.equal(result.recommendation, 'review')
    assert.equal(result.ai_recheck.final_result.overall_score, 76)
  } finally { changed.dispose() }
})

test('disposing a playing preview stops all pending work without publishing a result', async t => {
  t.mock.timers.enable({ apis: ['setTimeout'] })
  const fixture = createEmbedAuditPreview('approve', labels, { autoStart: true })
  const execution = fixture.preview.execute(() => {}, false, false)
  fixture.dispose()
  await assert.rejects(execution, { name: 'AbortError' })
  assert.equal(fixture.preview.context.value.audit_result, null)
})
