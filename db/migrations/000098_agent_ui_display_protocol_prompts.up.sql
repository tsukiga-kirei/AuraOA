-- 000098_agent_ui_display_protocol_prompts.up.sql
-- 背景：oa_creator / oa_analyst 的系统提示词只写了“输出 :::form-confirm / :::chart 块”，
-- 但从未给出键名与结构，模型只能凭“常见写法”猜测，导致卡片缺标题、明细丢失、字段串值。
-- 本迁移把两段卡片协议改为由系统变量注入（正文见 prompt_ui_protocols.go）：
--   {{form_confirm_protocol}} —— :::form-confirm 表单确认卡片协议
--   {{chart_protocol}}        —— :::chart 图表卡片协议
-- 任意智能体在提示词中写这两个变量即可复用，不必再依赖“调用 skill 才能拿到格式”。
--
-- 注意：本迁移只改「智能体系统提示词」这一处。
-- Skill 正文（E9 表字典、SQL 查数规范等）属于各自 skill 的专属内容，不进系统变量，
-- 也不在这里批量覆盖，避免冲掉租户已自定义的 skill 副本。

-- 1. 系统种子智能体接入两个协议变量（仅平台种子行，租户自建副本保持原样）
UPDATE agent_definitions
SET system_prompt = system_prompt || $protocol$

【Generative UI 卡片输出规范】
需要向用户输出表单确认卡片或图表卡片时，必须严格按下列两段协议执行，键名、层级、取值以协议为准，禁止自行发明：

{{form_confirm_protocol}}

{{chart_protocol}}
$protocol$,
    updated_at = NOW()
WHERE tenant_id IS NULL
  AND agent_code IN ('oa_creator', 'oa_analyst')
  AND system_prompt NOT LIKE '%{{form_confirm_protocol}}%';
