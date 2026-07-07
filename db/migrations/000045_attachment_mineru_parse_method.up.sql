-- 000045_attachment_mineru_parse_method.up.sql
-- 将 MinerU OCR 布尔开关升级为 parse_method（auto / txt / ocr）

INSERT INTO system_configs (key, value, remark) VALUES
    ('attachment.mineru_parse_method', 'ocr', 'MinerU 解析方式：auto / txt / ocr')
ON CONFLICT (key) DO NOTHING;

UPDATE system_configs AS pm
SET value = 'txt'
FROM system_configs AS ocr
WHERE pm.key = 'attachment.mineru_parse_method'
  AND ocr.key = 'attachment.mineru_enable_ocr'
  AND ocr.value IN ('false', '0');
