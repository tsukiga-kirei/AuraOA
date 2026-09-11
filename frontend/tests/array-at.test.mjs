import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import vm from 'node:vm'
import ts from 'typescript'

const source = fs.readFileSync(new URL('../plugins/array-at.client.ts', import.meta.url), 'utf8')
const js = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS } }).outputText

test('旧内核先补齐 at，再执行 Nuxt 路由的 matched.at(-1)', () => {
  const context = vm.createContext({ exports: {}, defineNuxtPlugin: plugin => plugin })
  vm.runInContext('delete Array.prototype.at', context)
  vm.runInContext(js, context)
  assert.ok(context.exports.default.order < -20)
  context.exports.default.setup()
  assert.equal(vm.runInContext('({ matched: ["parent", "summary"] }).matched.at(-1)', context), 'summary')
  for (const expression of ['[1,2].at(2)', '[1,2].at(-3)', '[1].at(Infinity)', '[1].at(-Infinity)', '[].at(-1)']) {
    assert.equal(vm.runInContext(expression, context), undefined)
  }
  assert.equal(vm.runInContext('[4,5].at()', context), 4)
  assert.equal(vm.runInContext('[4,5].at(NaN)', context), 4)
  assert.equal(vm.runInContext('[4,5].at(-1.9)', context), 5)
  assert.equal(vm.runInContext('Array.prototype.at.call({0:"value",length:1},0)', context), 'value')
  assert.equal(vm.runInContext('Object.getOwnPropertyDescriptor(Array.prototype,"at").enumerable', context), false)
  assert.throws(() => vm.runInContext('Array.prototype.at.call(null,0)', context), /null or undefined/)
})

test('现代浏览器保留原生 at，重复初始化不覆盖已有实现', () => {
  const context = vm.createContext({ exports: {}, defineNuxtPlugin: plugin => plugin })
  const original = vm.runInContext('Array.prototype.at', context)
  vm.runInContext(js, context)
  context.exports.default.setup()
  context.exports.default.setup()
  assert.equal(vm.runInContext('Array.prototype.at', context), original)
})
