import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import ts from 'typescript'
import { marked } from 'marked'
import sanitizeHtml from 'sanitize-html'
const module = { exports: {} }
const source = ts.transpileModule(fs.readFileSync(new URL('../utils/markdown.ts', import.meta.url), 'utf8'), { compilerOptions: { module: ts.ModuleKind.CommonJS } }).outputText
new Function('require', 'exports', source)(name => name === 'marked' ? { marked } : { default: sanitizeHtml }, module.exports)
const { renderSafeMarkdown } = module.exports

test('model headings render during every partial update without waiting for completion', () => {
  for (const text of ['#分', '#分数', '#分数核对', '##结论']) assert.match(renderSafeMarkdown(text), /<h[12]>/)
  assert.match(renderSafeMarkdown('# 规范标题'), /<h1>规范标题<\/h1>/)
})
test('heading normalization preserves code fences and markdown sanitization', () => {
  const html = renderSafeMarkdown('```text\n#代码原文\n```\n#标题\n<script>alert(1)</script>')
  assert.match(html, /#代码原文/)
  assert.match(html, /<h1>标题<\/h1>/)
  assert.doesNotMatch(html, /<script|alert\(1\)/)
})
