-- 审核工作台系统提示词：与 archive_ 对称，统一为 audit_ 前缀，避免与归档模板 key 混淆或覆盖。
-- 已按 000007 基线插入 audit_* 的环境本迁移为 no-op（WHERE 无匹配行）。

UPDATE system_prompt_templates
SET prompt_key = 'audit_' || prompt_key,
    updated_at = now()
WHERE prompt_key IN (
    'system_reasoning_strict',
    'system_reasoning_standard',
    'system_reasoning_loose',
    'system_extraction_strict',
    'system_extraction_standard',
    'system_extraction_loose',
    'user_reasoning_strict',
    'user_reasoning_standard',
    'user_reasoning_loose',
    'user_extraction_strict',
    'user_extraction_standard',
    'user_extraction_loose'
);
