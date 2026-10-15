import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import ts from 'typescript'
import { parse, compileScript } from 'vue/compiler-sfc'
import * as vue from 'vue'

const { descriptor } = parse(fs.readFileSync(new URL('../components/RuleDataQueryEditor.vue', import.meta.url), 'utf8'))
const compiled = ts.transpileModule(compileScript(descriptor, { id: 'query-reference-test', inlineTemplate: true }).content, {
  compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS },
}).outputText

test('inherited bindings expand only on override and restore without changing library defaults', async () => {
  const definition = { name: '历史报销', parameters: [
    { name: 'start', label: '开始日期', type: 'date', default_binding: { source: 'system', value: 'year_start', days_offset: -7 } },
    { name: 'company', label: '付款公司', type: 'string', default_binding: { source: 'field' } },
  ] }
  const original = JSON.stringify(definition)
  const module = { exports: {} }
  const sourceEditor = vue.defineComponent({
    props: ['modelValue'], emits: ['update:modelValue'],
    setup: (props, { emit }) => () => vue.h('source-editor', {
      binding: props.modelValue,
      change: value => emit('update:modelValue', value),
    }),
  })
  const requireStub = name => {
    if (name === 'vue') return vue
    if (name === './QueryValueSourceEditor.vue') return { default: sourceEditor }
    if (name === '~/composables/useAuditDataQueryApi') return {
      useAuditDataQueryApi: () => ({ versions: async () => [{ version_no: 1, definition }] }),
    }
    if (name === 'ant-design-vue') return { message: { error: error => { throw new Error(error) } } }
    return new Proxy({}, { get: () => vue.defineComponent({ render: () => null }) })
  }
  new Function('require', 'module', 'exports', 'ref', 'computed', 'watch', 'useI18n', 'useAuth', compiled)(
    requireStub, module, module.exports, vue.ref, vue.computed, vue.watch,
    () => ({ t: key => key }),
    () => ({ menus: vue.ref([{ path: '/admin/tenant/memory' }]) }),
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
  const references = vue.ref([{ query_id: 'query', version_no: 1, enabled: true, bindings: {
    company: { source: 'field', field: 'main:company' },
  } }])
  const app = renderer.createApp({ render: () => vue.h(module.exports.default, {
    modelValue: references.value, fieldOptions: [],
    'onUpdate:modelValue': value => { references.value = value },
  }) })
  app.config.warnHandler = () => {}
  const root = { children: [] }
  const nodes = () => { const walk = node => [node, ...(node.children || []).flatMap(walk)]; return walk(root) }
  const textOf = node => (node.text || '') + (node.children || []).map(textOf).join('')
  const button = key => nodes().find(node => node.type === 'a-button' && textOf(node).includes(key))
  app.mount(root)
  try {
    await new Promise(resolve => setImmediate(resolve))
    await vue.nextTick()
    await vue.nextTick()
    assert.equal(nodes().filter(node => node.type === 'source-editor').length, 1, 'only workflow fields need editing initially')
    assert.ok(button('query.overrideBinding'))
    button('query.overrideBinding').props.onClick()
    await vue.nextTick()
    assert.deepEqual(references.value[0].bindings.start, definition.parameters[0].default_binding)
    const editor = nodes().find(node => node.type === 'source-editor' && node.props.binding.source === 'system')
    editor.props.change({ ...editor.props.binding, days_offset: 3 })
    await vue.nextTick()
    assert.equal(references.value[0].bindings.start.days_offset, 3)
    assert.equal(JSON.stringify(definition), original, 'override must not mutate the published definition')
    button('query.resetBinding').props.onClick()
    await vue.nextTick()
    assert.equal('start' in references.value[0].bindings, false)
    assert.equal(references.value[0].bindings.company.field, 'main:company')
    assert.equal(nodes().filter(node => node.type === 'source-editor').length, 1)
  } finally {
    app.unmount()
  }
})
