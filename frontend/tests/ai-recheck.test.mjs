import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import ts from 'typescript'
import { parse, compileScript } from 'vue/compiler-sfc'
import * as vue from 'vue'

function compileComponent(name, globals = {}) {
  const { descriptor } = parse(fs.readFileSync(new URL(`../components/${name}.vue`, import.meta.url), 'utf8'))
  const compiled = ts.transpileModule(compileScript(descriptor, { id: name, inlineTemplate: true }).content, {
    compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS },
  }).outputText
  const module = { exports: {} }
  new Function('require', 'module', 'exports', 'computed', 'ref', 'watch', 'useI18n', ...Object.keys(globals), compiled)(
    id => id === 'vue' ? { ...vue, Transition: vue.BaseTransition } : id.includes('useI18n') ? { useI18n: () => ({ t: key => key }) } : id.includes('markdown') ? { renderSafeMarkdown: text => text } : new Proxy({}, { get: (_, name) => () => vue.h('icon', { name }) }), module, module.exports, vue.computed, vue.ref, vue.watch, () => ({ t: key => key }), ...Object.values(globals),
  )
  return module.exports.default
}

function mountConfig(initial, extraProps = {}) {
  const composable = { exports: {} }
  const systemVariableCode = ts.transpileModule(fs.readFileSync(new URL('../composables/usePromptSystemVariables.ts', import.meta.url), 'utf8'), {
    compilerOptions: { module: ts.ModuleKind.CommonJS },
  }).outputText
  new Function('module', 'exports', 'computed', 'useI18n', systemVariableCode)(composable, composable.exports, vue.computed, () => ({ t: key => key }))
  const component = compileComponent('AIRecheckConfigEditor', { usePromptSystemVariables: composable.exports.usePromptSystemVariables })
  const renderer = vue.createRenderer({
    createElement: type => ({ type, props: {}, children: [] }), createText: text => ({ text }), createComment: text => ({ type: 'comment', text }),
    setText: (node, text) => { node.text = text }, setElementText: (node, text) => { node.text = text },
    parentNode: node => node.parent, nextSibling: () => null,
    insert: (node, parent) => { node.parent = parent; parent.children.push(node) },
    remove: node => { node.parent.children = node.parent.children.filter(child => child !== node) },
    patchProp: (node, key, _old, value) => { node.props[key] = value },
  })
  const selected = vue.ref(initial), updates = []
  const app = renderer.createApp({ setup: () => () => vue.h(component, {
    modelValue: selected.value, defaultSystem: '默认系统提示词', defaultUser: '{{initial_result}}', ...extraProps,
    'onUpdate:modelValue': value => { selected.value = value; updates.push(value) },
  }) })
  app.config.warnHandler = () => {}
  app.component('a-switch', vue.defineComponent({ props: ['checked'], emits: ['change'], setup: (props, { emit }) => () => vue.h('switch', { checked: props.checked, toggle: value => emit('change', value) }) }))
  app.component('PromptTextarea', vue.defineComponent({ props: ['value'], emits: ['update:value'], setup: (props, { emit, expose }) => {
    let start, end
    expose({ insertAtCursor: variable => {
      const text = props.value || '', from = start ?? text.length, to = end ?? from
      emit('update:value', text.slice(0, from) + variable + text.slice(to))
      start = end = from + variable.length
    } })
    return () => vue.h('textarea', { value: props.value, edit: value => emit('update:value', value), select: (from, to = from) => { start = from; end = to } })
  } }))
  app.component('PromptVariableBar', compileComponent('PromptVariableBar'))
  app.component('a-tooltip', vue.defineComponent({ setup: (_, { slots }) => () => vue.h('tooltip', {}, slots.default?.()) }))
  app.component('a-button', vue.defineComponent({ setup: (_, { attrs, slots }) => () => vue.h('button', attrs, slots.default?.()) }))
  const root = { children: [] }; app.mount(root)
  const all = node => [node, ...(node.children || []).flatMap(all)]
  return { app, selected, updates, find: type => all(root).filter(node => node.type === type) }
}

test('opening legacy config does not enable or mutate it; explicit switch uses independent defaults', async () => {
  const view = mountConfig(undefined)
  try {
    assert.equal(view.updates.length, 0)
    assert.equal(view.find('switch')[0].props.checked, false)
    assert.equal(view.find('textarea').length, 0)
    view.find('switch')[0].props.toggle(true); await vue.nextTick()
    assert.equal(view.selected.value.enabled, true)
    assert.equal(view.selected.value.system_prompt, '默认系统提示词')
    assert.equal(view.find('textarea').length, 2)
    view.find('textarea')[0].props.edit('租户自定义复核'); await vue.nextTick()
    view.find('switch')[0].props.toggle(false); await vue.nextTick()
    assert.equal(view.selected.value.system_prompt, '租户自定义复核')
    view.find('switch')[0].props.toggle(true); await vue.nextTick()
    assert.equal(view.find('textarea')[0].props.value, '租户自定义复核')
  } finally { view.app.unmount() }
})

test('shared variable buttons insert optional data and dates into their own editors at the remembered selection', async () => {
  const view = mountConfig({ enabled: true, system_prompt: '系统说明', user_prompt: '前文后文' })
  const label = node => node.type === 'comment' ? '' : (node.text || '') + (node.children || []).map(label).join('')
  const buttons = key => view.find('button').filter(node => label(node).trim() === key)
  try {
    for (const key of ['rules', 'initial_result', 'audit_context', 'original_system_prompt', 'reasoning_result', 'current_date', 'current_time', 'current_datetime', 'weekday']) {
      assert.equal(buttons(`{{${key}}}`).length, 2, `both editors offer ${key}`)
    }
    view.find('textarea')[1].props.select(2)
    buttons('{{audit_context}}')[1].props.onClick(); await vue.nextTick()
    assert.equal(view.selected.value.user_prompt, '前文{{audit_context}}后文')
    assert.equal(view.selected.value.system_prompt, '系统说明')
    view.find('textarea')[0].props.select(0, 2)
    buttons('{{current_date}}')[0].props.onClick(); await vue.nextTick()
    assert.equal(view.selected.value.system_prompt, '{{current_date}}说明')
    assert.equal(view.selected.value.user_prompt, '前文{{audit_context}}后文')
  } finally { view.app.unmount() }
})

test('reset restores both recheck prompts without disabling recheck', async () => {
  const view = mountConfig({ enabled: true, system_prompt: '系统草稿', user_prompt: '用户草稿' })
  try {
    view.find('button')[0].props.onClick(); await vue.nextTick()
    assert.deepEqual(JSON.parse(JSON.stringify(view.selected.value)), { enabled: true, system_prompt: '默认系统提示词', user_prompt: '{{initial_result}}' })
  } finally { view.app.unmount() }
})


test('the read-only code panel displays the backend contract verbatim, outside the page explanation', () => {
  const contract = '实际输出约定\n{"type":"object","oneOf":[]}'
  const view = mountConfig({ enabled: true, system_prompt: '正文', user_prompt: '{{rules}}\n{{initial_result}}' }, { outputInstructions: contract })
  try {
    assert.equal(view.find('pre')[0].text, contract)
    assert.equal(view.find('aside').length, 1)
    assert.ok(view.find('icon').some(node => node.props.name === 'SafetyCertificateOutlined'))
    assert.equal(view.selected.value.system_prompt, '正文')
  } finally { view.app.unmount() }
})

function mountReadOnlyComponent(name, initial) {
  const props = vue.reactive(initial)
  const renderer = vue.createRenderer({
    createElement: type => ({ type, props: {}, children: [] }), createText: text => ({ text }), createComment: text => ({ type: 'comment', text }),
    setText: (node, text) => { node.text = text }, setElementText: (node, text) => { node.text = text },
    parentNode: node => node.parent, nextSibling: () => null,
    insert: (node, parent) => { node.parent = parent; parent.children.push(node) }, remove: node => { node.parent.children = node.parent.children.filter(child => child !== node) },
    patchProp: (node, key, _old, value) => { node.props[key] = value },
  })
  const component = compileComponent(name, { useId: vue.useId })
  const app = renderer.createApp({ setup: () => () => vue.h(component, props) })
  app.config.warnHandler = () => {}
  app.component('AiMarkdownStream', vue.defineComponent({ props: ['text', 'format'], setup: props => () => vue.h('stream', { format: props.format }, props.text) }))
  const root = { children: [] }; app.mount(root)
  const all = node => [node, ...(node.children || []).flatMap(all)]
  return { app, props, find: type => all(root).filter(node => node.type === type) }
}

test('unchanged and reused results show no final recheck card; actual modifications show one', async () => {
  const view = mountReadOnlyComponent('AIRecheckResult', { record: { status: 'completed', changed: false, reuse_original: true } })
  try {
    assert.equal(view.find('section').length, 0)
    view.props.record = { status: 'completed', changed: true, original_result: { overall_score: 95, recommendation: 'approve' }, final_result: { overall_score: 75, recommendation: 'review' }, reason: '纠正对应关系' }
    await vue.nextTick()
    assert.equal(view.find('section').length, 1)
    const labels = view.find('strong')
    const scores = view.find('b')
    assert.ok(labels[0].props.class.includes('recheck-result__label--success'))
    assert.ok(scores[0].props.class.includes('recheck-result__score--success'))
    assert.ok(labels[1].props.class.includes('recheck-result__label--warning'))
    assert.ok(scores[1].props.class.includes('recheck-result__score--warning'))

    view.props.record = { status: 'completed', changed: false, original_result: {}, final_result: {} }
    await vue.nextTick()
    assert.equal(view.find('section').length, 0)
  } finally { view.app.unmount() }
})

test('the next output stage collapses the previous stage and still allows reopening it', async () => {
  const view = mountReadOnlyComponent('ReviewStageStreams', { result: { id: 'demo', status: 'reasoning', ai_reasoning: '#分析' } })
  try {
    assert.equal(view.find('button')[0].props['aria-expanded'], true)
    view.props.result.structured_output = '{"overall_score":80}'
    view.props.result.status = 'extracting'
    await vue.nextTick()
    assert.equal(view.find('button')[0].props['aria-expanded'], false)
    assert.equal(view.find('button')[1].props['aria-expanded'], true)
    view.props.result.recheck_output = '{"reuse_original":true}'
    view.props.result.status = 'rechecking'
    await vue.nextTick()
    assert.equal(view.find('button')[1].props['aria-expanded'], false)
    assert.equal(view.find('button')[2].props['aria-expanded'], true)
    view.find('button')[0].props.onClick(); await vue.nextTick()
    assert.equal(view.find('button')[0].props['aria-expanded'], true)
  } finally { view.app.unmount() }
})

test('recheck retry opens the last stage immediately, before the first response chunk arrives', () => {
  const view = mountReadOnlyComponent('ReviewStageStreams', { result: { id: 'retry', status: 'rechecking', ai_reasoning: '# 初审分析', structured_output: '{"overall_score":96}', recheck_output: '' } })
  try {
    assert.equal(view.find('button').length, 3)
    assert.equal(view.find('button')[0].props['aria-expanded'], false)
    assert.equal(view.find('button')[1].props['aria-expanded'], false)
    assert.equal(view.find('button')[2].props['aria-expanded'], true)
  } finally { view.app.unmount() }
})

test('manual expansion and collapse override stage following, and a new run restores defaults', async () => {
  const view = mountReadOnlyComponent('ReviewStageStreams', { result: { id: 'first', status: 'reasoning', deep_thinking: '思考正文', ai_reasoning: '分析正文' } })
  const expanded = () => view.find('button').map(button => button.props['aria-expanded'])
  try {
    assert.deepEqual(expanded(), [false, true])
    view.find('button')[0].props.onClick(); await vue.nextTick()
    assert.deepEqual(expanded(), [true, true])
    view.props.result.status = 'extracting'; view.props.result.structured_output = '{'
    await vue.nextTick()
    assert.deepEqual(expanded(), [true, false, true])
    view.find('button')[2].props.onClick(); await vue.nextTick()
    view.props.result.structured_output = '{"overall_score":96}'
    await vue.nextTick()
    assert.deepEqual(expanded(), [true, false, false])
    view.props.result.status = 'rechecking'; await vue.nextTick()
    assert.deepEqual(expanded(), [true, false, false, true])
    view.props.result.recheck_output = '{"reuse_original":true}'
    await vue.nextTick()
    assert.deepEqual(expanded(), [true, false, false, true])
    view.props.result = { id: 'second', status: 'reasoning', deep_thinking: '新思考', ai_reasoning: '新分析' }
    await vue.nextTick()
    assert.deepEqual(expanded(), [false, true])
  } finally { view.app.unmount() }
})

test('uncontrolled result panels still toggle, while controlled panels follow saved choices', async () => {
  for (const [name, props] of [
    ['ResultReasoningPanel', { title: '推理分析', text: '正文' }],
    ['ResultRecheckPanel', { record: { status: 'completed', reuse_original: true } }],
  ]) {
    const view = mountReadOnlyComponent(name, props)
    try {
      assert.equal(view.find('button')[0].props['aria-expanded'], false)
      assert.ok(!view.find('section')[0].props.class?.includes(`${name.replace(/([a-z])([A-Z])/g, '$1-$2').toLowerCase()}--expanded`))
      view.find('button')[0].props.onClick(); await vue.nextTick()
      assert.equal(view.find('button')[0].props['aria-expanded'], true)
      assert.ok(view.find('section')[0].props.class?.includes(`${name.replace(/([a-z])([A-Z])/g, '$1-$2').toLowerCase()}--expanded`))
      view.find('button')[0].props.onClick(); await vue.nextTick()
      assert.equal(view.find('button')[0].props['aria-expanded'], false)
      view.props.expanded = true; await vue.nextTick()
      assert.equal(view.find('button')[0].props['aria-expanded'], true)
      assert.ok(view.find('section')[0].props.class?.includes(`${name.replace(/([a-z])([A-Z])/g, '$1-$2').toLowerCase()}--expanded`))
      view.props.expanded = false; await vue.nextTick()
      assert.equal(view.find('button')[0].props['aria-expanded'], false)
    } finally { view.app.unmount() }
  }
})
