-- 000098_agent_ui_display_protocol_prompts.down.sql
-- 回滚：移除系统提示词中追加的 Generative UI 协议引用。
-- Skill 正文未在本迁移中改动，无需回滚。

UPDATE agent_definitions
SET system_prompt = REPLACE(system_prompt, $protocol$

【Generative UI 卡片输出规范】
需要向用户输出表单确认卡片或图表卡片时，必须严格按下列两段协议执行，键名、层级、取值以协议为准，禁止自行发明：

{{form_confirm_protocol}}

{{chart_protocol}}
$protocol$, ''),
    updated_at = NOW()
WHERE tenant_id IS NULL
  AND agent_code IN ('oa_creator', 'oa_analyst');
