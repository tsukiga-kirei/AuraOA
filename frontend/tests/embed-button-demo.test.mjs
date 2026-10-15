import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import vm from 'node:vm'

const html = fs.readFileSync(new URL('../../docs/oa-configurations/assets/aura-embed-mobile-button-demo.html', import.meta.url), 'utf8')
const script = html.match(/<script>([\s\S]*?)<\/script>/)[1]
const core = script.slice(0, script.indexOf('/* 拖拽与长按 Tooltip 系统 */'))
const tooltip = script.slice(script.indexOf('function buttonTooltipText('), script.indexOf('function hideTooltip()'))
const retry = script.slice(script.indexOf('function retryDemoButton('), script.indexOf('// 点击、拖拽与长按分别处理'))
const click = script.slice(script.indexOf('function handleButtonClick('), script.indexOf("stageGroup.addEventListener('click'"))

// 执行演示页的真实状态转换，模拟动画时钟，检查连续更新和异步重试边界。
function runtime({ style = 'modern', reduced = false, systemReduced = false, hover = true } = {}) {
  const timers = [], animations = [], announcements = { textContent: '' }
  let now = 0
  const state = { style, reduced, opened: 0, hints: [], animations }
  const context = vm.createContext({
    document: {
      body: { getAttribute: key => key === 'data-button-style' ? state.style : String(state.reduced) },
      getElementById: id => id === 'buttonStatusAnnouncement' ? announcements : null,
    },
    window: { matchMedia: query => ({ matches: query.includes('prefers-reduced-motion') ? systemReduced : hover }) },
    Date: { now: () => now },
    setTimeout(fn, ms) { const timer = { fn, at: now + ms }; timers.push(timer); return timer },
    clearTimeout(timer) { const index = timers.indexOf(timer); if (index >= 0) timers.splice(index, 1) },
    sceneGeneration: 0, galleryGeneration: 0,
    dragSession: { ignoreNextClick: false, hasMoved: false, longPressTriggered: false },
    activeDialogScene: 'suggest-only',
    hideTooltip() {},
    showTooltipForButton(btn, text) { state.hints.push(text) },
    openOptimizedEmbedDialog() { state.opened++ },
  })
  vm.runInContext(core + '\n' + tooltip + '\n' + retry + '\n' + click, context)
  function animate(target, frames, options) {
    let resolve
    const motion = {
      target, frames, options, cancelled: false,
      finished: new Promise(done => { resolve = done }),
      cancel() { this.cancelled = true },
      finish() { resolve() },
    }
    animations.push(motion)
    return motion
  }
  function button(status, text, loading = false, feature = 'audit', gallery = false) {
    const markup = context.buildStatusButton('btnAudit_stage', feature, status, text, loading, false)
    const attrs = Object.fromEntries([...markup.matchAll(/([\w-]+)="([^"]*)"/g)].map(match => [match[1], match[2]]))
    // 仅取 button 自身的属性，避免 SVG 的 aria-hidden 和 class 覆盖外层。
    const outer = markup.slice(0, markup.indexOf('>'))
    Object.assign(attrs, Object.fromEntries([...outer.matchAll(/([\w-]+)="([^"]*)"/g)].map(match => [match[1], match[2]])))
    const classes = new Set(attrs.class.split(/\s+/))
    const icon = { innerHTML: context.iconSvgMarkup(feature, status, loading) }
    const copy = { innerHTML: '', animate: (frames, options) => animate('copy', frames, options) }
    icon.animate = (frames, options) => animate('icon', frames, options)
    const btn = {
      id: gallery ? 'g-error' : 'btnAudit_stage', isConnected: true,
      attributes: attrs, icon, copy,
      style: { setProperty() {}, removeProperty() {} },
      classList: {
        add(...names) { names.forEach(name => classes.add(name)) },
        remove(...names) { names.forEach(name => classes.delete(name)) },
        contains: name => classes.has(name),
        toggle(name, enabled) { enabled ? classes.add(name) : classes.delete(name) },
      },
      getAttribute: key => attrs[key] ?? null,
      setAttribute(key, value) { attrs[key] = String(value) },
      removeAttribute(key) { delete attrs[key] },
      querySelector(selector) { return selector === '.aura-status-icon' ? icon : copy },
      closest(selector) { return selector === '#galleryGroup' ? (gallery ? {} : null) : btn },
      getBoundingClientRect() { return { width: btn.visualWidth ?? (classes.has('aura-status-button--icon-only') ? 44 : attrs['data-aura-expanded'] === 'true' ? 120 : 86) } },
      animate: (frames, options) => animate('button', frames, options),
    }
    return btn
  }
  function advance(ms) {
    const end = now + ms
    while (true) {
      timers.sort((a, b) => a.at - b.at)
      if (!timers.length || timers[0].at > end) break
      const timer = timers.shift(); now = timer.at; timer.fn()
    }
    now = end
  }
  return { context, state, button, advance, announcements }
}

test('demo: modern states keep one icon and distinguish retry from audit rejection', () => {
  const { context } = runtime()
  for (const [feature, status, loading, text] of [
    ['audit', 'green', false, '审核通过 (95分)'],
    ['audit', 'yellow', false, '建议关注 (78分)'],
    ['audit', 'red', false, '建议退回 (45分)'],
    ['summary', 'green', false, '查看流程总结'],
    ['summary', 'gray', true, '总结加载中...'],
    ['audit', 'loading', false, '审核分析中'],
    ['audit', 'disabled', false, '保存查看AI智审'],
    ['audit', 'error', false, '连接失败，点击重试'],
  ]) {
    const markup = context.buildStatusButton('demo', feature, status, text, loading, false)
    assert.equal((markup.match(/<svg\b/g) || []).length, 1)
    if (loading || status === 'loading' || status === 'disabled' || status === 'error') assert.match(markup, /aura-status-button--icon-only/)
    if (status === 'error') { assert.match(markup, /data-icon-kind="retry"/); assert.match(markup, /aria-disabled="false"/); assert.doesNotMatch(markup, /aria-haspopup/); }
    if (status === 'red') assert.match(markup, /data-icon-kind="reject"/)
  }
})

test('demo: initial result expands within 550ms and exposes accessible status', () => {
  const { context, button, state, announcements } = runtime()
  const btn = button('loading', '审核分析中')
  context.morphModernStatusButton(btn, 'audit', 'green', '审核通过 (95分)', false)
  const width = state.animations.find(motion => motion.target === 'button')
  assert.equal(width.frames[0].width, '44px')
  assert.equal(width.frames[1].width, '86px')
  assert.ok(width.options.delay + width.options.duration <= 550)
  assert.equal(btn.getAttribute('aria-busy'), 'false')
  assert.equal(announcements.textContent, '审核通过 (95分)')
  assert.ok(btn.classList.contains('aura-status-button--reveal'))
})

test('demo: result refresh does not collapse and identical refresh does not animate', () => {
  const { context, button, state } = runtime()
  const btn = button('green', '审核通过 (95分)')
  context.morphModernStatusButton(btn, 'audit', 'yellow', '建议关注 (78分)', false)
  const width = state.animations.find(motion => motion.target === 'button')
  assert.equal(width.frames[0].width, '86px')
  assert.equal(width.frames[1].width, '86px')
  assert.ok(btn.classList.contains('aura-status-button--update'))
  assert.ok(!btn.classList.contains('aura-status-button--reveal'))
  const count = state.animations.length
  context.morphModernStatusButton(btn, 'audit', 'yellow', '建议关注 (78分)', false)
  assert.equal(state.animations.length, count)
})

test('demo: score-only updates preserve the icon and skip icon animation', () => {
  const { context, button, state } = runtime()
  const btn = button('green', '审核通过 (95分)')
  const oldIcon = btn.icon.innerHTML
  context.morphModernStatusButton(btn, 'audit', 'green', '审核通过 (96分)', false)
  assert.equal(btn.icon.innerHTML, oldIcon)
  assert.ok(state.animations.every(motion => motion.target !== 'icon'))
  assert.match(btn.copy.innerHTML, /96/)
})

test('demo: interrupted completion resumes at visible width and stale callbacks cannot clear new motion', async () => {
  const { context, button, state } = runtime()
  const btn = button('loading', '审核分析中')
  context.morphModernStatusButton(btn, 'audit', 'green', '审核通过 (95分)', false)
  const old = [...state.animations]
  btn.visualWidth = 62
  context.morphModernStatusButton(btn, 'audit', 'red', '建议退回 (45分)', false)
  assert.ok(old.every(motion => motion.cancelled))
  const current = btn._auraModernAnimations[0]
  assert.equal(current.frames[0].width, '62px')
  old[0].finish()
  await Promise.resolve()
  assert.equal(btn._auraModernAnimations[0], current)
  current.finish()
  await Promise.resolve()
  assert.equal(btn._auraModernAnimations.length, 0)
  assert.ok(!btn.classList.contains('aura-status-button--morph'))
})

test('demo: manual and system reduced motion both render the final result immediately', () => {
  for (const options of [{ reduced: true }, { systemReduced: true }]) {
    const { context, button, state } = runtime(options)
    const btn = button('loading', '审核分析中')
    context.morphModernStatusButton(btn, 'audit', 'green', '审核通过 (95分)', false)
    assert.equal(state.animations.length, 0)
    assert.match(btn.getAttribute('aria-label'), /^审核通过 \(95分\)/)
    assert.ok(!btn.classList.contains('aura-status-button--icon-only'))
  }
})

test('demo: retry blocks duplicate requests while loading and recovers', () => {
  const { context, button, advance, state } = runtime()
  const btn = button('error', '连接失败，点击重试', false, 'audit', true)
  const event = { target: btn, preventDefault() {}, stopPropagation() {} }
  context.handleButtonClick(event)
  assert.equal(btn.getAttribute('aria-busy'), 'true')
  assert.equal(btn.getAttribute('aria-disabled'), 'true')
  assert.equal(state.opened, 0)
  context.handleButtonClick(event)
  assert.equal(btn.getAttribute('data-aura-state'), 'gray')
  assert.equal(btn.getAttribute('aria-expanded'), 'false')
  advance(750)
  assert.equal(btn.getAttribute('data-aura-state'), 'loading')
  advance(950)
  assert.equal(btn.getAttribute('data-aura-state'), 'green')
  assert.equal(btn.getAttribute('aria-disabled'), 'false')
  assert.equal(state.opened, 0)
})

test('demo: switching gallery invalidates pending retry callbacks', () => {
  const { context, button, advance } = runtime()
  const btn = button('error', '连接失败，点击重试', false, 'audit', true)
  context.retryDemoButton(btn)
  context.galleryGeneration++
  advance(2000)
  assert.equal(btn.getAttribute('data-aura-state'), 'gray')
})

test('demo: long press suppresses accidental activation and locked buttons explain prerequisites', () => {
  const { context, button, state } = runtime()
  const btn = button('green', '审核通过 (95分)')
  btn._auraSuppressClickUntil = 1200
  context.handleButtonClick({ target: btn, preventDefault() {}, stopPropagation() {} })
  assert.equal(state.opened, 0)
  const locked = button('disabled', '保存查看AI智审')
  context.handleButtonClick({ target: locked })
  assert.match(state.hints.at(-1), /请先保存流程/)
  assert.equal(locked.getAttribute('data-aura-expanded'), 'false')
  assert.equal(state.opened, 0)
})

test('demo: desktop hover changes the label and click still opens details', () => {
  const { context, button, state, advance } = runtime()
  const btn = button('green', '审核通过 (95分)')
  const icon = btn.icon.innerHTML
  context.queueModernButtonHover(btn, true, 'mouse')
  advance(70)
  assert.equal(btn.getAttribute('data-aura-expanded'), 'true')
  assert.equal(btn.getAttribute('aria-expanded'), 'false')
  assert.match(btn.copy.innerHTML, /审核通过/)
  assert.doesNotMatch(btn.copy.innerHTML, /95/)
  assert.equal(state.animations[0].frames[0].width, '86px')
  assert.equal(state.animations[0].frames[1].width, '120px')
  context.handleButtonClick({ target: btn })
  assert.equal(state.opened, 1)
  context.queueModernButtonHover(btn, false, 'mouse')
  advance(110)
  assert.equal(btn.getAttribute('data-aura-expanded'), 'false')
  assert.match(btn.copy.innerHTML, /<b>95<\/b>/)
  assert.equal(btn.icon.innerHTML, icon)
  assert.equal(btn.getAttribute('aria-haspopup'), 'dialog')
})

test('demo: status refresh preserves the expanded reading mode', () => {
  const { context, button } = runtime()
  const btn = button('green', '审核通过 (95分)')
  context.setModernButtonExpanded(btn, true)
  context.morphModernStatusButton(btn, 'audit', 'yellow', '建议关注 (78分)', false)
  assert.equal(btn.getAttribute('data-aura-expanded'), 'true')
  assert.match(btn.copy.innerHTML, /建议关注/)
  assert.doesNotMatch(btn.copy.innerHTML, /78/)
  context.setModernButtonExpanded(btn, false)
  assert.match(btn.copy.innerHTML, /<b>78<\/b>/)
})

test('demo: loading and summary labels remain inside the same single-icon button', () => {
  const { context, button, state } = runtime()
  for (const [feature, status, loading, text, label] of [
    ['audit', 'gray', true, '加载中...', '处理中'],
    ['audit', 'loading', false, '审核分析中', '分析中'],
    ['summary', 'green', false, '查看流程总结', '总结已完成'],
  ]) {
    const btn = button(status, text, loading, feature)
    const icon = btn.icon.innerHTML
    context.setModernButtonExpanded(btn, true)
    assert.match(btn.copy.innerHTML, new RegExp(label))
    assert.equal(btn.icon.innerHTML, icon)
    assert.ok(!btn.classList.contains('aura-status-button--icon-only'))
  }
  assert.equal(state.opened, 0)
})

test('demo: rapid label toggles resume at visible width and reduced motion settles immediately', () => {
  const { context, button, state } = runtime()
  const btn = button('green', '审核通过 (95分)')
  context.setModernButtonExpanded(btn, true)
  const old = [...state.animations]
  btn.visualWidth = 98
  context.setModernButtonExpanded(btn, false)
  assert.ok(old.every(motion => motion.cancelled))
  assert.equal(btn._auraModernAnimations[0].frames[0].width, '98px')
  state.reduced = true
  context.setModernButtonExpanded(btn, true)
  assert.equal(btn._auraModernAnimations.length, 0)
  assert.equal(btn.getAttribute('data-aura-expanded'), 'true')
  assert.match(btn.copy.innerHTML, /审核通过/)
})

test('demo: touch events and mobile capability never change the label but clicks open details', () => {
  for (const hover of [true, false]) {
    const { context, button, state, advance } = runtime({ hover })
    const btn = button('green', '审核通过 (95分)')
    context.queueModernButtonHover(btn, true, hover ? 'touch' : 'mouse')
    advance(200)
    assert.equal(btn.getAttribute('data-aura-expanded'), 'false')
    if (!hover) {
      context.setModernButtonExpanded(btn, true)
      assert.equal(btn.getAttribute('data-aura-expanded'), 'false')
    }
    context.handleButtonClick({ target: btn })
    assert.equal(state.opened, 1)
  }
})

test('demo: moving past a button or starting a drag cancels hover expansion', () => {
  const { context, button, advance } = runtime()
  const btn = button('green', '审核通过 (95分)')
  context.queueModernButtonHover(btn, true, 'mouse')
  context.queueModernButtonHover(btn, false, 'mouse')
  advance(200)
  assert.equal(btn.getAttribute('data-aura-expanded'), 'false')
  context.dragSession.isDown = true
  context.queueModernButtonHover(btn, true, 'mouse')
  advance(100)
  assert.equal(btn.getAttribute('data-aura-expanded'), 'false')
})

test('demo: classic colors, icons and disabled-error behavior stay intact', () => {
  const { context } = runtime({ style: 'classic' })
  assert.equal(context.buttonTheme('green', false).surface, '#f1fbf5')
  assert.doesNotMatch(context.iconSvgMarkup('audit', 'loading', false), /aura-modern/)
  assert.match(context.buildStatusButton('demo', 'audit', 'error', '连接失败', false, false), /aria-disabled="true"/)
})
