-- 012_process_archive_configs.sql
-- 演示数据：归档复盘配置 + 归档规则
-- 依赖：003_tenants.sql（租户数据）
-- 注意：archive_ 前缀的 system_prompt_templates 已经在 000007 迁移中初始化
--
-- 外键依赖：
--   process_archive_configs.tenant_id → tenants(id)
--   archive_rules.tenant_id           → tenants(id)
--   archive_rules.config_id           → process_archive_configs(id)
--
-- UUID 约定：
--   process_archive_configs : d7000000-0000-0000-0000-00000000000x
--   archive_rules           : d8000000-0000-0000-0000-00000000000x

-- ============================================================
-- DEMO_HQ (a0...01) 归档复盘配置（3个流程）
-- access_control 结构：allowed_roles / allowed_members / allowed_departments
-- ============================================================
INSERT INTO process_archive_configs
    (id, tenant_id, process_type, process_type_label, main_table_name,
     main_fields, detail_tables, field_mode, kb_mode, ai_config, user_permissions, access_control, status)
VALUES
(
    'd7000000-0000-0000-0000-000000000001',
    'a0000000-0000-0000-0000-000000000001',
    '采购审批', '采购归档复盘', 'formtable_main_1',
    '[
        {"field_key":"sqbm","field_name":"申请部门","field_type":"text"},
        {"field_key":"sqr","field_name":"申请人","field_type":"text"},
        {"field_key":"cgje","field_name":"采购金额","field_type":"float"},
        {"field_key":"gys","field_name":"供应商名称","field_type":"text"},
        {"field_key":"htbh","field_name":"合同编号","field_type":"text"}
    ]'::jsonb,
    '[{"table_name":"formtable_main_1_dt1","table_label":"采购明细","fields":[
        {"field_key":"wpmc","field_name":"物品名称","field_type":"text"},
        {"field_key":"sl","field_name":"数量","field_type":"int"},
        {"field_key":"xj","field_name":"小计","field_type":"float"}
    ]}]'::jsonb,
    'all', 'rules_only',
    (SELECT jsonb_build_object(
        'audit_strictness',        'standard',
        'system_reasoning_prompt',  (SELECT content FROM system_prompt_templates WHERE prompt_key = 'archive_system_reasoning_standard'),
        'system_extraction_prompt', (SELECT content FROM system_prompt_templates WHERE prompt_key = 'archive_system_extraction_standard'),
        'user_reasoning_prompt',    (SELECT content FROM system_prompt_templates WHERE prompt_key = 'archive_user_reasoning_standard'),
        'user_extraction_prompt',   (SELECT content FROM system_prompt_templates WHERE prompt_key = 'archive_user_extraction_standard')
    )),
    '{"allow_custom_fields":true,"allow_custom_rules":true,"allow_modify_strictness":true}'::jsonb,
    '{"allowed_roles":[],"allowed_members":[],"allowed_departments":[]}'::jsonb,
    'active'
),
(
    'd7000000-0000-0000-0000-000000000002',
    'a0000000-0000-0000-0000-000000000001',
    '合同审批', '合同归档合规检查', 'formtable_main_2',
    '[
        {"field_key":"htmc","field_name":"合同名称","field_type":"text"},
        {"field_key":"htje","field_name":"合同金额","field_type":"float"},
        {"field_key":"dfjg","field_name":"对方机构","field_type":"text"},
        {"field_key":"htlx","field_name":"合同类型","field_type":"select"}
    ]'::jsonb,
    '[]'::jsonb,
    'selected', 'rules_only',
    (SELECT jsonb_build_object(
        'audit_strictness',        'strict',
        'system_reasoning_prompt',  (SELECT content FROM system_prompt_templates WHERE prompt_key = 'archive_system_reasoning_strict'),
        'system_extraction_prompt', (SELECT content FROM system_prompt_templates WHERE prompt_key = 'archive_system_extraction_strict'),
        'user_reasoning_prompt',    (SELECT content FROM system_prompt_templates WHERE prompt_key = 'archive_user_reasoning_strict'),
        'user_extraction_prompt',   (SELECT content FROM system_prompt_templates WHERE prompt_key = 'archive_user_extraction_strict')
    )),
    '{"allow_custom_fields":false,"allow_custom_rules":true,"allow_modify_strictness":false}'::jsonb,
    '{"allowed_roles":[],"allowed_members":[],"allowed_departments":[]}'::jsonb,
    'active'
),
(
    'd7000000-0000-0000-0000-000000000003',
    'a0000000-0000-0000-0000-000000000001',
    '费用报销', '费用报销事后复核', 'formtable_main_3',
    '[
        {"field_key":"bxr","field_name":"报销人","field_type":"text"},
        {"field_key":"bxje","field_name":"报销金额","field_type":"float"},
        {"field_key":"bxlx","field_name":"报销类型","field_type":"select"}
    ]'::jsonb,
    '[]'::jsonb,
    'all', 'rules_only',
    (SELECT jsonb_build_object(
        'audit_strictness',        'standard',
        'system_reasoning_prompt',  (SELECT content FROM system_prompt_templates WHERE prompt_key = 'archive_system_reasoning_standard'),
        'system_extraction_prompt', (SELECT content FROM system_prompt_templates WHERE prompt_key = 'archive_system_extraction_standard'),
        'user_reasoning_prompt',    (SELECT content FROM system_prompt_templates WHERE prompt_key = 'archive_user_reasoning_standard'),
        'user_extraction_prompt',   (SELECT content FROM system_prompt_templates WHERE prompt_key = 'archive_user_extraction_standard')
    )),
    '{"allow_custom_fields":true,"allow_custom_rules":true,"allow_modify_strictness":true}'::jsonb,
    '{"allowed_roles":[],"allowed_members":[],"allowed_departments":[]}'::jsonb,
    'active'
);

-- ============================================================
-- DEMO_HQ 归档规则（采购审批）
-- ============================================================
INSERT INTO archive_rules
    (id, tenant_id, config_id, process_type, rule_content, rule_scope, enabled, source, related_flow)
VALUES
    ('d8000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001',
     'd7000000-0000-0000-0000-000000000001', '采购审批',
     '归档资料中必须包含供应商入库审核结果', 'mandatory', TRUE, 'manual', FALSE),
    ('d8000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001',
     'd7000000-0000-0000-0000-000000000001', '采购审批',
     '审批流中必须经过采购部主管和财务部主管节点', 'mandatory', TRUE, 'manual', TRUE),
    ('d8000000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000001',
     'd7000000-0000-0000-0000-000000000001', '采购审批',
     '归档合同编号必须与采购申请表单中填写的一致', 'default_on', TRUE, 'manual', FALSE);

-- ============================================================
-- DEMO_HQ 归档规则（合同审批）
-- ============================================================
INSERT INTO archive_rules
    (id, tenant_id, config_id, process_type, rule_content, rule_scope, enabled, source, related_flow)
VALUES
    ('d8000000-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000001',
     'd7000000-0000-0000-0000-000000000002', '合同审批',
     '归档合同附件必须为加盖公章的扫描件', 'mandatory', TRUE, 'manual', FALSE),
    ('d8000000-0000-0000-0000-000000000005', 'a0000000-0000-0000-0000-000000000001',
     'd7000000-0000-0000-0000-000000000002', '合同审批',
     '所有金额超过 100 万的合同必须有法务部审批通过的记录', 'mandatory', TRUE, 'manual', TRUE);

-- ============================================================
-- DEMO_HQ 归档规则（费用报销）
-- ============================================================
INSERT INTO archive_rules
    (id, tenant_id, config_id, process_type, rule_content, rule_scope, enabled, source, related_flow)
VALUES
    ('d8000000-0000-0000-0000-000000000006', 'a0000000-0000-0000-0000-000000000001',
     'd7000000-0000-0000-0000-000000000003', '费用报销',
     '报销归档中必须包含完整的电子发票报销入账记录', 'mandatory', TRUE, 'manual', FALSE);
