import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import vm from 'node:vm'

const html = fs.readFileSync(new URL('../../docs/oa-configurations/assets/aura-embed-mobile-button-demo.html', import.meta.url), 'utf8')
const script = html.match(/<script>([\s\S]*?)<\/script>/)[1]
const core = script.slice(0, script.indexOf('/* 拖拽与长按 Tooltip 系统 */'))
const tooltip = script.slice(script.indexOf('function buttonTooltipText('), script.indexOf('function hideTooltip()'))
const click = script.slice(script.indexOf('function handleButtonClick('), script.indexOf("stageGroup.addEventListener('click'"))

// 执行演示页的真实状态转换，模拟动画时钟，检查连续更新、可访问性和实际错误行为。
function runtime({ style = 'modern', reduced = false, systemReduced = false, hover = true } = {}) {
  const timers = [], animations = [], announcements = { textContent: '' }
  let now = 0
  function schedule(fn, ms) { const timer = { fn, at: now + ms }; timers.push(timer); return timer }
  function cancel(timer) { const index = timers.indexOf(timer); if (index >= 0) timers.splice(index, 1) }
  const state = { style, reduced, opened: 0, hints: [], animations, listeners: {}, buttons: [] }
  const context = vm.createContext({
    document: {
      addEventListener(name, callback) { state.listeners[name] = callback },
      querySelectorAll() { return state.buttons.filter(btn => btn.getAttribute('data-aura-expanded') === 'true') },
      body: { getAttribute: key => key === 'data-button-style' ? state.style : String(state.reduced) },
      getElementById: id => id === 'buttonStatusAnnouncement' ? announcements : null,
      createElement: tag => ({
        tagName: tag, className: '', style: {}, removed: false,
        setAttribute() {}, remove() { this.removed = true },
        animate: (frames, options) => animate(tag, frames, options),
        getContext: () => new Proxy({}, { get: () => () => {} }),
      }),
    },
    window: { matchMedia: query => ({ matches: query.includes('prefers-reduced-motion') ? systemReduced : hover }) },
    Date: { now: () => now },
    setTimeout: schedule, clearTimeout: cancel,
    requestAnimationFrame: fn => schedule(() => fn(now), 16), cancelAnimationFrame: cancel,
    performance: { now: () => now },
    sceneGeneration: 0, galleryGeneration: 0,
    dragSession: { ignoreNextClick: false, hasMoved: false, longPressTriggered: false },
    activeDialogScene: 'suggest-only',
    hideTooltip() {},
    showTooltipForButton(btn, text) { state.hints.push(text) },
    openOptimizedEmbedDialog() { state.opened++ },
  })
  vm.runInContext(core + '\n' + tooltip + '\n' + click, context)
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
    const copy = { style: { removeProperty() {} }, innerHTML: context.modernButtonCopyMarkup(feature, status, text, loading), animate: (frames, options) => animate('copy', frames, options) }
    icon.animate = (frames, options) => animate('icon', frames, options)
    const number = { textContent: context.parseButtonLabel(text).scoreMatch?.[1] || '' }
    const caption = { classList: { add() {} }, animate: (frames, options) => animate('caption', frames, options) }
    const paths = Array.from({ length: 2 }, () => ({ animate: (frames, options) => animate('path', frames, options) }))
    const glyph = { querySelectorAll: () => paths, animate: (frames, options) => animate('glyph', frames, options) }
    icon.querySelector = () => glyph
    const children = []
    const holder = { appendChild(node) { children.push(node) } }
    const btn = {
      id: gallery ? 'g-error' : 'btnAudit_stage', isConnected: true,
      attributes: attrs, icon, copy, number, children,
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
      appendChild(node) { children.push(node) },
      querySelector(selector) {
        if (selector === '.aura-status-icon') return icon
        if (selector === '.aura-status-score b') return context.parseButtonLabel(attrs['data-aura-copy']).scoreMatch ? number : null
        if (selector === '.aura-status-caption') return context.parseButtonLabel(attrs['data-aura-copy']).scoreMatch ? caption : null
        if (selector === '.aura-status-score' || selector === '.aura-modern-sheen') return holder
        return copy
      },
      closest(selector) { return selector === '#galleryGroup' ? (gallery ? {} : null) : btn },
      getBoundingClientRect() { return { width: btn.visualWidth ?? (classes.has('aura-status-button--icon-only') ? 64 : 156) } },
      animate: (frames, options) => animate('button', frames, options),
    }
    state.buttons.push(btn)
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

test('demo: modern states keep one icon and distinguish unavailable from audit rejection', () => {
  const { context } = runtime()
  for (const [feature, status, loading, text] of [
    ['audit', 'green', false, '审核通过 (95分)'],
    ['audit', 'yellow', false, '建议关注 (78分)'],
    ['audit', 'red', false, '建议退回 (45分)'],
    ['summary', 'green', false, '查看流程总结'],
    ['summary', 'gray', true, '总结加载中...'],
    ['audit', 'loading', false, '审核分析中'],
    ['audit', 'disabled', false, '保存查看AI智审'],
    ['audit', 'error', false, '加载异常'],
  ]) {
    const markup = context.buildStatusButton('demo', feature, status, text, loading, false)
    assert.equal((markup.match(/<svg\b/g) || []).length, 1)
    if (loading) assert.match(markup, /aura-status-button--icon-only/)
    if (!loading && status === 'loading') { assert.doesNotMatch(markup, /aura-status-button--icon-only/); assert.match(markup, />分析中</); assert.doesNotMatch(markup, /aura-status-description/); }
    if (status === 'error') { assert.match(markup, /data-icon-kind="unavailable"/); assert.match(markup, /data-aura-disclosure="true"/); assert.doesNotMatch(markup, /aria-haspopup/); }
    if (status === 'red') assert.match(markup, /data-icon-kind="reject"/)
  }
})

test('demo: initial result expands within 550ms and exposes accessible status', () => {
  const { context, button, state, announcements } = runtime()
  const btn = button('gray', '加载中...', true)
  context.morphModernStatusButton(btn, 'audit', 'green', '审核通过 (95分)', false)
  const width = state.animations.find(motion => motion.target === 'button')
  assert.equal(width.frames[0].width, '64px')
  assert.equal(width.frames.at(-1).width, '156px')
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
  assert.equal(width.frames[0].width, '156px')
  assert.equal(width.frames.at(-1).width, '156px')
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

test('demo: unavailable status does not show a tooltip, retry or open details', () => {
  const { context, button, state, advance } = runtime()
  const btn = button('error', '加载异常')
  context.handleButtonClick({ target: btn })
  advance(2000)
  assert.equal(btn.getAttribute('data-aura-state'), 'error')
  assert.equal(btn.getAttribute('data-aura-expanded'), 'true')
  assert.equal(state.hints.length, 0)
  assert.equal(state.opened, 0)
  context.handleButtonClick({ target: btn })
  assert.equal(state.opened, 0)
})

test('demo: suppressed clicks and locked buttons do not open details or show tooltips', () => {
  const { context, button, state } = runtime()
  const btn = button('green', '审核通过 (95分)')
  btn._auraSuppressClickUntil = 1200
  context.handleButtonClick({ target: btn, preventDefault() {}, stopPropagation() {} })
  assert.equal(state.opened, 0)
  const locked = button('disabled', '保存查看AI智审')
  context.handleButtonClick({ target: locked })
  assert.equal(state.hints.length, 0)
  assert.equal(state.opened, 0)
})

test('demo: result caption and score remain visible together when opening details', () => {
  const { context, button, state } = runtime()
  const btn = button('green', '审核通过 (95分)')
  const copy = btn.copy.innerHTML
  assert.match(copy, /aura-status-caption[^>]*>审核通过/)
  assert.match(copy, /<b>95<\/b>/)
  context.handleButtonClick({ target: btn })
  assert.equal(state.opened, 1)
  assert.equal(btn.copy.innerHTML, copy)
  context.morphModernStatusButton(btn, 'audit', 'yellow', '建议关注 (78分)', false)
  assert.match(btn.copy.innerHTML, /建议关注/)
  assert.match(btn.copy.innerHTML, /<b>78<\/b>/)
})

test('demo: scoreless business states preserve their actual labels', () => {
  const { context } = runtime()
  for (const [feature, status, text] of [
    ['audit', 'red', '审核异常'],
    ['summary', 'red', '总结异常'],
    ['summary', 'disabled', '未开启总结'],
    ['audit', 'gray', '查看并生成审核'],
    ['summary', 'green', '查看流程总结'],
    ['audit', 'green', '审核通过'],
  ]) {
    const markup = context.modernButtonCopyMarkup(feature, status, text, false, true)
    if (context.isSecondaryButton(feature, status, false)) {
      assert.ok(markup.includes(feature === 'summary' ? 'AI总结' : 'AI审核'))
      assert.doesNotMatch(markup, /aura-status-description/)
    } else assert.ok(markup.includes(feature === 'summary' && status === 'green' ? '>总结<' : text))
    assert.doesNotMatch(markup, /aura-status-score/)
    assert.equal(context.isIconOnlyButton(feature, status, false), context.isSecondaryButton(feature, status, false))
  }
})

test('demo: loading stays unavailable while analysis can open progress', () => {
  const { context, button, state } = runtime()
  const fetching = button('gray', '加载中...', true)
  context.handleButtonClick({ target: fetching })
  assert.equal(state.opened, 0)
  assert.equal(fetching.getAttribute('aria-haspopup'), null)
  assert.equal(fetching.getAttribute('aria-busy'), 'true')
  const analyzing = button('loading', '审核分析中')
  context.handleButtonClick({ target: analyzing })
  assert.equal(state.opened, 1)
  assert.equal(analyzing.getAttribute('aria-busy'), 'true')
  assert.equal(analyzing.getAttribute('aria-haspopup'), 'dialog')
})

test('demo: classic colors, icons and disabled-error behavior stay intact', () => {
  const { context } = runtime({ style: 'classic' })
  assert.equal(context.buttonTheme('green', false).surface, '#f1fbf5')
  assert.doesNotMatch(context.iconSvgMarkup('audit', 'loading', false), /aura-modern/)
  assert.match(context.buildStatusButton('demo', 'audit', 'error', '连接失败', false, false), /aria-disabled="true"/)
})

// 完成特效拥有独立生命周期，宽度过渡结束后仍可播放，状态改变时必须立即清理。
test('demo: success confetti finishes and a new state cancels particles and score frames', () => {
  const { context, button, advance } = runtime()
  const btn = button('loading', '审核分析中')
  context.morphModernStatusButton(btn, 'audit', 'green', '审核通过 (95分)', false)
  advance(700)
  const canvas = btn.children.find(node => node.tagName === 'canvas')
  assert.ok(canvas)
  assert.ok(Number(btn.number.textContent) > 0)
  context.morphModernStatusButton(btn, 'audit', 'loading', '审核分析中', false)
  assert.equal(canvas.removed, true)
  advance(2000)
  assert.equal(btn.getAttribute('data-aura-state'), 'loading')
  assert.equal(btn.children.filter(node => node.tagName === 'canvas').length, 1)
})

test('demo: pass, review, return and summary have different completion effects', () => {
  for (const [feature, status, text, expected] of [
    ['audit', 'green', '审核通过 (95分)', 'success'],
    ['audit', 'yellow', '建议关注 (78分)', 'warning'],
    ['audit', 'red', '建议退回 (45分)', 'reject'],
    ['summary', 'green', '查看流程总结', 'summary'],
  ]) {
    const { context, button, advance, state } = runtime()
    assert.equal(context.modernEffectKind(feature, status, text, false), expected)
    const btn = button('loading', '分析中', false, feature)
    context.morphModernStatusButton(btn, feature, status, text, false)
    advance(1100)
    if (expected === 'success' || expected === 'reject') assert.ok(btn.children.some(node => node.tagName === 'canvas'))
    else assert.ok(btn.children.some(node => node.className === 'aura-modern-scan'))
    if (expected === 'warning') assert.ok(btn.children.some(node => node.className === 'aura-modern-mark'))
    if (expected === 'reject') assert.ok(state.animations.some(motion => motion.target === 'caption'))
    if (feature === 'audit') assert.equal(btn.number.textContent, Number(context.parseButtonLabel(text).scoreMatch[1]))
    if (feature === 'summary') assert.match(btn.copy.innerHTML, />总结</)
  }
})

test('demo: reduced motion removes all particles and settles the score', () => {
  const { context, button, advance, state } = runtime()
  const btn = button('loading', '分析中')
  context.morphModernStatusButton(btn, 'audit', 'green', '审核通过 (95分)', false)
  advance(700)
  state.reduced = true
  context.cancelModernButtonMotion(btn)
  assert.equal(btn.number.textContent, '95')
  assert.ok(btn.children.every(node => node.removed))
  advance(2000)
  assert.equal(btn.number.textContent, '95')
})

test('demo: modern hover and touch tooltip entry returns before creating a popup', () => {
  const { context, button } = runtime()
  const popup = script.slice(script.indexOf('function showTooltipForButton('), script.indexOf('function setupDraggableFloatContainer('))
  vm.runInContext(popup, context)
  let created = 0
  context.document.createElement = () => { created++; throw new Error('Unexpected tooltip') }
  context.showTooltipForButton(button('red', '建议退回 (45分)'), 'AI 流程审核 · 建议退回 (45分)')
  assert.equal(created, 0)
})

test('demo: exception and analysis descriptions identify the AI feature', () => {
  const { context } = runtime()
  for (const [feature, status, label, description] of [
    ['audit', 'error', '加载异常', 'AI审核状态获取失败'],
    ['summary', 'error', '加载异常', 'AI总结状态获取失败'],
    ['audit', 'red', '审核异常', 'AI审核任务执行异常'],
    ['summary', 'disabled', '未开启总结', '该流程未启用AI总结'],
  ]) {
    const copy = context.modernButtonCopyMarkup(feature, status, label, false, true)
    assert.ok(copy.includes(feature === 'summary' ? 'AI总结' : 'AI审核'))
    assert.doesNotMatch(copy, /aura-status-description/)
    const markup = context.buildStatusButton('demo', feature, status, label, false, false)
    assert.ok(markup.includes('aria-label="' + label + '，' + description + '"'))
  }
})

test('demo: while the dialog is open, state updates settle without new particle or score work', () => {
  const { context, button, state, advance } = runtime()
  const btn = button('gray', '加载中...', true)
  context.buttonEffectsSuspended = true
  context.morphModernStatusButton(btn, 'audit', 'green', '审核通过 (95分)', false)
  advance(2000)
  assert.equal(state.animations.length, 0)
  assert.equal(btn.children.length, 0)
  assert.match(btn.copy.innerHTML, /<b>95<\/b>/)
  context.buttonEffectsSuspended = false
  context.morphModernStatusButton(btn, 'audit', 'red', '建议退回 (45分)', false)
  advance(500)
  assert.ok(btn.children.some(node => node.tagName === 'canvas'))
})

test('demo: shared-image genie slices keep continuous edges and finite geometry', () => {
  const hosts = []
  const context = vm.createContext({
    document: {
      body: { appendChild(node) { hosts.push(node) } },
      createElement: () => ({ style: {}, children: [], setAttribute() {}, appendChild(node) { this.children.push(node) }, remove() { this.removed = true } }),
    },
    getComputedStyle: () => ({ backgroundColor: 'rgb(255, 255, 255)', borderRadius: '22px' }),
  })
  vm.runInContext(script.slice(script.indexOf('function createDemoGenie('), script.indexOf('var dialogPanel =')), context)
  const box = { left: 100, top: 80, width: 780, height: 860 }
  const panel = { getBoundingClientRect: () => box, querySelectorAll: () => [] }
  const layer = context.createDemoGenie(panel, { left: 950, top: 100, width: 52, height: 52 })
  const slices = hosts[0].children
  const bandHeight = box.height / slices.length
  assert.ok(slices.length >= 24 && slices.length <= 48)
  assert.ok(slices.every(slice => slice.children.length === 0))
  assert.ok(slices.every(slice => slice.style.backgroundImage === slices[0].style.backgroundImage))
  function vertex(matrix, x, y) {
    const w = matrix[3] * x + matrix[7] * y + 1
    return [(matrix[0] * x + matrix[4] * y + matrix[12]) / w, (matrix[1] * x + matrix[5] * y + matrix[13]) / w]
  }
  for (const progress of [0, .2, .65, .9, 1]) {
    layer.apply(progress, false, 1)
    const matrices = slices.map(slice => slice.style.transform.slice(9, -1).split(',').map(Number))
    assert.ok(matrices.every(matrix => matrix.length === 16 && matrix.every(Number.isFinite)))
    for (let i = 1; i < matrices.length; i++) {
      for (const x of [0, box.width]) {
        const bottom = vertex(matrices[i - 1], x, bandHeight)
        const top = vertex(matrices[i], x, 0)
        assert.ok(Math.abs(bottom[0] - top[0]) < 1e-6)
        assert.ok(Math.abs(bottom[1] - top[1]) < 1e-6)
      }
    }
  }
  layer.destroy()
  assert.equal(hosts[0].removed, true)
})

test('demo: analysis shows only 分析中 while fetching keeps a single icon', () => {
  const { context } = runtime()
  for (const feature of ['audit', 'summary']) {
    const copy = context.modernButtonCopyMarkup(feature, 'loading', '审核分析中', false)
    assert.match(copy, />分析中</)
    assert.doesNotMatch(copy, /aura-status-description|审核分析中|总结分析中|点击查看进度/)
    assert.equal(context.modernButtonCopyMarkup(feature, 'gray', '加载中...', true), '')
    assert.equal(context.isIconOnlyButton(feature, 'gray', true), true)
  }
})

test('demo: secondary states begin as icons and hover reveals one concise line', async () => {
  const { context, button, advance } = runtime()
  for (const [feature, status, label, description] of [
    ['audit', 'disabled', '保存查看AI智审', '保存后查看AI审核'],
    ['audit', 'error', '加载异常', 'AI审核加载异常'],
    ['audit', 'red', '审核异常', 'AI审核执行异常'],
    ['summary', 'disabled', '未开启总结', '未开启AI总结'],
  ]) {
    const btn = button(status, label, false, feature)
    assert.ok(btn.classList.contains('aura-status-button--icon-only'))
    assert.equal(btn.copy.innerHTML, '')
    context.queueSecondaryButtonHover(btn, true, 'mouse')
    advance(60)
    assert.equal(btn.getAttribute('data-aura-expanded'), 'true')
    assert.ok(btn.copy.innerHTML.includes(description))
    assert.doesNotMatch(btn.copy.innerHTML, /aura-status-description/)
    assert.equal(btn.getAttribute('aria-expanded'), 'true')
    const width = btn._auraModernAnimations[0]
    assert.equal(width.frames[0].width, '64px')
    assert.equal(width.frames[1].width, '156px')
    width.finish()
    await Promise.resolve()
    context.queueSecondaryButtonHover(btn, false, 'mouse')
    advance(120)
    assert.ok(btn.copy.innerHTML.includes(description), 'copy fades with the shell before it is removed')
    btn._auraModernAnimations[0].finish()
    await Promise.resolve()
    assert.equal(btn.copy.innerHTML, '')
    assert.ok(btn.classList.contains('aura-status-button--icon-only'))
  }
})

test('demo: mobile first tap reveals text; only a second tap on an available state opens details', () => {
  const { context, button, state } = runtime({ hover: false })
  const btn = button('gray', '查看并生成审核')
  context.queueSecondaryButtonHover(btn, true, 'touch')
  assert.equal(btn.getAttribute('data-aura-expanded'), 'false')
  context.handleButtonClick({ target: btn })
  assert.equal(btn.getAttribute('data-aura-expanded'), 'true')
  assert.equal(state.opened, 0)
  context.handleButtonClick({ target: btn })
  assert.equal(state.opened, 1)
  const loading = button('gray', '加载中...', true)
  context.handleButtonClick({ target: loading })
  context.handleButtonClick({ target: loading })
  assert.equal(loading.copy.innerHTML, '')
  assert.equal(loading.getAttribute('data-aura-disclosure'), null)
  assert.ok(loading.classList.contains('aura-status-button--icon-only'))
  assert.equal(state.opened, 1)
})

test('demo: interrupted disclosure starts at the visible width and stale completion cannot collapse it', async () => {
  const { context, button } = runtime()
  const btn = button('error', '加载异常')
  context.setSecondaryButtonExpanded(btn, true)
  const old = btn._auraModernAnimations[0]
  btn.visualWidth = 90
  context.setSecondaryButtonExpanded(btn, false)
  const current = btn._auraModernAnimations[0]
  assert.equal(old.cancelled, true)
  assert.equal(current.frames[0].width, '90px')
  old.finish()
  await Promise.resolve()
  assert.equal(btn._auraModernAnimations[0], current)
  current.finish()
  await Promise.resolve()
  assert.equal(btn.copy.innerHTML, '')
})

test('demo: state updates preserve an expanded secondary label and primary results stay visible', () => {
  const { context, button } = runtime({ reduced: true })
  const btn = button('disabled', '保存查看AI智审')
  context.setSecondaryButtonExpanded(btn, true)
  context.morphModernStatusButton(btn, 'audit', 'error', '加载异常', false)
  assert.equal(btn.getAttribute('data-aura-expanded'), 'true')
  assert.match(btn.copy.innerHTML, /AI审核加载异常/)
  context.morphModernStatusButton(btn, 'audit', 'green', '审核通过 (95分)', false)
  assert.equal(btn.getAttribute('data-aura-disclosure'), 'false')
  assert.match(btn.copy.innerHTML, /95/)
  assert.equal(context.setSecondaryButtonExpanded(btn, false), false)
  assert.ok(!btn.classList.contains('aura-status-button--icon-only'))
})

test('demo: clicking outside collapses secondary content without a popup', () => {
  const { context, button, state } = runtime({ reduced: true })
  const btn = button('error', '加载异常')
  context.setSecondaryButtonExpanded(btn, true)
  state.listeners.pointerdown({ target: { closest: () => null } })
  assert.equal(btn.getAttribute('data-aura-expanded'), 'false')
  assert.equal(btn.copy.innerHTML, '')
  assert.equal(state.opened, 0)
})
