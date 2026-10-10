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
