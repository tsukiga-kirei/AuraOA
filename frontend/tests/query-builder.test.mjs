import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import ts from 'typescript'
const source = fs.readFileSync(new URL('../utils/queryBuilder.ts', import.meta.url), 'utf8')
const js = ts.transpileModule(source, {
  compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ESNext },
}).outputText
const { compileQueryBuilder } = await import(
  `data:text/javascript;base64,${Buffer.from(js).toString('base64')}`
)
const schema = ['requestid', 'person', 'amount', 'expense_date'].map((key) => ({
  key,
  label: key,
  data_type: 'text',
}))
const base = () => ({
  source_type: 'workflow',
  table: 'formtable_main_123',
  columns: [
    { field: 'person', alias: '申请人', aggregate: '' },
    { field: 'amount', alias: '累计金额', aggregate: 'SUM' },
  ],
  filters: [
    {
      field: 'expense_date',
      operator: '>=',
      parameter: 'start_date',
      label: '开始日期',
      type: 'date',
    },
  ],
  order_alias: '累计金额',
  order_dir: 'DESC',
})

test('workflow aggregate uses archived EXISTS without multiplying source rows', () => {
  const result = compileQueryBuilder(base(), 'mysql', schema)
  assert.match(result.sql, /SUM\(d\.`amount`\) AS `累计金额`/)
  assert.match(result.sql, /EXISTS \(SELECT 1 FROM `workflow_requestbase`/)
  assert.match(result.sql, /currentnodetype` = '3'/)
  assert.match(result.sql, /GROUP BY d\.`person`/)
  assert.deepEqual(result.tables, ['formtable_main_123', 'workflow_requestbase'])
  assert.deepEqual(result.parameters, [{ name: 'start_date', label: '开始日期', type: 'date' }])
})
test('Oracle dates use explicit conversion and identifiers preserve Chinese aliases', () => {
  const result = compileQueryBuilder(base(), 'oracle', schema)
  assert.match(result.sql, /d\."EXPENSE_DATE" >= TO_DATE\(:start_date, 'YYYY-MM-DD'\)/)
  assert.match(result.sql, /AS "累计金额"/)
})
test('model table count and list conditions retain bind placeholders', () => {
  const builder = base()
  builder.source_type = 'table'
  builder.columns = [{ field: '*', alias: '记录数', aggregate: 'COUNT' }]
  builder.order_alias = ''
  builder.filters = [
    {
      field: 'person',
      operator: 'IN',
      parameter: 'people',
      label: '人员列表',
      type: 'string_list',
    },
  ]
  const result = compileQueryBuilder(builder, 'dm', schema)
  assert.match(result.sql, /COUNT\(\*\) AS "记录数"/)
  assert.match(result.sql, /IN \(:people\)/)
  assert.equal(result.sql.includes('EXISTS'), false)
})
test('unsafe identifiers, unknown fields, duplicate aliases and invalid list types fail closed', () => {
  for (const change of [
    (b) => {
      b.table = 'foo;DELETE'
    },
    (b) => {
      b.columns[0].field = 'unknown'
    },
    (b) => {
      b.columns[1].alias = b.columns[0].alias
    },
    (b) => {
      b.columns[0].alias = '名字` FROM x'
    },
    (b) => {
      b.filters[0].operator = 'IN'
    },
    (b) => {
      b.order_alias = '不存在'
    },
  ]) {
    const builder = base()
    change(builder)
    assert.throws(() => compileQueryBuilder(builder, 'mysql', schema))
  }
})
