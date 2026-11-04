import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import ts from 'typescript'
import { parse, compileScript } from 'vue/compiler-sfc'
import * as vue from 'vue'

/** 以 CommonJS 方式加载 TS 工具模块（无外部依赖） */
function loadTs(relativePath, args = {}) {
  const module = { exports: {} }
  const code = ts.transpileModule(fs.readFileSync(new URL(relativePath, import.meta.url), 'utf8'), {
    compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS },
  }).outputText
  new Function('module', 'exports', ...Object.keys(args), code)(module, module.exports, ...Object.values(args))
  return module.exports
}

const protocol = loadTs('../utils/chatUiProtocol.ts')

/** 编译单文件组件，按需注入 Nuxt 自动导入的全局函数 */
function compileCard(name, globals = {}) {
  const { descriptor } = parse(fs.readFileSync(new URL(`../components/Chat/Cards/${name}.vue`, import.meta.url), 'utf8'))
  const compiled = ts.transpileModule(compileScript(descriptor, { id: name, inlineTemplate: true }).content, {
    compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS },
  }).outputText
  const module = { exports: {} }
  const requireShim = id => {
    if (id === 'vue') return vue
    if (id === '~/utils/chatUiProtocol') return protocol
    if (id === 'ant-design-vue') return { message: { warning() {}, success() {}, error() {} } }
    if (id === '@ant-design/icons-vue') {
      return new Proxy({}, { get: (_, icon) => vue.defineComponent({ name: String(icon), setup: () => () => vue.h('icon') }) })
    }
    throw new Error(`unexpected import: ${id}`)
  }
  new Function('require', 'module', 'exports', 'useI18n', 'computed', 'ref', 'watch', 'reactive', ...Object.keys(globals), compiled)(
    requireShim,
    module,
    module.exports,
    () => ({ t: (key, fallback) => fallback || key }),
    vue.computed,
    vue.ref,
    vue.watch,
    vue.reactive,
    ...Object.values(globals),
  )
  return module.exports.default
}

function mountFormCard(payload) {
  const FormConfirmCard = compileCard('FormConfirmCard')
  const seen = []
  const sent = []

  const renderer = vue.createRenderer({
    createElement: type => ({ type, props: {}, children: [] }),
    createText: text => ({ text }),
    createComment: text => ({ type: 'comment', text }),
    setText: (node, text) => { node.text = text },
    setElementText: (node, text) => { node.text = text },
    parentNode: node => node.parent,
    nextSibling: () => null,
    insert: (node, parent) => { node.parent = parent; parent.children.push(node) },
    remove: node => { node.parent.children = node.parent.children.filter(child => child !== node) },
    patchProp: (node, key, _old, value) => { node.props[key] = value },
  })

  const stub = tag => vue.defineComponent({
    name: tag,
    props: { value: null, options: null, dataSource: null, columns: null, disabled: null },
    emits: ['update:value', 'change'],
    setup(props, { slots }) {
      seen.push({ tag, props: { ...props } })
      return () => vue.h(tag, {}, slots.default?.())
    },
  })

  const app = renderer.createApp({
    setup: () => () => vue.h(FormConfirmCard, { data: payload, onSend: text => sent.push(text) }),
  })
  app.config.warnHandler = () => {}
  for (const tag of ['a-textarea', 'a-select', 'a-date-picker', 'a-input-number', 'a-input', 'a-table', 'a-tag']) {
    app.component(tag, stub(tag))
  }
  app.component('a-button', vue.defineComponent({
    props: { type: null, disabled: null },
    setup(props, { slots, attrs }) {
      return () => vue.h('button', { disabled: props.disabled, onClick: attrs.onClick }, slots.default?.())
    },
  }))

  const root = { children: [] }
  app.mount(root)
  const all = node => [node, ...(node.children || []).flatMap(all)]
  return { app, seen, sent, nodes: all(root), text: () => all(root).map(node => node.text || '').join('') }
}

test('renders a guessed payload without dropping description or detail tables', () => {
  const view = mountFormCard({
    title: '费用报销单',
    description: '请核对以下报销信息',
    fields: [
      { label: '申请日期', type: 'date', value: '2026-11-05' },
      { label: '金额', type: 'number', value: 120 },
    ],
    details: { title: '费用明细', columns: ['费用类型', '金额'], rows: [{ 费用类型: '交通费', 金额: 120 }] },
  })
  try {
    const text = view.text()
    assert.match(text, /费用报销单/)
    assert.match(text, /请核对以下报销信息/)
    assert.match(text, /费用明细/)
    // 两个字段各自独立渲染（此前缺 name 会共用同一模型键）
    assert.equal(view.seen.filter(item => item.tag === 'a-date-picker').length, 1)
    assert.equal(view.seen.filter(item => item.tag === 'a-input-number').length, 1)
    assert.equal(view.seen.filter(item => item.tag === 'a-table').length, 1)
  } finally {
    view.app.unmount()
  }
})

test('confirm emits the canonical confirmation message with real field keys', () => {
  const view = mountFormCard({
    workflow_id: '188',
    workflow_name: '物品领用申请',
    fields: [
      { name: 'sqrq', label: '申请日期', type: 'date', value: '2026-11-05' },
      { name: 'wp_type', label: '物品类型', type: 'select', value: '0', options: [{ label: '办公文具', value: '0' }] },
    ],
  })
  try {
    const nodeText = node => (node.text || '') + (node.children || []).map(nodeText).join('')
    const buttons = view.nodes.filter(node => node.type === 'button' && node.props.onClick)
    const confirmButton = buttons.find(node => nodeText(node).includes('确认创建'))
    assert.ok(confirmButton, 'expected a confirm button')
    confirmButton.props.onClick()
    assert.equal(view.sent.length, 1)
    assert.match(view.sent[0], /^【用户已确认表单】流程ID: 188/)
    assert.match(view.sent[0], /"sqrq":"2026-11-05"/)
    assert.match(view.sent[0], /"wp_type":"0"/)
  } finally {
    view.app.unmount()
  }
})

test('data_table variant renders rows and export button', () => {
  const view = mountFormCard({
    variant: 'data_table',
    title: '各部门流程数量',
    columns: ['部门', '数量'],
    rows: [{ 部门: '市场部', 数量: 12 }],
  })
  try {
    const table = view.seen.find(item => item.tag === 'a-table')
    assert.equal(table.props.dataSource.length, 1)
    assert.equal(table.props.columns[0].dataIndex, '部门')
    assert.match(view.text(), /各部门流程数量/)
  } finally {
    view.app.unmount()
  }
})
