# 审核工作台功能文档

> OA 智审平台 — 审核工作台（`/dashboard`）前端功能规范与接口对接

---

## 1. 页面总体结构

审核工作台是业务用户的核心工作页面，分为上下两部分：

### 1.1 上方：统计卡片区（四个页签）

四个可点击的统计卡片，点击切换下方列表的数据源：

| 页签 | 数据含义 | 数据来源 | 关键约束 |
|------|---------|---------|---------|
| 待审核 | OA 系统中分配给当前用户的待审批流程 | OA 数据库实时同步 | 包含所有待办，无论是否使用过 AI 审核 |
| 已通过 | 用户**通过审核工作台**处理并通过的流程 | 本平台审核记录 | 仅展示经过 AI 审核的流程，直接在 OA 中通过的不在此列 |
| 已驳回 | 用户**通过审核工作台**处理并驳回的流程 | 本平台审核记录 | 同上，仅经过 AI 审核的 |
| 已归档 | 已完结并归档的流程（经过审核工作台处理的） | 本平台归档记录 | 可查看完整审核历史链 |

**数据隔离**：每个用户只能看到自己相关的流程。待审核来自 OA 系统的个人待办；已通过/已驳回/已归档来自本平台记录的该用户的审核操作历史。

### 1.2 下方：主内容区（左右分栏）

| 区域 | 内容 |
|------|------|
| 左侧面板 | 流程列表（搜索、筛选、分页） |
| 右侧面板 | 选中流程的审核详情 / AI 审核结果 |

---

## 2. 左侧面板：流程列表

### 2.1 列表项展示

每个流程项展示以下信息：

| 字段 | 说明 | 变更说明 |
|------|------|---------|
| 标题 | 流程标题 | 不变 |
| 申请人 · 部门 · 提交时间 | 基本信息 | 不变 |
| 当前审批节点 | 当前流程走到哪个审批节点 | **新增**，替代原来的金额展示 |
| 流程分类标签 | 如"采购审批"、"费用报销" | 不变 |
| OA 跳转按钮 | 跳转到 OA 系统查看原始流程 | 不变 |

**移除项**：
- ~~金额展示~~：不是每个流程都涉及金额，改为展示当前审批节点
- ~~紧急程度标签~~：移除"一般"、"紧急"等标签，不需要此功能

### 2.2 搜索与筛选

- 关键词搜索：按标题、申请人模糊搜索（已有）
- **新增：流程分类筛选**：下拉选择流程分类（采购审批、费用报销、合同审批、人事审批等），支持多选

### 2.3 批量审核（新增）

仅在"待审核"页签下显示：

- 列表项左侧新增复选框，支持勾选多个流程
- 列表顶部显示"批量审核"按钮（勾选后激活）
- 点击后调用批量审核接口（见 `AI_INTERACTION_API.md` 第 4 节）
- 显示批量审核进度条（总数、已完成、进行中、失败）
- 批量审核完成后，每个流程可单独查看结果

---

## 3. 右侧面板：审核详情

### 3.1 待审核模式

选中流程后，右侧显示：

**未审核时**：
- 流程基本信息（标题、申请人、部门、提交时间、当前审批节点）
- "开始 AI 审核"按钮
- "跳转 OA 系统"按钮

**审核中**：
- 两阶段进度展示：
  - 阶段一：AI 深度分析中...（脉冲动画）
  - 阶段二：结构化结果提取中...
- 可提前展示部分结果（当阶段一完成时）

**审核完成后**：
- **建议横幅**：显示建议选项（建议通过/建议退回/建议驳回/建议人工复核）+ 综合得分 + 耗时 + trace_id
- **规则校验详情**：逐条规则展示通过/不通过状态及判断理由，强制规则标记"强制"徽章
- **AI 推理分析**：完整的推理分析文本，含风险点和改进建议
- 操作按钮：跳转 OA、重新审核

### 3.2 已通过/已驳回模式

选中流程后，右侧显示：

- 历史只读标记（"历史记录 · 只读"徽章）
- 该流程最后一次 AI 审核的完整结果（建议、得分、规则校验、推理分析）
- "查看审核历史链"按钮 → 打开抽屉，展示该流程的所有审核快照（时间线形式）
- "跳转 OA"按钮

### 3.3 已归档模式

与已通过/已驳回类似，但：

- 标记为"已归档 · 只读"
- 审核历史链展示该流程从首次审核到最终归档的完整链路
- 每个快照节点可展开查看当次审核的详细结果

---

## 4. 审核历史链（抽屉）

点击"查看审核历史链"打开右侧抽屉，展示：

- 时间线形式，每个节点代表一次 AI 审核
- 每个节点显示：建议选项标签、得分、审核时间、是否被用户采纳
- **新增**：点击节点可展开查看该次审核的完整详情（规则校验 + 推理分析）
- 最新的审核在最上方

---

## 5. 接口对接清单

### 5.1 待审核列表

**GET** `/api/audit/todo`

```
Query: ?page=1&size=20&search=keyword&process_type=采购审批,费用报销
```

**响应**:
```json
{
  "processes": [
    {
      "process_id": "WF-2025-001",
      "title": "办公设备采购申请",
      "applicant": "张明",
      "department": "研发部",
      "submit_time": "2025-06-10 09:30",
      "process_type": "采购审批",
      "status": "pending",
      "current_node": "财务总监审批",
      "oa_url": "https://oa.example.com/workflow/view/WF-2025-001"
    }
  ],
  "total": 12,
  "page": 1,
  "size": 20,
  "process_types": ["采购审批", "费用报销", "合同审批", "人事审批", "预算审批", "工程审批"]
}
```

**变更**：
- 新增 `current_node` 字段（当前审批节点）
- 新增 `process_type` 查询参数（流程分类筛选，逗号分隔多选）
- 移除 `amount` 和 `urgency` 字段
- 新增 `process_types` 响应字段（可选的流程分类列表，用于筛选下拉框）

### 5.2 已通过/已驳回/已归档列表

**GET** `/api/audit/history`

```
Query: ?status=approved|rejected|archived&page=1&size=20&search=keyword&process_type=采购审批
```

**响应**：同 5.1 结构，`status` 为对应值。

**关键约束**：此接口仅返回经过审核工作台 AI 审核处理过的流程。用户在 OA 中直接通过但未使用本平台审核的流程不会出现在此列表中。

### 5.3 执行 AI 审核

见 `AI_INTERACTION_API.md` 第 3.1 节。

### 5.4 批量审核

见 `AI_INTERACTION_API.md` 第 4 节。

### 5.5 获取审核历史链

**GET** `/api/audit/chain/{process_id}`

**响应**:
```json
{
  "process_id": "WF-2025-050",
  "chain": [
    {
      "snapshot_id": "SN-A003",
      "trace_id": "TR-20250510-M1N2",
      "recommendation": "approve",
      "action_label": "建议通过",
      "score": 95,
      "created_at": "2025-05-10 09:15",
      "adopted": true,
      "rule_checks": {
        "total": 3,
        "passed": 3,
        "failed": 0,
        "details": [...]
      },
      "ai_analysis": {
        "summary": "...",
        "full_reasoning": "..."
      }
    }
  ]
}
```

每个快照包含完整的审核结果，前端可按需展开。

### 5.6 审核反馈

见 `AI_INTERACTION_API.md` 第 5 节。

---

## 6. 租户管理员配置变更

### 6.1 新增审批流程时

- ~~审批路径~~：移除 `flow_path` 输入
- **新增：主表表名**（`main_table_name`）：OA 数据库中该流程对应的主表名称，用于数据同步

### 6.2 字段配置变更

原来的字段列表不区分主表和明细表，现在需要区分：

**主表字段**：每个流程都有，对应 OA 主表
**明细表字段**：有的流程有，有的没有，有的有多个明细表

数据结构变更：

```json
{
  "main_table_name": "formtable_main_001",
  "main_fields": [
    { "field_key": "amount", "field_name": "采购金额", "field_type": "number", "selected": true }
  ],
  "detail_tables": [
    {
      "table_name": "formtable_main_001_dt1",
      "table_label": "采购明细",
      "fields": [
        { "field_key": "item_name", "field_name": "物品名称", "field_type": "text", "selected": true }
      ]
    }
  ]
}
```

**UI 变更**：
- 字段不再直接平铺展示
- 改为右侧"+"按钮，点击弹出字段选择弹框
- 弹框内分"主表字段"和"明细表字段"两个区域
- 明细表为数组形式，可添加多个明细表
- 选择好的字段回显到配置界面上

### 6.3 审核规则变更

编辑规则时新增：

- **是否关联审批流**（`related_flow: boolean`）：标识该规则校验时是否需要参考已过的审批流节点信息
- 例如"金额超过50万需总经理审批"这条规则，需要关联审批流来验证总经理是否已审批

### 6.4 AI 配置变更

- **服务商和模型关联系统设置**：AI 服务商和模型列表不再硬编码，改为使用系统设置中配置给对应租户可用的模型，考虑多模型的情况，在系统配置中进行修改可以配置多模型
- **多模型编排**：支持为推理阶段和提取阶段配置不同模型（见 `AI_INTERACTION_API.md` 6.3 节）
- **提示词变量**：在提示词编辑区域增加变量插入功能，可插入 `{{main_table}}`、`{{detail_tables}}`、`{{rules}}`、`{{flow_history}}`、`{{current_node}}` 等变量
- **交互模式选择**：`two_phase`（两阶段，默认）或 `single_pass`（单次）

---

## 7. 个人设置 — 审核工作台配置

参考租户管理的配置结构，用户可在个人设置中进行个性化调整（受 `user_permissions` 控制）：

- 字段选择覆盖（当 `allow_custom_fields` 为 true）
- 自定义规则叠加（当 `allow_custom_rules` 为 true）
- 审核尺度调整（当 `allow_modify_strictness` 为 true）
- 规则开关覆盖（对 `default_on` / `default_off` 规则的启用/禁用）

**不涉及的不需要改变**：主表表名、明细表配置、AI 模型选择、提示词等属于租户管理员权限，个人设置中不展示。

---

## 8. Mock 数据变更清单

为配合上述功能变更，需要更新的 mock 数据：

| 数据 | 变更内容 | 状态 |
|------|---------|------|
| `OAProcess` 类型 | 新增 `current_node`、`oa_url`；`amount` 和 `urgency` 改为可选（deprecated） | ✅ 已完成 |
| `mockProcesses` | 每条数据增加 `current_node`；`amount`、`urgency` 保留但标记 deprecated | 待完成 |
| `mockApprovedProcesses` | 同上 | 待完成 |
| `mockRejectedProcesses` | 同上 | 待完成 |
| `mockArchivedOAProcesses` | 同上 | 待完成 |
| `AuditResultV2` 类型（新增） | 新接口，`recommendation` 为对象（含 `action`、`action_label`、`score`、`confidence`）；含 `rule_checks`、`ai_analysis`、`meta` 结构 | ✅ 已完成 |
| `AuditResult` 类型（legacy） | `recommendation` 枚举扩展（新增 `return`、`review`）；新增 v2 兼容字段（`action_label`、`confidence`、`risk_points` 等）为可选 | ✅ 已完成 |
| `ChecklistResult` 类型 | 新增 `related_flow` 字段 | ✅ 已完成 |
| `mockAuditResult` | 适配 `AuditResultV2` 新结构 | 待完成 |
| `mockHistoricalResults` | 适配新结构 | 待完成 |
| `ProcessAuditConfig` | 新增 `main_table_name`、`detail_tables`；字段区分主表/明细表 | 待完成 |
| 审核规则 mock 数据 | 新增 `related_flow` 字段 | 待完成 |
| AI 配置 | 新增 `reasoning_model`、`extraction_model`、`interaction_mode` | 待完成 |
| 批量审核 | 新增 `mockBatchAuditResult` | 待完成 |
