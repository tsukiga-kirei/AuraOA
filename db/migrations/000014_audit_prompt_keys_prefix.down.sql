-- 回滚：去掉审核台提示词 audit_ 前缀（仅影响 12 条标准键名）

UPDATE system_prompt_templates
SET prompt_key = regexp_replace(prompt_key, '^audit_', ''),
    updated_at = now()
WHERE prompt_key ~ '^audit_(system|user)_(reasoning|extraction)_(strict|standard|loose)$';
