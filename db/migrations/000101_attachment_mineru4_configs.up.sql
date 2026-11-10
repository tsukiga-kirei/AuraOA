-- MinerU 4.0 使用独立配置，升级后仍保留管理员原先选择的 OCR 引擎。
INSERT INTO system_configs (key, value, remark) VALUES
('attachment.mineru4_endpoint', '', 'MinerU 4.0 V1 解析服务根地址'),
('attachment.mineru4_api_key', '', 'MinerU 4.0 API Key（可选）'),
('attachment.mineru4_tier', 'standard', 'MinerU 4.0 PDF/图片解析档位：flash/basic/standard/advanced'),
('attachment.mineru4_ocr_mode', 'auto', 'MinerU 4.0 OCR 模式：auto/txt/ocr')
ON CONFLICT (key) DO NOTHING;
