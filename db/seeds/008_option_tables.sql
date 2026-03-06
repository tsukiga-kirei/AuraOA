-- 008_option_tables.sql
-- Seed data: OA type options, DB driver options, AI deploy type options, AI provider options

-- ============================================================
-- OA系统类型选项
-- ============================================================
INSERT INTO oa_type_options (code, label, sort_order) VALUES
    ('weaver_e9',      '泛微 Ecology E9', 1),
    ('weaver_ebridge', '泛微 E-Bridge',   2),
    ('zhiyuan_a8',     '致远 A8+',        3),
    ('landray_ekp',    '蓝凌 EKP',        4),
    ('custom',         '自定义 OA',       99);

-- ============================================================
-- 数据库驱动选项
-- ============================================================
INSERT INTO db_driver_options (code, label, default_port, sort_order) VALUES
    ('mysql',      'MySQL',      3306, 1),
    ('oracle',     'Oracle',     1521, 2),
    ('postgresql', 'PostgreSQL', 5432, 3),
    ('sqlserver',  'SQL Server', 1433, 4);

-- ============================================================
-- AI部署类型选项
-- ============================================================
INSERT INTO ai_deploy_type_options (code, label, sort_order) VALUES
    ('local', '本地部署', 1),
    ('cloud', '云端API',  2);

-- ============================================================
-- AI服务商选项
-- ============================================================
INSERT INTO ai_provider_options (code, label, deploy_type, sort_order) VALUES
    ('xinference',     'Xinference',     'local', 1),
    ('ollama',         'Ollama',         'local', 2),
    ('vllm',           'vLLM',           'local', 3),
    ('aliyun_bailian', '阿里云百炼',     'cloud', 10),
    ('deepseek',       'DeepSeek',       'cloud', 11),
    ('zhipu',          '智谱 AI',        'cloud', 12),
    ('openai',         'OpenAI',         'cloud', 13),
    ('azure_openai',   'Azure OpenAI',   'cloud', 14);
