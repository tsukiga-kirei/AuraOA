-- 001_oa_ai_seeds.sql
-- Seed data: OA database connections and AI model configs
-- References mock data from frontend for consistency

-- ============================================================
-- OA数据库连接
-- ============================================================
INSERT INTO oa_database_connections (id, name, oa_type, oa_type_label, driver, host, port, database_name, username, password, pool_size, connection_timeout, test_on_borrow, status, sync_interval, enabled, description)
VALUES
    (
        'b0000000-0000-0000-0000-000000000001',
        '总部泛微E9数据库', 'weaver_e9', '泛微 Ecology E9',
        'mysql', '192.168.1.100', 3306, 'ecology',
        'oa_reader', '********', 20, 30, TRUE,
        'connected', 30, TRUE,
        '总部泛微E9 OA系统主数据库，用于流程数据同步'
    ),
    (
        'b0000000-0000-0000-0000-000000000002',
        '华东分公司E9数据库', 'weaver_e9', '泛微 Ecology E9',
        'mysql', '192.168.2.100', 3306, 'ecology_east',
        'oa_reader', '********', 10, 30, TRUE,
        'connected', 60, TRUE,
        '华东分公司泛微E9数据库'
    ),
    (
        'b0000000-0000-0000-0000-000000000003',
        '测试环境数据库', 'weaver_e9', '泛微 Ecology E9',
        'oracle', 'localhost', 1521, 'ecology_test',
        'test_reader', '********', 5, 15, FALSE,
        'disconnected', 120, FALSE,
        '用于系统测试和演示的OA数据库连接'
    );

-- ============================================================
-- AI模型配置
-- ============================================================
INSERT INTO ai_model_configs (id, provider, provider_label, model_name, display_name, deploy_type, endpoint, api_key_configured, max_tokens, context_window, cost_per_1k_tokens, status, enabled, description, capabilities)
VALUES
    (
        'c0000000-0000-0000-0000-000000000001',
        'xinference', 'Xinference',
        'Qwen2.5-72B', 'Qwen2.5-72B（本地）', 'local',
        'http://192.168.1.50:9997/v1', FALSE,
        8192, 131072, 0,
        'online', TRUE,
        '通义千问2.5 72B 参数大模型，通过 Xinference 本地私有部署，数据不出域',
        '["text","code","reasoning","analysis"]'
    ),
    (
        'c0000000-0000-0000-0000-000000000002',
        'xinference', 'Xinference',
        'Qwen2.5-32B', 'Qwen2.5-32B（本地）', 'local',
        'http://192.168.1.50:9997/v1', FALSE,
        4096, 65536, 0,
        'online', TRUE,
        '通义千问2.5 32B 参数大模型，通过 Xinference 部署，适合轻量级审核任务',
        '["text","code","reasoning"]'
    ),
    (
        'c0000000-0000-0000-0000-000000000003',
        'aliyun_bailian', '阿里云百炼',
        'qwen-plus', 'Qwen-Plus（阿里云百炼）', 'cloud',
        'https://dashscope.aliyuncs.com/compatible-mode/v1', TRUE,
        16384, 131072, 0.008,
        'online', TRUE,
        '阿里云百炼 Qwen-Plus 大模型，云端部署，性价比高',
        '["text","code","reasoning","analysis"]'
    ),
    (
        'c0000000-0000-0000-0000-000000000004',
        'aliyun_bailian', '阿里云百炼',
        'qwen-max', 'Qwen-Max（阿里云百炼）', 'cloud',
        'https://dashscope.aliyuncs.com/compatible-mode/v1', TRUE,
        8192, 131072, 0.02,
        'online', FALSE,
        '阿里云百炼 Qwen-Max 旗舰模型，适合复杂合同和法务审核',
        '["text","code","reasoning","vision","analysis"]'
    ),
    (
        'c0000000-0000-0000-0000-000000000005',
        'xinference', 'Xinference',
        'DeepSeek-V3', 'DeepSeek-V3（本地）', 'local',
        'http://192.168.1.51:9997/v1', FALSE,
        8192, 65536, 0,
        'maintenance', FALSE,
        'DeepSeek V3 大模型，通过 Xinference 部署，擅长代码和推理任务',
        '["text","code","reasoning"]'
    ),
    (
        'c0000000-0000-0000-0000-000000000006',
        'deepseek', 'DeepSeek',
        'deepseek-chat', 'DeepSeek Chat（云端）', 'cloud',
        'https://api.deepseek.com/v1', TRUE,
        8192, 65536, 0.001,
        'online', TRUE,
        'DeepSeek Chat 云端模型，性价比极高',
        '["text","code","reasoning"]'
    ),
    (
        'c0000000-0000-0000-0000-000000000007',
        'ollama', 'Ollama',
        'qwen2.5:14b', 'Qwen2.5-14B (Ollama)', 'local',
        'http://192.168.1.52:11434/v1', FALSE,
        4096, 32768, 0,
        'offline', FALSE,
        'Ollama 本地部署 Qwen2.5 14B 轻量模型',
        '["text","reasoning"]'
    );
