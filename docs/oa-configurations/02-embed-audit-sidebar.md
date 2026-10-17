# OA 嵌入 AI 审核侧边栏

在泛微 E9 审批页通过 **iframe** 嵌入 AuraOA 固定页 `/embed/audit`，由 **OA 自定义 JS** 把当前流程 `requestid` 传给嵌入页，**无需用户登录 AuraOA**。

> 只读展示 AI 建议，不修改 OA 流程状态。

---

## 1. 能力概览

| 项目 | 说明 |
|------|------|
| **嵌入地址** | `https://<aura-frontend>/embed/audit`（固定，不带参数） |
| **流程编号** | OA 脚本 `WfForm.getBaseInfo().requestid` → `postMessage` |
| **用户登录** | 不需要；OA 父页传 `embed_token`，Nuxt 写入 httpOnly Cookie 后代理 Go API |
| **AuraOA** | 租户管理 → **OA 嵌入** 生成密钥；规则配置 → **OA 嵌入审核/总结** 开关 |

---

## 2. 前置条件

1. 租户已绑定 OA 只读库  
2. 目标流程已配置 AI 审核规则并开启 **OA 嵌入审核**  
3. 迁移 `000046_tenant_embed_config`、`000051_embed_change_detection` 已执行
4. 在 **系统管理 → 租户管理 → OA 嵌入** 中已生成租户嵌入密钥  

---

## 3. AuraOA 配置

### 3.1 租户嵌入密钥

路径：**系统管理 → 租户管理 → 选择租户 → OA 嵌入**

1. 开启 **启用 OA 嵌入**
2. 点击 **生成密钥** 或 **重置密钥**
3. 立即复制弹窗中的明文密钥（仅显示一次）
4. 将密钥配置到 OA 脚本 `aura-embed-notify.js` 的 `EMBED_ACCESS_TOKEN`

页面同时展示：

- 租户名称 / `tenant_code` / `tenant_id`
- 嵌入地址：`/embed/audit`、`/embed/summary`

### 3.2 流程开关

**租户管理 → 规则配置 → 权限 → OA 嵌入审核**

同一位置可配置自动刷新：

| 开关 | 默认 | 说明 |
|------|------|------|
| 首次打开自动审核 | 开 | 没有历史结果时执行 |
| 业务数据变化后自动审核 | 开 | 只比较审核实际使用的主表、明细和附件版本 |
| 退回或重新提交后自动审核 | 开 | 退回链路单独处理 |
| 普通审批推进后自动审核 | 关 | 批准、批注、转发、抄送或节点变化；开启会增加 Token 消耗 |

附件换版使用主表附件 `docId` 以及 `DocImageFile` 最新 `versionid` / `imagefileid`
共同判断；判断阶段不会下载附件或调用 MinerU。

---

## 4. 泛微 E9：自定义页面 JS（必配）

跨域 iframe **无法**读取 `parent.location`，必须在 OA 流程里挂脚本，用 **WfForm** 取 `requestid` 再 `postMessage` 给 AuraOA。

### 4.1 复制脚本

仓库内示例（改 `AURA_EMBED_ORIGIN` 后上传到 OA 静态目录）：

**[assets/aura-embed-notify.js](./assets/aura-embed-notify.js)**

核心逻辑：

| 方向 | type | 说明 |
|------|------|------|
| iframe → OA | `aura-oa-request-requestid` | 嵌入页加载后主动要 requestid |
| OA → iframe | `aura-oa-requestid` | `{ requestid: '598488', embed_token: 'aura_emb_...' }` |
| OA JS → AuraOA | `POST /api/embed/events` | OA 点击保存或提交时直接安排后台审核和总结检查 |

脚本在页面就绪后直接注册 `WfForm.OPER_SAVE` 和 `WfForm.OPER_SUBMIT`，不创建隐藏 iframe。
已有流程直接传点击时冻结的
`WfForm.getBaseInfo().requestid`；首次新建流程没有 requestid 时，同时传 workflowid 和人员诊断标识，
由 AuraOA 记录操作前高水位并在后台解析。人员标识只辅助多候选消歧，不等同于流程创建人。
事件使用唯一嵌入密钥通过简单异步 POST 发送，最多等待 800ms；请求完成、超时或 AuraOA 不可用都会
放行 OA。事件只安排延迟检查，不等待 AI，浏览器也不轮询 requestid。

### 4.2 在流程里启用

1. 打开流程 → **基础设置** → **自定义页面**  
2. **自定义页面地址** 填上传后的路径，例如：  
   `/oa-front/workflow/AuraOA/aura-embed-notify.js?v=1`  
3. 勾选 **是否启用** → 保存  

同一流程下多个表单可共用该 js；`v=1` 用于发版后刷新缓存。

### 4.3 页面里放 iframe

iframe 的 `id` 必须包含在脚本里 `IFRAME_IDS` 中（默认包含 `aura-embed-audit`；若同页还嵌入流程总结，也可同时包含 `aura-embed-summary`）：

```html
<iframe
  id="aura-embed-audit"
  title="AI 审核"
  src="https://aura.example.com/embed/audit"
  style="width:380px;height:100%;border:0;"
></iframe>
```

脚本通过「自定义页面」自动加载，**无需**再在 HTML 里单独 `<script src=...>`（若门户页不在流程表单内，则需在门户 HTML 单独引入同一 js）。

### 4.4 修改脚本顶部两项

```javascript
var AURA_EMBED_ORIGIN = 'https://aura.example.com'; // 改成实际 AuraOA 前端地址
var EMBED_ACCESS_TOKEN = 'aura_emb_...'; // 租户管理 → OA 嵌入 → 生成密钥
var IFRAME_IDS = ['aura-embed-audit', 'aura-embed-summary'];
```

---

## 5. 移动端集成（方案B：状态胶囊按钮 + 弹窗查看详情）

泛微 E9 移动端（EMobile、企业微信移动审批）**没有 PC 端那样的常驻协同区/侧边栏**。直接在移动端表单嵌入常驻 iframe 会破坏表单排版。因此移动端采用**状态胶囊按钮 + 弹窗抽屉**的最佳实践集成。

### 5.1 移动端交互与时序

```
移动端表单加载
    → 读取 WfForm.getBaseInfo().requestid 与人员标识
    → 自动注册保存/提交事件（WfForm.OPER_SAVE / OPER_SUBMIT 变更感知）
    → 异步请求 GET /api/embed/context 轻量预检状态
    → 渲染状态胶囊按钮到 #getMyBt 容器（红/黄/绿指示灯）
    → 若脚本 `AUTO_RUN_BEFORE_OPEN = true` 且预检 `should_auto_audit`，后台 POST execute（`form_open`）
审批人点击胶囊按钮
    → 调用 weaJs.showDialog(...) 打开全屏/抽屉弹窗
    → URL 自带 ?requestid=...&embed_token=...&oa_user_id=...
    → AuraOA 嵌入页直接初始化展示审核结论与规则明细
    → 嵌入页将运行中/完成状态 postMessage 给 OA 父页（电脑端 iframe 弹窗可即时改灯）
审批人关闭弹窗或回到表单
    → 监听 pageshow / visibilitychange / focus，以及 weaJs 关闭回调
    → 再次 GET /api/embed/context?prefer_cached=true 刷新胶囊灯色
    → 若任务仍在运行，则短时轮询直到出结论或超时
```

### 5.2 状态指示灯定义

状态按钮采用按状态区分的低饱和色底、轻阴影、浅色状态图标和独立分数区，末尾箭头表示可打开详情；首次渲染按“按钮出现 → 图标绘制 → 文字/分数/箭头出现”的顺序过渡，并以低频呼吸光保持可见性。错误状态使用稳重的正红色，避免与普通提示色混淆。
移动端与开启 `SHOW_ON_DESKTOP` 的 PC 弹窗入口共用外观；触屏点击区至少 44px 高，
悬浮位置适配底部安全区，双按钮在窄屏自动换行。已有 OA 部署需重新导出并替换脚本、更新引用版本参数后生效。

| 状态 | 颜色 | 按钮文案示例 | 含义说明 |
|:---|:---|:---|:---|
| **通过** | 绿色对勾 | `审核通过 · 95分` | AI 建议批准 |
| **关注** | 琥珀色实心叹号 | `建议关注 · 78分` | AI 建议人工复核，存在提示项 |
| **预警** | 红色叉号 | `建议退回 · 45分` | 触发高风险规则或阻断项 |
| **审核中** | 蓝色单星旋转，可点开 | `审核分析中...` | AI 正在分析，点击可查看进度 |
| **加载中** | 蓝色半环持续转圈 | `加载中...` | 正在读取状态，暂不可点 |
| **待审核/无结论**| 灰色星形 | `AI审核详情` | 尚未生成审核结论，点击弹窗可手动发起 |
| **未配置** | 灰色锁形 | `未开启审核` | 该流程类型尚未在 AuraOA 中启用审核 |

### 5.3 移动端配置与部署

1. **导出移动端脚本**：
   在 **系统管理 → 租户管理 → 选择租户 → OA 嵌入**：
   - 终端类型切换至 **移动端（状态按钮+弹窗）**
   - 选择 **流程审核**、**流程总结** 或 **全部功能**；全部功能会展示审核与总结两个独立状态按钮，未启用的功能保留为不可用入口，不会启动任务。
   - 点击 **导出移动端脚本**，获取已注入当前租户 Origin 与 Token 的审核脚本 `aura-embed-audit-mobile-notify.js`、总结脚本 `aura-embed-summary-mobile-notify.js` 或全部功能脚本 `aura-embed-mobile-notify.js`。总结脚本只查询总结配置，不依赖审核规则。
   - 状态按钮默认只读取状态；待生成或正在分析时仍可点击，由详情页启动自动分析或接续任务。
   - 配置区分别提供 `BUTTON_STYLE_MOBILE` 与 `BUTTON_STYLE_DESKTOP`：`'modern'` 为柔光按钮，`'classic'` 为经典风格，默认均为柔光；未识别的值回退经典风格。风格按设备标识选择，与窗口宽度无关；电脑端仍须开启 `SHOW_ON_DESKTOP` 才显示。
   - 柔光风格沿用演示页的瓷白胶囊、主题色轮廓、局部柔光与连续 SVG 描边。审核结果采用“结论在上、评分在下”（不带“分”），总结显示“总结”，分析只显示“分析中”；小屏按钮高 56px，其他尺寸高 64px，图标与字体同步适配。
   - 柔光结果通过 `BUTTON_EXPANDED_MOBILE` / `BUTTON_EXPANDED_DESKTOP` 分别控制默认展开：移动端默认 `false`，展示结果动画后收起；电脑端默认 `true`。设为 `false` 时电脑端悬停或键盘聚焦可展开，移动端可用入口单击打开详情。
   - 柔光加载中始终只有花瓣流转图标，不展开。辅助状态默认只显示图标，电脑端悬停/聚焦展开一行说明；移动端可用的待生成、详情或执行异常入口单击进入详情，未开启、保存后可用及无效状态点击只展示说明。展开本身不查询、不重试、不触发审核；点空白处或按 Esc 收起，不显示额外提示气泡。
   - 左侧停靠时图标固定左端、内容向右展开；右侧停靠镜像为图标固定右端、内容向左展开。文字裁切、光效和触碰回弹支点随方向同步切换，避免图标跟随收起过程横向滚动。
   - 所有柔光状态在动画开始前用同主题的隐藏副本按最终文案、图标和评分位数确定目标宽度，动画期间解除各状态的最小宽度钳制；加载、分析、结果在同一个按钮上从当前长度直接伸缩，不重复弹出，动画结束保留该宽度，避免印章落定或文字计数后再次补长；视口变化时重新计算以适配小屏。右侧停靠的扫光、高亮、落印与粒子方向同步镜像，粒子从真实图标端点发出。
   - 柔光结果按实际状态播放不同反馈：通过纸屑、关注扫光与评分高亮、退回分数下落、叉号压入、红色光环向内收拢并落印（无纸屑、无回弹）、总结文档描绘。重复轮询结果不重播；状态改变、页面离开或电脑端打开弹窗时清理旧效果；遵循系统“减少动态效果”设置。
   - 经典风格保持原有紧凑图标/评分、主题色小光晕与悬停/移动端长按提示，既有点击逻辑不变。
   - 移动端始终使用泛微 `weaJs.showDialog`，保留 `moduleName: workflow`、全屏尺寸。打开详情期间按钮保持显示和可拖动，不依赖关闭回调恢复可见性，也不设置等待关闭才能释放的点击锁；确认 `callback` 与关闭/取消 `onCancel` 仅用于刷新对应功能状态；移动端不注入自定义 iframe 弹层、不锁定 OA 页面滚动、不播放弹层卷曲动画。
   - 已移除 `USE_NATIVE_MOBILE_DIALOG` 开关，避免移动端绕过泛微集成逻辑。缺少 `weaJs.showDialog` 时保留原有 `window.open` 独立页面降级；原生 API 抛错时保留按钮并提示重试。旧脚本需重新导出、覆盖并更新引用版本参数。
   - 脚本配置区的 `var AUTO_RUN_BEFORE_OPEN = false;` 默认点开详情才审。改为 `true` 后，预检到需要自动审/总结时会在打开详情前发起后台任务（`trigger_detail=form_open`），按钮变为「审核分析中... / 总结分析中...」，点开可查看进度。仍受流程配置「打开即审 / 数据变化」等开关约束；没有 `requestid` 的编辑态不会预审。
   - 审核/总结结束后，详情页会通知父页；关闭弹窗或回到表单时脚本会再次拉取状态，刷新胶囊灯色。需重新导出并覆盖 OA 脚本（更新 `?v=`）后生效。
   - 更新脚本后覆盖 OA 静态文件并更新引用版本参数（例如 `?v=5`），避免继续加载旧缓存。
   - 仓库静态模板见：[assets/aura-embed-mobile-notify.js](./assets/aura-embed-mobile-notify.js)
2. **上传与启用**：
   - 将脚本上传至 OA 静态目录（如 `/oa-front/workflow/AuraOA/aura-embed-mobile-notify.js`）
   - 在流程 **基础设置 → 自定义页面**（或移动端表单脚本设置）中填入地址并勾选启用
3. **表单挂载位（可选）**：
   - 表单设计器中预留 `<div id="getMyBt"></div>`（与 `oa-front` 既有业务流程标准统一）
   - 若表单未配置该元素，脚本会按终端分别读取 `FLOAT_POSITION_MOBILE` / `FLOAT_POSITION_DESKTOP` 创建浮动按钮。支持四个角：`'top-left'`（左上角）、`'top-right'`（右上角）、`'bottom-left'`（左下角，默认）、`'bottom-right'`（右下角）。上方位置避开工具栏，四个角均适配安全区；无效值回退左下角，手动拖动后按拖动位置显示

按钮风格可以分别配置，例如移动端使用柔光、电脑端使用经典：

```javascript
var BUTTON_STYLE_MOBILE = 'modern';
var BUTTON_STYLE_DESKTOP = 'classic';
var FLOAT_POSITION_MOBILE = 'bottom-right'; // 移动端右下角
var FLOAT_POSITION_DESKTOP = 'top-left'; // 电脑端左上角
// 仅当需要电脑端也使用本脚本时开启：
var SHOW_ON_DESKTOP = true;
```

4. **控制电脑端是否启用这份移动端脚本**：
   - 脚本配置区的 `var SHOW_ON_DESKTOP = false;` 默认让电脑端跳过按钮、状态请求和事件注册；电脑端继续使用原 PC 嵌入通知脚本与侧栏。
   - 改为 `true` 后，电脑端也显示按钮，仍按“存在 `getMyBt` 则嵌入，否则按 `FLOAT_POSITION_DESKTOP` 悬浮”的规则定位。
   - 电脑端点击后使用居中详情弹窗，宽度为 `min(740px, 94vw)`，高度为当前可视区域的 86%，最大高度 880px；电脑端保留卷曲展开/收起效果，移动端由泛微原生弹窗管理。
   - 终端判断依据浏览器设备标识并兼容 iPad 桌面标识，不会因电脑窗口或侧栏较窄而启用移动端按钮。
   - 若同时开启 `AUTO_RUN_BEFORE_OPEN`，电脑端弹窗入口也会在进入表单后预审，不占用详情页交互队列。
   - 若遇到 `matched.at is not a function`，需更新部署 AuraOA 前端以加载路由初始化前的兼容补丁；仅替换 OA 脚本不能修复嵌入页的浏览器兼容性。

---

## 6. PC 端消息时序（简图）

```
OA 页面加载
    → 直接注册保存/提交事件
点击保存或提交
    → 立即冻结 requestid/workflow_id/人员标识/occurred_at_ms
    → 简单异步 POST /api/embed/events（action: save_requested | submit_requested）
    → 延迟读取 OA → 按指纹决定是否执行 AI
可见嵌入页加载
    → postMessage({ type: 'aura-oa-request-requestid' })  →  OA 父页
OA aura-embed-notify.js
    → WfForm.getBaseInfo().requestid
    → postMessage({ type: 'aura-oa-requestid', requestid, embed_token })  →  iframe
嵌入页收到 requestid + embed_token
    → POST /api/embed/session
    → 调 /api/embed/context
    → 在展示旧结果前轻量比较 OA/规则指纹（不识别附件正文、不调用 AI）
    → 未变化则展示已有结果，已变化或已有后台任务则进入交互队列
```

iframe `load`、WfForm 就绪轮询、`hashchange` 时 OA 脚本会**主动再推一次**，避免切换流程后编号不更新。

---

## 7. 可选：同源读父 URL

仅当 OA 与 AuraOA **同协议、同域名、同端口** 时，嵌入页可轮询 `parent.location`；生产一般为跨域，**以第 4 节与第 5 节 JS 为准**。

---

## 布局预览与滚动边界

- `/embed/audit-preview` 提供独立的虚拟数据预览，可切换电脑、侧栏、手机宽度（也可拖动宽度滑块）、审核结论、长内容、空评论、分页、身份只读及异常场景。
- 预览复用正式页的 `EmbedAuditPage`、规则、推理和评论组件。模拟重新审核、流式分析及评论增删改仅作用于当前页面内存，不访问 OA、业务 API 或 AI；切换场景或刷新后重置。审核中场景停留在推理阶段，点击重新审核可观看完整模拟进度。
- 正式 `/embed/audit` 保持原有嵌入鉴权、双视角、配置版本、SSE 与 API 行为。本次没有接口契约变更。
- 弹窗内最右侧滚动条属于 iframe 的审核文档；若 OA 外侧仍有滚动条，那一条由 OA 自身容器控制，跨域嵌入页无法修改。本次样式只作用于审核文档，未修改 OA 弹窗脚本、工作台、归档或流程总结页。
- 审核文档使用覆盖式细圆角滑块，取消原生纵向滚动槽，避免白色边条；正文横向滚动条保持同色系细圆角样式。滚动停止约 800ms 后隐藏滑块，拖动或键盘操作期间保留显示；页面滚轮、触摸滚动以及滑块拖动、方向键、翻页键、Home/End 均可使用。
- AI 推理分析、深度思考和审核中的流式正文采用整页滚动；已完成结果的标题在正文经过视口时吸顶，随时可收起。Markdown 表格和代码块仍可在内容过宽时横向滚动。
- 姓名部门与流程节点以内容宽度决定是否同排，不够时自然换行，长文字不省略。评论默认展示已有内容，“分享看法”才展开满意度与编辑区；收起保留未发送草稿，编辑自己的评论会自动展开输入。
- 视角切换保留灰色滑轨与白色选中滑块，按完整文案宽度左对齐排列，不等宽挤压；窄屏自然换行。“执行配置 vN”保留蓝色标签样式。
- 规则版本有更新时统一显示“有新版审核规则”，不再同时显示“结论可能已过期”。后端 `stale` 单独出现时显示“审核相关信息有变化”，只提示变化，不直接要求重审，也不将原因归结为表单内容修改；与规则更新同时出现时，仍只展示一条规则更新提示，详情补充相关信息变化说明。提示与是否允许使用新版重新执行分开判断，操作权限保持原有配置。
- 审批日志、节点变化是否触发自动审核，由规则配置的“普通审批推进后自动审核”（`auto_audit_on_flow_change`，默认关闭）控制，并非由提示词变量是否引用决定；提示词变量用于组织 AI 输入。本次仅调整页面表现，没有修改这些后端判断或开关。
- 本次按需求未运行测试、类型检查、生产构建或自动化交互验收，预览供人工查看。

## 8. 允许被 iframe 嵌入

AuraOA 前端需允许 OA 域名嵌入（CSP `frame-ancestors` 等，勿全局 `X-Frame-Options: DENY`）。

---

## 9. 多租户说明

同一套 AuraOA 实例可为多个租户分别生成嵌入密钥。运行时由 `embed_token` 识别租户，再由 `process_id → process_type` 命中该租户下的流程规则配置。

---

## 10. 常见问题

| 现象 | 处理 |
|------|------|
| 一直「正在读取 OA 流程编号」 | 检查自定义页面 js 是否启用、`AURA_EMBED_ORIGIN` 是否与 iframe src 域名一致、iframe `id` 是否匹配 |
| 移动端看不到状态按钮 | 检查移动端自定义脚本是否加载，表单是否有 `#getMyBt` 占位，或按 `FLOAT_POSITION_MOBILE` 查看配置的角落是否有悬浮按钮 |
| 移动端按钮点击无反应 | 检查移动端环境是否支持 `weaJs.showDialog`，或查看控制台弹窗报错 |
| 提示缺少嵌入访问令牌 | 检查 OA 脚本是否配置 `EMBED_ACCESS_TOKEN`，以及 postMessage 或 URL 参数是否携带 `embed_token` |
| 401 嵌入访问令牌无效 | 在租户管理中重置密钥，并同步更新 OA 脚本 |
| Nuxt DevTools 跨域 SecurityError | 开发环境正常现象，生产可关 DevTools |

---

## 11. 相关文档

- PC 端示例脚本：[assets/aura-embed-notify.js](./assets/aura-embed-notify.js)  
- 移动端示例脚本：[assets/aura-embed-mobile-notify.js](./assets/aura-embed-mobile-notify.js)  
- [嵌入审核 API](../api/embed.md)  
- [OA 系统对接说明](../oa-integration.md)
