-- 000079_attachment_aliyun_ocr_configs.up.sql
-- 增加阿里云 OCR 统一文字识别服务配置，支持在 MinerU 与阿里云 OCR 间选配。

INSERT INTO system_configs (key, value, remark) VALUES
    ('attachment.ocr_provider',               'mineru',                           'OCR 解析服务提供商（mineru / aliyun）'),
    ('attachment.aliyun_ocr_endpoint',        'ocr-api.cn-hangzhou.aliyuncs.com', '阿里云 OCR 服务接入地址'),
    ('attachment.aliyun_ocr_access_key_id',   '',                                 '阿里云 AccessKey ID'),
    ('attachment.aliyun_ocr_access_key_secret', '',                               '阿里云 AccessKey Secret'),
    ('attachment.aliyun_ocr_type',            'General',                          '阿里云 OCR 识别类型（General / Advanced）')
ON CONFLICT (key) DO NOTHING;
