import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import ts from 'typescript'

const transpile = source => ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS } }).outputText
const exports = {}
new Function('exports', transpile(fs.readFileSync(new URL('../utils/reviewProgress.ts', import.meta.url), 'utf8')))(exports)
const { isReviewRunning, mergeReviewProgress } = exports

test('polling preserves live text only until the saved reasoning becomes available', () => {
  const live = { id: 'job', status: 'reasoning', ai_reasoning: '旧尝试：规则核验', deep_thinking: '旧思考' }
  assert.equal(mergeReviewProgress(live, { id: 'job', status: 'reasoning', ai_reasoning: '' }).ai_reasoning, live.ai_reasoning)
  const saved = { id: 'job', status: 'extracting', ai_reasoning: '规则核验', deep_thinking: '思考' }
  assert.deepEqual(mergeReviewProgress(live, saved), saved)
  for (const status of ['completed', 'failed', 'cancelled']) {
    assert.equal(mergeReviewProgress(live, { id: 'job', status, ai_reasoning: '' }).ai_reasoning, '')
  }
  assert.equal(mergeReviewProgress(live, { id: 'new-job', status: 'pending', ai_reasoning: '' }).ai_reasoning, '')
})

for (const [path, selection] of [
  ['components/EmbedAuditPage.vue', 'process'],
  ['pages/dashboard.vue', 'process'],
  ['pages/archive.vue', { process_id: 'process' }],
]) {
  test(`${path}: replay, polling and retries never append another copy of reasoning`, () => {
    const source = fs.readFileSync(new URL(`../${path}`, import.meta.url), 'utf8')
    const start = source.indexOf('const startSSE = ')
    const end = source.indexOf('\n}\n', start) + 3
    const currentResult = { value: { id: 'job', status: 'reasoning', ai_reasoning: '' } }
    const eventSourceStream = { value: null }
    class EventSource {
      listeners = {}
      addEventListener(name, listener) { this.listeners[name] = listener }
      close() {}
    }
    const globals = {
      currentResult, eventSourceStream, EventSource, isReviewRunning,
      selectedProcess: { value: selection }, token: { value: '' }, localStorage: { getItem: () => '' },
      useRuntimeConfig: () => ({ public: { apiBase: '' } }),
      preview: null, process: { client: true }, streamJobId: { value: '' },
      disconnectStream: () => {}, useEmbedAuth: () => ({ appendEmbedTokenQuery: value => value }),
    }
    new Function(...Object.keys(globals), transpile(source.slice(start, end)) + '\nstartSSE("job", "process")')(...Object.values(globals))
    const stream = eventSourceStream.value
    const content = '## 规则核验\n规则1：通过\n总分：100'
    stream.onmessage({ data: content.slice(0, 8) })
    currentResult.value = mergeReviewProgress(currentResult.value, { id: 'job', status: 'extracting', ai_reasoning: content })
    stream.onmessage({ data: content })
    stream.onmessage({ data: content }) // 重连重放同一累计快照
    assert.equal(currentResult.value.ai_reasoning, content)
    currentResult.value.status = 'reasoning'
    stream.listeners.thinking({ data: '旧尝试的思考' })
    stream.onmessage({ data: '' }) // 切换模型先清空两侧
    stream.listeners.thinking({ data: '' })
    assert.equal(currentResult.value.ai_reasoning, '')
    assert.equal(currentResult.value.deep_thinking, '')
    stream.onmessage({ data: content })
    currentResult.value = mergeReviewProgress(currentResult.value, { id: 'job', status: 'completed', ai_reasoning: '最终分析', deep_thinking: '最终思考' })
    stream.onmessage({ data: content })
    stream.listeners.thinking({ data: '迟到的思考' })
    assert.equal(currentResult.value.ai_reasoning, '最终分析')
    assert.equal(currentResult.value.deep_thinking, '最终思考')
  })
}
