-- 010_process_audit_configs.sql
-- 演示数据：流程审核配置 + 审核规则
-- 依赖：003_tenants.sql（租户数据）
-- 注意：system_prompt_templates 数据已迁移至 000007_audit_configs_rules_presets.up.sql
--
-- 外键依赖：
--   process_audit_configs.tenant_id → tenants(id)
--   audit_rules.tenant_id           → tenants(id)
--   audit_rules.config_id           → process_audit_configs(id)
--
-- UUID 约定：
--   process_audit_configs : d1000000-0000-0000-0000-00000000000x
--   audit_rules           : d2000000-0000-0000-0000-00000000000x

-- ============================================================
-- DEMO_HQ (a0...01) 流程审核配置（3个流程）
-- ai_config 结构：audit_strictness / system_reasoning_prompt /
--                 system_extraction_prompt / user_reasoning_prompt / user_extraction_prompt
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
    (SELECT jsonb_build_object(
        'audit_strictness',        'standard',
        'system_reasoning_prompt',  (SELECT content FROM system_prompt_templates WHERE prompt_key = 'audit_system_reasoning_standard'),
        'system_extraction_prompt', (SELECT content FROM system_prompt_templates WHERE prompt_key = 'audit_system_extraction_standard'),
        'user_reasoning_prompt',    (SELECT content FROM system_prompt_templates WHERE prompt_key = 'audit_user_reasoning_standard'),
        'user_extraction_prompt',   (SELECT content FROM system_prompt_templates WHERE prompt_key = 'audit_user_extraction_standard')
    )),
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
    (SELECT jsonb_build_object(
        'audit_strictness',        'strict',
        'system_reasoning_prompt',  (SELECT content FROM system_prompt_templates WHERE prompt_key = 'audit_system_reasoning_strict'),
        'system_extraction_prompt', (SELECT content FROM system_prompt_templates WHERE prompt_key = 'audit_system_extraction_strict'),
        'user_reasoning_prompt',    (SELECT content FROM system_prompt_templates WHERE prompt_key = 'audit_user_reasoning_strict'),
        'user_extraction_prompt',   (SELECT content FROM system_prompt_templates WHERE prompt_key = 'audit_user_extraction_strict')
    )),
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
    (SELECT jsonb_build_object(
        'audit_strictness',        'standard',
        'system_reasoning_prompt',  (SELECT content FROM system_prompt_templates WHERE prompt_key = 'audit_system_reasoning_standard'),
        'system_extraction_prompt', (SELECT content FROM system_prompt_templates WHERE prompt_key = 'audit_system_extraction_standard'),
        'user_reasoning_prompt',    (SELECT content FROM system_prompt_templates WHERE prompt_key = 'audit_user_reasoning_standard'),
        'user_extraction_prompt',   (SELECT content FROM system_prompt_templates WHERE prompt_key = 'audit_user_extraction_standard')
    )),
    '{"allow_custom_fields":true,"allow_custom_rules":true,"allow_modify_strictness":true}'::jsonb,
    'active'
);

-- ============================================================
-- DEMO_BR1 (a0...02) 流程审核配置（1个流程）
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
    (SELECT jsonb_build_object(
        'audit_strictness',        'loose',
        'system_reasoning_prompt',  (SELECT content FROM system_prompt_templates WHERE prompt_key = 'audit_system_reasoning_loose'),
        'system_extraction_prompt', (SELECT content FROM system_prompt_templates WHERE prompt_key = 'audit_system_extraction_loose'),
        'user_reasoning_prompt',    (SELECT content FROM system_prompt_templates WHERE prompt_key = 'audit_user_reasoning_loose'),
        'user_extraction_prompt',   (SELECT content FROM system_prompt_templates WHERE prompt_key = 'audit_user_extraction_loose')
    )),
    '{"allow_custom_fields":false,"allow_custom_rules":true,"allow_modify_strictness":false}'::jsonb,
    'active'
);

-- ============================================================
-- DEMO_HQ 审核规则（采购审批）
-- ============================================================
INSERT INTO audit_rules
    (id, tenant_id, config_id, process_type, rule_content, rule_scope, enabled, source, related_flow)
VALUES
    ('d2000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001',
     'd1000000-0000-0000-0000-000000000001', '采购审批',
     '单笔采购金额超过 50,000 元必须附有经审批的采购合同', 'mandatory', TRUE, 'manual', FALSE),
    ('d2000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001',
     'd1000000-0000-0000-0000-000000000001', '采购审批',
     '供应商必须在合格供应商名录中', 'mandatory', TRUE, 'manual', FALSE),
    ('d2000000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000001',
     'd1000000-0000-0000-0000-000000000001', '采购审批',
     '采购明细中每项物品的单价不得超过市场参考价的 120%', 'default_on', TRUE, 'manual', FALSE),
    ('d2000000-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000001',
     'd1000000-0000-0000-0000-000000000001', '采购审批',
     '采购原因说明不得少于 20 字', 'default_on', TRUE, 'manual', FALSE),
    ('d2000000-0000-0000-0000-000000000005', 'a0000000-0000-0000-0000-000000000001',
     'd1000000-0000-0000-0000-000000000001', '采购审批',
     '同一供应商连续三个月累计采购金额超过 200,000 元需额外审批', 'default_off', TRUE, 'manual', FALSE),
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
