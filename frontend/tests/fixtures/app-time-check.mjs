import assert from 'node:assert/strict'
import fs from 'node:fs'
import vm from 'node:vm'
import { createRequire } from 'node:module'
import ts from 'typescript'
const source=fs.readFileSync(new URL('../../utils/appTime.ts',import.meta.url),'utf8')
const js=ts.transpileModule(source,{compilerOptions:{target:ts.ScriptTarget.ES2022,module:ts.ModuleKind.CommonJS,esModuleInterop:true}}).outputText
let timeZone='Asia/Shanghai'
const exports={}
vm.runInNewContext(js,{exports,require:createRequire(import.meta.url),useRuntimeConfig:()=>({public:{timeZone}})})
const {appDayjs,formatDateTimeInAppZone}=exports
for(const input of ['2026-09-11 16:43','2026-09-11T16:43:00','2026/09/11 16:43']){
 assert.equal(appDayjs(input).toISOString(),'2026-09-11T08:43:00.000Z')
 assert.equal(formatDateTimeInAppZone(input,'en-GB',{dateStyle:'short',timeStyle:'short'}),'11/09/2026, 16:43')
}
assert.equal(appDayjs('2026-09-11').toISOString(),'2026-09-10T16:00:00.000Z')
assert.equal(appDayjs('2026-09-11T08:43:00Z').toISOString(),'2026-09-11T08:43:00.000Z')
assert.equal(appDayjs('2026-09-11T10:43:00+02:00').toISOString(),'2026-09-11T08:43:00.000Z')
assert.equal(appDayjs(new Date('2026-09-11T08:43:00Z')).toISOString(),'2026-09-11T08:43:00.000Z')
assert.equal(appDayjs(0).valueOf(),0)
assert.notEqual(formatDateTimeInAppZone(0),'-')
assert.equal(formatDateTimeInAppZone(null),'-')
assert.equal(formatDateTimeInAppZone('not-a-date'),'not-a-date')
timeZone='America/New_York'
assert.equal(appDayjs('2026-03-08 00:00').toISOString(),'2026-03-08T05:00:00.000Z')
assert.equal(appDayjs('2026-03-09 00:00').toISOString(),'2026-03-09T04:00:00.000Z')
assert.equal(appDayjs('2026-11-01 00:00').toISOString(),'2026-11-01T04:00:00.000Z')
assert.equal(appDayjs('2026-11-02 00:00').toISOString(),'2026-11-02T05:00:00.000Z')
