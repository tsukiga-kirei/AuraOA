-- 010_process_audit_configs.sql
-- Seed data: 流程审核配置 + 审核规则 + 系统提示词模板
-- Run after 003_tenants.sql (depends on tenants)
-- Run after 001_oa_ai_seeds.sql (ai_config references ai_model_configs)
--
-- 外键依赖：
--   process_audit_configs.tenant_id → tenants(id)
--   audit_rules.tenant_id → tenants(id)
--   audit_rules.config_id → process_audit_configs(id)
--
-- UUID 约定：
--   process_audit_configs: d1000000-0000-0000-0000-00000000000x
--   audit_rules:           d2000000-0000-0000-0000-00000000000x
--   system_prompt_templates: d4000000-0000-0000-0000-00000000000x

-- ============================================================
-- 系统提示词模板（全局，12 条记录）
-- 6 条系统提示词（3 尺度 × 2 阶段）+ 6 条用户提示词（3 尺度 × 2 阶段）
-- ============================================================
INSERT INTO system_prompt_templates
    (id, prompt_key, prompt_type, phase, strictness, content, description)
VALUES
-- 系统提示词（严格，推理阶段）
('d4000000-0000-0000-0000-000000000001', 'system_reasoning_strict', 'system', 'reasoning', 'strict',
'你是 OA 智能审核系统的深度推理引擎，工作于【严格】审核模式。你的任务是对 OA 流程表单数据进行全面、严格的合规性分析。

工作流程：
1. 仔细阅读并理解所有提供的表单数据（主表和明细表）
2. 以零容忍标准逐条对照审核规则检查合规性
3. 如果提供了审批流信息，结合审批流上下文分析流程合理性
4. 识别数据中的所有风险点、异常模式和逻辑矛盾，包括轻微偏差
5. 对每条规则给出独立的专业判断

分析要求：
- 对任何不符合规则的项目，无论轻重，必须明确标记为不通过
- 模糊或边界情况一律倾向于判定为不通过，要求提供充分说明
- 对缺失或不完整的信息一律视为违规
- 提供详细的违规证据和判断依据
- 合规性优先于业务便利性，不接受以业务需要为由的豁免

请以自由文本格式输出完整的分析过程和推理结论。',
'系统推理提示词（严格）：零容忍，重视细节，合规优先'),

-- 系统提示词（标准，推理阶段）
('d4000000-0000-0000-0000-000000000002', 'system_reasoning_standard', 'system', 'reasoning', 'standard',
'你是 OA 智能审核系统的深度推理引擎，工作于【标准】审核模式。你的任务是对 OA 流程表单数据进行全面的合规性分析。

工作流程：
1. 仔细阅读并理解所有提供的表单数据（主表和明细表）
2. 逐条对照审核规则检查表单数据的合规性
3. 如果提供了审批流信息，结合审批流上下文分析流程合理性
4. 识别数据中的潜在风险点、异常模式和逻辑矛盾
5. 对每条规则给出独立的专业判断

分析要求：
- 保持客观中立，以事实和数据为依据
- 对明确违规的项目标记为不通过，并给出不合规理由
- 存疑项需说明理由并给出改进建议
- 轻微偏差可标注，但需结合业务合理性综合判断
- 关注数据之间的关联性和一致性

请以自由文本格式输出完整的分析过程和推理结论。',
'系统推理提示词（标准）：平衡合规与业务合理性'),

-- 系统提示词（宽松，推理阶段）
('d4000000-0000-0000-0000-000000000003', 'system_reasoning_loose', 'system', 'reasoning', 'loose',
'你是 OA 智能审核系统的深度推理引擎，工作于【宽松】审核模式。你的任务是对 OA 流程表单数据进行合规性分析，聚焦重大风险。

工作流程：
1. 仔细阅读并理解所有提供的表单数据（主表和明细表）
2. 以宽容视角逐条对照审核规则，聚焦实质性违规
3. 如果提供了审批流信息，结合审批流上下文判断是否存在重大流程异常
4. 识别显著风险点，对技术性细节偏差保持包容
5. 以推动业务正常流转为导向给出判断

分析要求：
- 重点关注实质性、重大违规项
- 轻微偏差或技术性问题仅记录，不建议退回
- 模糊或边界情况倾向于宽容判定，优先通过
- 以业务合理性为核心评判依据
- 仅在存在明显重大违规时建议退回

请以自由文本格式输出完整的分析过程和推理结论。',
'系统推理提示词（宽松）：聚焦重大违规，以推动业务流转为导向'),

-- 系统提示词（严格，提取阶段）
('d4000000-0000-0000-0000-000000000004', 'system_extraction_strict', 'system', 'extraction', 'strict',
'你是 OA 智能审核系统的结构化提取引擎，工作于【严格】审核模式。你的任务是将推理分析结果转化为标准化的 JSON 格式输出。

评分规则（严格模式）：
- 任何违规项扣分权重加倍
- overall_score 80 分以上建议通过（approve）
- overall_score 60-80 分建议人工复核（review）
- overall_score 60 分以下建议退回（return）

请严格按照以下 JSON Schema 输出：
{
  "recommendation": "approve | return | review",
  "overall_score": 0-100,
  "rule_results": [
    {
      "rule_content": "规则内容",
      "passed": true/false,
      "reason": "判断理由"
    }
  ],
  "risk_points": ["风险点描述"],
  "suggestions": ["改进建议"],
  "confidence": 0-100
}

字段说明：
- recommendation: 综合建议（approve=通过, return=退回, review=人工复核）
- overall_score: 综合评分（0-100），越高表示合规程度越好
- rule_results: 逐条规则校验结果，必须覆盖所有规则
- risk_points: 发现的风险点，需具体可定位
- suggestions: 改进建议，需具体可操作
- confidence: 结论置信度（0-100）

仅输出 JSON，不要包含其他文字。',
'系统提取提示词（严格）：严格评分阈值，违规扣分加倍'),

-- 系统提示词（标准，提取阶段）
('d4000000-0000-0000-0000-000000000005', 'system_extraction_standard', 'system', 'extraction', 'standard',
'你是 OA 智能审核系统的结构化提取引擎，工作于【标准】审核模式。你的任务是将推理分析结果转化为标准化的 JSON 格式输出。

评分规则（标准模式）：
- 明确违规项按正常权重扣分
- overall_score 70 分以上建议通过（approve）
- overall_score 50-70 分建议人工复核（review）
- overall_score 50 分以下建议退回（return）

请严格按照以下 JSON Schema 输出：
{
  "recommendation": "approve | return | review",
  "overall_score": 0-100,
  "rule_results": [
    {
      "rule_content": "规则内容",
      "passed": true/false,
      "reason": "判断理由"
    }
  ],
  "risk_points": ["风险点描述"],
  "suggestions": ["改进建议"],
  "confidence": 0-100
}

字段说明：
- recommendation: 综合建议（approve=通过, return=退回, review=人工复核）
- overall_score: 综合评分（0-100），越高表示合规程度越好
- rule_results: 逐条规则校验结果，必须覆盖所有规则
- risk_points: 发现的风险点，需具体可定位
- suggestions: 改进建议，需具体可操作
- confidence: 结论置信度（0-100）

仅输出 JSON，不要包含其他文字。',
'系统提取提示词（标准）：标准评分阈值'),

-- 系统提示词（宽松，提取阶段）
('d4000000-0000-0000-0000-000000000006', 'system_extraction_loose', 'system', 'extraction', 'loose',
'你是 OA 智能审核系统的结构化提取引擎，工作于【宽松】审核模式。你的任务是将推理分析结果转化为标准化的 JSON 格式输出。

评分规则（宽松模式）：
- 仅重大违规项扣分，轻微问题不纳入扣分
- overall_score 50 分以上建议通过（approve）
- overall_score 30-50 分建议人工复核（review）
- overall_score 30 分以下建议退回（return）

请严格按照以下 JSON Schema 输出：
{
  "recommendation": "approve | return | review",
  "overall_score": 0-100,
  "rule_results": [
    {
      "rule_content": "规则内容",
      "passed": true/false,
      "reason": "判断理由"
    }
  ],
  "risk_points": ["风险点描述"],
  "suggestions": ["改进建议"],
  "confidence": 0-100
}

字段说明：
- recommendation: 综合建议（approve=通过, return=退回, review=人工复核）
- overall_score: 综合评分（0-100），越高表示合规程度越好
- rule_results: 逐条规则校验结果，必须覆盖所有规则
- risk_points: 发现的风险点，需具体可定位
- suggestions: 改进建议，需具体可操作
- confidence: 结论置信度（0-100）

仅输出 JSON，不要包含其他文字。',
'系统提取提示词（宽松）：宽松评分阈值，轻微问题不扣分'),

-- 用户提示词 — 严格（推理阶段）
('d4000000-0000-0000-0000-000000000007', 'user_reasoning_strict', 'user', 'reasoning', 'strict',
'请以【严格】标准审核以下 OA 流程数据。

审核尺度要求：
- 任何不符合规则的项目必须标记为不通过
- 模糊或边界情况倾向于判定为不通过
- 不接受缺失信息或不完整说明
- 所有金额、日期、人员信息必须完整准确

主表数据：
{{main_table}}

明细表数据：
{{detail_tables}}

审核规则：
{{rules}}

审批流信息：
{{flow_history}}

流程图节点：
{{flow_graph}}

当前审批节点：{{current_node}}

请逐条对照规则进行严格审核，给出详细的通过/不通过判断及理由。',
'用户推理提示词（严格）：宁可误判也不放过'),

-- 用户提示词 — 严格（提取阶段）
('d4000000-0000-0000-0000-000000000008', 'user_extraction_strict', 'user', 'extraction', 'strict',
'基于以下推理分析结果和审核规则，请以【严格】标准提取结构化审核结论。

评分标准：任何违规项扣分权重加倍，80 分以下建议退回。

推理分析结果：
{{reasoning_result}}

审核规则：
{{rules}}

请严格按 JSON Schema 输出结构化结论。',
'用户提取提示词（严格）：严格评分标准'),

-- 用户提示词 — 标准（推理阶段）
('d4000000-0000-0000-0000-000000000009', 'user_reasoning_standard', 'user', 'reasoning', 'standard',
'请以【标准】尺度审核以下 OA 流程数据。

审核尺度要求：
- 明确违规项判定为不通过
- 存疑项需说明理由并给出建议
- 轻微偏差可标注但不强制不通过
- 关注核心合规要素，兼顾业务合理性

主表数据：
{{main_table}}

明细表数据：
{{detail_tables}}

审核规则：
{{rules}}

审批流信息：
{{flow_history}}

流程图节点：
{{flow_graph}}

当前审批节点：{{current_node}}

请逐条对照规则进行审核，对每条规则给出通过/不通过的判断及理由。',
'用户推理提示词（标准）：明确违规退回，存疑项给出建议'),

-- 用户提示词 — 标准（提取阶段）
('d4000000-0000-0000-0000-00000000000a', 'user_extraction_standard', 'user', 'extraction', 'standard',
'基于以下推理分析结果和审核规则，请以【标准】尺度提取结构化审核结论。

评分标准：明确违规项按正常权重扣分，60 分以下建议退回，60-80 分建议复核。

推理分析结果：
{{reasoning_result}}

审核规则：
{{rules}}

请严格按 JSON Schema 输出结构化结论。',
'用户提取提示词（标准）：标准评分标准'),

-- 用户提示词 — 宽松（推理阶段）
('d4000000-0000-0000-0000-00000000000b', 'user_reasoning_loose', 'user', 'reasoning', 'loose',
'请以【宽松】标准审核以下 OA 流程数据。

审核尺度要求：
- 仅对明显违规项建议退回
- 轻微问题仅提示，不影响通过建议
- 关注重大合规风险，忽略细节偏差
- 以推动业务正常流转为导向

主表数据：
{{main_table}}

明细表数据：
{{detail_tables}}

审核规则：
{{rules}}

审批流信息：
{{flow_history}}

流程图节点：
{{flow_graph}}

当前审批节点：{{current_node}}

请重点检查是否存在重大违规项，给出审核判断及简要理由。',
'用户推理提示词（宽松）：仅关注重大违规'),

-- 用户提示词 — 宽松（提取阶段）
('d4000000-0000-0000-0000-00000000000c', 'user_extraction_loose', 'user', 'extraction', 'loose',
'基于以下推理分析结果和审核规则，请以【宽松】标准提取结构化审核结论。

评分标准：仅重大违规项扣分，40 分以下才建议退回，轻微问题仅作提示。

推理分析结果：
{{reasoning_result}}

审核规则：
{{rules}}

请严格按 JSON Schema 输出结构化结论。',
'用户提取提示词（宽松）：宽松评分标准');

-- ============================================================
-- DEMO_HQ (a0...01) 流程审核配置
-- ai_config 新结构：system_reasoning_prompt / system_extraction_prompt / user_reasoning_prompt / user_extraction_prompt
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
        'audit_strictness', 'standard',
        'system_reasoning_prompt', (SELECT content FROM system_prompt_templates WHERE prompt_key = 'system_reasoning_standard'),
        'system_extraction_prompt', (SELECT content FROM system_prompt_templates WHERE prompt_key = 'system_extraction_standard'),
        'user_reasoning_prompt', (SELECT content FROM system_prompt_templates WHERE prompt_key = 'user_reasoning_standard'),
        'user_extraction_prompt', (SELECT content FROM system_prompt_templates WHERE prompt_key = 'user_extraction_standard')
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
        'audit_strictness', 'strict',
        'system_reasoning_prompt', (SELECT content FROM system_prompt_templates WHERE prompt_key = 'system_reasoning_strict'),
        'system_extraction_prompt', (SELECT content FROM system_prompt_templates WHERE prompt_key = 'system_extraction_strict'),
        'user_reasoning_prompt', (SELECT content FROM system_prompt_templates WHERE prompt_key = 'user_reasoning_strict'),
        'user_extraction_prompt', (SELECT content FROM system_prompt_templates WHERE prompt_key = 'user_extraction_strict')
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
        'audit_strictness', 'standard',
        'system_reasoning_prompt', (SELECT content FROM system_prompt_templates WHERE prompt_key = 'system_reasoning_standard'),
        'system_extraction_prompt', (SELECT content FROM system_prompt_templates WHERE prompt_key = 'system_extraction_standard'),
        'user_reasoning_prompt', (SELECT content FROM system_prompt_templates WHERE prompt_key = 'user_reasoning_standard'),
        'user_extraction_prompt', (SELECT content FROM system_prompt_templates WHERE prompt_key = 'user_extraction_standard')
    )),
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
    (SELECT jsonb_build_object(
        'audit_strictness', 'loose',
        'system_reasoning_prompt', (SELECT content FROM system_prompt_templates WHERE prompt_key = 'system_reasoning_loose'),
        'system_extraction_prompt', (SELECT content FROM system_prompt_templates WHERE prompt_key = 'system_extraction_loose'),
        'user_reasoning_prompt', (SELECT content FROM system_prompt_templates WHERE prompt_key = 'user_reasoning_loose'),
        'user_extraction_prompt', (SELECT content FROM system_prompt_templates WHERE prompt_key = 'user_extraction_loose')
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
