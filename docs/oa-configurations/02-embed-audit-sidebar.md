# OA 嵌入 AI 审核侧边栏

在泛微 E9 审批页通过 **iframe** 嵌入 AuraOA **固定展示页**，页面自动从 **OA 父页面 URL** 解析 `requestid`，**无需用户登录 AuraOA**。

> 只读展示 AI 建议，不修改 OA 流程状态。

---

## 1. 能力概览

| 项目 | 说明 |
|------|------|
| **固定嵌入地址** | `https://<aura-frontend>/embed/audit`（**不要**带 `process_id` 参数） |
| **流程识别** | 页面加载后主动读取 `window.parent.location.href` 中的 `requestid` |
| **用户登录** | **不需要**；API 经 Nuxt 服务端代理，使用嵌入令牌鉴权 |
| **AuraOA 配置** | 租户规则 → 对应流程 → **OA 嵌入审核** 开关 |
| **服务配置** | `config.yaml` + 前端环境变量 `EMBED_ACCESS_TOKEN` / `EMBED_TENANT_CODE` |

泛微审批页 URL 示例（`requestid` 在 hash 内）：

```
http://10.x.x.x:8081/spa/workflow/static4form/index.html#/main/workflow/req?requestid=598488&...
```

嵌入 iframe **src 固定为**：

```html
<iframe src="https://aura.example.com/embed/audit" style="width:360px;height:100%;border:0;"></iframe>
```

---

## 2. 前置条件

1. 租户已绑定 OA 只读库  
2. 目标流程已配置 AI 审核规则  
3. 已执行迁移 `000041_embed_audit_context`  
4. 已在 **AuraOA 后端** 与 **前端环境变量** 中配置嵌入令牌（见第 3 节）  
5. 目标流程已打开 **OA 嵌入审核** 开关  

---

## 3. AuraOA 侧配置

### 3.1 嵌入令牌（无需用户 JWT）

**Go 服务** `config.yaml`：

```yaml
embed:
  tenant_code: "your-tenant-code"   # 与 tenants.code 一致
  access_token: "请更换为随机长字符串"
```

**前端环境变量**（与后端 `access_token` 相同，仅存在于服务端，不打包进浏览器）：

```bash
EMBED_ACCESS_TOKEN=请更换为随机长字符串
EMBED_TENANT_CODE=your-tenant-code
```

浏览器只访问 Nuxt 的 `/api/embed/*` 代理；令牌由 **Nuxt Server** 注入请求头 `X-Embed-Token`，不会暴露给 OA 页面用户。

> 自动审核任务在后台以**租户管理员**身份写入 `audit_logs`（`trigger_source=embed_auto`）。

### 3.2 启用流程嵌入

**租户管理 → 规则配置 → 权限 → OA 嵌入审核**

### 3.3 嵌入行为（可选）

`process_audit_configs.embed_config`：

```json
{
  "auto_audit_on_open": true,
  "auto_audit_on_stale": true
}
```

---

## 4. requestid 如何获取

### 4.1 同源 iframe（推荐）

OA 页与 AuraOA **同协议、同 host、同端口** 时，嵌入页每 300ms 轮询读取：

```javascript
window.parent.location.href
```

从完整 URL 的 **query 或 hash** 中解析 `requestid`。

### 4.2 跨域 iframe（常见）

浏览器**禁止**子页面读取 `parent.location`，需 OA 侧脚本将父页 URL 或 requestid 发给 iframe：

**方式 A — postMessage 传 requestid：**

```javascript
var iframe = document.getElementById('aura-embed-audit');
function notifyAura() {
  var hash = location.hash || '';
  var q = hash.indexOf('?') >= 0 ? hash.slice(hash.indexOf('?') + 1) : '';
  var requestid = new URLSearchParams(q).get('requestid');
  if (requestid && iframe && iframe.contentWindow) {
    iframe.contentWindow.postMessage({ type: 'aura-oa-requestid', requestid: requestid }, 'https://aura.example.com');
  }
}
iframe.addEventListener('load', notifyAura);
// hash 变化时再发一次
window.addEventListener('hashchange', notifyAura);
```

**方式 B — postMessage 传完整 URL：**

```javascript
iframe.contentWindow.postMessage({ type: 'aura-oa-url', url: location.href }, 'https://aura.example.com');
```

嵌入页已内置上述两种消息的监听。

---

## 5. 泛微 E9 OA 侧配置

### 5.1 最小 HTML

```html
<iframe
  id="aura-embed-audit"
  title="AI 审核"
  src="https://aura.example.com/embed/audit"
  style="width:380px;height:100%;border:0;"
></iframe>
<script src="/path/to/aura-embed-notify.js"></script>
```

`aura-embed-notify.js` 内容见第 4.2 节（在 **OA 域名下** 执行，可读自身 URL）。

### 5.2 挂载位置

- 流程表单页扩展脚本 / CustomPage  
- 门户右侧栏  
- 审批布局模板  

### 5.3 允许被 iframe 嵌入

确保 AuraOA 前端响应头允许 OA 域名嵌入（如 CSP `frame-ancestors` 或勿设置 `X-Frame-Options: DENY`）。

---

## 6. 数据与来源区分

同一 `requestid` 在工作台与嵌入页**共用** `audit_process_snapshots` 最新结论；每次审核在 `audit_logs` 新增一条，用 `trigger_source` 区分：

| trigger_source | 含义 |
|----------------|------|
| `embed_auto` | 嵌入页自动审核 |
| `embed_manual` | 嵌入页「重新审核」 |
| `workbench_manual` | 审核工作台 |

---

## 7. 常见问题

**Q：嵌入页一直「正在读取 OA 流程编号」**  
→ 跨域且 OA 未 postMessage；或父 URL 中没有 `requestid`。

**Q：503 嵌入审核未配置**  
→ 检查 `EMBED_ACCESS_TOKEN`、`EMBED_TENANT_CODE` 与 `config.yaml` 的 `embed.*` 是否一致。

**Q：仍要传 process_id 吗？**  
→ **不要**。固定地址 `/embed/audit` 即可；仅本地调试时可临时在父页同源环境下手动改 hash 测试。

---

## 8. 相关文档

- [嵌入审核 API](../api/embed.md)  
- [OA 系统对接说明](../oa-integration.md)
