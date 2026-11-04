import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import ts from 'typescript'
import { parse, compileScript } from 'vue/compiler-sfc'
import * as vue from 'vue'

function loadTs(relativePath) {
  const module = { exports: {} }
  const code = ts.transpileModule(fs.readFileSync(new URL(relativePath, import.meta.url), 'utf8'), {
    compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS },
  }).outputText
  new Function('module', 'exports', code)(module, module.exports)
  return module.exports
}

const protocol = loadTs('../utils/chatUiProtocol.ts')

const chartCalls = []

function compileChartCard() {
  const { descriptor } = parse(fs.readFileSync(new URL('../components/Chat/Cards/ChartCard.vue', import.meta.url), 'utf8'))
  const compiled = ts.transpileModule(compileScript(descriptor, { id: 'ChartCard', inlineTemplate: true }).content, {
    compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS },
  }).outputText

  const stubs = {
    BarChartOutlined: vue.defineComponent({ setup: () => () => vue.h('icon') }),
    LineChartOutlined: vue.defineComponent({ setup: () => () => vue.h('icon') }),
    PieChartOutlined: vue.defineComponent({ setup: () => () => vue.h('icon') }),
    RadarChartOutlined: vue.defineComponent({ setup: () => () => vue.h('icon') }),
    TableOutlined: vue.defineComponent({ setup: () => () => vue.h('icon') }),
    CameraOutlined: vue.defineComponent({ setup: () => () => vue.h('icon') }),
    DownloadOutlined: vue.defineComponent({ setup: () => () => vue.h('icon') }),
    DotChartOutlined: vue.defineComponent({ setup: () => () => vue.h('icon') }),
  }

  const requireShim = id => {
    if (id === 'vue') return vue
    if (id === '~/utils/chatUiProtocol') return protocol
    if (id === 'ant-design-vue') return { message: { warning() {}, success() {}, error() {} } }
    if (id === '@ant-design/icons-vue') {
      return new Proxy(stubs, { get: (target, name) => target[name] || vue.defineComponent({ setup: () => () => vue.h('icon') }) })
    }
    if (id === 'echarts/core' || id === 'echarts/charts' || id === 'echarts/components' || id === 'echarts/renderers') {
      return new Proxy({}, { get: (_, name) => (name === 'use' ? () => {} : {}) })
    }
    if (id === 'vue-echarts') {
      return {
        default: vue.defineComponent({
          props: { option: null, style: null },
          setup(props) {
            chartCalls.push(props.option)
            return () => vue.h('v-chart')
          },
        }),
      }
    }
    throw new Error(`unexpected import: ${id}`)
  }

  const module = { exports: {} }
  new Function('require', 'module', 'exports', 'useI18n', 'useThemeColors', compiled)(
    requireShim,
    module,
    module.exports,
    () => ({ t: (key, fallback) => fallback || key }),
    () => ({ chartColors: vue.ref({}), isDark: vue.ref(false) }),
  )
  return module.exports.default
}

function mountChart(payload) {
  chartCalls.length = 0
  const ChartCard = compileChartCard()
  const renderer = vue.createRenderer({
    createElement: type => ({ type, props: {}, children: [], style: {} }),
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

  const seen = []
  const app = renderer.createApp({
    setup: () => () => vue.h(ChartCard, { data: payload }),
  })
  app.config.warnHandler = () => {}
  for (const tag of ['a-table', 'a-dropdown', 'a-menu', 'a-menu-item']) {
    app.component(tag, vue.defineComponent({
      props: { dataSource: null, columns: null },
      setup(props, { slots }) {
        seen.push({ tag, props: { ...props } })
        return () => vue.h(tag, {}, [slots.default?.(), slots.overlay?.()])
      },
    }))
  }
  app.component('a-button', vue.defineComponent({ setup: (_, { slots }) => () => vue.h('button', {}, slots.default?.()) }))
  const root = { children: [] }
  app.mount(root)
  const all = node => [node, ...(node.children || []).flatMap(all)]
  return { app, seen, text: () => all(root).map(node => node.text || '').join(' ') }
}

test('bar payload renders axis categories and numeric series', () => {
  const view = mountChart({
    type: 'bar',
    title: '各部门流程发起量',
    xAxis: ['市场部', '研发部'],
    series: [{ name: '数量', data: [142, 98] }],
  })
  try {
    assert.match(view.text(), /各部门流程发起量/)
    const option = chartCalls.at(-1)
    assert.deepEqual(option.xAxis.data, ['市场部', '研发部'])
    assert.deepEqual(option.series[0].data, [142, 98])
  } finally {
    view.app.unmount()
  }
})

test('pie payload built from [{name,value}] objects renders named slices', () => {
  const view = mountChart({
    type: 'pie',
    title: '流程办结状态占比',
    series: [{ name: '状态占比', data: [{ name: '已归档', value: 660141 }, { name: '流转中', value: 17092 }] }],
  })
  try {
    const option = chartCalls.at(-1)
    assert.deepEqual(option.series[0].data.map(item => item.name), ['已归档', '流转中'])
    assert.deepEqual(option.series[0].data.map(item => item.value), [660141, 17092])
  } finally {
    view.app.unmount()
  }
})

test('radar payload with singular indicator still gets filled max values', () => {
  const view = mountChart({
    type: 'radar',
    title: '部门负载',
    indicator: [{ name: '行政规范科' }, { name: '管理部' }],
    series: [{ name: '待办', data: [795, 520] }],
  })
  try {
    const option = chartCalls.at(-1)
    assert.deepEqual(option.radar.indicator.map(item => item.name), ['行政规范科', '管理部'])
    assert.ok(option.radar.indicator.every(item => item.max >= 795))
  } finally {
    view.app.unmount()
  }
})

test('payload without dimensions shows an empty state instead of a blank canvas', () => {
  const view = mountChart({ type: 'bar', title: '空图' })
  try {
    assert.equal(chartCalls.length, 0)
    assert.match(view.text(), /暂无可绘制的数据/)
  } finally {
    view.app.unmount()
  }
})
