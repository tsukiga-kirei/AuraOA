-- 让默认审核推理模板显式注入附件识别内容。
-- 已有流程若保存了旧 ai_config，运行时仍会由 prompt builder 兜底追加附件段落。

UPDATE system_prompt_templates
SET content = replace(
        content,
        E'明细表数据：\n{{detail_tables}}\n\n审核规则：',
        E'明细表数据：\n{{detail_tables}}\n\n附件识别内容：\n{{attachments}}\n\n审核规则：'
    ),
    updated_at = now()
WHERE prompt_key LIKE 'audit_user_reasoning_%'
  AND content NOT LIKE '%{{attachments}}%';

UPDATE system_prompt_templates
SET content = replace(
        content,
        E'明细表数据：\n{{detail_tables}}\n\n审核规则：',
        E'明细表数据：\n{{detail_tables}}\n\n附件识别内容：\n{{attachments}}\n\n审核规则：'
    ),
    updated_at = now()
WHERE prompt_key LIKE 'archive_user_reasoning_%'
  AND content NOT LIKE '%{{attachments}}%';
