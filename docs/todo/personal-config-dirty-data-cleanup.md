# TODO：个人配置脏数据清理

> 文档版本：v0.1 | 创建日期：2026-03-17  
> 记录租户修改后，用户个人配置中可能产生的脏数据现状、改进点及实现清理功能时的注意事项。

---

## 一、当前内容

### 1.1 数据存储结构

用户个人配置存储在 `user_personal_configs` 表，主要 JSONB 字段：

| 字段 | 说明 | 脏数据来源 |
|------|------|------------|
| `audit_details` | 审核工作台各流程的个性化配置 | 流程配置删除、字段同步、规则删除、权限关闭 |
| `archive_details` | 归档复盘各流程的个性化配置 | 同上 |
| `cron_details` | 定时任务偏好（如默认邮箱） | 较少涉及租户变更 |

每个流程详情（`AuditDetailItem` / `ArchiveDetailItem`）内嵌：

| 子字段 | 类型 | 说明 | 脏数据场景 |
|-------|------|------|------------|
| `field_config.field_overrides` | `[]string` | 用户额外勾选的字段 key | **租户同步字段后删除某字段**，该 key 仍保留 |
| `rule_config.rule_toggle_overrides` | `[]{rule_id, enabled}` | 用户对**租户通用规则**的开关覆盖 | **租户删除某条通用规则**，该 rule_id 仍保留 |
| `rule_config.custom_rules` | `[]{id, content, ...}` | 用户自定义规则 | 无直接脏数据（不依赖租户实体） |
| `ai_config.strictness_override` | `string` | 用户覆盖的审核尺度 | 无直接脏数据 |

### 1.2 租户设置的通用规则（与本 TODO 的关系）

**租户通用规则**：租户管理员在「规则配置」中为审核工作台/归档复盘配置的规则，存储在 `audit_rules` / `archive_rules` 表。用户可在个人设置中对每条规则进行开关覆盖（启用/禁用），覆盖结果保存在 `rule_config.rule_toggle_overrides` 中，以 `rule_id` 关联。

当租户**删除**某条通用规则时：
- 该规则在 `audit_rules` / `archive_rules` 中已不存在
- 用户配置中的 `rule_toggle_overrides` 仍可能包含该 `rule_id`
- 该 `rule_id` 即成为脏数据，需在展示时过滤，并在清理功能中持久化移除

### 1.3 当前「忽略」逻辑（不报错、不展示，但数据仍保留）

| 位置 | 逻辑 | 效果 |
|------|------|------|
| `user_personal_config_service.go` 合并字段 | 遍历**租户当前字段列表**，对每个字段查 `userAddedFieldKeys` | 若 `field_overrides` 中有租户已删除的 key，该 key 不会出现在任何字段上，**自然被忽略** |
| `user_config_management_handler.go` `filterToggleOverrides` | 遍历 toggles，仅保留 `rule_id` 存在于 `ruleMap` 中的项 | 若 `rule_toggle_overrides` 中有租户已删除的 rule_id（通用规则），**过滤掉不展示**，但**原始 JSON 未修改** |
| 权限关闭时的清空 | `applyFieldOverridesPerm` / `applyCustomRulesPerm` 等 | 权限关闭时**返回空**给前端，但**不写回数据库** |

**结论**：脏数据只存在于**读取路径的过滤逻辑**中，**没有写回清理**。数据库中的 `audit_details` / `archive_details` 会持续累积无效 key。

---

## 二、需要改进的点

1. **字段 `field_overrides`**：租户同步字段后删除某字段，用户配置中该 `field_key` 应被清理。
2. **规则 `rule_toggle_overrides`**：租户删除某条**通用规则**后，用户配置中该 `rule_id` 应被清理。
3. **流程配置删除**：租户删除整个流程配置（或归档访问控制不再包含该用户），用户 `audit_details` / `archive_details` 中对应流程条目应被清理。
4. **权限关闭**：租户关闭某权限后，是否要**持久化清空**用户对应内容（当前仅读时忽略，未写回）。

---

## 三、若实现删除/清理功能，可能遇到的问题

### 3.1 数据库扫描方式

| 方案 | 描述 | 优点 | 缺点 |
|------|------|------|------|
| **逐条扫描** | 按 `tenant_id` 查 `user_personal_configs`，逐条解析 JSON 后校验 | 实现简单，可精确控制 | 大量用户时性能差，需批量分页 |
| **批量 + 定时任务** | 定期 job 扫描全表，按租户/流程分批处理 | 不阻塞主流程 | 延迟清理，需额外 job 基础设施 |
| **写时清理** | 在用户保存/读取时顺带清理并写回 | 无额外扫描，实时生效 | 需在保存/读取路径增加校验逻辑，可能增加延迟 |
| **混合** | 读时过滤（保持现状）+ 保存时校验并清理 | 读路径不变，写路径做一次清理 | 未再次保存的用户脏数据仍保留 |

**建议**：优先考虑「写时清理」——在用户**保存个人配置**时，对 `field_overrides`、`rule_toggle_overrides` 做校验，剔除无效项后写回。若需「历史脏数据」一次性清理，可单独写迁移脚本或定时任务。

### 3.2 需要去除或调整的「忽略」逻辑

若实现持久化清理，以下逻辑需同步调整，避免重复处理或语义冲突：

| 文件 | 函数/逻辑 | 当前行为 | 调整建议 |
|------|-----------|----------|----------|
| `user_config_management_handler.go` | `filterToggleOverrides` | 读时过滤，不写回 | 若改为写时清理，此处可保留（仅影响展示），或考虑在「管理端」也触发一次清理写回 |
| `user_config_management_handler.go` | `applyFieldOverridesPerm` / `applyCustomRulesPerm` | 权限关闭时返回空 | 若改为写回清空，需在**保存接口**中增加对应逻辑 |
| `user_personal_config_service.go` | 合并字段时的 `userAddedFieldKeys` | 仅遍历租户字段，忽略多余 key | 若保存时清理，可在此处或保存前统一过滤 `field_overrides` 中不在租户字段列表的 key |

**注意**：`filterToggleOverrides` 同时做了「过滤已删除规则」和「过滤未修改项」两项，后者是业务需求（只展示用户修改过的），与脏数据清理无关，不应一并删除。

### 3.3 其他潜在问题

1. **并发**：用户保存时清理，若同时租户正在修改字段/规则，可能产生竞态。需在「校验时」使用的租户字段/规则列表与「保存时」一致（同一事务或快照）。
2. **归档访问控制**：用户失去某归档流程的访问权后，对应 `archive_details` 条目是否保留？当前管理端 `user-configs` 已做过滤不展示，但若要做持久化清理，需明确策略（立即删除 vs 保留待恢复）。
3. **审计**：清理属于「变更用户数据」，是否需记录审计日志，需评估。
4. **迁移**：历史脏数据需一次性清理时，建议写独立 migration 或脚本，避免与业务逻辑耦合。

---

## 四、后续行动项

- [ ] 确定清理策略：写时清理 / 定时任务 / 混合
- [ ] 在保存个人配置接口中增加 `field_overrides`、`rule_toggle_overrides` 校验与清理逻辑
- [ ] 评估是否对「权限关闭」做持久化清空
- [ ] 评估是否对「流程删除/访问权变更」做持久化清理
- [ ] 如需历史清理，编写迁移脚本或定时任务
- [ ] 更新相关文档与测试用例
