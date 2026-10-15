import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import ts from 'typescript'
import { parse, compileScript } from 'vue/compiler-sfc'
import * as vue from 'vue'

const transpile = source => ts.transpileModule(source, {
  compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS },
}).outputText
const helpers = {}
new Function('exports', transpile(fs.readFileSync(new URL('../utils/queryTrigger.ts', import.meta.url), 'utf8')))(helpers)

test('trigger field metadata selects suitable types and empty checks require no value', () => {
  for (const [field_type, type] of [['number', 'number'], ['decimal', 'number'], ['timestamp', 'date'],
    ['checkbox', 'boolean'], ['select', 'choice'], ['file', 'file'], ['textarea', 'text']]) {
    assert.equal(helpers.queryTriggerType({ field_type }), type)
  }
  assert.equal(helpers.queryTriggerType(), 'text')
  assert.equal(helpers.queryTriggerError(), undefined)
  assert.equal(helpers.queryTriggerError({ field: '', value_type: 'number', operator: 'eq' }), 'query.triggerFieldRequired')
  assert.equal(helpers.queryTriggerError({ field: 'main:amount', value_type: 'number', operator: 'gte', value: '' }), 'query.triggerValueRequired')
  assert.equal(helpers.queryTriggerError({ field: 'main:amount', value_type: 'number', operator: 'gte', value: '0' }), undefined)
  assert.equal(helpers.queryTriggerError({ field: 'main:files', value_type: 'file', operator: 'is_empty' }), undefined)
  assert.equal(helpers.queryTriggerError({ field: 'main:files', value_type: 'file', operator: 'gt', value: '1' }), 'query.triggerOperatorRequired')
})

test('trigger editor changes input controls with field type and clears stale comparisons', async () => {
  const { descriptor } = parse(fs.readFileSync(new URL('../components/QueryTriggerEditor.vue', import.meta.url), 'utf8'))
  const compiled = transpile(compileScript(descriptor, { id: 'query-trigger-test', inlineTemplate: true }).content)
  const module = { exports: {} }
  new Function('require', 'module', 'exports', 'computed', 'useI18n', compiled)(
    name => name === 'vue' ? vue : helpers, module, module.exports, vue.computed, () => ({ t: key => key }),
  )
  const renderer = vue.createRenderer({
    createElement: type => ({ type, props: {}, children: [] }),
    createText: text => ({ text }), createComment: text => ({ text }),
    setText: (node, text) => { node.text = text }, setElementText: (node, text) => { node.text = text },
    parentNode: node => node.parent,
    nextSibling: node => node.parent?.children[node.parent.children.indexOf(node) + 1] || null,
    insert: (node, parent, anchor) => {
      if (node.parent) node.parent.children = node.parent.children.filter(child => child !== node)
      node.parent = parent
      const index = anchor ? parent.children.indexOf(anchor) : -1
      if (index < 0) parent.children.push(node)
      else parent.children.splice(index, 0, node)
    },
    remove: node => { node.parent.children = node.parent.children.filter(child => child !== node) },
    patchProp: (node, key, _old, value) => { node.props[key] = value },
  })
  const trigger = vue.ref()
  const fields = ['number', 'date', 'checkbox', 'file'].map(field_type => ({ value: `main:${field_type}`, label: field_type, field_type }))
  const app = renderer.createApp({ render: () => vue.h(module.exports.default, {
    modelValue: trigger.value, fieldOptions: fields,
    'onUpdate:modelValue': value => { trigger.value = value },
  }) })
  app.config.warnHandler = () => {}
  const root = { children: [] }
  const nodes = () => { const walk = node => [node, ...(node.children || []).flatMap(walk)]; return walk(root) }
  const byType = type => nodes().find(node => node.type === type)
  const fieldSelect = () => nodes().find(node => node.props?.['aria-label'] === 'query.triggerField')
  app.mount(root)
  try {
    assert.equal(fieldSelect(), undefined)
    byType('a-switch').props.onChange(true)
    await vue.nextTick()
    fieldSelect().props.onChange('main:number')
    await vue.nextTick()
    assert.equal(trigger.value.value_type, 'number')
    byType('a-input-number').props['onUpdate:value']('5000.25')
    await vue.nextTick()
    assert.equal(trigger.value.value, '5000.25')
    fieldSelect().props.onChange('main:date')
    await vue.nextTick()
    assert.equal(trigger.value.value, '')
    assert.ok(byType('a-date-picker'))
    assert.equal(byType('a-input-number'), undefined)
    fieldSelect().props.onChange('main:checkbox')
    await vue.nextTick()
    assert.equal(trigger.value.value_type, 'boolean')
    assert.equal(byType('a-date-picker'), undefined)
    fieldSelect().props.onChange('main:file')
    await vue.nextTick()
    assert.equal(trigger.value.operator, 'is_empty')
    assert.equal(nodes().some(node => node.props?.['aria-label'] === 'query.triggerValue'), false)
    byType('a-switch').props.onChange(false)
    await vue.nextTick()
    assert.equal(trigger.value, undefined)
    assert.equal(fieldSelect(), undefined)
  } finally {
    app.unmount()
  }
})
