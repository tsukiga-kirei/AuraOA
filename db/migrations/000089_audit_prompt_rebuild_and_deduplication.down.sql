-- 000089_audit_prompt_rebuild_and_deduplication.down.sql
-- 回滚审核与归档提示词重构

-- 审核用户推理提示词回滚
UPDATE system_prompt_templates
SET content = E'请以【标准】尺度审核以下 OA 流程数据。\n\n审核尺度要求：\n- 明确违规项判定为不通过\n- 存疑项需说明理由并给出建议\n- 轻微偏差可标注但不强制不通过\n- 关注核心合规要素，兼顾业务合理性\n\n主表数据：\n{{main_table}}\n\n明细表数据：\n{{detail_tables}}\n\n附件识别内容：\n{{attachments}}\n\n审核规则：\n{{rules}}\n\n审批流信息：\n{{flow_history}}\n\n流程图节点：\n{{flow_graph}}\n\n当前审批节点：{{current_node}}\n\n请逐条对照规则进行审核，对每条规则给出通过/不通过的判断及理由。',
    updated_at = now()
WHERE prompt_key = 'audit_user_reasoning_standard';

UPDATE system_prompt_templates
SET content = E'基于以下推理分析结果和审核规则，请以【标准】尺度提取结构化审核结论。\n\n评分标准：明确违规项按正常权重扣分，60 分以下建议退回，60-80 分建议复核。\n\n推理分析结果：\n{{reasoning_result}}\n\n审核规则：\n{{rules}}\n\n请严格按 JSON Schema 输出结构化结论。',
    updated_at = now()
WHERE prompt_key = 'audit_user_extraction_standard';
