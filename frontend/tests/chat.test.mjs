import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import vm from 'node:vm'
import { createRequire } from 'node:module'
import ts from 'typescript'
import * as vue from 'vue'

const require = createRequire(import.meta.url)
async function sourceModule(path) {
  const source = fs.readFileSync(new URL(path, import.meta.url), 'utf8')
  let js = ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ESNext } }).outputText
  for (const dep of ['marked', 'sanitize-html']) js = js.replaceAll(`from '${dep}'`, `from '${import.meta.resolve(dep)}'`)
  return import(`data:text/javascript;base64,${Buffer.from(js).toString('base64')}`)
}
const { readSSE } = await sourceModule('../utils/sse.ts')
const { renderSafeMarkdown } = await sourceModule('../utils/markdown.ts')
const locales = await Promise.all(['zh-CN', 'en-US'].map(name => sourceModule(`../locales/${name}.ts`)))

test('Chat UI translation keys exist in both languages', () => {
  const paths = ['components/Chat/ChatNavigation.vue', 'components/Chat/ChatThread.vue', 'components/Chat/ToolActivity.vue', 'components/ChatAllocationPanel.vue', 'pages/chat.vue', 'pages/admin/tenant/agents.vue']
  for (const path of paths) {
    const source = fs.readFileSync(new URL('../' + path, import.meta.url), 'utf8')
    for (const [, key] of source.matchAll(/\bt\('([^']+)'/g)) {
      for (const locale of locales) assert.ok(locale.default[key], `${path}: missing ${key}`)
    }
  }
})

test('A slower previous search cannot replace a newer search or another tenant', async () => {
  const exports = {}
  const shared = new Map()
  const pending = []
  const activeRole = vue.ref({ tenant_id: 'tenant-a', role: 'business' })
  const js = ts.transpileModule(fs.readFileSync(new URL('../composables/useChatSession.ts', import.meta.url), 'utf8'), { compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS } }).outputText
  const scope = vue.effectScope()
  const session = scope.run(() => {
    vm.runInNewContext(js, { ...vue, exports, URLSearchParams, console,
      useState: (key, init) => { if (!shared.has(key)) shared.set(key, vue.ref(init())); return shared.get(key) },
      useAuth: () => ({ activeRole, currentUser: vue.ref({ username: 'tester' }), authFetch: () => new Promise(resolve => pending.push(resolve)) }),
      useI18n: () => ({ t: key => key }),
    })
    return exports.useChatSession()
  })
  const old = session.fetchSessions('旧')
  const fresh = session.fetchSessions('新')
  pending[1]({ items: [{ id: 'fresh', agent_code: 'agent-b' }], total: 1 }); await fresh
  pending[0]({ items: [{ id: 'old', agent_code: 'agent-a' }], total: 1 }); await old
  assert.equal(session.sessions.value[0].id, 'fresh')
  const other = session.fetchSessions('新')
  activeRole.value = { tenant_id: 'tenant-b', role: 'business' }
  pending[2]({ items: [{ id: 'tenant-a' }], total: 1 }); await other
  assert.equal(session.sessions.value.length, 0)
  scope.stop()
})
const wire = 'event: reasoning\r\ndata: {"content":"分析"}\r\n\r\nevent: delta\ndata: {"content":"你好"}\n\nevent: done\ndata: {"token_usage":{"total_tokens":12}}\n\n'
const fragmented = text => new ReadableStream({ start(controller) { for (const byte of new TextEncoder().encode(text)) controller.enqueue(new Uint8Array([byte])); controller.close() } })

test('SSE retains event names and Chinese UTF-8 across single-byte chunks', async () => {
  const events = []
  await readSSE(fragmented(wire), (event, data) => events.push([event, data]))
  assert.equal(events[0][0], 'reasoning'); assert.equal(events[0][1].content, '分析')
  assert.equal(events[1][1].content, '你好'); assert.equal(events[2][0], 'done')
})

test('Markdown preserves GFM and removes active HTML and unsafe links', () => {
  const html = renderSafeMarkdown('# 标题\n\n| 名称 | 金额 |\n| --- | --- |\n| 报销 | 120 |\n\n- [x] 完成\n\n<script>alert(1)</script><img src=x onerror=alert(1)><a href="javascript:alert(1)">点此</a><iframe srcdoc="bad"></iframe>\n\n[文档](https://example.org)')
  assert.match(html, /<h1>标题/); assert.match(html, /<table>/); assert.match(html, /type="checkbox"/)
  assert.doesNotMatch(html, /<script|<iframe|onerror|javascript:/i)
  assert.match(html, /rel="noopener noreferrer"/)
})

test('Chat streaming uses authorized fetch and reactive tool/message state', async () => {
  const events = wire.replace('event: done', 'event: tool_start\ndata: {"tool_call_id":"call1","tool_code":"get_process","status":"running","ui_kind":"process"}\n\nevent: tool_result\ndata: {"tool_call_id":"call1","tool_code":"get_process","status":"success","payload":{"process_id":"42"}}\n\nevent: done')
  let requests = 0
  const exports = {}
  const source = fs.readFileSync(new URL('../composables/useChatStream.ts', import.meta.url), 'utf8')
  const js = ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS } }).outputText
  vm.runInNewContext(js, { ...vue, exports, crypto: globalThis.crypto, AbortController, TextDecoder, console, onBeforeUnmount() {},
    require: name => name === '~/utils/sse' ? { readSSE } : require(name),
    useAuth: () => ({ authStreamFetch: async (path, options) => { requests++; assert.match(path, /sessions\/test\/messages\/stream/); assert.equal(options.method, 'POST'); return { body: fragmented(events) } } }),
    useI18n: () => ({ t: key => key }), useChatSession: () => ({ sessions: vue.ref([]), currentDetail: vue.ref(null) }),
  })
  const stream = exports.useChatStream()
  const messages = vue.ref([])
  let updates = 0
  const stop = vue.watch(messages, () => updates++, { deep: true, flush: 'sync' })
  await stream.sendStreamMessage('test', '问题', messages)
  stop()
  assert.equal(requests, 1); assert.equal(messages.value[1].content, '你好')
  assert.equal(messages.value[1].reasoning_content, '分析')
  assert.equal(messages.value[1].tool_calls.length, 1)
  assert.equal(messages.value[1].tool_calls[0].payload.process_id, '42')
  assert.equal(messages.value[1].token_usage.total_tokens, 12)
  assert.equal(stream.streaming.value, false); assert.ok(updates > 5)
})
