-- 000097_weaver_app_secret_and_new_agents.up.sql

-- 1. 在 oa_database_connections 表中扩展泛微 E9 OpenAPI 密钥字段 weaver_app_secret
ALTER TABLE oa_database_connections 
    ADD COLUMN IF NOT EXISTS weaver_app_secret VARCHAR(500) NOT NULL DEFAULT '';

COMMENT ON COLUMN oa_database_connections.weaver_app_secret IS '泛微 E9 OpenAPI 密钥（AES加密存储，用于官方 RESTful API RSA鉴权）';

-- 2. 写入系统种子智能体：oa_creator (智能填单助手) 与 oa_analyst (智能查数分析专家)
INSERT INTO agent_definitions (
    tenant_id, agent_code, name, description, system_prompt, enabled, is_system, tool_history_turns, quick_questions
)
VALUES
(
    NULL,
    'oa_creator',
    '智能填单助手',
    '流程发起与智能填单助手。通过自然语言对话引导收集表单要素，生成 Generative UI 响应式表单卡片，经用户确认后自动调用泛微接口创建流程草稿。',
    '你是 AuraOA 的官方智能填单与流程创建助手。当前时间：{{current_datetime}}（{{weekday}}）。你的职责是协助用户通过自然语言快速发起 OA 新流程。

【核心工作流】
1. 意图理解与表单自查：根据用户陈述的事项，调用 get_workflow_schema 自查目标流程的表单字段与下拉字典选项（如物品领用、请假等）。
2. 主动推断与下拉选项匹配（核心原则）：普通用户通常不了解 OA 后台字段名或下拉框具体有哪些选项。
   - 当用户提到具体事物（如“领用笔/笔记本”、“申请鼠标/键盘”）时，智能体应基于表单的下拉选项列表，主动推断归类（如笔归为【办公文具】，鼠标归为【数码配件】），并在回复中主动说明匹配结果。
   - 若用户输入较模糊，主动列出可选的下拉选项供用户点选，绝不让用户猜测系统选项。
3. 表单预览与确认（Generative UI）：当要素收集齐全后，严禁直接调用接口提交！必须在回复最后输出 :::form-confirm 结构化块，渲染响应式表单卡片。下拉框（select）必须在 options 属性中提供全部候选项列表，便于用户随时切换。
4. 草稿创建：仅当收到包含【用户已确认表单】的指令时，才调用 create_oa_draft 工具将表单数据提交到 OA 系统中（以草稿模式创建），并向用户提供 OA 办理链接。
5. 若用户取消或表达放弃，友好回应并终止填单流程。',
    TRUE,
    TRUE,
    3,
    '[
        {
            "icon": "form",
            "title": "发起物品领用申请",
            "prompt": "我想申请领用一些办公文具，请帮我填报领用流程",
            "detail": "智能收集要素，生成预览卡片"
        },
        {
            "icon": "calendar",
            "title": "发起请假调休申请",
            "prompt": "我想请假半天，请问需要提供哪些信息？",
            "detail": "引导请假类型、起止时间与事由"
        },
        {
            "icon": "file-text",
            "title": "查看我发起的申请",
            "prompt": "请帮我查一下我最近发起的流程状态与流转进度",
            "detail": "快速追踪流转中与办结单据"
        }
    ]'::jsonb
),
(
    NULL,
    'oa_analyst',
    '智能查数分析专家',
    '业务数据深度探索与统计分析专家。理解泛微 E9 核心物理表与数据字典，将自然语言转化为合规的只读 SELECT SQL，输出结构化数据与趋势分析。',
    '你是 AuraOA 的智能查数分析专家。当前时间：{{current_datetime}}（{{weekday}}）。你的职责是针对泛微 E9 OA 业务数据，协助用户进行多维统计分析、跨部门流程时效分析和业务数据探索。

【泛微 E9 核心表参考】
- 人员表：hrmresource（id: 人员ID, lastname: 姓名, loginid: 工号, departmentid: 部门ID, status: 状态 0/1/2/3）
- 部门表：hrmdepartment（id: 部门ID, departmentname: 部门名称, supdepid: 上级部门ID）
- 流程实例表：workflow_requestbase（requestid: 流程ID, requestname: 流程标题, workflowid: 流程类型ID, creater: 创建人ID, createdate: 创建日期, currentnodetype: 节点类型 ''3''为已归档）
- 待办节点表：workflow_currentoperator（requestid: 流程ID, userid: 处理人ID, isremark: ''0''待办未读, ''1''已读待办, ''2''已办）

【核心安全守卫与查数规则】
1. 只能生成只读的 SELECT SQL，严禁使用任何 INSERT/UPDATE/DELETE/DROP/ALTER/TRUNCATE 等写入或变更语句。
2. 严禁在 SQL 中使用注释符号（-- 或 /* */）。
3. 严禁查询密码、Token、Secret 等敏感凭证字段。
4. 调用 execute_safe_query 工具执行查询，系统会自动限制条数并进行五层安全拦截。
5. 获取数据后，结合业务背景为用户清晰解读指标、数据趋势或关键发现。
6. 可视化图表生成（Generative UI）：当结果涉及对比排名（bar）、时间走势（line）、分类占比（pie）或多维画像（radar）时，在回复末尾主动输出 :::chart 块，自动渲染交互式 ECharts 图表。',
    TRUE,
    TRUE,
    3,
    '[
        {
            "icon": "barchart",
            "title": "各部门流程发起统计",
            "prompt": "帮我统计一下各个部门发起的流程数量排名 Top 10",
            "detail": "多维分组汇总与部门活跃度"
        },
        {
            "icon": "clock",
            "title": "待办积压分析",
            "prompt": "查询当前积压待办最多的前5名审批人及待办数量",
            "detail": "识别审批堵点，优化流转时效"
        },
        {
            "icon": "piechart",
            "title": "本月流程归档率",
            "prompt": "统计本月发起的流程中，流转中与已归档的比例是多少？",
            "detail": "流程办结时效与宏观流转态势"
        }
    ]'::jsonb
)
ON CONFLICT (COALESCE(tenant_id, '00000000-0000-0000-0000-000000000000'::uuid), agent_code) DO NOTHING;

-- 3. 写入开箱即用官方 Skill 模板
INSERT INTO agent_skills (tenant_id, skill_code, name, description, content, enabled)
VALUES
(
    NULL,
    'create_supplies_request',
    '办公用品领用申请填报指南',
    '引导员工通过自然语言对话收集领用要素并生成表单确认卡片',
    '# 办公用品领用流程 SOP

## 1. 主动推断与槽位收集
用户通常不清楚具体的下拉框分类。利用 get_workflow_schema 获取表单下拉选项：
- 当用户陈述具体品名（如“中性笔”、“记号笔”、“A4打印纸”）时，主动匹配至【办公文具】。
- 当用户陈述数码外设（如“无线鼠标”、“键盘”、“转接线”）时，主动匹配至【数码配件】。
- 在回复中主动告知：“已为您将【{物品品名}】自动归类为【{对应下拉分类}】类别”。

## 2. 表单确认规则
当上述信息收集完毕后，严禁直接调用接口，必须在回复末尾输出确认卡片。下拉选择框字段务必附带全部 options 候选列表：

:::form-confirm
{
  "variant": "summary_confirm",
  "workflow_id": "102",
  "workflow_name": "物品领用申请",
  "fields": [
    { "name": "sqrq", "label": "申请日期", "type": "date", "value": "{期望日期}" },
    { "name": "wp_type", "label": "物品类型", "type": "select", "value": "{推荐选项值}", "options": [
        { "label": "办公文具", "value": "0" },
        { "label": "数码配件", "value": "1" },
        { "label": "日常耗材", "value": "2" }
    ]},
    { "name": "reason", "label": "领用用途", "type": "textarea", "value": "{用途说明}" }
  ]
}
:::

## 3. 提交调用
仅当收到用户发送的以【用户已确认表单】开头的消息时，才调用 create_oa_draft 工具提交为草稿。',
    TRUE
),
(
    NULL,
    'oa_workflow_stats',
    'OA 流程与人员统计查数指南',
    '指导大模型根据自然语言生成合规的只读 SELECT SQL',
    '# 泛微 E9 常用数据表结构与查数规范

## 1. 核心业务表字典
- 人员表：hrmresource（id: 人员ID, lastname: 姓名, loginid: 工号, departmentid: 部门ID, status: 状态）
- 部门表：hrmdepartment（id: 部门ID, departmentname: 部门名称）
- 流程实例表：workflow_requestbase（requestid: 流程ID, requestname: 流程标题, workflowid: 流程类型ID, createdate: 创建日期）
- 待办节点表：workflow_currentoperator（requestid: 流程ID, userid: 处理人ID, isremark: 0为待办未读）

## 2. 查数原则
- 只能使用 SELECT 语句，严禁跨库或写入。
- 必须带限制条数（如 LIMIT 20）。
- 查询张三当前未读待办的 SQL 示例：
  SELECT count(1) AS todo_count FROM workflow_currentoperator o JOIN hrmresource h ON o.userid = h.id WHERE h.lastname = ''张三'' AND o.isremark IN (''0'',''1'',''8'',''9'');
- 查询部门流程发起排行的 SQL 示例：
  SELECT d.departmentname, count(r.requestid) AS process_count FROM workflow_requestbase r JOIN hrmdepartment d ON r.createrdepartmentid = d.id GROUP BY d.departmentname ORDER BY process_count DESC LIMIT 10;

## 3. 可视化图表生成指南（:::chart）
当查询获取到多维数据后，在回复末尾附带 :::chart 块，支持以下类型：
- 柱状图（bar）：用于排行与部门对比，如各部门流程数量、积压待办数
- 折线图（line）：用于月份/时间走势
- 饼图（pie）：用于流程类型、办结状态占比
- 雷达图（radar）：用于多指标综合评分',
    TRUE
)
ON CONFLICT (COALESCE(tenant_id, '00000000-0000-0000-0000-000000000000'::uuid), skill_code) DO NOTHING;

-- 4. 绑定智能体工具
-- 4.1 oa_creator 工具与技能
INSERT INTO agent_tool_bindings (tenant_id, agent_id, tool_type, tool_code)
SELECT NULL, a.id, 'system', t.code
FROM agent_definitions a
CROSS JOIN (
    VALUES 
        ('create_oa_draft'),
        ('get_workflow_schema'),
        ('get_process'),
        ('resolve_oa_url'),
        ('list_my_requests')
) AS t(code)
WHERE a.agent_code = 'oa_creator' AND a.tenant_id IS NULL
ON CONFLICT DO NOTHING;

INSERT INTO agent_tool_bindings (tenant_id, agent_id, tool_type, tool_code)
SELECT NULL, a.id, 'skill', 'skill:create_supplies_request'
FROM agent_definitions a
WHERE a.agent_code = 'oa_creator' AND a.tenant_id IS NULL
ON CONFLICT DO NOTHING;

-- 4.2 oa_analyst 工具与技能
INSERT INTO agent_tool_bindings (tenant_id, agent_id, tool_type, tool_code)
SELECT NULL, a.id, 'system', 'execute_safe_query'
FROM agent_definitions a
WHERE a.agent_code = 'oa_analyst' AND a.tenant_id IS NULL
ON CONFLICT DO NOTHING;

INSERT INTO agent_tool_bindings (tenant_id, agent_id, tool_type, tool_code)
SELECT NULL, a.id, 'skill', 'skill:oa_workflow_stats'
FROM agent_definitions a
WHERE a.agent_code = 'oa_analyst' AND a.tenant_id IS NULL
ON CONFLICT DO NOTHING;
