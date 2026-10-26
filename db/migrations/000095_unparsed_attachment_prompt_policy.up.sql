-- 标准、宽松尺度中，已获取但无法解析的附件暂按无问题处理；仅更新默认提示词。
-- 已保存配置、已发布版本和历史结果仍沿用原内容，通过采用新版模板并发布生效。
WITH policy AS (
    SELECT $policy$【已获取附件解析失败的处理】
已成功获取但因格式、兼容性或解析服务异常而无法解析正文的附件，暂按无问题处理。仅依赖该附件正文、且没有其他证据表明存在问题的检查项按通过处理；不得仅因此扣分、列为风险、要求补充材料、给出复核/退回或部分合规/不合规结论。可以说明该附件未解析、暂按无问题处理，但不得编造已读取的内容或核验事实。本条仅覆盖已获取附件的解析失败，不适用于附件未提供、获取失败，或其他已读取材料能够证实的问题；其他事项仍按原规则判断。$policy$ AS content
)
UPDATE system_prompt_templates AS templates
SET content = replace(templates.content, '【规则适用前提】', policy.content || E'\n\n【规则适用前提】'),
    updated_at = now()
FROM policy
WHERE templates.prompt_key IN (
    'audit_system_reasoning_standard', 'audit_system_reasoning_loose',
    'audit_system_extraction_standard', 'audit_system_extraction_loose',
    'archive_system_reasoning_standard', 'archive_system_reasoning_loose',
    'archive_system_extraction_standard', 'archive_system_extraction_loose'
)
AND position('【已获取附件解析失败的处理】' IN templates.content) = 0;
