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
const pcFactory = vm.runInNewContext(ts.transpileModule(
  extract('const getEmbedScriptConfig =', 'const getEmbedMobileScriptConfig =')
  + extract('const buildEmbedNotifyScript =', 'const buildEmbedMobileNotifyScript =')
  + '\nbuildEmbedNotifyScript',
  { compilerOptions: { target: ts.ScriptTarget.ES2022 } },
).outputText, {
  t: key => key,
  embedOrigin: { value: 'https://aura.example.com' },
  embedAuditUrl: { value: 'https://aura.example.com/embed/audit' },
  embedSummaryUrl: { value: 'https://aura.example.com/embed/summary' },
})

function addListener(map, type, fn) {
  if (!map[type]) map[type] = []
  map[type].push(fn)
}

// 执行实际导出的完整脚本，模拟 OA 容器并记录按钮文案、请求和点击去向。
function captureButtonText(state, html) {
  const accessibleLabel = String(html).match(/class="aura-status-sr [^"]*"[^>]*>([^<]*)/)
  if (accessibleLabel) {
    state.text = accessibleLabel[1]
    return
  }
  const label = String(html).match(/class="aura-status-text"[^>]*>([^<]*)/)
  if (!label) return
  const score = String(html).match(/<b>([^<]+)<\/b><small>([^<]*)<\/small>/)
  state.text = score ? `${label[1]} (${score[1]}${score[2]})` : label[1]
}

function createButtonStub(state, id = '') {
  const btn = {
    id,
    isConnected: true,
    focus() {},
    style: { width: '', transition: '', setProperty() {} },
    classList: { add() {}, remove() {}, contains: name => name === 'aura-status-button' },
    attributes: {},
    parentNode: null,
    offsetWidth: 120,
    innerHTML: '',
    setAttribute(k, v) { this.attributes[k] = v; if (k === 'id') this.id = v },
    getAttribute(k) { return this.attributes[k] },
    querySelector() { return null },
    getBoundingClientRect() { return { left: 16, top: 640, width: 80, height: 44, right: 96, bottom: 684 } },
    addEventListener() {},
    insertBefore() {},
  }
  Object.defineProperty(btn, 'innerHTML', {
    get() { return this._html || '' },
    set(value) {
      this._html = value
      captureButtonText(state, value)
    },
  })
  return btn
}

async function run(source, payload, httpOK = true, device = {}) {
  const state = {
    requests: [], requestInits: [], dialogs: [], dialogOpts: [], messages: [], text: '', click: null, elements: [], styles: [], queries: [], appended: [],
    windowListeners: {}, docListeners: {}, timeouts: [], intervals: [], timerId: 0,
  }
  const nodesById = {}
  const groupEl = {
    firstChild: null,
    style: { opacity: '', transform: '', pointerEvents: '' },
    inert: false,
    attributes: {},
    getAttribute(k) { return this.attributes[k] },
    setAttribute(k, v) { this.attributes[k] = v },
    addEventListener(type, fn) {
      if (type === 'click') {
        state.click = (id) => fn({ target: nodesById[id] || groupEl.firstChild || createButtonStub(state, 'auraMobileEmbedBtn') })
      }
    },
    insertBefore(node) { node.parentNode = this; this.firstChild = node },
    appendChild(node) { node.parentNode = this; this.firstChild = this.firstChild || node },
  }
  const element = {
    length: 1, ready: fn => fn(), html: () => element, append: html => { state.appended.push(html); return element },
    find: () => element, text: value => { state.text = value; return element },
    off: () => element, on: (event, fn) => { state.click = fn; return element },
    children: () => [groupEl],
  }
  const document = {
    visibilityState: 'visible',
    activeElement: null,
    getElementById: id => nodesById[id] || null,
    head: { appendChild: node => state.styles.push(node) },
    documentElement: { style: { overflow: 'auto' } },
    body: {
      style: { overflow: 'scroll' }, children: [groupEl],
      appendChild(node) { state.elements.push(node); this.children.push(node); node.parentNode = this },
    },
    addEventListener: (type, fn) => addListener(state.docListeners, type, fn),
    removeEventListener(type, fn) { state.docListeners[type] = (state.docListeners[type] || []).filter(item => item !== fn) },
    createElement: tag => {
      const node = {
        tag, tagName: tag.toUpperCase(), style: {}, children: [], attributes: {}, isConnected: true,
        getBoundingClientRect() { return { left: 140, top: 50, width: 740, height: 600, right: 880, bottom: 650 } },
        querySelectorAll() { return [] },
        setAttribute(k, v) { this.attributes[k] = v },
        appendChild(child) { this.children.push(child); child.parentNode = this },
        focus() { document.activeElement = this },
        remove() { this.removed = true; this.isConnected = false; if(this.parentNode) this.parentNode.children = this.parentNode.children.filter(n => n !== this) },
      }
      if (tag === 'iframe') Object.defineProperty(node, 'src', {
        get() { return this._src },
        set(value) { this._src = value; state.dialogs.push(value) },
      })
      Object.defineProperty(node, 'firstChild', { get() { return this.children[0] || null } })
      Object.defineProperty(node, 'innerHTML', {
        get() { return this._html || '' },
        set(value) {
          this._html = value
          captureButtonText(state, value)
          const idMatch = String(value).match(/\sid="([^"]+)"/)
          const btn = createButtonStub(state, idMatch ? idMatch[1] : '')
          if (btn.id) nodesById[btn.id] = btn
          this.children = [btn]
        },
      })
      return node
    },
  }
  const win = {
    weaJs: {
      showDialog: (url, opts) => {
        if (device.nativeThrows) throw new Error('OA unavailable')
        state.dialogs.push(url)
        state.dialogOpts.push(opts)
      },
    },
    addEventListener: (type, fn) => addListener(state.windowListeners, type, fn),
    removeEventListener(type, fn) { state.windowListeners[type] = (state.windowListeners[type] || []).filter(item => item !== fn) },
    matchMedia: () => ({ matches: !!device.reducedMotion }),
    open(url, target) { state.openedWindow = { url, target }; return {} },
  }
  if (device.noNative) delete win.weaJs
  state.group = groupEl
  state.document = document
  state.nodes = nodesById
  state.frames = new Map()
  if (device.motion) {
    win.requestAnimationFrame = fn => { const id = ++state.timerId; state.frames.set(id, fn); return id }
    win.cancelAnimationFrame = id => state.frames.delete(id)
  }
  state.advanceAnimation = time => { const pending = [...state.frames.values()]; state.frames.clear(); pending.forEach(fn => fn(time)) }
  vm.runInNewContext(source, {
    navigator: { userAgent: device.ua || 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_0 like Mac OS X)', maxTouchPoints: device.touch || 0 },
    document,
    getComputedStyle: () => ({ backgroundColor: '#f8fafc', borderRadius: '22px', overflowY: 'visible' }),
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
    window: win,
    fetch: async (url, init) => {
      state.requests.push(url)
      state.requestInits.push(init || {})
      return { ok: httpOK, status: httpOK ? 200 : 500, json: async () => payload }
    },
    console: { log() {}, warn() {} },
    setTimeout: (fn, ms) => {
      const id = ++state.timerId
      state.timeouts.push({ id, fn, ms })
      return id
    },
    clearTimeout: (id) => {
      state.timeouts = state.timeouts.filter(item => item.id !== id)
    },
    setInterval: (fn, ms) => {
      const id = ++state.timerId
      state.intervals.push({ id, fn, ms })
      return id
    },
    clearInterval: (id) => {
      state.intervals = state.intervals.filter(item => item.id !== id)
    },
  })
  await new Promise(resolve => setImmediate(resolve))
  await new Promise(resolve => setImmediate(resolve))
  const due = state.timeouts.filter(item => item.ms < 1000)
  state.timeouts = state.timeouts.filter(item => item.ms >= 1000)
  due.forEach(item => item.fn())
  await new Promise(resolve => setImmediate(resolve))
  return state
}

async function flush(state) {
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
      assert.equal(shown.dialogOpts.length, 0, 'desktop must not use the mobile OA dialog')
      const overlay = shown.elements[0], dialog = overlay.children[0], frame = dialog.children[1]
      assert.equal(dialog.className, 'aura-embed-dialog-panel')
      const css = shown.styles.map(node => node.textContent).join('')
      assert.match(css, /width:min\(740px,\s*94vw\)/)
      assert.match(css, /height:86vh/)
      assert.match(css, /background:#f8fafc/)
      assert.equal(new URL(frame.src).pathname, '/embed/' + type)
      dialog.children[0].children[1].onclick()
      assert.equal(overlay.removed, true)
      await flush(shown)
      assert.ok(shown.requests.length >= 2, 'closing the desktop dialog must refresh the capsule')
      assert.equal(new URL(shown.requests.at(-1)).searchParams.get('prefer_cached'), 'true')
    })
    test(origin + ' ' + type + ': mounting behavior and iPad detection stay compatible', async () => {
      const inline = await run(script, { supported: true })
      assert.equal(inline.appended.length, 0)
      const floating = await run(script, { supported: true }, true, { noContainer: true })
      assert.match(floating.appended[0], /id="auraMobileEmbedFloatContainer"/)
      const styles = floating.styles.map(node => node.textContent).join('')
      assert.match(floating.appended[0], /aura-float--bottom-left/)
      assert.match(styles, /#auraMobileEmbedFloatContainer\{position:fixed/)
      assert.match(styles, /aura-float--bottom-left/)
      assert.match(styles, /aura-float--top-right/)
      assert.match(styles, /safe-area-inset-bottom/)
      assert.match(styles, /safe-area-inset-top/)
      assert.match(script, /var FLOAT_POSITION_MOBILE = 'bottom-left'/)
      assert.match(script, /var FLOAT_POSITION_DESKTOP = 'bottom-left'/)
      const mobileTopRight = await run(script.replace("var FLOAT_POSITION_MOBILE = 'bottom-left'", "var FLOAT_POSITION_MOBILE = 'top-right'"), { supported: true }, true, { noContainer: true })
      assert.match(mobileTopRight.appended[0], /aura-float--top-right/)
      const desktopCfg = script
        .replace('var SHOW_ON_DESKTOP = false', 'var SHOW_ON_DESKTOP = true')
        .replace("var FLOAT_POSITION_DESKTOP = 'bottom-left'", "var FLOAT_POSITION_DESKTOP = 'top-right'")
      const desktopTopRight = await run(desktopCfg, { supported: true }, true, { ua: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)', noContainer: true })
      assert.match(desktopTopRight.appended[0], /aura-float--top-right/)
      assert.match(styles, /auraButtonEnter/)
      assert.match(styles, /auraIconEnter/)
      assert.match(styles, /auraTextEnter/)
      assert.match(styles, /auraStatusBreath/)
      assert.match(styles, /auraResultHalo/)
      assert.doesNotMatch(styles, /auraBorderBreath/)
      assert.doesNotMatch(styles, /:hover\{[^}]*border-color:var\(--aura-status-color\)/)
      assert.match(styles, /--aura-status-surface/)
      assert.match(script, /aura-running-star/)
      assert.match(script, /auraStarSpin/)
      assert.doesNotMatch(script, /aura-running-ring|aura-running-core/)
      assert.match(styles, /aura-is-dragging/)
      assert.match(styles, /aura-is-compact/)
      assert.match(styles, /aura-embed-tooltip/)
      assert.match(script, /setupDraggableFloatContainer/)
      assert.match(script, /dragSession/)
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
        assert.equal(state.dialogOpts.length, 1, 'mobile must use the Fanwei native dialog')
        assert.equal(state.dialogOpts[0].moduleName, 'workflow')
        assert.equal(state.dialogOpts[0].style.width, '100%')
        assert.equal(state.dialogOpts[0].style.height, '100%')
        assert.equal(state.elements.length, 0, 'mobile must not create an iframe overlay')
        assert.equal(state.group.style.opacity, '')
        assert.equal(state.group.inert, false)
      })
    }
    test(`${origin} ${type}: configuration absence preserves the business explanation`, async () => {
      const state = await run(script, { supported: false, reason: 'no_config', message: '此流程尚未配置 AI ' + label })
      assert.equal(state.text, '未开启' + label)
      state.click()
      assert.equal(state.messages[0], '此流程尚未配置 AI ' + label)
      assert.equal(state.dialogs.length, 0)
    })
    test(`${origin} ${type}: uses the request belong user as the current operator`, async () => {
      const state = await run(script, complete)
      const request = new URL(state.requests[0])
      assert.equal(request.searchParams.get('oa_user_id'), '23')
    })
    test(`${origin} ${type}: pending auto-run and running jobs remain clickable`, async () => {
      for (const fields of [{ ['should_auto_' + type]: true }, { running_job_id: 'job-1' }]) {
        const state = await run(script, { supported: true, ...fields })
        state.click()
        assert.equal(state.dialogs.length, 1)
        assert.equal(state.requests.length, 1, 'status button must not start an AI job')
        assert.ok(state.intervals.length >= 1, 'opening details should watch for a later result')
      }
    })
    test(`${origin} ${type}: AUTO_RUN_BEFORE_OPEN starts a background job without opening`, async () => {
      const enabled = script.replace('var AUTO_RUN_BEFORE_OPEN = false', 'var AUTO_RUN_BEFORE_OPEN = true')
      const skipped = await run(enabled, { supported: true })
      assert.ok(!skipped.requests.some(url => String(url).includes('/execute')))
      const state = await run(enabled, { supported: true, ['should_auto_' + type]: true })
      await flush(state)
      const executeUrl = state.requests.find(url => String(url).includes('/execute'))
      assert.ok(executeUrl, 'form load must POST execute when AUTO_RUN_BEFORE_OPEN is true')
      assert.equal(new URL(executeUrl).pathname, type === 'summary' ? '/api/embed/summary/execute' : '/api/embed/execute')
      const init = state.requestInits[state.requests.indexOf(executeUrl)]
      assert.equal(init.method, 'POST')
      const body = JSON.parse(init.body)
      assert.equal(body.process_id, '614309')
      assert.equal(body.trigger_detail, 'form_open')
      assert.equal(body.trigger_source, type === 'summary' ? 'summary_embed_auto' : 'embed_auto')
      assert.match(state.text, new RegExp(label + '分析中'))
      state.click()
      assert.equal(state.dialogs.length, 1)
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
    test(`${origin} ${type}: embed status message updates the capsule immediately`, async () => {
      const state = await run(script, { supported: true, ['should_auto_' + type]: true })
      assert.match(state.text, new RegExp('查看并生成' + label))
      const listeners = state.windowListeners.message || []
      assert.ok(listeners.length)
      listeners.forEach(fn => fn({
        origin: 'https://aura.example.com',
        data: {
          type: 'aura-oa-embed-status',
          embed_type: type,
          requestid: '614309',
          running: false,
          has_result: true,
          status: 'completed',
          recommendation: 'return',
          overall_score: 45,
        },
      }))
      assert.equal(state.text, type === 'summary' ? '查看流程总结' : '建议退回 (45分)')
    })
    test(`${origin} ${type}: returning to the form refreshes cached status`, async () => {
      const state = await run(script, complete)
      const before = state.requests.length
      ;(state.docListeners.visibilitychange || []).forEach(fn => fn())
      state.timeouts.filter(item => item.ms === 300).forEach(item => item.fn())
      await flush(state)
      assert.ok(state.requests.length > before)
      assert.equal(new URL(state.requests.at(-1)).searchParams.get('prefer_cached'), 'true')
    })
  }
}

test('export all: requests both audit and summary context and renders dual buttons', async () => {
  const allScript = factory('all', 'test-embed-token')
  const state = await run(allScript, { supported: true, has_audit: true, audit_result: { recommendation: 'approve', overall_score: 98 }, has_summary: true, summary_result: { status: 'completed' } })
  const paths = state.requests.map(r => new URL(r).pathname)
  assert.ok(paths.includes('/api/embed/context'))
  assert.ok(paths.includes('/api/embed/summary/context'))
})

test('export all: AUTO_RUN_BEFORE_OPEN prefetches both features', async () => {
  const allScript = factory('all', 'test-embed-token').replace('var AUTO_RUN_BEFORE_OPEN = false', 'var AUTO_RUN_BEFORE_OPEN = true')
  const state = await run(allScript, { supported: true, should_auto_audit: true, should_auto_summary: true })
  await flush(state)
  const paths = state.requests.map(r => new URL(r).pathname)
  assert.ok(paths.includes('/api/embed/execute'))
  assert.ok(paths.includes('/api/embed/summary/execute'))
  const executeInits = state.requestInits.filter(init => init.method === 'POST')
  assert.equal(executeInits.length, 2)
  executeInits.forEach(init => {
    assert.equal(JSON.parse(init.body).trigger_detail, 'form_open')
  })
})

for (const [origin, source] of [
  ['export', pcFactory('all', 'test-embed-token')],
  ['template', fs.readFileSync(new URL('../../docs/oa-configurations/assets/aura-embed-notify.js', import.meta.url), 'utf8')],
]) {
  test(`${origin} pc: passes the request belong user to iframe context and operation events`, async () => {
    const state = { listeners: {}, checks: {}, posted: [], requests: [] }
    const iframe = {
      contentWindow: { postMessage: (payload, target) => state.posted.push({ payload, target }) },
      addEventListener() {},
    }
    vm.runInNewContext(source, {
      document: { getElementById: () => iframe },
      window: { addEventListener: (type, fn) => { state.listeners[type] = fn } },
      WfForm: {
        getBaseInfo: () => ({ requestid: '2110118', workflowid: '815', f_weaver_belongto_userid: '2480' }),
        registerCheckEvent: (type, fn) => { state.checks[type] = fn },
        OPER_SAVE: 'save', OPER_SUBMIT: 'submit',
      },
      jQuery: () => ({ ready: fn => fn() }),
      fetch: async (url, options) => { state.requests.push({ url, options }); return { ok: true } },
      console: { log() {}, warn() {} },
      setTimeout: () => 1,
      clearTimeout() {},
      setInterval: () => 1,
      clearInterval() {},
    })

    const direct = { postMessage: (payload, target) => state.posted.push({ payload, target }) }
    state.listeners.message({
      origin: 'https://aura.example.com',
      data: { type: 'aura-oa-request-requestid' },
      source: direct,
    })
    assert.equal(state.posted.at(-1).payload.oa_current_user_id, '2480')

    await new Promise(resolve => state.checks.submit(resolve))
    const body = new URLSearchParams(state.requests.at(-1).options.body)
    assert.equal(body.get('oa_current_user_id'), '2480')
    assert.equal(body.get('action'), 'submit_requested')
  })
}

for (const origin of ['export', 'template']) {
  for (const type of ['audit', 'summary', 'all']) {
    const source = origin === 'export' ? factory(type, 'test-token') : staticSource.replace("var EMBED_TYPE = 'audit'", `var EMBED_TYPE = '${type}'`)
    const payload = { supported: true, has_audit: true, audit_result: { recommendation: 'approve', overall_score: 95 }, has_summary: true, summary_result: { status: 'completed' } }
    const trigger = type === 'all' ? 'auraMobileEmbedSummaryBtn' : 'auraMobileEmbedBtn'
    const desktopSource = source.replace('var SHOW_ON_DESKTOP = false', 'var SHOW_ON_DESKTOP = true')
    const desktop = { ua: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)', motion: true }

    test(`${origin} ${type}: animated close restores OA scrolling and buttons exactly once`, async () => {
      const state = await run(desktopSource, payload, true, desktop)
      state.click(trigger)
      const overlay = state.elements.find(node => node.id === 'auraDesktopEmbedDialog')
      const close = overlay.children[0].children[0].children[1]
      assert.equal(state.document.body.style.overflow, 'hidden')
      assert.equal(state.group.inert, true)
      assert.equal(state.elements.filter(node => node.className === 'aura-embed-genie-layer').length, 1)
      state.advanceAnimation(0)
      state.advanceAnimation(300)
      state.advanceAnimation(650)
      assert.equal(state.group.style.opacity, '0')
      assert.equal(state.frames.size, 0)
      const before = state.requests.length
      state.click(trigger)
      assert.equal(state.dialogs.length, 1, 'repeated clicks must not load a second iframe')
      close.onclick()
      close.onclick()
      state.advanceAnimation(700)
      state.advanceAnimation(1300)
      await flush(state)
      assert.equal(overlay.removed, true)
      assert.equal(state.document.body.style.overflow, 'scroll')
      assert.equal(state.document.documentElement.style.overflow, 'auto')
      assert.equal(state.group.style.opacity, '')
      assert.equal(state.group.style.transform, '')
      assert.equal(state.group.inert, false)
      assert.equal(state.frames.size, 0)
      assert.equal(state.requests.length, before + 1)
      assert.ok(state.elements.filter(node => node.className === 'aura-embed-genie-layer').every(node => node.removed))
      assert.equal(state.windowListeners.resize.length, 0)
      assert.equal(state.docListeners.keydown.length, 0)
    })

    test(`${origin} ${type}: close during entry and page exit release animation resources`, async () => {
      const state = await run(desktopSource, payload, true, desktop)
      state.click(trigger)
      state.advanceAnimation(0)
      state.advanceAnimation(150)
      const overlay = state.elements.find(node => node.id === 'auraDesktopEmbedDialog')
      overlay.onclick({ target: overlay })
      state.advanceAnimation(200)
      state.advanceAnimation(800)
      assert.equal(overlay.removed, true)
      assert.equal(state.group.style.opacity, '')
      state.click(trigger)
      const before = state.requests.length
      ;[...state.windowListeners.pagehide].forEach(fn => fn())
      await flush(state)
      assert.equal(state.frames.size, 0)
      assert.equal(state.group.inert, false)
      assert.equal(state.document.body.style.overflow, 'scroll')
      assert.equal(state.requests.length, before, 'leaving OA must not start another refresh')
    })

    test(`${origin} ${type}: mobile native dialog keeps buttons visible on callback and failure without creating an iframe`, async () => {
      const state = await run(source, payload)
      state.click(trigger)
      assert.equal(state.dialogOpts.length, 1)
      assert.equal(state.group.style.opacity, '')
      const before = state.requests.length
      state.dialogOpts[0].callback()
      state.dialogOpts[0].callback()
      await flush(state)
      assert.equal(state.group.style.opacity, '')
      assert.equal(state.group.inert, false)
      assert.equal(state.requests.length, before + 1)
      const failed = await run(source, payload, true, { nativeThrows: true })
      failed.click(trigger)
      assert.equal(failed.dialogOpts.length, 0)
      const overlay = failed.elements.find(node => node.id === 'auraDesktopEmbedDialog')
      assert.equal(overlay, undefined)
      assert.equal(failed.messages[0], '详情打开失败，请稍后重试')
      assert.equal(failed.document.body.style.overflow, 'scroll')
      assert.equal(failed.group.style.opacity, '')
      assert.equal(failed.group.inert, false)
      failed.click(trigger)
      assert.equal(failed.messages.length, 2, 'failed native opening must release the click lock')
    })

    test(`${origin} ${type}: native cancel only refreshes once and never changes button visibility`, async () => {
      const state = await run(source, payload)
      state.click(trigger)
      const first = state.dialogOpts[0]
      const before = state.requests.length
      first.onCancel()
      first.callback()
      await flush(state)
      assert.equal(state.group.style.opacity, '')
      assert.equal(state.group.style.transform, '')
      assert.equal(state.group.style.pointerEvents, '')
      assert.equal(state.group.inert, false)
      assert.equal(state.requests.length, before + 1)
      state.click(trigger)
      assert.equal(state.dialogOpts.length, 2)
      first.onCancel()
      assert.equal(state.group.style.opacity, '')
      state.dialogOpts[1].onCancel()
      assert.equal(state.group.inert, false)
    })

    test(`${origin} ${type}: mobile buttons stay visible and reopen even without close callbacks or route events`, async () => {
      const state = await run(source, payload)
      const before = state.requests.length
      state.click(trigger)
      // 模拟泛微右上角 X 静默关闭：不调用 callback、onCancel，也不派发路由事件。
      assert.equal(state.group.style.opacity, '')
      assert.equal(state.group.style.transform, '')
      assert.equal(state.group.style.pointerEvents, '')
      assert.equal(state.group.inert, false)
      state.click(trigger)
      assert.equal(state.dialogOpts.length, 2, 'missing native close callbacks must never lock the entry')
      assert.equal(state.group.style.opacity, '')
      assert.equal(state.group.inert, false)
      assert.equal(state.requests.length, before, 'opening must not pretend that the previous dialog emitted a close event')
      assert.equal((state.windowListeners.hashchange || []).length, 0)
      assert.equal((state.windowListeners.popstate || []).length, 0)
      assert.equal((state.windowListeners.pagehide || []).length, 0)
    })

    test(`${origin} ${type}: reduced motion skips layers and resize settles an in-flight surface`, async () => {
      const state = await run(desktopSource, payload, true, { ...desktop, reducedMotion: true })
      state.click(trigger)
      assert.equal(state.frames.size, 0)
      assert.equal(state.elements.filter(node => node.className === 'aura-embed-genie-layer').length, 0)
      assert.equal(state.group.style.opacity, '0')
      const resizing = await run(desktopSource, payload, true, desktop)
      resizing.click(trigger)
      resizing.advanceAnimation(0)
      ;[...resizing.windowListeners.resize].forEach(fn => fn())
      assert.equal(resizing.frames.size, 0)
      assert.equal(resizing.group.style.opacity, '0')
      const overlay = resizing.elements.find(node => node.id === 'auraDesktopEmbedDialog')
      overlay.children[0].children[0].children[1].onclick()
      ;[...resizing.windowListeners.resize].forEach(fn => fn())
      assert.equal(overlay.removed, true)
      assert.equal(resizing.group.inert, false)
    })

    test(`${origin} ${type}: mobile without Fanwei API keeps the original new-window fallback`, async () => {
      const state = await run(source, payload, true, { noNative: true })
      state.click(trigger)
      assert.equal(state.openedWindow.target, '_blank')
      assert.equal(new URL(state.openedWindow.url).pathname, type === 'audit' ? '/embed/audit' : '/embed/summary')
      assert.equal(state.elements.length, 0)
      assert.equal(state.group.style.opacity, '')
      assert.equal(state.group.inert, false)
      assert.equal(state.document.body.style.overflow, 'scroll')
    })

    test(`${origin} ${type}: device detection keeps mobile native even when desktop buttons are enabled`, async () => {
      for (const device of [
        { ua: 'Mozilla/5.0 (Linux; Android 13)' },
        { ua: 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)' },
        { ua: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15)', touch: 5 },
      ]) {
        const state = await run(desktopSource, payload, true, device)
        state.click(trigger)
        assert.equal(state.dialogOpts.length, 1)
        assert.equal(state.elements.length, 0)
        assert.equal(state.document.documentElement.style.overflow, 'auto')
        const before = state.requests.length
        state.dialogOpts[0].callback()
        await flush(state)
        assert.equal(state.group.inert, false)
        assert.equal(state.group.style.opacity, '')
        assert.equal(state.requests.length, before + 1)
      }
      const state = await run(desktopSource, payload, true, { ua: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)', touch: 10 })
      state.click(trigger)
      assert.equal(state.dialogOpts.length, 0, 'Windows touch screens remain desktop')
      assert.ok(state.elements.some(node => node.id === 'auraDesktopEmbedDialog'))
    })

    test(`${origin} ${type}: exported runtime remains identical to the standalone script`, () => {
      assert.equal(source.slice(source.indexOf('  var MSG_STATUS =')).trim(), staticSource.slice(staticSource.indexOf('  var MSG_STATUS =')).trim())
    })
  }
}
