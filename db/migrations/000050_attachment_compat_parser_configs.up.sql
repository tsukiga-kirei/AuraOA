-- 000050_attachment_compat_parser_configs.up.sql
-- 增加旧版 Office / OFD 兼容解析服务配置，并扩展附件默认白名单。

INSERT INTO system_configs (key, value, remark) VALUES
    ('attachment.compat_endpoint',          'http://document-parser:8090', '兼容格式解析服务地址'),
    ('attachment.compat_api_key',           '',                            '兼容格式解析服务 API Key（可选）'),
    ('attachment.legacy_office_enabled',    'false',                       '是否启用 DOC、XLS、PPT 旧版 Office 解析'),
    ('attachment.ofd_enabled',              'false',                       '是否启用 OFD 解析'),
    ('attachment.visual_fallback_enabled',  'true',                        'OFD 无文字层时是否转 PDF 并回退 MinerU')
ON CONFLICT (key) DO NOTHING;

-- 仅升级历史默认值，避免覆盖管理员自定义的文件类型白名单。
UPDATE system_configs
SET value = 'pdf,png,jpg,jpeg,bmp,gif,tiff,webp,txt,csv,md,docx,xlsx,pptx,doc,xls,ppt,ofd'
WHERE key = 'attachment.supported_types'
  AND value IN (
      'pdf,png,jpg,jpeg,bmp,gif,tiff,webp,docx,xlsx,txt',
      'pdf,png,jpg,jpeg,bmp,gif,tiff,webp,txt,docx,xlsx'
  );
