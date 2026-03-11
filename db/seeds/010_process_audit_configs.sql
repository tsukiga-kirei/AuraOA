-- 010_process_audit_configs.sql
-- Seed data: 流程审核配置 + 审核规则 + 审核尺度预设
-- Run after 003_tenants.sql (depends on tenants)
-- Run after 001_oa_ai_seeds.sql (ai_config references ai_model_configs)
--
-- 外键依赖：
--   process_audit_configs.tenant_id → tenants(id)
--   audit_rules.tenant_id → tenants(id)
--   audit_rules.config_id → process_audit_configs(id)
--   strictness_presets.tenant_id → tenants(id)
--
-- UUID 约定：
--   process_audit_configs: d1000000-0000-0000-0000-00000000000x
--   audit_rules:           d2000000-0000-0000-0000-00000000000x
--   strictness_presets:    d3000000-0000-0000-0000-00000000000x

-- ============================================================
-- DEMO_HQ (a0...01) 流程审核配置
-- ============================================================
INSERT INTO process_audit_configs
    (id, tenant_id, process_type, process_type_label, main_table_name,
     main_fields, detail_tables, field_mode, kb_mode, ai_config, user_permissions, status)
VALUES
(
    'd1000000-0000-0000-0000-000000000001',
    'a0000000-0000-0000-0000-000000000001',
    '采购审批', '采购审批流程', 'formtable_main_1',
    '[
        {"field_key":"sqbm","field_name":"申请部门","field_type":"text"},
        {"field_key":"sqr","field_name":"申请人","field_type":"text"},
        {"field_key":"cgje","field_name":"采购金额","field_type":"float"},
        {"field_key":"gys","field_name":"供应商名称","field_type":"text"},
        {"field_key":"htbh","field_name":"合同编号","field_type":"text"},
        {"field_key":"cgyy","field_name":"采购原因","field_type":"textarea"},
        {"field_key":"fj","field_name":"附件","field_type":"attachment"}
    ]'::jsonb,
    '[{"table_name":"formtable_main_1_dt1","table_label":"采购明细","fields":[
        {"field_key":"wpmc","field_name":"物品名称","field_type":"text"},
        {"field_key":"gg","field_name":"规格型号","field_type":"text"},
        {"field_key":"sl","field_name":"数量","field_type":"int"},
        {"field_key":"dj","field_name":"单价","field_type":"float"},
        {"field_key":"xj","field_name":"小计","field_type":"float"}
    ]}]'::jsonb,
    'all', 'rules_only',
    '{
        "audit_strictness":"standard",
        "system_prompt":"你是一个专业的采购审核助手，请根据审核规则逐条检查以下采购申请。",
        "user_prompt_template":"请审核以下{{process_type}}流程：\n字段数据：{{fields}}\n审核规则：{{rules}}",
        "reasoning_instruction":"请逐条对照规则，给出通过/不通过的判断及理由。",
        "extraction_instruction":"请提取关键信息：采购金额、供应商、合同编号。"
    }'::jsonb,
    '{"allow_custom_fields":true,"allow_custom_rules":true,"allow_modify_strictness":true}'::jsonb,
    'active'
),
(
    'd1000000-0000-0000-0000-000000000002',
    'a0000000-0000-0000-0000-000000000001',
    '合同审批', '合同审批流程', 'formtable_main_2',
    '[
        {"field_key":"htmc","field_name":"合同名称","field_type":"text"},
        {"field_key":"htje","field_name":"合同金额","field_type":"float"},
        {"field_key":"qsrq","field_name":"签署日期","field_type":"date"},
        {"field_key":"dfjg","field_name":"对方机构","field_type":"text"},
        {"field_key":"htlx","field_name":"合同类型","field_type":"select"},
        {"field_key":"htfj","field_name":"合同扫描件","field_type":"image"},
        {"field_key":"flfj","field_name":"法律意见书","field_type":"attachment"}
    ]'::jsonb,
    '[]'::jsonb,
    'selected', 'rules_only',
    '{
        "audit_strictness":"strict",
        "system_prompt":"你是一个专业的合同审核助手，请严格检查合同条款的合规性。",
        "user_prompt_template":"请审核以下{{process_type}}：\n合同信息：{{fields}}\n审核规则：{{rules}}",
        "reasoning_instruction":"请逐条检查合同条款，重点关注金额、期限、违约责任。",
        "extraction_instruction":"请提取：合同金额、签署日期、对方机构、合同类型。"
    }'::jsonb,
    '{"allow_custom_fields":true,"allow_custom_rules":false,"allow_modify_strictness":false}'::jsonb,
    'active'
),
(
    'd1000000-0000-0000-0000-000000000003',
    'a0000000-0000-0000-0000-000000000001',
    '费用报销', '费用报销流程', 'formtable_main_3',
    '[
        {"field_key":"bxr","field_name":"报销人","field_type":"text"},
        {"field_key":"bxje","field_name":"报销金额","field_type":"float"},
        {"field_key":"bxlx","field_name":"报销类型","field_type":"select"},
        {"field_key":"fpsm","field_name":"发票说明","field_type":"textarea"},
        {"field_key":"fpzp","field_name":"发票照片","field_type":"image"}
    ]'::jsonb,
    '[{"table_name":"formtable_main_3_dt1","table_label":"报销明细","fields":[
        {"field_key":"fymx","field_name":"费用项目","field_type":"text"},
        {"field_key":"je","field_name":"金额","field_type":"float"},
        {"field_key":"rq","field_name":"日期","field_type":"date"},
        {"field_key":"sm","field_name":"说明","field_type":"textarea"}
    ]}]'::jsonb,
    'all', 'rules_only',
    '{
        "audit_strictness":"standard",
        "system_prompt":"你是一个费用报销审核助手，请检查报销单据的合规性。",
        "user_prompt_template":"请审核以下{{process_type}}：\n报销信息：{{fields}}\n审核规则：{{rules}}",
        "reasoning_instruction":"请检查金额是否合理、发票是否齐全、报销类型是否匹配。",
        "extraction_instruction":"请提取：报销金额、报销类型、报销人。"
    }'::jsonb,
    '{"allow_custom_fields":true,"allow_custom_rules":true,"allow_modify_strictness":true}'::jsonb,
    'active'
);

-- ============================================================
-- DEMO_BR1 (a0...02) 流程审核配置
-- ============================================================
INSERT INTO process_audit_configs
    (id, tenant_id, process_type, process_type_label, main_table_name,
     main_fields, detail_tables, field_mode, kb_mode, ai_config, user_permissions, status)
VALUES
(
    'd1000000-0000-0000-0000-000000000004',
    'a0000000-0000-0000-0000-000000000002',
    '采购审批', '分公司采购审批', 'formtable_main_1',
    '[
        {"field_key":"sqbm","field_name":"申请部门","field_type":"text"},
        {"field_key":"sqr","field_name":"申请人","field_type":"text"},
        {"field_key":"cgje","field_name":"采购金额","field_type":"float"},
        {"field_key":"gys","field_name":"供应商","field_type":"text"}
    ]'::jsonb,
    '[]'::jsonb,
    'all', 'rules_only',
    '{
        "audit_strictness":"loose",
        "system_prompt":"你是分公司采购审核助手。",
        "user_prompt_template":"请审核以下{{process_type}}：\n{{fields}}\n规则：{{rules}}",
        "reasoning_instruction":"请简要检查各项规则。",
        "extraction_instruction":"请提取采购金额和供应商。"
    }'::jsonb,
    '{"allow_custom_fields":false,"allow_custom_rules":true,"allow_modify_strictness":false}'::jsonb,
    'active'
);

-- ============================================================
-- DEMO_HQ 审核规则（采购审批）
-- ============================================================
INSERT INTO audit_rules
    (id, tenant_id, config_id, process_type, rule_content, rule_scope, enabled, source, related_flow)
VALUES
    -- 强制规则：不可关闭
    ('d2000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001',
     'd1000000-0000-0000-0000-000000000001', '采购审批',
     '单笔采购金额超过 50,000 元必须附有经审批的采购合同', 'mandatory', TRUE, 'manual', FALSE),
    ('d2000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001',
     'd1000000-0000-0000-0000-000000000001', '采购审批',
     '供应商必须在合格供应商名录中', 'mandatory', TRUE, 'manual', FALSE),
    -- 默认开启：用户可关闭
    ('d2000000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000001',
     'd1000000-0000-0000-0000-000000000001', '采购审批',
     '采购明细中每项物品的单价不得超过市场参考价的 120%', 'default_on', TRUE, 'manual', FALSE),
    ('d2000000-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000001',
     'd1000000-0000-0000-0000-000000000001', '采购审批',
     '采购原因说明不得少于 20 字', 'default_on', TRUE, 'manual', FALSE),
    -- 默认关闭：用户可开启
    ('d2000000-0000-0000-0000-000000000005', 'a0000000-0000-0000-0000-000000000001',
     'd1000000-0000-0000-0000-000000000001', '采购审批',
     '同一供应商连续三个月累计采购金额超过 200,000 元需额外审批', 'default_off', TRUE, 'manual', FALSE),
    -- 文件导入规则
    ('d2000000-0000-0000-0000-000000000006', 'a0000000-0000-0000-0000-000000000001',
     'd1000000-0000-0000-0000-000000000001', '采购审批',
     '紧急采购须在 24 小时内补齐完整审批手续', 'default_on', TRUE, 'file_import', TRUE);

-- ============================================================
-- DEMO_HQ 审核规则（合同审批）
-- ============================================================
INSERT INTO audit_rules
    (id, tenant_id, config_id, process_type, rule_content, rule_scope, enabled, source, related_flow)
VALUES
    ('d2000000-0000-0000-0000-000000000007', 'a0000000-0000-0000-0000-000000000001',
     'd1000000-0000-0000-0000-000000000002', '合同审批',
     '合同金额超过 100,000 元必须经法务部审核', 'mandatory', TRUE, 'manual', FALSE),
    ('d2000000-0000-0000-0000-000000000008', 'a0000000-0000-0000-0000-000000000001',
     'd1000000-0000-0000-0000-000000000002', '合同审批',
     '合同必须包含违约责任条款', 'mandatory', TRUE, 'manual', FALSE),
    ('d2000000-0000-0000-0000-000000000009', 'a0000000-0000-0000-0000-000000000001',
     'd1000000-0000-0000-0000-000000000002', '合同审批',
     '合同签署日期不得早于审批通过日期', 'default_on', TRUE, 'manual', FALSE);

-- ============================================================
-- DEMO_HQ 审核规则（费用报销）
-- ============================================================
INSERT INTO audit_rules
    (id, tenant_id, config_id, process_type, rule_content, rule_scope, enabled, source, related_flow)
VALUES
    ('d2000000-0000-0000-0000-000000000010', 'a0000000-0000-0000-0000-000000000001',
     'd1000000-0000-0000-0000-000000000003', '费用报销',
     '单笔报销金额超过 5,000 元必须附发票原件照片', 'mandatory', TRUE, 'manual', FALSE),
    ('d2000000-0000-0000-0000-000000000011', 'a0000000-0000-0000-0000-000000000001',
     'd1000000-0000-0000-0000-000000000003', '费用报销',
     '差旅费报销须附行程单和住宿发票', 'default_on', TRUE, 'manual', FALSE),
    ('d2000000-0000-0000-0000-000000000012', 'a0000000-0000-0000-0000-000000000001',
     'd1000000-0000-0000-0000-000000000003', '费用报销',
     '餐饮招待费单次不得超过 2,000 元', 'default_on', TRUE, 'manual', FALSE);

-- ============================================================
-- DEMO_BR1 审核规则（采购审批）
-- ============================================================
INSERT INTO audit_rules
    (id, tenant_id, config_id, process_type, rule_content, rule_scope, enabled, source, related_flow)
VALUES
    ('d2000000-0000-0000-0000-000000000013', 'a0000000-0000-0000-0000-000000000002',
     'd1000000-0000-0000-0000-000000000004', '采购审批',
     '分公司单笔采购金额超过 20,000 元须总部审批', 'mandatory', TRUE, 'manual', FALSE),
    ('d2000000-0000-0000-0000-000000000014', 'a0000000-0000-0000-0000-000000000002',
     'd1000000-0000-0000-0000-000000000004', '采购审批',
     '采购申请须注明预算来源', 'default_on', TRUE, 'manual', FALSE);

-- ============================================================
-- DEMO_HQ 审核尺度预设（每个租户 3 条：strict / standard / loose）
-- ============================================================
INSERT INTO strictness_presets
    (id, tenant_id, strictness, reasoning_instruction, extraction_instruction)
VALUES
    ('d3000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001',
     'strict',
     '请以最严格的标准逐条审核，任何不符合规则的项目都必须标记为不通过，并给出详细的不合规理由。对于模糊或边界情况，倾向于判定为不通过。',
     '请完整提取所有关键字段信息，包括金额、日期、人员、部门、合同编号等，不得遗漏。'),
    ('d3000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001',
     'standard',
     '请按照标准流程逐条审核，对于明确违规的项目标记为不通过，对于轻微偏差可给出建议但不强制不通过。',
     '请提取主要关键字段信息，包括核心金额、关键日期和主要人员信息。'),
    ('d3000000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000001',
     'loose',
     '请以宽松标准审核，仅对严重违规项目标记为不通过，对于轻微问题给出提醒即可。',
     '请提取核心金额和关键人员信息即可。');

-- ============================================================
-- DEMO_BR1 审核尺度预设
-- ============================================================
INSERT INTO strictness_presets
    (id, tenant_id, strictness, reasoning_instruction, extraction_instruction)
VALUES
    ('d3000000-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000002',
     'strict',
     '分公司严格审核模式：逐条检查，不合规即不通过。',
     '完整提取所有字段。'),
    ('d3000000-0000-0000-0000-000000000005', 'a0000000-0000-0000-0000-000000000002',
     'standard',
     '分公司标准审核模式：按规则检查，轻微偏差可建议通过。',
     '提取主要字段信息。'),
    ('d3000000-0000-0000-0000-000000000006', 'a0000000-0000-0000-0000-000000000002',
     'loose',
     '分公司宽松审核模式：仅检查重大违规。',
     '提取核心金额信息。');
