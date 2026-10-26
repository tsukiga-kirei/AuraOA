import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import ts from 'typescript'

const module = { exports: {} }
const source = fs.readFileSync(new URL('../utils/configPublish.ts', import.meta.url), 'utf8')
new Function('module', 'exports', ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS } }).outputText)(module, module.exports)
const { configFormFingerprint, publishAfterSavingChanges } = module.exports

test('changed content waits for successful save before publishing', async () => {
  const events = []
  let release
  const pending = publishAfterSavingChanges(true, async () => {
    events.push('save')
    await new Promise(resolve => { release = resolve })
    events.push('saved')
    return true
  }, async () => { events.push('publish'); return { version_no: 4 } })
  assert.deepEqual(events, ['save'])
  release()
  assert.deepEqual(await pending, { version_no: 4 })
  assert.deepEqual(events, ['save', 'saved', 'publish'])
})

test('failed or cancelled save never publishes; rejected save propagates', async () => {
  let published = false
  const publish = async () => { published = true }
  assert.equal(await publishAfterSavingChanges(true, async () => false, publish), undefined)
  await assert.rejects(publishAfterSavingChanges(true, async () => { throw new Error('save failed') }, publish), /save failed/)
  assert.equal(published, false)
})

test('unchanged content skips saving', async () => {
  let saved = false
  assert.equal(await publishAfterSavingChanges(false, async () => { saved = true }, async () => 'existing'), 'existing')
  assert.equal(saved, false)
})

test('form comparison ignores metadata and key ordering, includes every module edit', () => {
  const initial = { id: '1', updated_at: 'before', process_type: 'project', ai_config: { recheck: { enabled: false }, enable_thinking: true }, summary_blocks: [{ title: 'Summary', enabled: true }], embed_config: { auto_audit_on_open: true } }
  const reordered = { ...initial, updated_at: 'after', ai_config: { enable_thinking: true, recheck: { enabled: false } } }
  for (const kind of ['audit', 'archive', 'summary']) {
    assert.equal(configFormFingerprint(kind, initial), configFormFingerprint(kind, reordered))
    const changed = structuredClone(initial)
    if (kind === 'summary') changed.summary_blocks[0].title = 'Updated'
    else changed.ai_config.recheck.enabled = true
    assert.notEqual(configFormFingerprint(kind, initial), configFormFingerprint(kind, changed))
  }
  const changed = { ...initial, embed_config: { auto_audit_on_open: false } }
  assert.notEqual(configFormFingerprint('audit', initial), configFormFingerprint('audit', changed))
})
