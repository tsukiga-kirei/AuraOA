# 认证系统代码分析报告

## 1. 概述

本文档分析 OA 智审系统的认证机制，包括 Token 管理、刷新逻辑、过期处理等核心功能。

---

## 2. Token 架构

### 2.1 双令牌机制

系统采用 Access Token + Refresh Token 双令牌架构：

| Token 类型 | 默认有效期 | 配置项 | 用途 |
|-----------|-----------|--------|------|
| Access Token | 2 小时 | `jwt.access_token_ttl` / `auth.access_token_ttl_hours` | API 请求认证 |
| Refresh Token | 7 天 (168h) | `jwt.refresh_token_ttl` / `auth.refresh_token_ttl_days` | 静默刷新 Access Token |

**配置文件位置**: `go-service/config.yaml`

```yaml
jwt:
  secret: "change-me-in-production"
  access_token_ttl: 2h
  refresh_token_ttl: 168h
```

### 2.2 Token 生成逻辑

**文件**: `go-service/internal/pkg/jwt/jwt.go`

```go
// GenerateAccessTokenWithTTL 使用指定 TTL 签发访问令牌。
func GenerateAccessTokenWithTTL(claims *JWTClaims, ttl time.Duration) (string, error) { ... }

// GenerateRefreshTokenWithTTL 使用指定 TTL 签发刷新令牌。
func GenerateRefreshTokenWithTTL(userID string, jti string, ttl time.Duration) (string, string, error) { ... }
```

---

## 3. 已修复的问题

### ✅ 问题 1: Token TTL 与系统配置不同步（已修复）

**严重程度**: 高

**原问题描述**:
- 数据库 `system_configs` 表中存储了 `auth.access_token_ttl_hours` 和 `auth.refresh_token_ttl_days` 配置
- 但 JWT 生成代码直接从 `config.yaml` 读取 `jwt.access_token_ttl` 和 `jwt.refresh_token_ttl`
- 两套配置互不关联，修改数据库配置不会影响实际 Token 有效期

**修复方案**:

1. **JWT 包新增 TTL 辅助函数** (`go-service/internal/pkg/jwt/jwt.go`):
   - `GetAccessTokenTTL()` / `GetRefreshTokenTTL()` — 从 config.yaml 读取 TTL
   - `GenerateAccessTokenWithTTL()` / `GenerateRefreshTokenWithTTL()` — 支持外部传入 TTL

2. **AuthService 注入 SystemConfigRepo** (`go-service/internal/service/auth_service.go`):
   - `getAccessTokenTTL()` — 优先从数据库读取 `auth.access_token_ttl_hours`，降级使用 config.yaml
   - `getRefreshTokenTTL()` — 优先从数据库读取 `auth.refresh_token_ttl_days`，降级使用 config.yaml
   - Login / Refresh / SwitchRole 均使用数据库优先的 TTL

3. **main.go 更新** (`go-service/cmd/server/main.go`):
   - `NewAuthService` 新增 `systemConfigRepo` 参数注入

**修改文件**:
- `go-service/internal/pkg/jwt/jwt.go`
- `go-service/internal/service/auth_service.go`
- `go-service/cmd/server/main.go`

---

### ✅ 问题 2: 前端 Token 过期后未正确清理本地状态（已修复）

**严重程度**: 高

**原问题描述**:
用户反馈 "到期之后也没有自动退出，左下角还显示着用户，本地缓存也有 auth_state"。
原因是只有路由切换或 API 请求时才会触发校验，用户停留在同一页面不操作时 Token 过期不会被检测。

**修复方案**:

新增 `frontend/composables/useTokenGuard.ts` — 主动 Token 过期检测守卫：

1. **定时检查**（每 5 分钟）：解析 access_token 的 exp 字段，剩余有效期 < 5 分钟时主动调用 refresh
2. **visibilitychange 事件**：页面从后台切回前台时立即检查 Token 状态
3. **Token 丢失检测**：access_token 丢失但 refresh_token 有效时尝试恢复，否则登出
4. **刷新失败处理**：refresh 也失败时调用 `logout()` 清除所有本地状态并跳转登录页

在 `frontend/app.vue` 中通过 `onMounted` 启动守卫，`onUnmounted` 停止。

**修改文件**:
- `frontend/composables/useTokenGuard.ts`（新增）
- `frontend/app.vue`

---

### ✅ 问题 3: 前端错误消息国际化（已修复）

**严重程度**: 中

**原问题描述**:
`useAuth.ts` 中的错误消息（如 "网络连接失败，请检查网络"、"登录已过期，请重新登录"）为硬编码中文，
不支持英文环境。

**修复方案**:

1. **新增 i18n 键** (`frontend/locales/zh-CN.ts` & `frontend/locales/en-US.ts`):
   - `auth.sessionExpired` / `auth.networkError` / `auth.requestFailed` 等通用提示
   - `auth.error.*` 系列错误码对应的翻译

2. **useAuth.ts 国际化改造**:
   - 新增 `getErrorMessageByCode(code)` — 优先从 i18n messages 获取翻译，降级使用硬编码映射
   - 新增 `getI18nText(key)` — 从 localStorage 中的 locale 判断语言并返回翻译文案
   - `authFetch` 中所有错误消息改用 i18n 函数

**修改文件**:
- `frontend/composables/useAuth.ts`
- `frontend/locales/zh-CN.ts`
- `frontend/locales/en-US.ts`

---

### ✅ 问题 4: Session 缓存与 Token 有效期不一致（已修复）

**严重程度**: 中

**原问题描述**:
- Session 缓存 TTL: 2 小时
- Refresh Token TTL: 7 天
- Session 过期后刷新 Token 需要查库重建 claims，增加数据库压力

**修复方案**:

1. **Session 缓存 TTL 延长至与 Refresh Token 一致**:
   - `getSessionCacheTTL()` 方法优先从数据库读取 `auth.refresh_token_ttl_days`，降级使用 config.yaml 的 refresh_token_ttl
   - Login 和 SwitchRole 中的 `s.rdb.Set(...)` 改用 `s.getSessionCacheTTL()`

2. **Refresh 降级查库后重建 Session 缓存**:
   - 当 session 缓存失效需要查库重建 claims 时，刷新成功后将新的 session 数据写回 Redis
   - 避免后续刷新再次触发数据库查询

**修改文件**:
- `go-service/internal/service/auth_service.go`

---

### 🟡 问题 5: Refresh Token 提前失效的可能原因（排查建议）

**严重程度**: 中

**问题描述**: 用户反馈 "没到 7 天就过期了"

**可能原因分析**:

1. **Redis 黑名单机制**: 用户在其他设备登出会导致 Refresh Token 被吊销
2. **前端本地时间不准确**: `isRefreshTokenValid()` 使用 `Date.now() / 1000` 与 token exp 比较
3. **浏览器清理 localStorage**: 隐私模式或浏览器设置可能清除存储

**排查建议**:
1. 检查用户是否在多设备登录后在其他设备登出
2. 检查用户本地时间是否准确
3. 检查是否使用隐私模式浏览

---

## 4. 认证流程图

```
┌─────────────────────────────────────────────────────────────────┐
│                        用户登录流程                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  用户输入凭证 ──► 后端验证 ──► 生成 Access + Refresh Token      │
│                      │         (TTL 优先从 DB 读取)             │
│                      ▼                                          │
│              写入 Redis Session (TTL = Refresh Token TTL)       │
│                      │                                          │
│                      ▼                                          │
│              返回 Token 给前端                                   │
│                      │                                          │
│                      ▼                                          │
│              前端存储到 localStorage                             │
│              - token                                            │
│              - refresh_token                                    │
│              - auth_state (用户信息)                            │
│                                                                 │
│              启动 TokenGuard 定时检查                            │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                      Token 刷新流程                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  触发条件：                                                      │
│  - API 请求 401                                                 │
│  - TokenGuard 定时检查（每 5 分钟）                              │
│  - 页面从后台切回前台（visibilitychange）                        │
│                      │                                          │
│         ┌────────────┴────────────┐                             │
│         ▼                         ▼                             │
│  Refresh Token 有效          无效/过期                           │
│         │                         │                             │
│         ▼                         ▼                             │
│  调用 /api/auth/refresh      清除本地状态                        │
│         │                    跳转登录页                          │
│         ▼                                                       │
│  后端校验 Refresh Token                                         │
│  - 解析 JWT                                                     │
│  - 检查黑名单                                                   │
│  - 从 Session 或 DB 重建 claims                                 │
│  - 降级查库后重建 Session 缓存                                   │
│         │                                                       │
│         ▼                                                       │
│  返回新 Access Token (TTL 从 DB 读取)                           │
│         │                                                       │
│         ▼                                                       │
│  前端更新 localStorage                                          │
│  重试原请求                                                     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 5. 代码质量评估

### ✅ 优点

1. **完善的黑名单机制**: 登出时将 Token JTI 加入 Redis 黑名单，防止已登出 Token 被滥用
2. **并发刷新保护**: 前端使用 `isRefreshing` 标志和订阅队列，防止多个请求同时触发刷新
3. **降级策略**: Session 缓存失效时能降级查库重建 claims，并回写缓存
4. **多角色支持**: 支持用户在多租户间切换角色，切换时生成新 Token
5. **主动过期检测**: TokenGuard 定时 + visibilitychange 双重机制确保 Token 过期被及时发现
6. **配置统一**: Token TTL 优先从数据库读取，管理员修改系统设置后立即生效
7. **国际化支持**: 错误消息支持 zh-CN / en-US 双语，根据用户语言偏好自动切换

### ⚠️ 待改进

1. 建议为 Token 即将过期添加 UI 提示（如 toast 通知用户正在自动续期）
2. 可考虑在 Refresh Token 即将过期时（如剩余 1 天）提示用户重新登录以获取新的 Refresh Token

---

### ✅ 问题 6: 记住登录信息未持久化（已修复）

**严重程度**: 低

**原问题描述**:
登录页 `rememberMe` 选项仅为前端内存状态，页面刷新后用户需重新输入凭证。

**修复方案**:

新增 `restoreRemembered()` 函数（`frontend/pages/login.vue`）：

- 使用 `localStorage` key `login_remember` 持久化 `username`、`password`、`portal`、`tenant_id`
- 页面加载时自动恢复已记住的登录信息，`tenant_id` 在租户列表加载完成后回填
- 仅当 `portal` 值合法（`business` / `tenant_admin` / `system_admin`）时才恢复入口选择
- JSON 解析异常时静默忽略，不影响正常登录流程

**修改文件**:
- `frontend/pages/login.vue`

---

## 6. 修复状态

| 优先级 | 问题 | 状态 |
|-------|------|------|
| P0 | Token TTL 配置不同步 | ✅ 已修复 |
| P0 | 前端 Token 过期未自动清理 | ✅ 已修复 |
| P1 | Session 缓存 TTL 优化 | ✅ 已修复 |
| P1 | 前端错误消息国际化 | ✅ 已修复 |
| P2 | Refresh Token 提前失效排查 | 📋 已提供排查建议 |
| P3 | 记住登录信息未持久化 | ✅ 已修复 |
