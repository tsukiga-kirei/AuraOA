import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import vm from 'node:vm'
import ts from 'typescript'

const tenantSource = fs.readFileSync(new URL('../pages/admin/system/tenants.vue', import.meta.url), 'utf8')
const staticSource = fs.readFileSync(new URL('../../docs/oa-configurations/assets/aura-embed-mobile-notify.js', import.meta.url), 'utf8')
const extract = (start, end) => tenantSource.slice(tenantSource.indexOf(start), tenantSource.indexOf(end))
const factory = vm.runInNewContext(ts.transpileModule(
  extract('const getEmbedMobileScriptConfig =', 'const buildEmbedNotifyScript =')
  + extract('const buildEmbedMobileNotifyScript =', 'const downloadTextFile =')
  + '\nbuildEmbedMobileNotifyScript',
  { compilerOptions: { target: ts.ScriptTarget.ES2022 } },
).outputText, {
  t: key => key,
  embedOrigin: { value: 'https://aura.example.com' },
  embedAuditUrl: { value: 'https://aura.example.com/embed/audit' },
  embedSummaryUrl: { value: 'https://aura.example.com/embed/summary' },
})

// 执行实际导出的完整脚本，模拟 OA 容器并记录按钮文案、请求和点击去向。
async function run(source, payload, httpOK = true, device = {}) {
  const state = { requests: [], dialogs: [], messages: [], text: '', click: null, elements: [], queries: [], appended: [] }
  const element = {
    length: 1, ready: fn => fn(), html: () => element, append: html => { state.appended.push(html); return element },
    find: () => element, text: value => { state.text = value; return element },
    off: () => element, on: (event, fn) => { state.click = fn; return element },
  }
  vm.runInNewContext(source, {
    navigator: { userAgent: device.ua || 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_0 like Mac OS X)', maxTouchPoints: device.touch || 0 },
    document: {
      activeElement: null, getElementById: () => null,
      body: { appendChild: node => state.elements.push(node) },
      addEventListener() {}, removeEventListener() {},
      createElement: tag => ({ tag, style: {}, children: [], attributes: {},
        setAttribute(k,v) { this.attributes[k] = v },
        appendChild(node) { this.children.push(node) }, focus() {}, remove() { this.removed = true },
      }),
    },
    jQuery: selector => {
      state.queries.push(selector);
      if (selector === '#auraMobileEmbedFloatContainer' || (selector === '#getMyBt' && device.noContainer)) return { ...element, length: 0 };
      return element;
    },
    WfForm: {
      getBaseInfo: () => ({ requestid: '614309', f_weaver_belongto_userid: '23' }),
      showMessage: value => state.messages.push(value),
      registerCheckEvent: () => {}, OPER_SAVE: 'save', OPER_SUBMIT: 'submit',
    },
    window: { weaJs: { showDialog: url => state.dialogs.push(url) } },
    fetch: async url => {
      state.requests.push(url)
      return { ok: httpOK, status: httpOK ? 200 : 500, json: async () => payload }
    },
    console: { log() {}, warn() {} }, setTimeout, clearTimeout,
  })
  await new Promise(resolve => setImmediate(resolve))
  return state
}

for (const origin of ['export', 'template']) {
  for (const type of ['audit', 'summary']) {
    const script = origin === 'export'
      ? factory(type, 'test-embed-token')
      : staticSource.replace("var EMBED_TYPE = 'audit'", `var EMBED_TYPE = '${type}'`)
    test(origin + ' ' + type + ': desktop is opt-in and uses a responsive dialog', async () => {
      const device = { ua: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)' }
      const hidden = await run(script, { supported: true }, true, device)
      assert.equal(hidden.requests.length, 0)
      assert.equal(hidden.queries.length, 0)
      const shown = await run(script.replace('var SHOW_ON_DESKTOP = false', 'var SHOW_ON_DESKTOP = true'), { supported: true }, true, device)
      shown.click()
      assert.equal(shown.dialogs.length, 0, 'desktop must not use the mobile OA dialog')
      const overlay = shown.elements[0], dialog = overlay.children[0], frame = dialog.children[1]
      assert.match(dialog.style.cssText, /height:85vh/)
      assert.match(dialog.style.cssText, /max-width:100%/)
      assert.equal(new URL(frame.src).pathname, '/embed/' + type)
      dialog.children[0].children[1].onclick()
      assert.equal(overlay.removed, true)
    })
    test(origin + ' ' + type + ': mounting behavior and iPad detection stay compatible', async () => {
      const inline = await run(script, { supported: true })
      assert.equal(inline.appended.length, 0)
      const floating = await run(script, { supported: true }, true, { noContainer: true })
      assert.match(floating.appended[0], /position:fixed;bottom:16px;left:16px/)
      const ipad = await run(script, { supported: true }, true, { ua: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15)', touch: 5 })
      assert.equal(ipad.requests.length, 1)
    })
    const label = type === 'summary' ? '总结' : '审核'
    const result = type === 'summary' ? { status: 'completed', blocks: [] } : { recommendation: 'approve', overall_score: 95 }
    const complete = { supported: true, ['has_' + type]: true, [type + '_result']: result }

    for (const wrapped of [false, true]) {
      test(`${origin} ${type}: accepts ${wrapped ? 'Go envelope' : 'Nuxt data'} and opens the matching feature`, async () => {
        const state = await run(script, wrapped ? { code: 0, data: complete } : complete)
        const request = new URL(state.requests[0])
        assert.equal(request.pathname, type === 'summary' ? '/api/embed/summary/context' : '/api/embed/context')
        assert.equal(request.searchParams.get('requestid'), '614309')
        assert.equal(request.searchParams.get('oa_user_id'), '23')
        assert.equal(state.text, type === 'summary' ? '查看流程总结' : '审核通过 (95分)')
        state.click()
        assert.equal(new URL(state.dialogs[0]).pathname, '/embed/' + type)
      })
    }
    test(`${origin} ${type}: configuration absence preserves the business explanation`, async () => {
      const state = await run(script, { supported: false, reason: 'no_config', message: '此流程尚未配置 AI ' + label })
      assert.equal(state.text, '未开启' + label)
      state.click()
      assert.equal(state.messages[0], '此流程尚未配置 AI ' + label)
      assert.equal(state.dialogs.length, 0)
    })
    test(`${origin} ${type}: pending auto-run and running jobs remain clickable`, async () => {
      for (const fields of [{ ['should_auto_' + type]: true }, { running_job_id: 'job-1' }]) {
        const state = await run(script, { supported: true, ...fields })
        state.click()
        assert.equal(state.dialogs.length, 1)
        assert.equal(state.requests.length, 1, 'status button must not start an AI job')
      }
    })
    test(`${origin} ${type}: errors and malformed payloads do not appear successful`, async () => {
      for (const payload of [null, {}, { code: 403, data: complete }, { code: 0, data: null }]) {
        const state = await run(script, payload)
        assert.equal(state.text, '加载异常')
      }
      const state = await run(script, { supported: true, ['has_' + type]: true, [type + '_result']: { status: 'failed' } })
      assert.equal(state.text, label + '异常')
      state.click()
      assert.equal(state.dialogs.length, 1)
      const network = await run(script, null, false)
      network.click()
      assert.equal(network.dialogs.length, 1)
    })
  }
}
