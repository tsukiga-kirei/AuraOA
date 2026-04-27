# Bug 清单与优化建议

## 1. 概述

本文档汇总代码审查中发现的所有问题，按严重程度分类，并提供修复建议。

---

## 2. Bug 清单

### 🔴 严重问题 (P0)

#### BUG-001: Token TTL 配置不同步 ✅ 已修复

**位置**: 
- `go-service/internal/service/auth_service.go`
- `go-service/internal/pkg/jwt/jwt.go`
- `go-service/cmd/server/main.go`

**问题描述**:
数据库 `system_configs` 表中的 `auth.access_token_ttl_hours` 和 `auth.refresh_token_ttl_days` 配置此前未被 JWT 生成代码使用。

**修复内容**:
- ✅ **JWT 包新增辅助函数**: `GetAccessTokenTTL()` / `GetRefreshTokenTTL()` 从 config.yaml 读取 TTL；`GenerateAccessTokenWithTTL()` / `GenerateRefreshTokenWithTTL()` 支持外部传入 TTL
- ✅ **AuthService 注入 SystemConfigRepo**: 新增 `getAccessTokenTTL()` / `getRefreshTokenTTL()` 方法，优先从数据库读取，降级使用 config.yaml
- ✅ **Login 已修复**: 改用 `GenerateAccessTokenWithTTL(claims, s.getAccessTokenTTL())` 和 `GenerateRefreshTokenWithTTL(userID, "", s.getRefreshTokenTTL())`
- ✅ **Refresh 已修复**: 两处 token 生成（session 缓存命中 / 降级查库）均改用 `GenerateAccessTokenWithTTL(jwtClaims, s.getAccessTokenTTL())`
- ✅ **SwitchRole 已修复**: 改用 `GenerateAccessTokenWithTTL(claims, s.getAccessTokenTTL())`
- ✅ **main.go 已更新**: `NewAuthService` 新增 `systemConfigRepo` 参数

**状态**: ✅ 已修复

---

#### BUG-002: 前端 Token 过期后未自动清理本地状态 ✅ 已修复

**位置**: 
- `frontend/composables/useTokenGuard.ts`（新增）
- `frontend/app.vue`

**问题描述**:
Token 过期后，如果用户不切换路由或发起 API 请求，本地状态（左下角用户信息、localStorage 中的 auth_state）不会自动清除。

**修复内容**:

新增 `useTokenGuard` composable，实现主动 Token 过期检测：

1. **定时检查**（每 5 分钟）：解析 access_token 的 exp 字段，剩余有效期 < 5 分钟时主动调用 refresh
2. **visibilitychange 事件**：页面从后台切回前台时立即检查 Token 状态
3. **Token 丢失检测**：access_token 丢失但 refresh_token 有效时尝试恢复，否则登出
4. **刷新失败处理**：refresh 也失败时调用 `logout()` 清除所有本地状态并跳转登录页

在 `app.vue` 中通过 `onMounted` 启动守卫，`onUnmounted` 停止。

**状态**: ✅ 已修复

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

**状态**: 📋 待修复

---

#### BUG-004: Session 缓存 TTL 与 Refresh Token 不一致 ✅ 已修复

**位置**: `go-service/internal/service/auth_service.go`

**问题描述**:
Session 缓存 TTL 为 2 小时，而 Refresh Token 有效期为 7 天。Session 过期后刷新 Token 需要查库重建 claims。

**修复内容**:
- ✅ **Session 缓存 TTL 动态对齐**: 新增 `getSessionCacheTTL()` 方法，优先从数据库读取 `auth.refresh_token_ttl_days`，降级使用 config.yaml 的 refresh_token_ttl
- ✅ **Login 已修复**: `s.rdb.Set(..., s.getSessionCacheTTL())`
- ✅ **SwitchRole 已修复**: `s.rdb.Set(..., s.getSessionCacheTTL())`
- ✅ **Refresh 降级查库后重建缓存**: 当 session 缓存失效需要查库重建 claims 时，刷新成功后将新的 session 数据写回 Redis（使用 `s.getSessionCacheTTL()`），避免后续刷新再次触发数据库查询

**状态**: ✅ 已修复

---

#### BUG-005: 审批流信息未接入 ✅ 已修复

**位置**: `go-service/internal/pkg/oa/ecology9.go`、`go-service/internal/service/audit_prompt_builder.go`

**问题描述**:
提示词模板中的 `{{flow_history}}` 和 `{{flow_graph}}` 占位符此前使用固定文本 "（暂未提供）"。

**修复内容**:
- `FetchProcessFlow` 重构：审批历史仅取最后一次退回（logtype='3'）之后的有效路径，通过 `mapLogType` 将 E9 LOGTYPE 代码映射为可读操作类型（批准/提交/退回/转发等）
- 新增 `fetchFlowRouteGraph`：查询 `workflow_nodelink` + `rule_base` 获取流程路由图（节点连接关系和出口条件），兼容 Oracle/DM（TO_CHAR）和 MySQL（CAST）
- 提示词构建已注入真实 `flowHistory` 和 `flowGraph` 数据

**状态**: ✅ 已修复

---

#### BUG-005.5: 归档列表缓存键未包含日期字段 ✅ 已修复

**位置**: `go-service/internal/service/archive_review_service.go`

**问题描述**:
`ArchiveListParams` 的日期字段（`ArchiveDateStart`、`ArchiveDateEndExclusive`）标记为 `json:"-"`，导致 `cache.ComputeFilterHash` 序列化时忽略这些字段。不同日期范围的查询会生成相同的缓存键，返回错误的缓存数据。

**修复方案（已实施）**:
将 `ComputeFilterHash` 的入参从结构体改为显式构造的 `map[string]interface{}`，手动列出所有筛选字段（含日期字段），确保缓存键唯一。

**状态**: ✅ 已修复

---

#### BUG-005.6: FetchTodoListPaged COUNT 查询未去重导致待办总数偏大 ✅ 已修复

**位置**: `go-service/internal/pkg/oa/ecology9.go` — `FetchTodoListPaged`

**问题描述**:
`FetchTodoListPaged` 的 COUNT 查询使用 `COUNT(*)`，但由于 `workflow_currentoperator` 中同一流程可能存在多个审批节点记录，导致 COUNT 结果大于实际去重后的流程数。

**修复方案（已实施）**:
将 `COUNT(*)` 改为 `COUNT(DISTINCT r.requestid)`，与数据查询的 `SELECT DISTINCT` 保持一致。

**状态**: ✅ 已修复

---

#### BUG-005.7: resolveFieldSet 未选中字段的明细表泄漏全部字段 ✅ 已修复

**位置**: `go-service/internal/service/audit_review_service.go` — `resolveFieldSet`

**问题描述**:
当某张明细表没有任何字段被选中时（`dtSet` 为空 map），原逻辑跳过写入 `fieldSet[dt.TableName]`。下游 `formatGroupedDetailData` 在 `fieldSet` 中找不到该表名时，`allowedKeys` 为 `nil`，`filterRowFields` 对 `nil` 不做过滤，导致该表的所有字段全部输出给 AI。

**修复方案（已实施）**:
始终将 `dtSet`（即使为空 map）写入 `fieldSet`，使下游 `filterRowFields` 收到空 map 后返回 `nil`，`formatGroupedDetailData` 跳过该表。

**状态**: ✅ 已修复

---

#### BUG-008: 前端错误消息未国际化 ✅ 已修复

**位置**:
- `frontend/composables/useAuth.ts`
- `frontend/locales/zh-CN.ts`
- `frontend/locales/en-US.ts`

**问题描述**:
`useAuth.ts` 中的错误消息（如 "网络连接失败，请检查网络"、"登录已过期，请重新登录"、"请求失败"）为硬编码中文，不支持英文环境。

**修复内容**:
- 新增 `getErrorMessageByCode(code)` — 优先从 i18n messages 获取翻译，降级使用硬编码映射
- 新增 `getI18nText(key)` — 从 localStorage 中的 locale 判断语言并返回翻译文案
- `authFetch` 中所有错误消息改用 i18n 函数
- 新增 i18n 键：`auth.sessionExpired` / `auth.networkError` / `auth.requestFailed` / `auth.error.*` 系列

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

**状态**: 📋 待修复

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

**状态**: 📋 待修复

---

## 3. 优化建议汇总

### 3.1 安全性优化

| 编号 | 优化项 | 优先级 | 状态 |
|-----|-------|-------|------|
| SEC-001 | Token TTL 配置统一 | P0 | ✅ 已修复 |
| SEC-002 | 默认密码可配置化 | P1 | 📋 待修复 |
| SEC-003 | 首次登录强制改密 | P1 | 📋 待修复 |
| SEC-004 | 添加密码复杂度校验 | P2 | 📋 待修复 |

### 3.2 用户体验优化

| 编号 | 优化项 | 优先级 | 状态 |
|-----|-------|-------|------|
| UX-001 | Token 过期自动清理 | P0 | ✅ 已修复 |
| UX-002 | Token 即将过期预警刷新 | P1 | ✅ 已修复（含在 TokenGuard 中） |
| UX-003 | 页面可见性变化时检查登录态 | P1 | ✅ 已修复（含在 TokenGuard 中） |
| UX-004 | 前端错误消息国际化 | P1 | ✅ 已修复 |

### 3.3 功能完善

| 编号 | 优化项 | 优先级 | 状态 |
|-----|-------|-------|------|
| FEAT-001 | 接入审批流数据 | P1 | ✅ 已修复 |
| FEAT-002 | 添加操作审计日志 | P2 | 📋 待修复 |

### 3.4 代码质量

| 编号 | 优化项 | 优先级 | 状态 |
|-----|-------|-------|------|
| CODE-001 | 抽取租户管理员检查方法 | P2 | 📋 待修复 |
| CODE-002 | 重构字段合并逻辑 | P2 | 📋 待修复 |
| CODE-003 | 添加单元测试 | P2 | 📋 待修复 |

---

## 4. 修复优先级排序

### 第一阶段（立即修复）✅ 已完成

1. ✅ **BUG-001**: Token TTL 配置不同步（Login / Refresh / SwitchRole 全部修复）
2. ✅ **BUG-002**: 前端 Token 过期未自动清理
3. ✅ **BUG-004**: Session 缓存 TTL 优化
4. ✅ **BUG-008**: 前端错误消息国际化

### 第二阶段（1 周内）

5. 📋 **BUG-003**: 默认密码硬编码
6. ✅ ~~**BUG-005**: 审批流信息接入~~

### 第三阶段（2 周内）

7. 📋 **BUG-006**: 租户管理员保护逻辑重复
8. 📋 **BUG-007**: 字段合并逻辑重构
9. 📋 添加单元测试

---

## 5. 测试建议

### 5.1 认证系统测试用例

```
1. Token 过期测试
   - 等待 Access Token 过期，验证 TokenGuard 自动刷新
   - 等待 Refresh Token 过期，验证自动登出并清除本地状态
   - 在其他设备登出，验证当前设备 Token 失效
   - 修改数据库 auth.access_token_ttl_hours，验证新 Token 使用新 TTL

2. 并发刷新测试
   - 同时发起多个 API 请求触发 401
   - 验证只有一个刷新请求发出
   - 验证所有请求都能正确重试

3. 页面可见性测试
   - 将页面切到后台等待 Token 过期
   - 切回前台验证 TokenGuard 立即检查并刷新
   - 刷新失败时验证自动跳转登录页

4. 国际化测试
   - 切换语言为 en-US，验证错误消息显示英文
   - 切换语言为 zh-CN，验证错误消息显示中文
   - Token 过期登出提示使用当前语言
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
