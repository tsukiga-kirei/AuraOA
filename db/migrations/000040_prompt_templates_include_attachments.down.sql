UPDATE system_prompt_templates
SET content = replace(
        content,
        E'明细表数据：\n{{detail_tables}}\n\n附件识别内容：\n{{attachments}}\n\n审核规则：',
        E'明细表数据：\n{{detail_tables}}\n\n审核规则：'
    ),
    updated_at = now()
WHERE prompt_key LIKE 'audit_user_reasoning_%';

UPDATE system_prompt_templates
SET content = replace(
        content,
        E'明细表数据：\n{{detail_tables}}\n\n附件识别内容：\n{{attachments}}\n\n审核规则：',
        E'明细表数据：\n{{detail_tables}}\n\n审核规则：'
    ),
    updated_at = now()
WHERE prompt_key LIKE 'archive_user_reasoning_%';
