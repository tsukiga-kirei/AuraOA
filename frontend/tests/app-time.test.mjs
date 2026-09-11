import test from 'node:test'
import assert from 'node:assert/strict'
import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'
for(const tz of ['UTC','Asia/Shanghai','America/Los_Angeles']){
 test(`Application time parsing is independent of device timezone: ${tz}`,()=>{
  const result=spawnSync(process.execPath,[fileURLToPath(new URL('./fixtures/app-time-check.mjs',import.meta.url))],{env:{...process.env,TZ:tz},encoding:'utf8'})
  assert.equal(result.status,0,result.stderr||result.stdout)
 })
}
