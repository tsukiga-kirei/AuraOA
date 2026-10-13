import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import ts from 'typescript'
import { parse, compileScript } from 'vue/compiler-sfc'
import * as vue from 'vue'

const transpile = (source) => ts.transpileModule(source, {
  compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS },
}).outputText
const labelsExports = {}
new Function('exports', transpile(fs.readFileSync(new URL('../utils/queryColumnLabels.ts', import.meta.url), 'utf8')))(labelsExports)
const { descriptor } = parse(fs.readFileSync(new URL('../components/QueryBuilderEditor.vue', import.meta.url), 'utf8'))
const compiled = transpile(compileScript(descriptor, { id: 'query-builder-test', inlineTemplate: true }).content)

test('output selection fills the visible alias input through the actual Vue template event', async () => {
  const columns = [{ key: 'DZYWDJRKLC', label: '抵质押物登记入库流程', data_type: 'INT' }]
  const module = { exports: {} }
  const requireStub = (name) => {
    if (name === 'vue') return vue
    if (name === 'ant-design-vue') return { message: { success() {}, error(message) { throw new Error(message) } } }
    if (name === '~/composables/useAuditDataQueryApi') return { useAuditDataQueryApi: () => ({ schema: async () => ({ columns, dialect: 'dm' }) }) }
    if (name === '~/utils/queryColumnLabels') return labelsExports
    if (name === '~/utils/queryBuilder') return {}
    return new Proxy({}, { get: () => vue.defineComponent({ render: () => null }) })
  }
  new Function('require', 'module', 'exports', 'ref', 'computed', 'watch', 'onMounted', 'useI18n', compiled)(
    requireStub, module, module.exports, vue.ref, vue.computed, vue.watch, vue.onMounted,
    () => ({ t: (key) => key }),
  )
  const renderer = vue.createRenderer({
    createElement: (type) => ({ type, props: {}, children: [] }),
    createText: (text) => ({ text }), createComment: (text) => ({ text }),
    setText: (node, text) => { node.text = text }, setElementText: (node, text) => { node.text = text },
    parentNode: (node) => node.parent, nextSibling: () => null,
    insert: (node, parent) => { node.parent = parent; parent.children.push(node) },
    remove: (node) => { node.parent.children = node.parent.children.filter(child => child !== node) },
    patchProp: (node, key, _previous, value) => { node.props[key] = value },
  })
  const updates = []
  const app = renderer.createApp(module.exports.default, {
    definition: { tables: ['formtable_main_199'], parameters: [], dialect: 'dm', builder: {
      table: 'formtable_main_199', source_type: 'table', archived_only: false,
      columns: [{ field: '', alias: '', aggregate: '' }], filters: [], order_alias: '', order_dir: 'DESC',
    } },
    'onUpdate:builder': (builder) => updates.push(builder),
  })
  app.config.warnHandler = () => {}
  app.component('a-select', vue.defineComponent({
    props: ['value', 'options'], emits: ['update:value'],
    setup: (props, { emit }) => () => vue.h('select', { options: props.options, choose: value => emit('update:value', value) }),
  }))
  const root = { children: [] }
  app.mount(root)
  try {
    await vue.nextTick()
    await vue.nextTick()
    const all = (node) => [node, ...(node.children || []).flatMap(all)]
    const select = all(root).find(node => node.type === 'select' && node.props.options?.some(option => option.value === 'DZYWDJRKLC'))
    assert.ok(select, 'field selector must use the loaded OA schema')
    select.props.choose('DZYWDJRKLC')
    await vue.nextTick()
    const alias = all(root).find(node => node.type === 'a-input' && node.props.placeholder === 'query.builder.alias')
    assert.equal(alias.props.value, '抵质押物登记入库流程')
    assert.equal(updates.at(-1).columns[0].alias, '抵质押物登记入库流程')
  } finally {
    app.unmount()
  }
})
