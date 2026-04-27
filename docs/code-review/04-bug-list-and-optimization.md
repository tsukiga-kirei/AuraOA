# Bug 清单与优化建议

## 1. 概述

本文档汇总代码审查中发现的所有问题，按严重程度分类，并提供修复建议。

---

## 2. Bug 清单

### 🔴 严重问题 (P0)

#### BUG-001: Token TTL 配置不同步

**位置**: 
- `go-service/internal/pkg/jwt/jwt.go`
- `db/migrations/000004_system_configs.up.sql`

**问题描述**:
数据库 `system_configs` 表中的 `auth.access_token_ttl_hours` 和 `auth.refresh_token_ttl_days` 配置未被 JWT 生成代码使用。JWT 代码直接从 `config.yaml` 读取配置。

**影响**: 管理员在系统设置页面修改 Token 有效期无效。

**修复方案**:

```go
// jwt.go - 修改 GenerateAccessToken
func GenerateAccessToken(claims *JWTClaims) (string, error) {
    secret := viper.GetString("jwt.secret")
    
    // 优先从数据库读取配置
    ttl := getAccessTokenTTLFromDB()
    if ttl == 0 {
        ttl = viper.GetDuration("jwt.access_token_ttl")
    }
    if ttl == 0 {
        ttl = 2 * time.Hour
    }
    // ...
}

// 新增辅助函数
func getAccessTokenTTLFromDB() time.Duration {
    // 从 system_configs 表读取 auth.access_token_ttl_hours
    // 返回 time.Duration
}
```

**或者**: 移除数据库中的冗余配置，统一使用 config.yaml，并在文档中说明。

---

#### BUG-002: 前端 Token 过期后未自动清理本地状态

**位置**: 
- `frontend/composables/useAuth.ts`
- `frontend/middleware/auth.ts`

**问题描述**:
Token 过期后，如果用户不切换路由或发起 API 请求，本地状态（左下角用户信息、localStorage 中的 auth_state）不会自动清除。

**影响**: 用户体验差，可能误以为仍处于登录状态。

**修复方案**:

```typescript
// useAuth.ts - 添加定时检查
export const useAuth = () => {
    // ... 现有代码

    // 添加 Token 过期检查定时器
    const startTokenExpiryCheck = () => {
        const checkInterval = 5 * 60 * 1000 // 5 分钟
        
        setInterval(async () => {
            const t = token.value || localStorage.getItem('token')
            if (!t) return
            
            const exp = parseJwtExp(t)
            if (!exp) return
            
            const now = Date.now() / 1000
            const remaining = exp - now
            
            // Token 已过期
            if (remaining <= 0) {
                const refreshed = await tryRestoreAsync()
                if (!refreshed) {
                    clearLocalSession()
                    navigateTo('/login')
                }
            }
            // Token 即将过期（5分钟内），主动刷新
            else if (remaining < 300) {
                await doRefreshToken()
            }
        }, checkInterval)
    }

    // 页面可见性变化时检查
    const handleVisibilityChange = async () => {
        if (document.visibilityState === 'visible' && isAuthenticated.value) {
            const ok = await validateAccessToken()
            if (!ok) {
                const refreshed = await tryRestoreAsync()
                if (!refreshed) {
                    clearLocalSession()
                    navigateTo('/login')
                }
            }
        }
    }

    // 在 onMounted 中启动
    if (import.meta.client) {
        startTokenExpiryCheck()
        document.addEventListener('visibilitychange', handleVisibilityChange)
    }

    return {
        // ... 现有返回值
    }
}
```

---

### 🟡 中等问题 (P1)

#### BUG-003: 默认密码硬编码

**位置**: `go-service/internal/service/org_service.go`

**问题描述**:
创建组织成员时使用硬编码的默认密码 "123456"。

**影响**: 安全风险，密码过于简单。

**修复方案**:

```go
// org_service.go - CreateMember
password := req.Password
if password == "" {
    // 从系统配置读取默认密码
    defaultPwd, err := s.systemConfigRepo.FindByKey("auth.default_password")
    if err != nil || defaultPwd == "" {
        // 生成随机密码
        password = generateRandomPassword(12)
        // TODO: 发送邮件通知用户
    } else {
        password = defaultPwd
    }
}

// 标记需要首次登录改密
user.PasswordMustChange = true
```

---

#### BUG-004: Session 缓存 TTL 与 Refresh Token 不一致

**位置**: `go-service/internal/service/auth_service.go`

**问题描述**:
Session 缓存 TTL 为 2 小时，而 Refresh Token 有效期为 7 天。Session 过期后刷新 Token 需要查库重建 claims。

**影响**: 增加数据库查询压力。

**修复方案**:

```go
// auth_service.go - Login
// 将 Session 缓存 TTL 延长至 7 天
s.rdb.Set(context.Background(), sessionKey, string(sessionJSON), 7*24*time.Hour)

// 或者在 Refresh 成功后更新 Session 缓存
func (s *AuthService) Refresh(req *dto.RefreshRequest) (*dto.RefreshResponse, error) {
    // ... 生成新 token 后
    
    // 更新 Session 缓存
    sessionData := map[string]interface{}{...}
    sessionJSON, _ := json.Marshal(sessionData)
    s.rdb.Set(ctx, sessionKey, string(sessionJSON), 7*24*time.Hour)
}
```

---

#### BUG-005: 审批流信息未接入（已修复）

**位置**: `go-service/internal/pkg/oa/ecology9.go`、`go-service/internal/service/audit_prompt_builder.go`

**问题描述**:
提示词模板中的 `{{flow_history}}` 和 `{{flow_graph}}` 占位符此前使用固定文本 "（暂未提供）"。

**修复内容**:
- `FetchProcessFlow` 重构：审批历史仅取最后一次退回（logtype='3'）之后的有效路径，通过 `mapLogType` 将 E9 LOGTYPE 代码映射为可读操作类型（批准/提交/退回/转发等）
- 新增 `fetchFlowRouteGraph`：查询 `workflow_nodelink` + `rule_base` 获取流程路由图（节点连接关系和出口条件），兼容 Oracle/DM（TO_CHAR）和 MySQL（CAST）
- 提示词构建已注入真实 `flowHistory` 和 `flowGraph` 数据

**状态**: ✅ 已修复

---

#### BUG-005.5: 归档列表缓存键未包含日期字段（已修复）

**位置**: `go-service/internal/service/archive_review_service.go`

**问题描述**:
`ArchiveListParams` 的日期字段（`ArchiveDateStart`、`ArchiveDateEndExclusive`）标记为 `json:"-"`，导致 `cache.ComputeFilterHash` 序列化时忽略这些字段。不同日期范围的查询会生成相同的缓存键，返回错误的缓存数据。

**影响**: 用户切换日期筛选条件后，可能看到其他日期范围的归档数据，造成数据错乱。

**修复方案（已实施）**:
将 `ComputeFilterHash` 的入参从结构体改为显式构造的 `map[string]interface{}`，手动列出所有筛选字段（含日期字段），确保缓存键唯一。

**状态**: ✅ 已修复

---

#### BUG-005.6: FetchTodoListPaged COUNT 查询未去重导致待办总数偏大

**位置**: `go-service/internal/pkg/oa/ecology9.go` — `FetchTodoListPaged`

**问题描述**:
`FetchTodoListPaged` 的 COUNT 查询使用 `COUNT(*)`，但由于 `workflow_currentoperator` 中同一流程可能存在多个审批节点记录，导致 COUNT 结果大于实际去重后的流程数。而数据查询已使用 `SELECT DISTINCT`，造成 `total` 与实际返回条目数不一致，前端分页组件显示的总页数偏多。

**影响**: 前端分页显示的总条数偏大，末尾页可能为空页。

**修复方案（已实施）**:
将 `COUNT(*)` 改为 `COUNT(DISTINCT r.requestid)`，与数据查询的 `SELECT DISTINCT` 保持一致。

**状态**: ✅ 已修复

---

#### BUG-005.7: resolveFieldSet 未选中字段的明细表泄漏全部字段

**位置**: `go-service/internal/service/audit_review_service.go` — `resolveFieldSet`

**问题描述**:
当某张明细表没有任何字段被选中时（`dtSet` 为空 map），原逻辑跳过写入 `fieldSet[dt.TableName]`。下游 `formatGroupedDetailData` 在 `fieldSet` 中找不到该表名时，`allowedKeys` 为 `nil`，`filterRowFields` 对 `nil` 不做过滤，导致该表的所有字段全部输出给 AI。

**影响**: 租户管理员明确未选中任何字段的明细表仍会被完整发送到 AI 提示词中，违反字段选择语义，可能泄漏不应参与审核的数据。

**修复方案（已实施）**:
始终将 `dtSet`（即使为空 map）写入 `fieldSet`，使下游 `filterRowFields` 收到空 map 后返回 `nil`，`formatGroupedDetailData` 跳过该表。

```go
// 修复前
if len(dtSet) > 0 {
    fieldSet[dt.TableName] = dtSet
}

// 修复后
// 始终写入：空 map 表示该表无选中字段，后续过滤时跳过该表数据
fieldSet[dt.TableName] = dtSet
```

**状态**: ✅ 已修复

---

### 🟢 低优先级问题 (P2)

#### BUG-006: 租户管理员保护逻辑重复

**位置**: `go-service/internal/service/org_service.go`

**问题描述**:
租户管理员的保护检查在 UpdateMember 和 DeleteMember 中重复实现。

**修复方案**:

```go
// 抽取为独立方法
func (s *OrgService) isTenantAdmin(userID, tenantID uuid.UUID) bool {
    var tenant model.Tenant
    err := s.db.Where("admin_user_id = ? AND id = ?", userID, tenantID).First(&tenant).Error
    return err == nil
}

// 使用
if s.isTenantAdmin(member.UserID, member.TenantID) {
    return newServiceError(errcode.ErrParamValidation, "该成员是租户管理员，不允许此操作")
}
```

---

#### BUG-007: 字段合并逻辑复杂度高

**位置**: `go-service/internal/service/user_personal_config_service.go`

**问题描述**:
`GetFullAuditProcessConfig` 方法中字段合并逻辑嵌套较深，可读性差。

**修复方案**:

```go
// 抽取为独立函数
func mergeFieldConfig(tenantFields []TenantField, userOverrides []string, fieldMode string, allowCustom bool) []MergedField {
    // 清晰的合并逻辑
}

// 在主方法中调用
mainFields := mergeFieldConfig(rawMainFields, userDetail.FieldConfig.FieldOverrides, effectiveFieldMode, perms.AllowCustomFields)
```

---

## 3. 优化建议汇总

### 3.1 安全性优化

| 编号 | 优化项 | 优先级 | 预计工时 |
|-----|-------|-------|---------|
| SEC-001 | Token TTL 配置统一 | P0 | 2h |
| SEC-002 | 默认密码可配置化 | P1 | 1h |
| SEC-003 | 首次登录强制改密 | P1 | 4h |
| SEC-004 | 添加密码复杂度校验 | P2 | 2h |

### 3.2 用户体验优化

| 编号 | 优化项 | 优先级 | 预计工时 |
|-----|-------|-------|---------|
| UX-001 | Token 过期自动清理 | P0 | 4h |
| UX-002 | Token 即将过期预警刷新 | P1 | 2h |
| UX-003 | 页面可见性变化时检查登录态 | P1 | 1h |

### 3.3 功能完善

| 编号 | 优化项 | 优先级 | 预计工时 |
|-----|-------|-------|---------|
| FEAT-001 | ~~接入审批流数据~~ | ~~P1~~ | ~~8h~~ ✅ 已完成 |
| FEAT-002 | 添加操作审计日志 | P2 | 4h |

### 3.4 代码质量

| 编号 | 优化项 | 优先级 | 预计工时 |
|-----|-------|-------|---------|
| CODE-001 | 抽取租户管理员检查方法 | P2 | 0.5h |
| CODE-002 | 重构字段合并逻辑 | P2 | 2h |
| CODE-003 | 添加单元测试 | P2 | 8h |

---

## 4. 修复优先级排序

### 第一阶段（立即修复）

1. **BUG-001**: Token TTL 配置不同步
2. **BUG-002**: 前端 Token 过期未自动清理

### 第二阶段（1 周内）

3. **BUG-003**: 默认密码硬编码
4. **BUG-004**: Session 缓存 TTL 优化
5. ~~**BUG-005**: 审批流信息接入~~ ✅ 已修复

### 第三阶段（2 周内）

6. **BUG-006**: 租户管理员保护逻辑重复
7. **BUG-007**: 字段合并逻辑重构
8. 添加单元测试

---

## 5. 测试建议

### 5.1 认证系统测试用例

```
1. Token 过期测试
   - 等待 Access Token 过期，验证自动刷新
   - 等待 Refresh Token 过期，验证自动登出
   - 在其他设备登出，验证当前设备 Token 失效

2. 并发刷新测试
   - 同时发起多个 API 请求触发 401
   - 验证只有一个刷新请求发出
   - 验证所有请求都能正确重试

3. 本地时间偏差测试
   - 将本地时间调快，验证 Token 判断逻辑
```

### 5.2 配置合并测试用例

```
1. 字段合并测试
   - 租户 field_mode = "all"，用户无法减少字段
   - 租户 field_mode = "selected"，用户可新增字段
   - 用户权限锁定时，自定义字段被拒绝

2. 规则合并测试
   - mandatory 规则始终生效
   - default_on 规则可被用户关闭
   - default_off 规则可被用户开启
   - 租户删除规则后，用户覆盖自动清理
```

---

## 6. 监控建议

### 6.1 关键指标

| 指标 | 说明 | 告警阈值 |
|-----|------|---------|
| token_refresh_rate | Token 刷新频率 | > 100/min |
| token_refresh_failure_rate | 刷新失败率 | > 5% |
| session_cache_miss_rate | Session 缓存未命中率 | > 20% |
| login_failure_rate | 登录失败率 | > 10% |

### 6.2 日志增强

```go
// 建议在关键位置添加日志
pkglogger.Global().Info("Token 刷新成功",
    zap.String("userID", userID),
    zap.String("source", "session_cache|database"),
)

pkglogger.Global().Warn("Token 刷新失败",
    zap.String("userID", userID),
    zap.String("reason", "blacklisted|expired|invalid"),
)
```
