-- 001_oa_ai_seeds.sql
-- 演示数据：OA数据库连接配置
-- 注意：ai_model_configs 预置数据已迁移至 000005_system_options_oa_ai.up.sql

-- ============================================================
-- 演示 OA 数据库连接
-- UUID 约定：b0000000-0000-0000-0000-00000000000x
-- ============================================================
INSERT INTO oa_database_connections
    (id, name, oa_type, oa_type_label, driver, host, port, database_name,
     username, password, pool_size, connection_timeout, test_on_borrow,
     status, sync_interval, enabled, description)
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
