-- 000079_attachment_aliyun_ocr_configs.down.sql
-- 回滚阿里云 OCR 相关系统配置。

DELETE FROM system_configs WHERE key IN (
    'attachment.ocr_provider',
    'attachment.aliyun_ocr_endpoint',
    'attachment.aliyun_ocr_access_key_id',
    'attachment.aliyun_ocr_access_key_secret',
    'attachment.aliyun_ocr_type'
);
