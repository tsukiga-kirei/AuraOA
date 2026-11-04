import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import ts from 'typescript'

/** 以 CommonJS 方式加载前端的卡片协议工具（纯 TS，无外部依赖） */
function loadProtocolModule() {
  const module = { exports: {} }
  const code = ts.transpileModule(
    fs.readFileSync(new URL('../utils/chatUiProtocol.ts', import.meta.url), 'utf8'),
    { compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS } },
  ).outputText
  new Function('module', 'exports', code)(module, module.exports)
  return module.exports
}

const { parseChatChunks, normalizeFormConfirmPayload, normalizeChartPayload } = loadProtocolModule()

test('parses markdown and card blocks into ordered chunks', () => {
  const content = [
    '先给结论。',
    ':::form-confirm',
    '{"variant":"summary_confirm","workflow_name":"物品领用申请","fields":[]}',
    ':::',
    '补充说明。',
    ':::chart',
    '{"type":"bar","categories":["A"],"series":[{"name":"n","data":[1]}]}',
    ':::',
  ].join('\n')

  const chunks = parseChatChunks(content)
  assert.deepEqual(chunks.map(chunk => chunk.type), ['markdown', 'form-confirm', 'markdown', 'chart'])
  assert.equal(chunks[1].data.variant, 'summary_confirm')
  assert.equal(chunks[3].data.type, 'bar')
})

test('tolerates code fences, block aliases, trailing commas and spaced markers', () => {
  const content = [
    '::: form_confirm',
    '```json',
    '{',
    '  "workflow_name": "报销单",',
    '  "fields": [{ "label": "金额", "type": "number", "value": 120 },],',
    '}',
    '```',
    ':::',
  ].join('\n')

  const chunks = parseChatChunks(content)
  assert.equal(chunks.length, 1)
  assert.equal(chunks[0].type, 'form-confirm')
  assert.equal(chunks[0].data.workflow_name, '报销单')
  assert.equal(chunks[0].data.fields[0].name, 'field_0')
})

test('reports unclosed blocks as pending and malformed json as invalid instead of swallowing them', () => {
  const streaming = parseChatChunks(':::chart\n{"type":"bar",')
  assert.equal(streaming.at(-1).type, 'pending-block')

  const broken = parseChatChunks(':::form-confirm\n{not json}\n:::')
  assert.equal(broken[0].type, 'invalid-block')
  assert.equal(broken[0].content, '{not json}')
})

test('normalizes the guessed form payload into the renderer contract', () => {
  // 复现历史问题写法：title/description/details、fields 只有 label 没有 name
  const payload = normalizeFormConfirmPayload({
    title: '费用报销单',
    description: '请核对以下报销信息',
    workflow_id: '188',
    fields: [
      { label: '申请日期', type: 'datetime', value: '2026/11/5 10:00:00' },
      { label: '报销类型', type: 'dropdown', value: '差旅费', options: ['差旅费', '招待费'] },
      { label: '金额', type: 'number', value: '120.5' },
    ],
    details: {
      title: '费用明细',
      columns: ['费用类型', '金额'],
      rows: [{ 费用类型: '交通费', 金额: 120 }],
    },
  })

  assert.equal(payload.variant, 'summary_confirm')
  assert.equal(payload.workflow_name, '费用报销单')
  assert.equal(payload.description, '请核对以下报销信息')
  assert.deepEqual(payload.fields.map(field => field.name), ['field_0', 'field_1', 'field_2'])
  assert.deepEqual(payload.fields.map(field => field.type), ['date', 'select', 'number'])
  assert.equal(payload.fields[0].value, '2026-11-05')
  assert.equal(payload.fields[1].value, '差旅费')
  assert.deepEqual(payload.fields[1].options, [
    { label: '差旅费', value: '差旅费' },
    { label: '招待费', value: '招待费' },
  ])
  assert.equal(payload.details.length, 1)
  assert.equal(payload.details[0].title, '费用明细')
  assert.equal(payload.details[0].rows.length, 1)
})

test('maps select display labels back to their stored option values', () => {
  const payload = normalizeFormConfirmPayload({
    fields: [
      {
        name: 'wp_type',
        label: '物品类型',
        type: 'select',
        value: '办公文具',
        options: [
          { label: '办公文具', value: '0' },
          { label: '数码配件', value: '1' },
        ],
      },
    ],
  })
  assert.equal(payload.fields[0].value, '0')
})

test('treats rows without fields as a data_table variant', () => {
  const payload = normalizeFormConfirmPayload({
    columns: ['部门', '数量'],
    rows: [{ 部门: '市场部', 数量: 12 }],
  })
  assert.equal(payload.variant, 'data_table')
  assert.equal(payload.columns[0].dataIndex, '部门')
  assert.equal(payload.rows.length, 1)
})

test('normalizes chart aliases: xAxis, pie object points and singular indicator', () => {
  const bar = normalizeChartPayload({
    type: '柱状图',
    title: '部门发起量',
    xAxis: ['市场部', '研发部'],
    series: [{ name: '数量', data: [142, 98] }],
  })
  assert.equal(bar.type, 'bar')
  assert.deepEqual(bar.categories, ['市场部', '研发部'])
  assert.deepEqual(bar.series[0].data, [142, 98])

  const pie = normalizeChartPayload({
    type: 'pie',
    series: [{ name: '占比', data: [{ name: '已归档', value: 660141 }, { name: '流转中', value: 17092 }] }],
  })
  assert.deepEqual(pie.categories, ['已归档', '流转中'])
  assert.deepEqual(pie.series[0].data, [660141, 17092])

  const radar = normalizeChartPayload({
    type: 'radar',
    indicator: [{ name: '行政规范科' }, { name: '管理部' }],
    series: [{ name: '待办', data: [795, 520] }],
  })
  assert.deepEqual(radar.indicators.map(item => item.name), ['行政规范科', '管理部'])
  // 缺 max 时按数据量级自动补齐，避免雷达图空白
  assert.ok(radar.indicators.every(item => item.max >= 795))
})

test('keeps table columns/rows on chart payloads for the table view', () => {
  const chart = normalizeChartPayload({
    type: 'bar',
    columns: ['部门', '数量'],
    rows: [{ 部门: '市场部', 数量: 12 }],
  })
  assert.equal(chart.columns.length, 2)
  assert.equal(chart.rows.length, 1)
})
