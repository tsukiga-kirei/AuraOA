# 核心业务逻辑分析报告

## 1. 概述

本文档分析 OA 智审系统的核心业务逻辑，包括 OA 数据提取、规则组装、提示词构建、AI 审核执行等流程。

---

## 2. 业务流程架构

### 2.1 审核工作台流程

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          审核工作台完整流程                               │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  1. 用户选择流程类型                                                     │
│         │                                                               │
│         ▼                                                               │
│  2. 获取完整配置 (GetFullAuditProcessConfig)                            │
│     ├── 租户流程配置 (process_audit_configs)                            │
│     ├── 租户审核规则 (audit_rules)                                      │
│     ├── 用户个人配置 (user_personal_configs.audit_details)              │
│     └── 合并字段/规则/AI尺度                                            │
│         │                                                               │
│         ▼                                                               │
│  3. 用户输入流程 ID，发起审核                                            │
│         │                                                               │
│         ▼                                                               │
│  4. 从 OA 系统提取流程数据                                               │
│     ├── 连接 OA 数据库 (OAAdapter)                                      │
│     ├── 查询主表数据                                                    │
│     └── 查询明细表数据                                                  │
│         │                                                               │
│         ▼                                                               │
│  5. 规则合并 (MergeRules)                                               │
│     ├── 租户强制规则 (mandatory) → 始终生效                             │
│     ├── 租户默认规则 (default_on/off) → 应用用户 toggle                 │
│     └── 用户自定义规则 (custom) → 追加                                  │
│         │                                                               │
│         ▼                                                               │
│  6. 构建 AI 提示词                                                      │
│     ├── 推理阶段提示词 (BuildReasoningPrompt)                           │
│     │   └── 替换占位符: {{main_table}}, {{detail_tables}}, {{rules}}    │
│     └── 提取阶段提示词 (BuildExtractionPrompt)                          │
│         │                                                               │
│         ▼                                                               │
│  7. 调用 AI 模型 (两阶段)                                               │
│     ├── 阶段1: 推理分析 → 自由文本输出                                  │
│     └── 阶段2: 结构化提取 → JSON 输出                                   │
│         │                                                               │
│         ▼                                                               │
│  8. 解析 AI 响应，返回审核结果                                          │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 3. 关键代码分析

### 3.1 配置获取与合并

**文件**: `go-service/internal/service/user_personal_config_service.go`

**方法**: `GetFullAuditProcessConfig`

```go
func (s *UserPersonalConfigService) GetFullAuditProcessConfig(c *gin.Context, userID uuid.UUID, processType string) (*dto.FullAuditProcessConfigResponse, error) {
    // 1. 获取租户流程审核配置
    tenantCfg, err := s.configRepo.GetByProcessType(c, processType)
    
    // 2. 解析用户权限
    var perms model.UserPermissionsData
    json.Unmarshal(tenantCfg.UserPermissions, &perms)
    
    // 3. 获取租户审核规则
    tenantRules, err := s.auditRuleRepo.ListByConfigID(c, tenantCfg.ID)
    
    // 4. 获取用户个人配置
    userCfg, err := s.userConfigRepo.GetByTenantAndUser(c, tenantID, userID)
    
    // 5. 合并字段配置
    // - 租户 field_mode = "all" → 用户必须使用所有字段
    // - 租户 field_mode = "selected" → 用户可在租户选择基础上新增
    
    // 6. 合并规则配置
    // - mandatory 规则始终生效
    // - default_on/off 规则应用用户 toggle
    // - 用户自定义规则追加
    
    // 7. 合并 AI 尺度
    // - 用户覆盖优先（如果允许）
}
```

**✅ 逻辑正确性**: 配置合并逻辑清晰，权限控制完善。

---

### 3.2 规则合并逻辑

**文件**: `go-service/internal/service/rule_merge.go`

```go
func MergeRules(tenantRules []MergeableRule, userOverride *UserRuleOverride) []MergedRule {
    // 1. 构建用户 toggle 覆盖映射
    toggleMap := make(map[string]bool)
    for _, toggle := range userOverride.RuleToggleOverrides {
        toggleMap[toggle.RuleID] = toggle.Enabled
    }
    
    // 2. 处理租户规则
    for _, rule := range tenantRules {
        switch rule.GetRuleScope() {
        case "mandatory":
            merged.Enabled = true  // 强制规则始终生效
        case "default_on":
            merged.Enabled = true
            if userEnabled, exists := toggleMap[rule.GetID()]; exists {
                merged.Enabled = userEnabled  // 用户可关闭
            }
        case "default_off":
            merged.Enabled = false
            if userEnabled, exists := toggleMap[rule.GetID()]; exists {
                merged.Enabled = userEnabled  // 用户可开启
            }
        }
    }
    
    // 3. 添加用户私有规则
    for _, customRule := range userOverride.CustomRules {
        result = append(result, MergedRule{
            Scope:   "custom",
            Enabled: customRule.Enabled,
            Source:  "user",
        })
    }
    
    // 4. 按优先级排序: mandatory > custom > default_on > default_off
}
```

**✅ 逻辑正确性**: 规则合并优先级清晰，mandatory 规则不可被用户覆盖。

---

### 3.3 提示词构建

**文件**: `go-service/internal/service/audit_prompt_builder.go`

```go
func BuildReasoningPrompt(aiConfig *model.AIConfigData, processType string, processData *oa.ProcessData, rules string, currentNode string, fieldSet SelectedFieldSet) *ai.ChatRequest {
    mainDataStr := formatMainData(filterFields(processData.MainData, fieldSet["main"]))
    detailDataStr := formatGroupedDetailData(processData.DetailTables, fieldSet)

    userPrompt := aiConfig.UserReasoningPrompt
    userPrompt = strings.ReplaceAll(userPrompt, "{{process_type}}", processType)
    userPrompt = strings.ReplaceAll(userPrompt, "{{main_table}}", mainDataStr)
    userPrompt = strings.ReplaceAll(userPrompt, "{{detail_tables}}", detailDataStr)
    userPrompt = strings.ReplaceAll(userPrompt, "{{rules}}", rules)
    userPrompt = strings.ReplaceAll(userPrompt, "{{current_node}}", currentNode)
    userPrompt = strings.ReplaceAll(userPrompt, "{{flow_history}}", "（暂未提供）")  // ⚠️ 待实现
    userPrompt = strings.ReplaceAll(userPrompt, "{{flow_graph}}", "（暂未提供）")   // ⚠️ 待实现

    return &ai.ChatRequest{
        SystemPrompt: aiConfig.SystemReasoningPrompt,
        UserPrompt:   userPrompt,
        RequestType:  "audit",
    }
}
```

**⚠️ 待完善**: `{{flow_history}}` 和 `{{flow_graph}}` 占位符尚未实现，当前使用固定文本。

---

## 4. 发现的问题

### 🟡 问题 1: 审批流信息未接入

**严重程度**: 中

**问题描述**:
提示词模板中定义了 `{{flow_history}}` 和 `{{flow_graph}}` 占位符，但实际代码中使用固定文本 "（暂未提供）"。

**影响**:
- AI 无法获取审批流上下文
- 无法分析审批节点的完整性和合理性
- 降低审核准确性

**修复建议**:
1. 在 OA 适配器中实现审批流数据提取
2. 格式化审批流历史为结构化文本
3. 替换占位符时注入真实数据

---

### 🟡 问题 2: 字段过滤逻辑复杂度高

**严重程度**: 低

**问题描述**:
`GetFullAuditProcessConfig` 方法中字段合并逻辑较为复杂，涉及多层嵌套判断。

```go
// 字段同步逻辑：
// 如果租户是 'all' 模式，用户侧强制显示所有，且不允许自定义减少
// 如果租户是 'selected' 模式，用户侧默认包含租户选择的所有字段，且只能新增不能减少
effectiveFieldMode := tenantCfg.FieldMode
// ... 多层嵌套判断
```

**建议**: 考虑将字段合并逻辑抽取为独立函数，提高可读性和可测试性。

---

### 🟢 问题 3: 规则同步清理逻辑

**严重程度**: 低（已处理）

**代码位置**: `user_personal_config_service.go`

```go
// 规则同步逻辑：过滤掉已经不存在的租户规则覆盖
validRuleToggles := []model.RuleToggleOverride{}
tenantRuleMap := make(map[string]bool)
for _, tr := range tenantRules {
    tenantRuleMap[tr.ID.String()] = true
}
for _, ut := range userDetail.RuleConfig.RuleToggleOverrides {
    if tenantRuleMap[ut.RuleID] {
        validRuleToggles = append(validRuleToggles, ut)
    }
}
```

**✅ 已正确处理**: 当租户删除规则后，用户的 toggle 覆盖会被自动过滤，不会产生脏数据。

---

## 5. 访问控制逻辑分析

### 5.1 流程访问控制

**文件**: `user_personal_config_service.go` - `GetProcessList`

```go
// 访问控制规则：
// access_control 所有列表均为空 → 对所有租户成员开放
// 否则用户 ID/角色/部门命中任一列表即可访问

var ac model.AccessControlData
json.Unmarshal(cfg.AccessControl, &ac)

// 三列表均为空 → 公开
if len(ac.AllowedRoles) == 0 && len(ac.AllowedMembers) == 0 && len(ac.AllowedDepartments) == 0 {
    // 允许访问
}

// 检查成员 ID
if sliceContains(ac.AllowedMembers, member.ID.String()) { /* 允许 */ }

// 检查部门
if sliceContains(ac.AllowedDepartments, member.DepartmentID.String()) { /* 允许 */ }

// 检查角色
for _, r := range member.Roles {
    if sliceContains(ac.AllowedRoles, r.ID.String()) { /* 允许 */ }
}
```

**✅ 逻辑正确性**: 访问控制采用白名单机制，空列表表示公开，非空时需命中任一条件。

---

## 6. 数据流图

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           数据流向                                       │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐               │
│  │   OA 系统    │────►│  OA 适配器   │────►│  流程数据    │               │
│  │  (MySQL等)   │     │ (Weaver E9) │     │ (主表+明细)  │               │
│  └─────────────┘     └─────────────┘     └──────┬──────┘               │
│                                                  │                      │
│  ┌─────────────┐     ┌─────────────┐            │                      │
│  │ 租户配置     │────►│  配置合并    │◄───────────┤                      │
│  │ (字段/规则)  │     │   服务      │            │                      │
│  └─────────────┘     └──────┬──────┘            │                      │
│                             │                    │                      │
│  ┌─────────────┐            │                    │                      │
│  │ 用户个人配置 │────────────┤                    │                      │
│  │ (覆盖/自定义)│            │                    │                      │
│  └─────────────┘            ▼                    │                      │
│                      ┌─────────────┐            │                      │
│                      │  规则合并    │◄───────────┘                      │
│                      │ (MergeRules)│                                    │
│                      └──────┬──────┘                                    │
│                             │                                           │
│  ┌─────────────┐            │                                           │
│  │ 提示词模板   │────────────┤                                           │
│  │ (系统预置)   │            │                                           │
│  └─────────────┘            ▼                                           │
│                      ┌─────────────┐     ┌─────────────┐               │
│                      │ 提示词构建   │────►│  AI 模型    │               │
│                      │ (占位符替换) │     │ (推理+提取) │               │
│                      └─────────────┘     └──────┬──────┘               │
│                                                  │                      │
│                                                  ▼                      │
│                                          ┌─────────────┐               │
│                                          │  审核结果    │               │
│                                          │ (JSON结构)  │               │
│                                          └─────────────┘               │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 7. 代码质量评估

### ✅ 优点

1. **清晰的分层架构**: Handler → Service → Repository，职责分明
2. **完善的权限控制**: 用户权限锁定机制（AllowCustomFields/Rules/Strictness）
3. **灵活的规则系统**: 支持 mandatory/default_on/default_off/custom 四种作用域
4. **脏数据自动清理**: 租户规则删除后，用户覆盖自动过滤

### ⚠️ 待改进

1. 审批流信息尚未接入
2. 部分方法复杂度较高，可拆分
3. 缺少单元测试覆盖

---

## 8. 建议优化项

| 优先级 | 优化项 | 说明 |
|-------|-------|------|
| P1 | 接入审批流数据 | 实现 `{{flow_history}}` 和 `{{flow_graph}}` |
| P2 | 重构字段合并逻辑 | 抽取为独立函数，提高可读性 |
| P2 | 添加单元测试 | 覆盖规则合并、配置合并等核心逻辑 |
| P3 | 性能优化 | 考虑缓存热点配置数据 |
