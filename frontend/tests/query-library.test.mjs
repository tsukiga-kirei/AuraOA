import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import ts from 'typescript'

const source = fs.readFileSync(
  new URL('../composables/useAuditDataQueryApi.ts', import.meta.url),
  'utf8',
)
const js = ts.transpileModule(source, {
  compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ESNext },
}).outputText
const { useAuditDataQueryApi } = await import(
  `data:text/javascript;base64,${Buffer.from(js).toString('base64')}`
)

test('unpaged library loads every server page and retains keyword filtering', async () => {
  const entries = Array.from({ length: 235 }, (_, i) => ({ id: String(i) }))
  const calls = []
  globalThis.useAuth = () => ({
    authFetch: async (url, { query }) => {
      calls.push({ url, ...query })
      const offset = (query.page - 1) * query.page_size
      return { items: entries.slice(offset, offset + query.page_size), total: entries.length }
    },
  })
  try {
    assert.deepEqual(await useAuditDataQueryApi().listAll('预算'), entries)
    assert.deepEqual(
      calls.map(({ page }) => page),
      [1, 2, 3],
    )
    assert.ok(calls.every(({ keyword, page_size }) => keyword === '预算' && page_size === 100))
  } finally {
    delete globalThis.useAuth
  }
})

test('unpaged library reports later-page failure instead of displaying a partial directory', async () => {
  globalThis.useAuth = () => ({
    authFetch: async (_url, { query }) => {
      if (query.page === 2) throw new Error('读取失败')
      return { items: Array.from({ length: 100 }, (_, i) => ({ id: String(i) })), total: 150 }
    },
  })
  try {
    await assert.rejects(useAuditDataQueryApi().listAll(), /读取失败/)
  } finally {
    delete globalThis.useAuth
  }
})

const builderSource = fs.readFileSync(
  new URL('../utils/queryBuilder.ts', import.meta.url),
  'utf8',
)
const builderJs = ts.transpileModule(builderSource, {
  compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ESNext },
}).outputText
const { compileQueryBuilder } = await import(
  `data:text/javascript;base64,${Buffer.from(builderJs).toString('base64')}`
)

const labelSource = fs.readFileSync(new URL('../utils/queryColumnLabels.ts', import.meta.url), 'utf8')
const labelJs = ts.transpileModule(labelSource, {
  compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ESNext },
}).outputText
const { fillQueryBuilderLabels, fillQueryFieldNotes } = await import(
  `data:text/javascript;base64,${Buffer.from(labelJs).toString('base64')}`
)

test('reloading schema fills empty output and filter labels across identifier casing', () => {
  const columns = [{ key: 'AMOUNT', label: '金额', data_type: 'NUMBER' }]
  const builder = {
    columns: [{ field: 'amount', alias: ' ', aggregate: '' }, { field: 'AMOUNT', alias: '累计金额', aggregate: 'SUM' }],
    filters: [{ field: 'Amount', label: '' }, { field: 'amount', label: '最低金额' }],
  }
  fillQueryBuilderLabels(builder, columns)
  assert.deepEqual(builder.columns.map(c => c.alias), ['金额', '累计金额'])
  assert.deepEqual(builder.filters.map(f => f.label), ['金额', '最低金额'])
  fillQueryBuilderLabels(builder, [{ ...columns[0], label: '新金额' }])
  assert.equal(builder.columns[0].alias, '金额', 'reloading must retain an existing alias')
})

test('schema notes use Chinese defaults and preserve manual descriptions with different casing', () => {
  const notes = { 'FORMTABLE_MAIN_199.amount': '实际支付金额', 'other.amount': '另一张表' }
  const result = fillQueryFieldNotes(notes, 'formtable_main_199', [
    { key: 'AMOUNT', label: '金额' },
    { key: 'PERSON', label: '申请人' },
    { key: 'UNKNOWN', label: 'UNKNOWN' },
  ])
  assert.deepEqual(result, {
    'formtable_main_199.AMOUNT': '实际支付金额',
    'formtable_main_199.PERSON': '申请人',
    'other.amount': '另一张表',
  })
  assert.equal(notes['FORMTABLE_MAIN_199.amount'], '实际支付金额', 'source notes must not be mutated')
})

test('compileQueryBuilder handles varchar date without TO_DATE and native date with TO_DATE in oracle/dm', () => {
  const schema = [
    { key: 'sqrq', label: '申请日期', data_type: 'VARCHAR2' },
    { key: 'created_time', label: '创建时间', data_type: 'TIMESTAMP' },
    { key: 'nf', label: '年份', data_type: 'VARCHAR2' },
  ]
  const builder = {
    table: 'formtable_main_1',
    columns: [{ field: 'sqrq', alias: '申请日期', aggregate: '' }],
    filters: [
      {
        field: 'sqrq',
        operator: '>=',
        parameter: 'filter_1',
        label: '起始日期',
        type: 'date',
        binding: { source: 'system', value: 'year_start' },
      },
      {
        field: 'created_time',
        operator: '>=',
        parameter: 'filter_2',
        label: '创建时间',
        type: 'date',
        binding: { source: 'system', value: 'year_start' },
      },
      {
        field: 'nf',
        operator: '=',
        parameter: 'filter_3',
        label: '年份',
        type: 'string',
        binding: { source: 'system', value: 'current_year' },
      },
    ],
    order_alias: '',
    order_dir: 'DESC',
  }
  const result = compileQueryBuilder(builder, 'oracle', schema)
  assert.ok(result.sql.includes('d."SQRQ" >= :filter_1'), 'VARCHAR2 date column should not wrap TO_DATE')
  assert.ok(result.sql.includes('d."CREATED_TIME" >= TO_DATE(:filter_2, \'YYYY-MM-DD\')'), 'TIMESTAMP column should wrap TO_DATE')
  assert.ok(result.sql.includes('d."NF" = :filter_3'), 'current_year string column should match parameter')
  assert.equal(result.parameters.find(p => p.name === 'filter_3')?.default_binding?.value, 'current_year')
})
