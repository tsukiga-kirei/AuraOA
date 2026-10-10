import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import ts from 'typescript'

const source = fs.readFileSync(new URL('../utils/querySqlEditor.ts', import.meta.url), 'utf8')
const js = ts.transpileModule(source, {
  compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ESNext },
}).outputText
const { highlightQuerySql, formatQuerySql } = await import(
  `data:text/javascript;base64,${Buffer.from(js).toString('base64')}`
)

test('SQL syntax highlighting escapes all source HTML', () => {
  const result = highlightQuerySql("SELECT '<img src=x onerror=alert(1)>' AS `业务说明` WHERE id=:person")
  assert.equal(result.includes('<img'), false)
  assert.equal(result.includes('&lt;img'), true)
  assert.equal(result.includes('sql-parameter'), true)
})

test('SQL formatting preserves string literals, aliases, and named parameters', () => {
  const result = formatQuerySql("SELECT 'FROM  WHERE AND' AS `采购 FROM 说明`, amount AS 金额 FROM expense WHERE person=:person AND status='完  成'")
  assert.equal(result.includes("'FROM  WHERE AND'"), true)
  assert.equal(result.includes('`采购 FROM 说明`'), true)
  assert.equal(result.includes("'完  成'"), true)
  assert.equal(result.includes(':person'), true)
  assert.equal(result.includes('\nFROM'), true)
})
