import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import vm from 'node:vm'
import ts from 'typescript'

const tenantSource = fs.readFileSync(new URL('../pages/admin/system/tenants.vue', import.meta.url), 'utf8')
const modernStaticSource = fs.readFileSync(new URL('../../docs/oa-configurations/assets/aura-embed-mobile-notify.js', import.meta.url), 'utf8')
const classicStyle = source => source.replaceAll("STYLE_MOBILE = 'modern'", "STYLE_MOBILE = 'classic'").replaceAll("STYLE_DESKTOP = 'modern'", "STYLE_DESKTOP = 'classic'")
const staticSource = classicStyle(modernStaticSource)
const extract = (start, end) => tenantSource.slice(tenantSource.indexOf(start), tenantSource.indexOf(end))
const modernFactory = vm.runInNewContext(ts.transpileModule(
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
const factory = (...args) => classicStyle(modernFactory(...args))
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
    style: { width: '', transition: '', setProperty() {}, removeProperty(name) { delete this[name] } },
    classList: { add() {}, remove() {}, toggle() {}, contains: name => name === 'aura-status-button' },
    attributes: {},
    parentNode: null,
    offsetWidth: 120,
    innerHTML: '',
    setAttribute(k, v) { this.attributes[k] = v; if (k === 'id') this.id = v },
    getAttribute(k) { return this.attributes[k] },
    removeAttribute(k) { delete this.attributes[k] },
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
    groupListeners: {}, windowListeners: {}, docListeners: {}, timeouts: [], intervals: [], timerId: 0,
  }
  const nodesById = {}
  const groupEl = {
    firstChild: null,
    style: { opacity: '', transform: '', pointerEvents: '' },
    inert: false,
    attributes: {},
    getAttribute(k) { return this.attributes[k] },
    setAttribute(k, v) { this.attributes[k] = v },
    querySelectorAll() { return Object.values(nodesById).filter(node => node.parentNode === groupEl) },
    addEventListener(type, fn) {
      addListener(state.groupListeners, type, fn)
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
          const outer = String(value).slice(0, String(value).indexOf('>'))
          for (const match of outer.matchAll(/([\w-]+)="([^"]*)"/g)) btn.setAttribute(match[1], match[2])
          btn.innerHTML = value
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
    matchMedia: query => ({ matches: query.includes('prefers-reduced-motion') ? !!device.reducedMotion : !!device.hover }),
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
  if (device.exposeButtons) source = source.replace('  var dragSession =', `  window.__buttons = {
    buildStatusButton: buildStatusButton, iconSvgMarkup: iconSvgMarkup, parseButtonLabel: parseButtonLabel,
    modernButtonCopyMarkup: modernButtonCopyMarkup, modernSecondaryLabel: modernSecondaryLabel,
    morphModernStatusButton: morphModernStatusButton, setModernButtonExpanded: setModernButtonExpanded,
    handleModernButtonClick: handleModernButtonClick, cancelModernButtonMotion: cancelModernButtonMotion,
    queueSecondaryButtonHover: queueSecondaryButtonHover, bindModernButtonFeedback: bindModernButtonFeedback,
    setupDraggableFloatContainer: setupDraggableFloatContainer
  };\n  var dragSession =`)
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
      const response = device.payloadByPath?.[new URL(url).pathname] ?? payload
      return { ok: httpOK, status: httpOK ? 200 : 500, json: async () => response }
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
  state.buttonsApi = win.__buttons
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

// 柔光分支执行同一份正式/导出脚本；经典回归保留独立配置。
for (const origin of ['export', 'template']) {
  const sourceFor = type => origin === 'export' ? modernFactory(type, 'test-token') : modernStaticSource.replace("var EMBED_TYPE = 'audit'", `var EMBED_TYPE = '${type}'`)
  const complete = { supported: true, has_audit: true, audit_result: { recommendation: 'approve', overall_score: 95 }, has_summary: true, summary_result: { status: 'completed' } }

  test(`${origin} modern: mobile and desktop choose styles independently with invalid fallback`, async () => {
    for (const [mobile, desktop, device, expected] of [
      ['modern', 'classic', {}, 'modern'],
      ['modern', 'classic', { ua: 'Windows NT 10.0' }, 'classic'],
      ['classic', 'modern', {}, 'classic'],
      ['classic', 'modern', { ua: 'Windows NT 10.0', touch: 10 }, 'modern'],
      ['modern', 'classic', { ua: 'Macintosh', touch: 5 }, 'modern'],
      ['unknown', 'modern', {}, 'classic'],
      ['modern', 'unknown', { ua: 'Windows NT 10.0' }, 'classic'],
    ]) {
      const source = sourceFor('audit').replace('SHOW_ON_DESKTOP = false', 'SHOW_ON_DESKTOP = true')
        .replace("STYLE_MOBILE = 'modern'", `STYLE_MOBILE = '${mobile}'`)
        .replace("STYLE_DESKTOP = 'modern'", `STYLE_DESKTOP = '${desktop}'`)
      const state = await run(source, complete, true, device)
      assert.equal(state.group.getAttribute('data-aura-button-style'), expected)
      assert.equal(state.nodes.auraMobileEmbedBtn.innerHTML.includes('aura-modern-glyph'), expected === 'modern')
      state.click()
      assert.equal(state.dialogs.length, 1)
    }
  })

  for (const type of ['audit', 'summary', 'all']) {
    const source = sourceFor(type)
    const ids = type === 'all' ? ['auraMobileEmbedAuditBtn', 'auraMobileEmbedSummaryBtn'] : ['auraMobileEmbedBtn']
    test(`${origin} modern ${type}: real results retain feature routing and mobile native protocol`, async () => {
      const state = await run(source, complete)
      assert.equal(state.group.getAttribute('data-aura-button-style'), 'modern')
      for (const id of ids) {
        const btn = state.nodes[id]
        assert.equal((btn.innerHTML.match(/<svg\b/g) || []).length, 1)
        assert.equal(btn.getAttribute('data-aura-disclosure'), 'false')
        assert.match(btn.innerHTML, id.includes('Summary') || type === 'summary' ? />总结</ : /aura-status-caption.*审核通过/)
        state.click(id)
        const feature = id.includes('Summary') || type === 'summary' ? 'summary' : 'audit'
        assert.equal(new URL(state.dialogs.at(-1)).pathname, '/embed/' + feature)
        assert.equal(state.dialogOpts.at(-1).moduleName, 'workflow')
        assert.equal(state.group.style.opacity, '')
        assert.equal(state.group.getAttribute('data-aura-effects-paused'), undefined)
      }
      assert.equal(state.elements.length, 0)
    })

    test(`${origin} modern ${type}: mobile pending entry opens on the first tap without executing a task`, async () => {
      const state = await run(source, { supported: true, should_auto_audit: true, should_auto_summary: true })
      const before = state.requests.length
      for (const id of ids) {
        assert.equal(state.nodes[id].getAttribute('data-aura-expanded'), 'false')
        state.click(id)
        assert.equal(state.nodes[id].getAttribute('data-aura-expanded'), 'false')
        assert.equal(state.requests.length, before)
        assert.equal(state.dialogs.length, ids.indexOf(id) + 1)
      }
      assert.ok(state.requestInits.every(init => init.method !== 'POST'))
    })

    test(`${origin} modern ${type}: invalid context and unsupported states only disclose information`, async () => {
      for (const [payload, ok] of [[{ supported: false }, true], [{}, true]]) {
        const state = await run(source, payload, ok)
        const before = state.requests.length
        for (const id of ids) {
          state.click(id); state.click(id)
          assert.equal(state.nodes[id].getAttribute('data-aura-expanded'), 'true')
        }
        assert.equal(state.dialogs.length, 0)
        assert.equal(state.requests.length, before)
        assert.equal(state.messages.length, 0)
      }
    })

    test(`${origin} modern ${type}: analysis opens directly and transport failures retain the existing details fallback`, async () => {
      const running = await run(source, { supported: true, running_job_id: 'job-1' })
      for (const id of ids) {
        assert.match(running.nodes[id].innerHTML, />分析中</)
        assert.doesNotMatch(running.nodes[id].innerHTML, /aura-status-description|aura-status-caption/)
        assert.equal(running.nodes[id].getAttribute('data-aura-disclosure'), 'false')
        running.click(id)
      }
      assert.equal(running.dialogs.length, ids.length)
      const failed = await run(source, {}, false)
      for (const id of ids) {
        assert.equal(failed.nodes[id].getAttribute('data-aura-state'), 'gray')
        failed.click(id)
        assert.equal(failed.dialogs.length, ids.indexOf(id) + 1)
      }
      assert.equal(failed.dialogs.length, ids.length)
      assert.ok(failed.requestInits.every(init => init.method !== 'POST'))
    })

    test(`${origin} modern ${type}: desktop dialog pauses effects and restores them on close`, async () => {
      const state = await run(source.replace('SHOW_ON_DESKTOP = false', 'SHOW_ON_DESKTOP = true'), complete, true, { ua: 'Windows NT 10.0', reducedMotion: true })
      state.click(ids[0])
      assert.equal(state.group.getAttribute('data-aura-effects-paused'), 'true')
      const overlay = state.elements.find(node => node.id === 'auraDesktopEmbedDialog')
      overlay.children[0].children[0].children[1].onclick()
      assert.equal(state.group.getAttribute('data-aura-effects-paused'), 'false')
      assert.equal(state.document.body.style.overflow, 'scroll')
      assert.equal(state.group.style.opacity, '')
      assert.equal(state.group.inert, false)
    })
  }
}

// 保留真正的 WAAPI 调用和取消语义，以受控帧检查状态动画，避免仅检查 CSS 字符串。
function animatedButton(state, status, text, loading = false, feature = 'audit') {
  const api = state.buttonsApi, animations = [], children = [], sr = { textContent: text }
  function animate(target, frames, options) {
    let finish
    const motion = { target, frames, options, cancelled: false, finished: new Promise(resolve => { finish = resolve }),
      cancel() { this.cancelled = true }, finish() { finish() } }
    animations.push(motion)
    return motion
  }
  const markup = api.buildStatusButton('motion', feature, status, text, loading, 'aura-mobile-label', false)
  const outer = markup.slice(0, markup.indexOf('>'))
  const attrs = Object.fromEntries([...outer.matchAll(/([\w-]+)="([^"]*)"/g)].map(match => [match[1], match[2]]))
  const classes = new Set(attrs.class.split(/\s+/))
  const copy = { innerHTML: api.modernButtonCopyMarkup(feature, status, text, loading), style: { removeProperty() {} }, animate: (f, o) => animate('copy', f, o) }
  const paths = Array.from({ length: 2 }, () => ({ animate: (f, o) => animate('path', f, o) }))
  const glyph = { animate: (f, o) => animate('glyph', f, o), querySelectorAll: () => paths }
  const icon = { innerHTML: api.iconSvgMarkup(feature, status, loading, text), animate: (f, o) => animate('icon', f, o), querySelector: () => glyph }
  const caption = { classList: { add() {} }, animate: (f, o) => animate('caption', f, o) }
  const number = { textContent: api.parseButtonLabel(text).scoreMatch?.[1] || '' }
  const holder = { appendChild(node) { children.push(node) } }
  const btn = {
    isConnected: true, animations, children, copy, icon, number, sr,
    style: { removeProperty() {}, setProperty() {} },
    classList: {
      add(...names) { names.forEach(n => classes.add(n)) }, remove(...names) { names.forEach(n => classes.delete(n)) },
      toggle(n, flag) { flag ? classes.add(n) : classes.delete(n) }, contains: n => classes.has(n),
    },
    getAttribute: k => attrs[k] ?? null, setAttribute(k, v) { attrs[k] = String(v) }, removeAttribute(k) { delete attrs[k] },
    appendChild(node) { children.push(node) },
    querySelector(selector) {
      if (selector === '.aura-status-sr') return sr
      if (selector === '.aura-status-icon') return icon
      if (selector === '.aura-status-score b') return api.parseButtonLabel(attrs['data-aura-copy']).scoreMatch ? number : null
      if (selector === '.aura-status-caption') return api.parseButtonLabel(attrs['data-aura-copy']).scoreMatch ? caption : null
      if (selector === '.aura-status-score' || selector === '.aura-modern-sheen') return holder
      return copy
    },
    getBoundingClientRect() { return { width: btn.visibleWidth ?? (classes.has('aura-status-button--icon-only') ? 56 : 156), height: 56 } },
    animate: (f, o) => animate('button', f, o),
  }
  const createElement = state.document.createElement
  state.document.createElement = tag => {
    const node = createElement(tag)
    node.animate = (f, o) => animate(tag, f, o)
    if (tag === 'canvas') node.getContext = () => new Proxy({}, { get: () => () => {} })
    return node
  }
  return btn
}

for (const origin of ['export', 'template']) {
  const source = origin === 'export' ? modernFactory('audit', 'test-token') : modernStaticSource
  test(`${origin} modern motion: result expansion, score refresh and interruption cancel stale effects`, async () => {
    const state = await run(source, { supported: true }, true, { exposeButtons: true, motion: true })
    const api = state.buttonsApi, btn = animatedButton(state, 'gray', '加载中...', true)
    api.morphModernStatusButton(btn, 'audit', 'green', '审核通过 (95分)', false, 'aura-mobile-label')
    const initial = btn.animations[0]
    assert.equal(initial.frames[0].width, '56px')
    assert.equal(initial.frames.at(-1).width, '156px')
    assert.ok(initial.options.duration + initial.options.delay <= 550)
    assert.equal(btn.sr.textContent, '审核通过 (95分)')
    const stale = btn._auraEffects
    btn.visibleWidth = 124
    api.morphModernStatusButton(btn, 'audit', 'red', '建议退回 (45分)', false, 'aura-mobile-label')
    assert.equal(initial.cancelled, true)
    assert.ok(stale.animations.every(motion => motion.cancelled))
    const current = btn._auraModernAnimations[0]
    assert.equal(current.frames[0].width, '124px')
    initial.finish(); await flush(state)
    assert.equal(btn._auraModernAnimations[0], current)
    const count = btn.animations.length
    api.morphModernStatusButton(btn, 'audit', 'red', '建议退回 (45分)', false, 'aura-mobile-label')
    assert.equal(btn.animations.length, count, 'identical polling result must not replay')
    api.cancelModernButtonMotion(btn)
    assert.equal(btn.number.textContent, '45')
    assert.equal(state.frames.size, 0)
  })

  test(`${origin} modern motion: pass, attention, rejection and summary use their own feedback`, async () => {
    for (const [feature, status, text, effect] of [
      ['audit', 'green', '审核通过 (95分)', 'canvas'],
      ['audit', 'yellow', '建议关注 (78分)', 'aura-modern-mark'],
      ['audit', 'red', '建议退回 (45分)', 'caption'],
      ['summary', 'green', '查看流程总结', 'aura-modern-scan'],
    ]) {
      const state = await run(source, { supported: true }, true, { exposeButtons: true, motion: true })
      const btn = animatedButton(state, 'loading', '分析中', false, feature)
      state.buttonsApi.morphModernStatusButton(btn, feature, status, text, false, 'aura-mobile-label')
      for (const timer of [...state.timeouts].filter(timer => timer.ms < 1000)) timer.fn()
      if (effect === 'canvas') assert.ok(btn.children.some(n => n.tagName === 'CANVAS'))
      else if (effect === 'caption') assert.ok(btn.animations.some(a => a.target === 'caption'))
      else assert.ok(btn.children.some(n => n.className === effect))
      const children = [...btn.children]
      state.buttonsApi.cancelModernButtonMotion(btn)
      assert.ok(children.every(node => node.removed))
      assert.equal(state.frames.size, 0)
    }
  })

  test(`${origin} modern motion: loading never discloses; auxiliary content stays single line and interrupts smoothly`, async () => {
    const state = await run(source, { supported: true }, true, { exposeButtons: true, motion: true })
    const api = state.buttonsApi
    const loading = animatedButton(state, 'gray', '加载中...', true)
    assert.equal(api.handleModernButtonClick(loading), true)
    assert.equal(loading.getAttribute('data-aura-expanded'), null)
    const btn = animatedButton(state, 'error', '加载异常')
    assert.equal(api.handleModernButtonClick(btn), true)
    assert.equal(btn.getAttribute('data-aura-expanded'), 'true')
    assert.equal(btn.copy.innerHTML, '<span class="aura-status-inline-label">AI审核加载异常</span>')
    assert.equal(api.handleModernButtonClick(btn), true, 'unavailable state must not open a dialog')
    const old = btn._auraModernAnimations[0]
    btn.visibleWidth = 102
    api.setModernButtonExpanded(btn, false)
    assert.equal(old.cancelled, true)
    const current = btn._auraModernAnimations[0]
    assert.equal(current.frames[0].width, '102px')
    old.finish(); await flush(state)
    assert.equal(btn._auraModernAnimations[0], current)
    current.finish(); await flush(state)
    assert.equal(btn.copy.innerHTML, '')
    assert.ok(btn.classList.contains('aura-status-button--icon-only'))
  })

  test(`${origin} modern motion: reduced motion displays final result and keeps information accessible`, async () => {
    const state = await run(source, { supported: true }, true, { exposeButtons: true, motion: true, reducedMotion: true })
    const btn = animatedButton(state, 'loading', '分析中')
    state.buttonsApi.morphModernStatusButton(btn, 'audit', 'red', '建议退回 (45分)', false, 'aura-mobile-label')
    assert.equal(btn.animations.length, 0)
    assert.match(btn.copy.innerHTML, /aura-status-caption--stamp/)
    assert.match(btn.copy.innerHTML, />45</)
    assert.equal(btn.getAttribute('aria-busy'), 'false')
  })
}

for (const origin of ['export', 'template']) {
  const source = origin === 'export' ? modernFactory('all', 'test-token') : modernStaticSource.replace("var EMBED_TYPE = 'audit'", "var EMBED_TYPE = 'all'")
  test(`${origin}: four float corners work independently on both devices and button styles`, async () => {
    for (const style of ['modern', 'classic']) {
      for (const desktop of [false, true]) {
        for (const position of ['top-left', 'top-right', 'bottom-left', 'bottom-right', 'invalid']) {
          const config = (style === 'classic' ? classicStyle(source) : source)
            .replace('SHOW_ON_DESKTOP = false', 'SHOW_ON_DESKTOP = true')
            .replace(`FLOAT_POSITION_${desktop ? 'DESKTOP' : 'MOBILE'} = 'bottom-left'`, `FLOAT_POSITION_${desktop ? 'DESKTOP' : 'MOBILE'} = '${position}'`)
          const state = await run(config, { supported: true }, true, { noContainer: true, ...(desktop ? { ua: 'Windows NT 10.0' } : {}) })
          const expected = position === 'invalid' ? 'bottom-left' : position
          assert.match(state.appended[0], new RegExp(`class="aura-float--${expected}"`))
          const css = state.styles.map(node => node.textContent).join('')
          for (const [corner, vertical, horizontal] of [['top-left', 'top', 'left'], ['bottom-right', 'bottom', 'right']]) {
            const rules = [...css.matchAll(new RegExp(`#auraMobileEmbedFloatContainer\\.aura-float--${corner}\\{([^}]*)\\}`, 'g'))]
            assert.equal(rules.length, 2, 'desktop and small-screen offsets must both exist')
            assert.ok(rules.every(rule => rule[1].includes(`safe-area-inset-${vertical}`) && rule[1].includes(`safe-area-inset-${horizontal}`)))
          }
          assert.match(css, /aura-float--bottom-right \.aura-status-group \{ align-items: flex-end;/)
        }
      }
    }
  })

  test(`${origin}: dragging from any configured corner releases its original anchors`, async () => {
    for (const style of ['modern', 'classic']) {
      for (const corner of ['top-left', 'top-right', 'bottom-left', 'bottom-right']) {
        const state = await run(style === 'classic' ? classicStyle(source) : source, { supported: true }, true, { exposeButtons: true })
        const classes = new Set(['aura-float--' + corner]), listeners = {}, attrs = {}
        const container = {
          style: {}, offsetWidth: 156, offsetHeight: 56,
          getAttribute: key => attrs[key], setAttribute(key, value) { attrs[key] = value },
          getBoundingClientRect: () => ({ left: 16, top: 50 }),
          classList: { add: name => classes.add(name), remove: name => classes.delete(name) },
          addEventListener(name, fn) { listeners[name] = fn },
        }
        state.buttonsApi.setupDraggableFloatContainer(container)
        listeners.mousedown({ button: 0, clientX: 20, clientY: 50, target: state.nodes.auraMobileEmbedAuditBtn })
        state.windowListeners.mousemove.at(-1)({ clientX: 100, clientY: 110, cancelable: true, preventDefault() {} })
        assert.ok(![...classes].some(name => name.startsWith('aura-float--')))
        assert.equal(container.style.bottom, 'auto')
        assert.equal(container.style.right, 'auto')
        assert.equal(container.style.left, '96px')
        assert.equal(container.style.top, '110px')
        state.windowListeners.mouseup.at(-1)()
        assert.ok(!classes.has('aura-is-dragging'))
      }
    }
  })
}

for (const origin of ['export', 'template']) {
  const source = origin === 'export' ? modernFactory('audit', 'test-token') : modernStaticSource
  test(`${origin}: result expansion follows each device's configuration and preserves one-click actions`, async () => {
    for (const [mobileExpanded, desktopExpanded, desktop, expected] of [
      [false, true, false, false], [false, true, true, true],
      [true, false, false, true], [true, false, true, false],
    ]) {
      const config = source.replace('SHOW_ON_DESKTOP = false', 'SHOW_ON_DESKTOP = true')
        .replace('BUTTON_EXPANDED_MOBILE = false', `BUTTON_EXPANDED_MOBILE = ${mobileExpanded}`)
        .replace('BUTTON_EXPANDED_DESKTOP = true', `BUTTON_EXPANDED_DESKTOP = ${desktopExpanded}`)
      const state = await run(config, { supported: true }, true, { exposeButtons: true, ...(desktop ? { ua: 'Windows NT 10.0' } : {}) })
      for (const [feature, status, text] of [
        ['audit', 'green', '审核通过 (95分)'], ['audit', 'yellow', '建议关注 (78分)'],
        ['audit', 'red', '建议退回 (45分)'], ['summary', 'green', '查看流程总结'],
      ]) {
        const btn = animatedButton(state, status, text, false, feature)
        assert.equal(btn.getAttribute('data-aura-result'), 'true')
        assert.equal(btn.getAttribute('data-aura-expanded'), String(expected))
        assert.equal(btn.classList.contains('aura-status-button--icon-only'), !expected)
        assert.equal(btn.getAttribute('aria-expanded'), 'false', 'visual expansion is separate from the dialog')
        assert.equal(state.buttonsApi.handleModernButtonClick(btn), false, 'result always routes to details on first click')
      }
      for (const [status, loading] of [['gray', true], ['loading', false], ['disabled', false]]) {
        const btn = animatedButton(state, status, '实际业务状态', loading)
        assert.equal(btn.getAttribute('data-aura-result'), 'false')
        assert.equal(btn.classList.contains('aura-status-button--icon-only'), loading || status === 'disabled')
      }
    }
  })

  test(`${origin}: collapsed results show feedback once then shrink, retaining full content for desktop hover`, async () => {
    for (const desktop of [false, true]) {
      const config = desktop ? source.replace('SHOW_ON_DESKTOP = false', 'SHOW_ON_DESKTOP = true').replace('BUTTON_EXPANDED_DESKTOP = true', 'BUTTON_EXPANDED_DESKTOP = false') : source
      const state = await run(config, { supported: true }, true, { exposeButtons: true, motion: true, hover: desktop, ...(desktop ? { ua: 'Windows NT 10.0' } : {}) })
      const api = state.buttonsApi, btn = animatedButton(state, 'gray', '加载中...', true)
      api.morphModernStatusButton(btn, 'audit', 'green', '审核通过 (95分)', false, 'aura-mobile-label')
      assert.equal(btn.getAttribute('data-aura-expanded'), 'true')
      const collapse = state.timeouts.find(timer => timer.ms === 2600)
      assert.ok(collapse, 'first completion has a readable preview before collapsing')
      const count = btn.animations.length
      api.morphModernStatusButton(btn, 'audit', 'green', '审核通过 (95分)', false, 'aura-mobile-label')
      assert.equal(btn.animations.length, count)
      assert.equal(state.timeouts.filter(timer => timer.ms === 2600).length, 1)
      collapse.fn()
      assert.equal(btn.getAttribute('data-aura-expanded'), 'false')
      const shrinking = btn._auraModernAnimations[0]
      assert.equal(shrinking.frames.at(-1).width, '56px')
      shrinking.finish(); await flush(state)
      assert.ok(btn.classList.contains('aura-status-button--icon-only'))
      assert.match(btn.copy.innerHTML, />95</)
      assert.equal(api.handleModernButtonClick(btn), false)
      api.queueSecondaryButtonHover(btn, true, desktop ? 'mouse' : 'touch')
      const hover = state.timeouts.find(timer => timer.ms === 60)
      if (desktop) {
        assert.ok(hover); hover.fn()
        assert.equal(btn.getAttribute('data-aura-expanded'), 'true')
        assert.equal(btn.getAttribute('aria-expanded'), 'false')
        collapse.fn()
        assert.equal(btn.getAttribute('data-aura-expanded'), 'true', 'old timer must not override user hover')
      } else assert.equal(hover, undefined)
    }
  })

  test(`${origin}: new states cancel previous collapse timers and reduced motion skips the intro`, async () => {
    const state = await run(source, { supported: true }, true, { exposeButtons: true, motion: true })
    const api = state.buttonsApi, btn = animatedButton(state, 'loading', '分析中')
    api.morphModernStatusButton(btn, 'audit', 'green', '审核通过 (95分)', false, 'aura-mobile-label')
    const old = state.timeouts.find(timer => timer.ms === 2600)
    api.morphModernStatusButton(btn, 'audit', 'loading', '分析中', false, 'aura-mobile-label')
    assert.equal(state.timeouts.some(timer => timer.id === old.id), false)
    old.fn()
    assert.equal(btn.getAttribute('data-aura-expanded'), 'false')
    assert.equal(btn.classList.contains('aura-status-button--icon-only'), false, 'analysis remains expanded independently')
    const reduced = await run(source, { supported: true }, true, { exposeButtons: true, motion: true, reducedMotion: true })
    const quiet = animatedButton(reduced, 'loading', '分析中')
    reduced.buttonsApi.morphModernStatusButton(quiet, 'audit', 'green', '审核通过 (95分)', false, 'aura-mobile-label')
    assert.equal(quiet.getAttribute('data-aura-expanded'), 'false')
    assert.ok(quiet.classList.contains('aura-status-button--icon-only'))
    assert.equal(quiet.animations.length, 0)
    assert.equal(reduced.timeouts.some(timer => timer.ms === 2600), false)
  })

  test(`${origin}: auxiliary icons are distinct from result and analysis symbols, including same-state updates`, async () => {
    const state = await run(source, { supported: true }, true, { exposeButtons: true })
    const api = state.buttonsApi
    const cases = [
      ['gray', true, '加载中...', 'processing'], ['disabled', false, '保存查看AI智审', 'save'],
      ['error', false, '加载异常', 'load-error'], ['red', false, '审核异常', 'execution-error'],
      ['disabled', false, '未开启总结', 'feature-off'], ['gray', false, '查看并生成审核', 'generate'],
    ]
    for (const [status, loading, text, kind] of cases) {
      const markup = api.iconSvgMarkup('audit', status, loading, text)
      assert.match(markup, new RegExp(`data-icon-kind="${kind}"`))
      assert.equal((markup.match(/<svg\b/g) || []).length, 1)
    }
    const btn = animatedButton(state, 'disabled', '保存查看AI智审')
    api.morphModernStatusButton(btn, 'audit', 'disabled', '未开启审核', false, 'aura-mobile-label')
    assert.match(btn.icon.innerHTML, /data-icon-kind="feature-off"/)
    const failed = animatedButton(state, 'red', '建议退回 (45分)')
    api.morphModernStatusButton(failed, 'audit', 'red', '审核异常', false, 'aura-mobile-label')
    assert.match(failed.icon.innerHTML, /data-icon-kind="execution-error"/)
  })

  test(`${origin}: all mode still displays the unsupported feature without opening or executing it`, async () => {
    const all = source.replace(/var EMBED_TYPE = ['"]audit['"]/, "var EMBED_TYPE = 'all'")
    for (const unsupported of ['audit', 'summary']) {
      const data = {
        audit: { supported: true, has_audit: true, audit_result: { recommendation: 'approve', overall_score: 95 } },
        summary: { supported: true, has_summary: true, summary_result: { status: 'completed' } },
      }
      data[unsupported] = { supported: false, message: '该功能未配置' }
      const state = await run(all, {}, true, { payloadByPath: { '/api/embed/context': data.audit, '/api/embed/summary/context': data.summary } })
      const disabled = unsupported === 'audit' ? 'auraMobileEmbedAuditBtn' : 'auraMobileEmbedSummaryBtn'
      const enabled = unsupported === 'audit' ? 'auraMobileEmbedSummaryBtn' : 'auraMobileEmbedAuditBtn'
      assert.equal(state.nodes[disabled].getAttribute('data-aura-state'), 'disabled')
      assert.equal(state.nodes[enabled].getAttribute('data-aura-state'), 'green')
      state.click(disabled); state.click(disabled)
      assert.equal(state.dialogs.length, 0)
      state.click(enabled)
      assert.equal(state.dialogs.length, 1)
      assert.ok(state.requestInits.every(init => init.method !== 'POST'))
    }
  })
}
