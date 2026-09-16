import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import ts from 'typescript'

const source = fs.readFileSync(new URL('../composables/useEmbedApi.ts', import.meta.url), 'utf8')
const js = ts.transpileModule(source, {
  compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ESNext },
}).outputText

const mod = await import(`data:text/javascript;base64,${Buffer.from(js).toString('base64')}`)
const { extractEmbedErrorMessage, isRawHttpErrorMessage } = mod

test('isRawHttpErrorMessage detects raw ofetch and http status messages', () => {
  assert.equal(isRawHttpErrorMessage('[POST] "/api/embed/execute": 403'), true)
  assert.equal(isRawHttpErrorMessage('[GET] "/api/embed/context": 500'), true)
  assert.equal(isRawHttpErrorMessage('403 Forbidden'), true)
  assert.equal(isRawHttpErrorMessage('FetchError: [POST] "/api/embed/execute": 403'), true)

  assert.equal(isRawHttpErrorMessage('当前用户无权执行该审核流程'), false)
  assert.equal(isRawHttpErrorMessage('流程未启用 OA 嵌入审核'), false)
  assert.equal(isRawHttpErrorMessage(''), false)
  assert.equal(isRawHttpErrorMessage(null), false)
})

test('extractEmbedErrorMessage extracts business message from response data', () => {
  const errWithData = {
    statusCode: 403,
    message: '[POST] "/api/embed/execute": 403',
    data: {
      code: 40302,
      message: '当前用户无权执行该审核流程',
    },
  }
  assert.equal(extractEmbedErrorMessage(errWithData), '当前用户无权执行该审核流程')
})

test('extractEmbedErrorMessage extracts business message from statusMessage', () => {
  const errWithStatusMessage = {
    statusCode: 403,
    statusMessage: '流程未启用 OA 嵌入审核',
    message: '[POST] "/api/embed/execute": 403',
  }
  assert.equal(extractEmbedErrorMessage(errWithStatusMessage), '流程未启用 OA 嵌入审核')
})

test('extractEmbedErrorMessage falls back to friendly text on raw 403 without body', () => {
  const raw403Err = {
    statusCode: 403,
    message: '[POST] "/api/embed/execute": 403',
  }
  assert.equal(extractEmbedErrorMessage(raw403Err), '当前用户无权执行该操作，请联系管理员分配流程访问权限')
})

test('extractEmbedErrorMessage falls back to friendly text on raw 401', () => {
  const raw401Err = {
    statusCode: 401,
    message: '[GET] "/api/embed/context": 401',
  }
  assert.equal(extractEmbedErrorMessage(raw401Err), '嵌入访问令牌无效或已过期，请刷新 OA 页面')
})
