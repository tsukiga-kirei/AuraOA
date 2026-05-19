# OA 嵌入 AI 审核侧边栏

在泛微 E9 审批页通过 **iframe** 嵌入 AuraOA 固定页 `/embed/audit`，由 **OA 自定义 JS** 把当前流程 `requestid` 传给嵌入页，**无需用户登录 AuraOA**。

> 只读展示 AI 建议，不修改 OA 流程状态。

---

## 1. 能力概览

| 项目 | 说明 |
|------|------|
| **嵌入地址** | `https://<aura-frontend>/embed/audit`（固定，不带参数） |
| **流程编号** | OA 脚本 `WfForm.getBaseInfo().requestid` → `postMessage` |
| **用户登录** | 不需要；Nuxt 服务端用嵌入令牌代理 Go API |
| **AuraOA** | 租户规则 → **OA 嵌入审核** 开关 + `EMBED_*` 环境变量 |

---

## 2. 前置条件

1. 租户已绑定 OA 只读库  
2. 目标流程已配置 AI 审核规则并开启 **OA 嵌入审核**  
3. 迁移 `000041_embed_audit_context` 已执行  
4. `config.yaml` 与前端 `EMBED_ACCESS_TOKEN` / `EMBED_TENANT_CODE` 已配置  

---

## 3. AuraOA 配置

### 3.1 嵌入令牌

**Go** `config.yaml`：

```yaml
embed:
  tenant_code: "your-tenant-code"
  access_token: "随机长字符串"
```

**前端环境变量**（仅服务端，不暴露给浏览器）：

```bash
EMBED_ACCESS_TOKEN=与上面 access_token 相同
EMBED_TENANT_CODE=your-tenant-code
```

### 3.2 流程开关

**租户管理 → 规则配置 → 权限 → OA 嵌入审核**

---

## 4. 泛微 E9：自定义页面 JS（必配）

跨域 iframe **无法**读取 `parent.location`，必须在 OA 流程里挂脚本，用 **WfForm** 取 `requestid` 再 `postMessage` 给 AuraOA。

### 4.1 复制脚本

仓库内示例（改 `AURA_EMBED_ORIGIN` 后上传到 OA 静态目录）：

**[assets/aura-embed-notify.js](./assets/aura-embed-notify.js)**

核心逻辑（与你们现有「信贷助手」脚本风格一致）：

| 方向 | type | 说明 |
|------|------|------|
| iframe → OA | `aura-oa-request-requestid` | 嵌入页加载后主动要 requestid |
| OA → iframe | `aura-oa-requestid` | `{ requestid: '598488' }` |

OA 侧用 `WfForm.getBaseInfo().requestid`，**不必**从 URL hash 解析。

### 4.2 在流程里启用（与你截图一致）

1. 打开流程 → **基础设置** → **自定义页面**  
2. **自定义页面地址** 填上传后的路径，例如：  
   `/oa-front/workflow/AuraOA/aura-embed-notify.js?v=1`  
3. 勾选 **是否启用** → 保存  

同一流程下多个表单可共用该 js；`v=1` 用于发版后刷新缓存。

### 4.3 页面里放 iframe

iframe 的 `id` 必须与脚本里 `IFRAME_ID` 一致（默认 `aura-embed-audit`）：

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
var IFRAME_ID = 'aura-embed-audit';                  // 与 iframe id 一致
```

---

## 5. 消息时序（简图）

```
嵌入页加载
    → postMessage({ type: 'aura-oa-request-requestid' })  →  OA 父页
OA aura-embed-notify.js
    → WfForm.getBaseInfo().requestid
    → postMessage({ type: 'aura-oa-requestid', requestid })  →  iframe
嵌入页收到 requestid → 调 /api/embed/context → 展示/自动审核
```

iframe `load`、WfForm 就绪轮询、`hashchange` 时 OA 脚本会**主动再推一次**，避免切换流程后编号不更新。

---

## 6. 可选：同源读父 URL

仅当 OA 与 AuraOA **同协议、同域名、同端口** 时，嵌入页可轮询 `parent.location`；生产一般为跨域，**以第 4 节 JS 为准**。

---

## 7. 允许被 iframe 嵌入

AuraOA 前端需允许 OA 域名嵌入（CSP `frame-ancestors` 等，勿全局 `X-Frame-Options: DENY`）。

---

## 8. 常见问题

| 现象 | 处理 |
|------|------|
| 一直「正在读取 OA 流程编号」 | 检查自定义页面 js 是否启用、`AURA_EMBED_ORIGIN` 是否与 iframe src 域名一致、iframe `id` 是否匹配 |
| 控制台无 `[aura-embed]` 日志但仍失败 | 在 OA 脚本里临时 `console.log(getRequestId())` 确认 WfForm 是否已有 requestid |
| 503 嵌入未配置 | 核对 `EMBED_*` 与 `config.yaml` `embed.*` |
| Nuxt DevTools 跨域 SecurityError | 开发环境正常现象，生产可关 DevTools |

---

## 9. 相关文档

- 示例脚本：[assets/aura-embed-notify.js](./assets/aura-embed-notify.js)  
- [嵌入审核 API](../api/embed.md)  
- [OA 系统对接说明](../oa-integration.md)
