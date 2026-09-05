// 独立页面验收服务：全部账号、流程及回复均为测试数据，不连接业务系统。
import http from 'node:http'
import { randomUUID } from 'node:crypto'
const date = () => new Date().toISOString()
const tenant = {id:'11111111-1111-4111-8111-111111111111',name:'界面验收空间',code:'preview'}
const user = {id:'22222222-2222-4222-8222-222222222222',username:'preview',display_name:'界面验收',email:'',phone:'',locale:'zh-CN'}
const roles = ['business','tenant_admin','system_admin'].map((role,index)=>({id:String(index),role,tenant_id:tenant.id,tenant_name:tenant.name,label:role==='business'?'业务用户':role==='tenant_admin'?'租户管理员':'系统管理员'}))
let role = roles[0]
const token = () => 'eyJhbGciOiJub25lIn0.'+Buffer.from(JSON.stringify({sub:user.id,exp:Math.floor(Date.now()/1000)+86400,active_role:role})).toString('base64url')+'.preview'
const paths = ['/overview','/dashboard','/archive','/summary','/chat','/settings','/admin/tenant/agents','/admin/tenant/org','/admin/system/tenants']
const menus = () => paths.map(path=>({key:path,path,label:path}))
const agents = [{id:'a1',agent_code:'oa_query',name:'流程查询助手',description:'查找待办与审批记录，把分散的信息整理成清晰的答案。',is_system:true},{id:'a2',agent_code:'oa_assist',name:'审核协作助手',description:'结合审核规则与流程资料，协助你完成分析与决策。',is_system:true},{id:'a3',agent_code:'expense_analyst',name:'费用分析专员',description:'专注差旅与费用报销，按你的配置组合技能与查询工具。',is_system:false}]
const sessions = [{id:'s1',agent_id:'a1',agent_code:'oa_query',title:'本周费用报销与待办梳理',pinned:false,created_at:date(),updated_at:date()},{id:'s2',agent_id:'a2',agent_code:'oa_assist',title:'采购合同的审核要点',pinned:false,created_at:date(),updated_at:date()}]
const answer = `## 先处理这两项待办

本周共有 **3 项费用流程**需要关注。其中两项资料完整，可以优先处理；另一项还需要补充说明。

| 流程 | 金额 | 当前进展 | 建议 |
| --- | ---: | --- | --- |
| 华东项目差旅报销 | ¥ 2,860 | 部门负责人审核 | 核对行程后处理 |
| 客户拜访交通费 | ¥ 480 | 财务复核 | 票据完整 |
| 项目会议费用 | ¥ 1,200 | 补充材料 | 等待费用明细 |

### 建议的处理顺序

1. **核对差旅报销**：行程和票据日期一致，重点确认项目归属。
2. **完成交通费复核**：现有附件能够对应费用记录。
3. **跟进会议费用明细**：补齐参会人员与费用用途后再继续。

> 这些建议依据本次读取的流程信息整理，最终处理请结合实际业务情况。

### 可以直接使用的沟通草稿

请补充会议参会人员、费用用途及费用明细，方便继续完成复核。

你也可以继续问我：**“展开第一笔报销的审批记录”**。`
const toolCalls = [{tool_call_id:'c1',tool_code:'skill:expense_review',ui_kind:'skill',status:'success',payload:{content:'按费用类型、票据完整度和审批进度组织分析。'}},{tool_call_id:'c2',tool_code:'list_my_todos',ui_kind:'todo_list',status:'success',arguments:'{"page":1,"page_size":20}',payload:{items:[{process_id:'P1024',title:'华东项目差旅报销',applicant_name:'张宁',current_node_name:'部门负责人审核'},{process_id:'P1025',title:'客户拜访交通费',applicant_name:'陈悦',current_node_name:'财务复核'}]}},{tool_call_id:'c3',tool_code:'mcp:knowledge:search_policy',ui_kind:'mcp_generic',status:'success',payload:{content:[{type:'text',text:'已读取差旅费用说明中的交通费与住宿费条目。'}]}}]
const history = {s1:[{id:'m1',session_id:'s1',role:'user',content:'帮我梳理本周的费用报销，哪些需要优先处理？',status:'success',created_at:date()},{id:'m2',session_id:'s1',role:'assistant',content:answer,reasoning_content:'先读取待办，再按资料完整度与审批节点整理建议。',tool_calls:toolCalls,token_usage:{total_tokens:1836},status:'success',created_at:date()}],s2:[]}
const ok = (res,data) => {res.setHeader('Content-Type','application/json');res.end(JSON.stringify({code:0,message:'success',data}))}
const server = http.createServer(async(req,res)=>{
 const url = new URL(req.url,'http://localhost'); const path=url.pathname
 if(!path.startsWith('/api/')) { const proxy=http.request({hostname:'127.0.0.1',port:3100,path:req.url,method:req.method,headers:req.headers}, upstream=>{res.writeHead(upstream.statusCode,upstream.headers);upstream.pipe(res)});proxy.on('error',()=>{res.statusCode=502;res.end('Preview app is starting')});req.pipe(proxy);return }
 let raw='';for await(const chunk of req)raw+=chunk; const body=raw?JSON.parse(raw):{}
 if(path.includes('bootstrap'))return ok(res,{needs_setup:false})
 if(path==='/api/tenants/list')return ok(res,[tenant])
 if(path==='/api/auth/login'){role=roles.find(r=>r.role===body.preferred_role)||roles[0];return ok(res,{access_token:token(),refresh_token:token(),user,roles,active_role:role,permissions:[role.role]})}
 if(path==='/api/auth/me')return ok(res,{user,roles,active_role:role,permissions:[role.role],page_permissions:paths,org_roles:[],tenant_name:tenant.name})
 if(path==='/api/auth/menu')return ok(res,{menus:menus()})
 if(path==='/api/auth/switch-role'){role=roles.find(r=>r.id===body.role_id)||roles[0];return ok(res,{access_token:token(),active_role:role,permissions:[role.role],menus:menus()})}
 if(path==='/api/chat/agents')return ok(res,agents)
 if(path==='/api/chat/sessions'){
  if(req.method==='POST'){const a=agents.find(a=>a.agent_code===body.agent_code);if(!a){res.statusCode=400;return ok(res,{})};const s={id:randomUUID(),agent_id:a.id,agent_code:a.agent_code,title:'新对话',pinned:false,created_at:date(),updated_at:date()};sessions.unshift(s);history[s.id]=[];return ok(res,s)}
  const items=sessions.filter(s=>s.title.includes(url.searchParams.get('keyword')||''));return ok(res,{items,total:items.length,page:1,page_size:30})
 }
 const match=path.match(/^\/api\/chat\/sessions\/([^/]+)(.*)$/)
 if(match){const id=match[1],suffix=match[2];const session=sessions.find(s=>s.id===id)
  if(suffix==='/messages/stream'){
   res.writeHead(200,{'Content-Type':'text/event-stream','Cache-Control':'no-cache'});const send=(event,data)=>res.write(`event: ${event}\ndata: ${JSON.stringify(data)}\n\n`)
   history[id].push({id:randomUUID(),session_id:id,role:'user',content:body.content,created_at:date()})
   for(const tool of toolCalls){send('tool_start',{...tool,status:'running',payload:null});await new Promise(r=>setTimeout(r,150));send('tool_result',tool)}
   for(const content of answer.match(/.{1,35}/gs)||[]){if(res.destroyed)break;send('delta',{content});await new Promise(r=>setTimeout(r,40))}
   history[id].push({id:randomUUID(),session_id:id,role:'assistant',content:answer,tool_calls:toolCalls,status:'success',created_at:date()});session.title=body.content.slice(0,24);send('session',{session_id:id,title:session.title});send('done',{token_usage:{total_tokens:1836}});res.end();return
  }
  if(req.method==='PATCH'){session.title=body.title;return ok(res,{})}
  if(req.method==='DELETE'){sessions.splice(sessions.indexOf(session),1);return ok(res,{})}
  return ok(res,{session,messages:history[id]||[]})
 }
 if(path==='/api/tenant/settings/oa-jump-config')return ok(res,{enabled:false})
 if(path==='/api/audit/stats')return ok(res,{pending_ai_count:3})
 if(path==='/api/tenant/agents')return ok(res,agents.map(a=>({...a,enabled:true,system_prompt:'协助用户查询与分析流程信息。',tool_codes:['list_my_todos','get_process']})))
 if(path==='/api/tenant/mcp-servers'||path==='/api/tenant/skills')return ok(res,[])
 if(path.endsWith('agent-catalog'))return ok(res,{tool_catalog:[],agent_catalog:agents,skill_catalog:[]})
 return ok(res,{items:[],total:0,unread_count:0})
})
server.listen(3101,'127.0.0.1',()=>console.log('Isolated UI fixture: http://127.0.0.1:3101 (preview / preview)'))
