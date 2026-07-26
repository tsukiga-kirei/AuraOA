-- 000050_attachment_compat_parser_configs.down.sql

DELETE FROM system_configs
WHERE key IN (
    'attachment.compat_endpoint',
    'attachment.compat_api_key',
    'attachment.legacy_office_enabled',
    'attachment.ofd_enabled',
    'attachment.visual_fallback_enabled'
);

-- 只回退本次迁移写入的完整默认值，避免覆盖后续管理员自定义配置。
UPDATE system_configs
SET value = 'pdf,png,jpg,jpeg,bmp,gif,tiff,webp,docx,xlsx,txt'
WHERE key = 'attachment.supported_types'
  AND value = 'pdf,png,jpg,jpeg,bmp,gif,tiff,webp,txt,csv,md,docx,xlsx,pptx,doc,xls,ppt,ofd';
