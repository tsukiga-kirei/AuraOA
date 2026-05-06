-- 附件识别配置项
INSERT INTO system_configs (key, value, remark) VALUES
('attachment.recognition_enabled', 'false', '是否启用附件识别功能'),
('attachment.mineru_endpoint', 'http://mineru:8080', 'MinerU 文档解析服务地址'),
('attachment.mineru_api_key', '', 'MinerU API Key（可选）'),
('attachment.oa_api_endpoint', '', 'OA 系统附件接口地址（返回文件流，如：http://oa-server:8080/api/attachments）'),
('attachment.max_file_size_mb', '10', '最大文件大小限制（MB）'),
('attachment.supported_types', 'pdf,png,jpg,jpeg,bmp,gif,tiff,webp,docx,xlsx,txt', '支持的文件类型（逗号分隔）')
ON CONFLICT (key) DO NOTHING;
