# 认证系统代码分析报告

## 1. 概述

本文档分析 OA 智审系统的认证机制，包括 Token 管理、刷新逻辑、过期处理等核心功能。

---

## 2. Token 架构

### 2.1 双令牌机制

系统采用 Access Token + Refresh Token 双令牌架构：

| Token 类型 | 默认有效期 | 配置项 | 用途 |
|-----------|-----------|--------|------|
| Access Token | 2 小时 | `jwt.access_token_ttl` | API 请求认证 |
| Refresh Token | 7 天 (168h) | `jwt.refresh_token_ttl` | 静默刷新 Access Token |

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
// Access Token 生成
func GenerateAccessToken(claims *JWTClaims) (string, error) {
    ttl := viper.GetDuration("jwt.access_token_ttl")
    if ttl == 0 {
        ttl = 2 * time.Hour  // 默认 2 小时
    }
    // ...
}

// Refresh Token 生成
func GenerateRefreshToken(userID string, jti string) (string, string, error) {
    ttl := viper.GetDuration("jwt.refresh_token_ttl")
    if ttl == 0 {
        ttl = 7 * 24 * time.Hour  // 默认 7 天
    }
    // ...
}
```

---

## 3. 发现的问题

### 🔴 问题 1: Token TTL 与系统配置不同步

**严重程度**: 高

**问题描述**:
- 数据库 `system_configs` 表中存储了 `auth.access_token_ttl_hours` 和 `auth.refresh_token_ttl_days` 配置
- 但 JWT 生成代码直接从 `config.yaml` 读取 `jwt.access_token_ttl` 和 `jwt.refresh_token_ttl`
- **两套配置互不关联**，修改数据库配置不会影响实际 Token 有效期

**数据库配置** (`000004_system_configs.up.sql`):
```sql
('auth.access_token_ttl_hours', '2', 'Access Token 有效期（小时）'),
('auth.refresh_token_ttl_days', '7', 'Refresh Token 有效期（天）'),
```

**实际使用的配置** (`jwt.go`):
```go
ttl := viper.GetDuration("jwt.access_token_ttl")  // 从 config.yaml 读取
```

**影响**: 管理员在系统设置页面修改 Token 有效期无效，用户可能困惑为何设置不生效。

**修复建议**:
1. 在 JWT 生成时优先读取数据库配置，降级使用 config.yaml
2. 或移除数据库中的冗余配置项，统一使用 config.yaml

---

### 🔴 问题 2: 前端 Token 过期后未正确清理本地状态

**严重程度**: 高

**问题描述**:
用户反馈 "到期之后也没有自动退出，左下角还显示着用户，本地缓存也有 auth_state"

**根本原因分析**:

1. **前端路由守卫的校验时机问题** (`frontend/middleware/auth.ts`):
```typescript
// 第三步：本地有 JWT 时向后端校验
if (isAuthenticated.value) {
    let ok = await validateAccessToken()
    if (!ok) {
        const refreshed = await tryRestoreAsync()
        if (refreshed) {
            ok = await validateAccessToken()
        }
        if (!ok) {
            clearLocalSession()  // ✅ 这里会清理
        }
    }
}
```

2. **问题**: 只有在**路由切换时**才会触发校验，如果用户停留在同一页面不切换路由：
   - Token 过期后，页面不会自动刷新
   - 左下角用户信息来自 `currentUser` 响应式状态，不会自动清除
   - `auth_state` 在 localStorage 中持久化，不会自动清除

3. **authFetch 的 401 处理** (`frontend/composables/useAuth.ts`):
```typescript
if (statusCode === 401) {
    // ...
    const refreshOk = await doRefreshToken()
    if (!refreshOk) {
        await logout()  // ✅ 刷新失败会登出
        throw new Error('登录已过期，请重新登录')
    }
}
```

**但问题是**: 如果用户不发起任何 API 请求，就不会触发 401 处理。

**修复建议**:
1. 添加定时器定期检查 Token 有效期（如每 5 分钟）
2. 在 Token 即将过期时（如剩余 5 分钟）主动刷新
3. 使用 `visibilitychange` 事件，在页面重新可见时检查 Token

---

### 🟡 问题 3: Refresh Token 提前失效的可能原因

**严重程度**: 中

**问题描述**: 用户反馈 "没到 7 天就过期了"

**可能原因分析**:

1. **Redis 黑名单机制**:
```go
// Logout 时将 refresh_token JTI 加入黑名单
if req.RefreshJTI != "" {
    blacklistKey := fmt.Sprintf("blacklist:%s", req.RefreshJTI)
    s.rdb.Set(ctx, blacklistKey, "1", 7*24*time.Hour)
}
```
如果用户在其他设备登出，会导致 Refresh Token 被吊销。

2. **Session 缓存过期**:
```go
// 登录时缓存 session，TTL 2h
sessionKey := fmt.Sprintf("session:%s", user.ID.String())
s.rdb.Set(context.Background(), sessionKey, string(sessionJSON), 2*time.Hour)
```
Session 缓存 2 小时后过期，但 Refresh 逻辑会降级查库重建，这不是问题。

3. **前端本地判断逻辑**:
```typescript
const isRefreshTokenValid = (): boolean => {
    const rt = refreshToken.value || localStorage.getItem('refresh_token')
    if (!rt) return false
    const exp = parseJwtExp(rt)
    if (!exp) return false
    return exp > Date.now() / 1000  // 本地时间判断
}
```
**潜在问题**: 如果用户本地时间不准确（快于服务器时间），会导致提前判定过期。

4. **浏览器清理 localStorage**:
   - 隐私模式下 localStorage 会在关闭浏览器时清除
   - 某些浏览器设置可能定期清理存储

**排查建议**:
1. 检查用户是否在多设备登录后在其他设备登出
2. 检查用户本地时间是否准确
3. 检查是否使用隐私模式浏览

---

### 🟡 问题 4: Session 缓存与 Token 有效期不一致

**严重程度**: 中

**问题描述**:
- Session 缓存 TTL: 2 小时
- Access Token TTL: 2 小时
- Refresh Token TTL: 7 天

当 Session 缓存过期但 Refresh Token 仍有效时，刷新 Token 需要查库重建 claims：

```go
// Refresh 方法中的降级逻辑
if accessToken == "" {
    user, findErr := s.userRepo.FindByID(userID)
    // ... 重新查询用户和角色
}
```

**影响**: 增加数据库查询压力，但功能正常。

**优化建议**: 将 Session 缓存 TTL 延长至与 Refresh Token 一致（7 天），或在刷新成功后更新 Session 缓存。

---

## 4. 认证流程图

```
┌─────────────────────────────────────────────────────────────────┐
│                        用户登录流程                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  用户输入凭证 ──► 后端验证 ──► 生成 Access + Refresh Token      │
│                      │                                          │
│                      ▼                                          │
│              写入 Redis Session (TTL 2h)                        │
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
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                      Token 刷新流程                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  API 请求 401 ──► 检查 Refresh Token 有效性                     │
│                      │                                          │
│         ┌────────────┴────────────┐                             │
│         ▼                         ▼                             │
│     有效                       无效/过期                         │
│         │                         │                             │
│         ▼                         ▼                             │
│  调用 /api/auth/refresh      清除本地状态                        │
│         │                    跳转登录页                          │
│         ▼                                                       │
│  后端校验 Refresh Token                                         │
│  - 解析 JWT                                                     │
│  - 检查黑名单                                                   │
│  - 从 Session 或 DB 重建 claims                                 │
│         │                                                       │
│         ▼                                                       │
│  返回新 Access Token                                            │
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
3. **降级策略**: Session 缓存失效时能降级查库重建 claims
4. **多角色支持**: 支持用户在多租户间切换角色，切换时生成新 Token

### ⚠️ 待改进

1. Token TTL 配置应统一（数据库 vs config.yaml）
2. 需要添加主动 Token 过期检测机制
3. Session 缓存 TTL 应与业务需求对齐
4. 建议添加 Token 即将过期的预警刷新

---

## 6. 修复优先级

| 优先级 | 问题 | 建议修复时间 |
|-------|------|-------------|
| P0 | Token TTL 配置不同步 | 立即 |
| P0 | 前端 Token 过期未自动清理 | 立即 |
| P1 | Session 缓存 TTL 优化 | 1 周内 |
| P2 | 添加 Token 预警刷新 | 2 周内 |
