-- 000005_system_options_oa_ai.up.sql
-- 创建选项枚举表、OA数据库连接表、AI模型配置表，扩展租户表AI字段
-- 并初始化所有枚举选项和预置AI模型配置

-- ============================================================
-- oa_type_options — OA系统类型选项表
-- ============================================================
CREATE TABLE oa_type_options (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(), -- 主键UUID
    code       VARCHAR(50)  NOT NULL UNIQUE,                       -- 选项编码（程序内部使用）
    label      VARCHAR(100) NOT NULL,                              -- 选项显示名称（前端展示）
    sort_order INT          NOT NULL DEFAULT 0,                    -- 排序权重（越小越靠前）
    enabled    BOOLEAN      NOT NULL DEFAULT TRUE,                 -- 是否启用（禁用后前端不显示）
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now()                 -- 创建时间
);

-- ============================================================
-- db_driver_options — 数据库驱动类型选项表
-- ============================================================
CREATE TABLE db_driver_options (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(), -- 主键UUID
    code         VARCHAR(50)  NOT NULL UNIQUE,                       -- 驱动编码（对应Go数据库驱动名）
    label        VARCHAR(100) NOT NULL,                              -- 驱动显示名称
    default_port INT          NOT NULL DEFAULT 3306,                 -- 该数据库类型的默认端口
    sort_order   INT          NOT NULL DEFAULT 0,                    -- 排序权重
    enabled      BOOLEAN      NOT NULL DEFAULT TRUE,                 -- 是否启用
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now()                 -- 创建时间
);

-- ============================================================
-- ai_deploy_type_options — AI模型部署类型选项表
-- ============================================================
CREATE TABLE ai_deploy_type_options (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(), -- 主键UUID
    code       VARCHAR(50)  NOT NULL UNIQUE,                       -- 部署类型编码：local/cloud
    label      VARCHAR(100) NOT NULL,                              -- 部署类型显示名称
    sort_order INT          NOT NULL DEFAULT 0,                    -- 排序权重
    enabled    BOOLEAN      NOT NULL DEFAULT TRUE,                 -- 是否启用
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now()                 -- 创建时间
);

-- ============================================================
-- ai_provider_options — AI服务商选项表
-- ============================================================
CREATE TABLE ai_provider_options (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(), -- 主键UUID
    code        VARCHAR(100) NOT NULL UNIQUE,                       -- 服务商编码（与ai_model_configs.provider对应）
    label       VARCHAR(100) NOT NULL,                              -- 服务商显示名称
    deploy_type VARCHAR(50)  NOT NULL,                              -- 所属部署类型：local/cloud
    sort_order  INT          NOT NULL DEFAULT 0,                    -- 排序权重
    enabled     BOOLEAN      NOT NULL DEFAULT TRUE,                 -- 是否启用
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()                 -- 创建时间
);

-- ============================================================
-- oa_database_connections — OA数据库连接配置表
-- ============================================================
CREATE TABLE oa_database_connections (
    id                 UUID         PRIMARY KEY DEFAULT gen_random_uuid(), -- 主键UUID
    name               VARCHAR(200) NOT NULL,                              -- 连接名称（管理员自定义）
    oa_type            VARCHAR(50)  NOT NULL,                              -- OA系统类型编码（关联oa_type_options.code）
    oa_type_label      VARCHAR(100) DEFAULT '',                            -- OA系统类型显示名称（冗余字段，避免join）
    driver             VARCHAR(50)  NOT NULL DEFAULT 'mysql',              -- 数据库驱动类型（关联db_driver_options.code）
    host               VARCHAR(255) NOT NULL DEFAULT '',                   -- 数据库主机地址
    port               INT          NOT NULL DEFAULT 3306,                 -- 数据库端口
    database_name      VARCHAR(200) NOT NULL DEFAULT '',                   -- 数据库名称
    username           VARCHAR(200) NOT NULL DEFAULT '',                   -- 数据库登录用户名
    password           VARCHAR(500) NOT NULL DEFAULT '',                   -- 数据库登录密码（加密存储）
    pool_size          INT          NOT NULL DEFAULT 10,                   -- 连接池最大连接数
    connection_timeout INT          NOT NULL DEFAULT 30,                   -- 连接超时时间（秒）
    test_on_borrow     BOOLEAN      NOT NULL DEFAULT TRUE,                 -- 从连接池取出连接时是否先测试连通性
    status             VARCHAR(20)  NOT NULL DEFAULT 'disconnected',       -- 连接状态：connected/disconnected/error
    last_sync          TIMESTAMPTZ,                                        -- 最后一次成功同步时间
    sync_interval      INT          NOT NULL DEFAULT 30,                   -- 同步间隔（分钟）
    enabled            BOOLEAN      NOT NULL DEFAULT TRUE,                 -- 是否启用该连接
    description        TEXT         DEFAULT '',                            -- 连接描述说明
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),                -- 创建时间
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT now()                 -- 最后更新时间
);

-- ============================================================
-- ai_model_configs — AI模型配置表
-- ============================================================
CREATE TABLE ai_model_configs (
    id                 UUID          PRIMARY KEY DEFAULT gen_random_uuid(), -- 主键UUID
    provider           VARCHAR(100)  NOT NULL,                              -- 服务商编码（关联ai_provider_options.code）
    provider_label     VARCHAR(100)  DEFAULT '',                            -- 服务商显示名称（冗余字段，避免join）
    model_name         VARCHAR(100)  NOT NULL,                              -- 模型标识名（如 qwen2.5-72b，API调用时使用）
    display_name       VARCHAR(200)  NOT NULL,                              -- 模型显示名称（前端展示）
    deploy_type        VARCHAR(20)   NOT NULL DEFAULT 'local',              -- 部署类型：local/cloud
    endpoint           VARCHAR(500)  NOT NULL DEFAULT '',                   -- API接入端点URL
    api_key            VARCHAR(500)  DEFAULT '',                            -- API密钥（json序列化时隐藏）
    api_key_configured BOOLEAN       NOT NULL DEFAULT FALSE,                -- API密钥是否已配置（前端用于状态显示）
    max_tokens         INT           NOT NULL DEFAULT 8192,                 -- 单次请求最大输出Token数
    context_window     INT           NOT NULL DEFAULT 131072,               -- 模型上下文窗口大小（Token数）
    cost_per_1k_tokens DECIMAL(10,6) DEFAULT 0,                            -- 每1000 Token费用（元，本地部署填0）
    status             VARCHAR(20)   NOT NULL DEFAULT 'offline',            -- 模型状态：online/offline/maintenance
    enabled            BOOLEAN       NOT NULL DEFAULT TRUE,                 -- 是否启用（禁用后租户不可选择）
    description        TEXT          DEFAULT '',                            -- 模型描述说明
    capabilities       JSONB         NOT NULL DEFAULT '[]'::jsonb,          -- 模型能力列表（如["text","code","reasoning","vision"]）
    created_at         TIMESTAMPTZ   NOT NULL DEFAULT now(),                -- 创建时间
    updated_at         TIMESTAMPTZ   NOT NULL DEFAULT now()                 -- 最后更新时间
);

-- ============================================================
-- 扩展租户表：添加AI模型直接引用字段，替代原 ai_config JSONB 字段
-- ============================================================
ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS primary_model_id      UUID,                              -- 租户主用AI模型ID（外键在下方添加）
    ADD COLUMN IF NOT EXISTS fallback_model_id     UUID,                              -- 租户备用AI模型ID（主模型不可用时切换）
    ADD COLUMN IF NOT EXISTS max_tokens_per_request INT NOT NULL DEFAULT 8192,        -- 单次审核最大输出Token限制
    ADD COLUMN IF NOT EXISTS temperature           DECIMAL(3,2) NOT NULL DEFAULT 0.30, -- AI生成温度参数（0~1，越低越确定）
    ADD COLUMN IF NOT EXISTS timeout_seconds       INT NOT NULL DEFAULT 60,            -- AI请求超时时间（秒）
    ADD COLUMN IF NOT EXISTS retry_count           INT NOT NULL DEFAULT 3;             -- AI请求失败重试次数

-- 添加外键约束
ALTER TABLE tenants
    ADD CONSTRAINT fk_tenants_oa_db         FOREIGN KEY (oa_db_connection_id) REFERENCES oa_database_connections(id) ON DELETE SET NULL,
    ADD CONSTRAINT fk_tenants_primary_model  FOREIGN KEY (primary_model_id)   REFERENCES ai_model_configs(id) ON DELETE SET NULL,
    ADD CONSTRAINT fk_tenants_fallback_model FOREIGN KEY (fallback_model_id)  REFERENCES ai_model_configs(id) ON DELETE SET NULL;

-- 移除已被独立字段替代的冗余列
ALTER TABLE tenants DROP COLUMN IF EXISTS oa_type;
ALTER TABLE tenants DROP COLUMN IF EXISTS allow_custom_model;
ALTER TABLE tenants DROP COLUMN IF EXISTS ai_config;

-- ============================================================
-- 初始化枚举选项数据
-- ============================================================

-- OA系统类型
INSERT INTO oa_type_options (code, label, sort_order) VALUES
    ('weaver_e9',      '泛微 Ecology E9', 1),
    ('weaver_ebridge', '泛微 E-Bridge',   2),
    ('zhiyuan_a8',     '致远 A8+',        3),
    ('landray_ekp',    '蓝凌 EKP',        4),
    ('custom',         '自定义 OA',       99);

-- 数据库驱动类型
INSERT INTO db_driver_options (code, label, default_port, sort_order) VALUES
    ('mysql',      'MySQL',      3306, 1),
    ('oracle',     'Oracle',     1521, 2),
    ('postgresql', 'PostgreSQL', 5432, 3),
    ('sqlserver',  'SQL Server', 1433, 4),
    ('dm',         '达梦 DM',    5236, 5);

-- AI部署类型
INSERT INTO ai_deploy_type_options (code, label, sort_order) VALUES
    ('local', '本地部署', 1),
    ('cloud', '云端API',  2);

-- AI服务商
INSERT INTO ai_provider_options (code, label, deploy_type, sort_order) VALUES
    ('xinference',     'Xinference',   'local', 1),
    ('ollama',         'Ollama',       'local', 2),
    ('vllm',           'vLLM',         'local', 3),
    ('aliyun_bailian', '阿里云百炼',   'cloud', 10),
    ('deepseek',       'DeepSeek',     'cloud', 11),
    ('zhipu',          '智谱 AI',      'cloud', 12),
    ('openai',         'OpenAI',       'cloud', 13),
    ('azure_openai',   'Azure OpenAI', 'cloud', 14);

-- ============================================================
-- 数据库注释（中文）
-- ============================================================
COMMENT ON TABLE oa_type_options IS 'OA系统类型选项表';
COMMENT ON COLUMN oa_type_options.id IS '主键UUID';
COMMENT ON COLUMN oa_type_options.code IS '选项编码（程序内部使用）';
COMMENT ON COLUMN oa_type_options.label IS '选项显示名称（前端展示）';
COMMENT ON COLUMN oa_type_options.sort_order IS '排序权重（越小越靠前）';
COMMENT ON COLUMN oa_type_options.enabled IS '是否启用（禁用后前端不显示）';
COMMENT ON COLUMN oa_type_options.created_at IS '创建时间';

COMMENT ON TABLE db_driver_options IS '数据库驱动类型选项表';
COMMENT ON COLUMN db_driver_options.id IS '主键UUID';
COMMENT ON COLUMN db_driver_options.code IS '驱动编码（对应Go数据库驱动名）';
COMMENT ON COLUMN db_driver_options.label IS '驱动显示名称';
COMMENT ON COLUMN db_driver_options.default_port IS '该数据库类型的默认端口';
COMMENT ON COLUMN db_driver_options.sort_order IS '排序权重';
COMMENT ON COLUMN db_driver_options.enabled IS '是否启用';
COMMENT ON COLUMN db_driver_options.created_at IS '创建时间';

COMMENT ON TABLE ai_deploy_type_options IS 'AI模型部署类型选项表';
COMMENT ON COLUMN ai_deploy_type_options.id IS '主键UUID';
COMMENT ON COLUMN ai_deploy_type_options.code IS '部署类型编码：local/cloud';
COMMENT ON COLUMN ai_deploy_type_options.label IS '部署类型显示名称';
COMMENT ON COLUMN ai_deploy_type_options.sort_order IS '排序权重';
COMMENT ON COLUMN ai_deploy_type_options.enabled IS '是否启用';
COMMENT ON COLUMN ai_deploy_type_options.created_at IS '创建时间';

COMMENT ON TABLE ai_provider_options IS 'AI服务商选项表';
COMMENT ON COLUMN ai_provider_options.id IS '主键UUID';
COMMENT ON COLUMN ai_provider_options.code IS '服务商编码（与ai_model_configs.provider对应）';
COMMENT ON COLUMN ai_provider_options.label IS '服务商显示名称';
COMMENT ON COLUMN ai_provider_options.deploy_type IS '所属部署类型：local/cloud';
COMMENT ON COLUMN ai_provider_options.sort_order IS '排序权重';
COMMENT ON COLUMN ai_provider_options.enabled IS '是否启用';
COMMENT ON COLUMN ai_provider_options.created_at IS '创建时间';

COMMENT ON TABLE oa_database_connections IS 'OA数据库连接配置表';
COMMENT ON COLUMN oa_database_connections.id IS '主键UUID';
COMMENT ON COLUMN oa_database_connections.name IS '连接名称（管理员自定义）';
COMMENT ON COLUMN oa_database_connections.oa_type IS 'OA系统类型编码（关联oa_type_options.code）';
COMMENT ON COLUMN oa_database_connections.oa_type_label IS 'OA系统类型显示名称（冗余字段，避免join）';
COMMENT ON COLUMN oa_database_connections.driver IS '数据库驱动类型（关联db_driver_options.code）';
COMMENT ON COLUMN oa_database_connections.host IS '数据库主机地址';
COMMENT ON COLUMN oa_database_connections.port IS '数据库端口';
COMMENT ON COLUMN oa_database_connections.database_name IS '数据库名称';
COMMENT ON COLUMN oa_database_connections.username IS '数据库登录用户名';
COMMENT ON COLUMN oa_database_connections.password IS '数据库登录密码（加密存储）';
COMMENT ON COLUMN oa_database_connections.pool_size IS '连接池最大连接数';
COMMENT ON COLUMN oa_database_connections.connection_timeout IS '连接超时时间（秒）';
COMMENT ON COLUMN oa_database_connections.test_on_borrow IS '从连接池取出连接时是否先测试连通性';
COMMENT ON COLUMN oa_database_connections.status IS '连接状态：connected/disconnected/error';
COMMENT ON COLUMN oa_database_connections.last_sync IS '最后一次成功同步时间';
COMMENT ON COLUMN oa_database_connections.sync_interval IS '同步间隔（分钟）';
COMMENT ON COLUMN oa_database_connections.enabled IS '是否启用该连接';
COMMENT ON COLUMN oa_database_connections.description IS '连接描述说明';
COMMENT ON COLUMN oa_database_connections.created_at IS '创建时间';
COMMENT ON COLUMN oa_database_connections.updated_at IS '最后更新时间';

COMMENT ON TABLE ai_model_configs IS 'AI模型配置表';
COMMENT ON COLUMN ai_model_configs.id IS '主键UUID';
COMMENT ON COLUMN ai_model_configs.provider IS '服务商编码（关联ai_provider_options.code）';
COMMENT ON COLUMN ai_model_configs.provider_label IS '服务商显示名称（冗余字段，避免join）';
COMMENT ON COLUMN ai_model_configs.model_name IS '模型标识名（如 qwen2.5-72b，API调用时使用）';
COMMENT ON COLUMN ai_model_configs.display_name IS '模型显示名称（前端展示）';
COMMENT ON COLUMN ai_model_configs.deploy_type IS '部署类型：local/cloud';
COMMENT ON COLUMN ai_model_configs.endpoint IS 'API接入端点URL';
COMMENT ON COLUMN ai_model_configs.api_key IS 'API密钥（序列化时隐藏）';
COMMENT ON COLUMN ai_model_configs.api_key_configured IS 'API密钥是否已配置（前端用于状态显示）';
COMMENT ON COLUMN ai_model_configs.max_tokens IS '单次请求最大输出Token数';
COMMENT ON COLUMN ai_model_configs.context_window IS '模型上下文窗口大小（Token数）';
COMMENT ON COLUMN ai_model_configs.cost_per_1k_tokens IS '每1000 Token费用（元，本地部署填0）';
COMMENT ON COLUMN ai_model_configs.status IS '模型状态：online/offline/maintenance';
COMMENT ON COLUMN ai_model_configs.enabled IS '是否启用（禁用后租户不可选择）';
COMMENT ON COLUMN ai_model_configs.description IS '模型描述说明';
COMMENT ON COLUMN ai_model_configs.capabilities IS '模型能力列表（如["text","code","reasoning","vision"]）';
COMMENT ON COLUMN ai_model_configs.created_at IS '创建时间';
COMMENT ON COLUMN ai_model_configs.updated_at IS '最后更新时间';

-- 扩展 tenants 表新增字段注释
COMMENT ON COLUMN tenants.primary_model_id IS '租户主用AI模型ID';
COMMENT ON COLUMN tenants.fallback_model_id IS '租户备用AI模型ID（主模型不可用时切换）';
COMMENT ON COLUMN tenants.max_tokens_per_request IS '单次审核最大输出Token限制';
COMMENT ON COLUMN tenants.temperature IS 'AI生成温度参数（0~1，越低越确定）';
COMMENT ON COLUMN tenants.timeout_seconds IS 'AI请求超时时间（秒）';
COMMENT ON COLUMN tenants.retry_count IS 'AI请求失败重试次数';

-- ============================================================
-- 数据库注释（中文）
-- ============================================================
COMMENT ON TABLE oa_type_options IS 'OA系统类型选项表';
COMMENT ON COLUMN oa_type_options.id IS '主键UUID';
COMMENT ON COLUMN oa_type_options.code IS '选项编码（程序内部使用）';
COMMENT ON COLUMN oa_type_options.label IS '选项显示名称（前端展示）';
COMMENT ON COLUMN oa_type_options.sort_order IS '排序权重（越小越靠前）';
COMMENT ON COLUMN oa_type_options.enabled IS '是否启用（禁用后前端不显示）';
COMMENT ON COLUMN oa_type_options.created_at IS '创建时间';

COMMENT ON TABLE db_driver_options IS '数据库驱动类型选项表';
COMMENT ON COLUMN db_driver_options.id IS '主键UUID';
COMMENT ON COLUMN db_driver_options.code IS '驱动编码（对应Go数据库驱动名）';
COMMENT ON COLUMN db_driver_options.label IS '驱动显示名称';
COMMENT ON COLUMN db_driver_options.default_port IS '该数据库类型的默认端口';
COMMENT ON COLUMN db_driver_options.sort_order IS '排序权重';
COMMENT ON COLUMN db_driver_options.enabled IS '是否启用';
COMMENT ON COLUMN db_driver_options.created_at IS '创建时间';

COMMENT ON TABLE ai_deploy_type_options IS 'AI模型部署类型选项表';
COMMENT ON COLUMN ai_deploy_type_options.id IS '主键UUID';
COMMENT ON COLUMN ai_deploy_type_options.code IS '部署类型编码：local/cloud';
COMMENT ON COLUMN ai_deploy_type_options.label IS '部署类型显示名称';
COMMENT ON COLUMN ai_deploy_type_options.sort_order IS '排序权重';
COMMENT ON COLUMN ai_deploy_type_options.enabled IS '是否启用';
COMMENT ON COLUMN ai_deploy_type_options.created_at IS '创建时间';

COMMENT ON TABLE ai_provider_options IS 'AI服务商选项表';
COMMENT ON COLUMN ai_provider_options.id IS '主键UUID';
COMMENT ON COLUMN ai_provider_options.code IS '服务商编码（与ai_model_configs.provider对应）';
COMMENT ON COLUMN ai_provider_options.label IS '服务商显示名称';
COMMENT ON COLUMN ai_provider_options.deploy_type IS '所属部署类型：local/cloud';
COMMENT ON COLUMN ai_provider_options.sort_order IS '排序权重';
COMMENT ON COLUMN ai_provider_options.enabled IS '是否启用';
COMMENT ON COLUMN ai_provider_options.created_at IS '创建时间';

COMMENT ON TABLE oa_database_connections IS 'OA数据库连接配置表';
COMMENT ON COLUMN oa_database_connections.id IS '主键UUID';
COMMENT ON COLUMN oa_database_connections.name IS '连接名称（管理员自定义）';
COMMENT ON COLUMN oa_database_connections.oa_type IS 'OA系统类型编码（关联oa_type_options.code）';
COMMENT ON COLUMN oa_database_connections.oa_type_label IS 'OA系统类型显示名称（冗余字段，避免join）';
COMMENT ON COLUMN oa_database_connections.driver IS '数据库驱动类型（关联db_driver_options.code）';
COMMENT ON COLUMN oa_database_connections.host IS '数据库主机地址';
COMMENT ON COLUMN oa_database_connections.port IS '数据库端口';
COMMENT ON COLUMN oa_database_connections.database_name IS '数据库名称';
COMMENT ON COLUMN oa_database_connections.username IS '数据库登录用户名';
COMMENT ON COLUMN oa_database_connections.password IS '数据库登录密码（加密存储）';
COMMENT ON COLUMN oa_database_connections.pool_size IS '连接池最大连接数';
COMMENT ON COLUMN oa_database_connections.connection_timeout IS '连接超时时间（秒）';
COMMENT ON COLUMN oa_database_connections.test_on_borrow IS '从连接池取出连接时是否先测试连通性';
COMMENT ON COLUMN oa_database_connections.status IS '连接状态：connected/disconnected/error';
COMMENT ON COLUMN oa_database_connections.last_sync IS '最后一次成功同步时间';
COMMENT ON COLUMN oa_database_connections.sync_interval IS '同步间隔（分钟）';
COMMENT ON COLUMN oa_database_connections.enabled IS '是否启用该连接';
COMMENT ON COLUMN oa_database_connections.description IS '连接描述说明';
COMMENT ON COLUMN oa_database_connections.created_at IS '创建时间';
COMMENT ON COLUMN oa_database_connections.updated_at IS '最后更新时间';

COMMENT ON TABLE ai_model_configs IS 'AI模型配置表';
COMMENT ON COLUMN ai_model_configs.id IS '主键UUID';
COMMENT ON COLUMN ai_model_configs.provider IS '服务商编码（关联ai_provider_options.code）';
COMMENT ON COLUMN ai_model_configs.provider_label IS '服务商显示名称（冗余字段，避免join）';
COMMENT ON COLUMN ai_model_configs.model_name IS '模型标识名（如 qwen2.5-72b，API调用时使用）';
COMMENT ON COLUMN ai_model_configs.display_name IS '模型显示名称（前端展示）';
COMMENT ON COLUMN ai_model_configs.deploy_type IS '部署类型：local/cloud';
COMMENT ON COLUMN ai_model_configs.endpoint IS 'API接入端点URL';
COMMENT ON COLUMN ai_model_configs.api_key IS 'API密钥（json序列化时隐藏）';
COMMENT ON COLUMN ai_model_configs.api_key_configured IS 'API密钥是否已配置（前端用于状态显示）';
COMMENT ON COLUMN ai_model_configs.max_tokens IS '单次请求最大输出Token数';
COMMENT ON COLUMN ai_model_configs.context_window IS '模型上下文窗口大小（Token数）';
COMMENT ON COLUMN ai_model_configs.cost_per_1k_tokens IS '每1000 Token费用（元，本地部署填0）';
COMMENT ON COLUMN ai_model_configs.status IS '模型状态：online/offline/maintenance';
COMMENT ON COLUMN ai_model_configs.enabled IS '是否启用（禁用后租户不可选择）';
COMMENT ON COLUMN ai_model_configs.description IS '模型描述说明';
COMMENT ON COLUMN ai_model_configs.capabilities IS '模型能力列表（如["text","code","reasoning","vision"]）';
COMMENT ON COLUMN ai_model_configs.created_at IS '创建时间';
COMMENT ON COLUMN ai_model_configs.updated_at IS '最后更新时间';

COMMENT ON COLUMN tenants.primary_model_id IS '租户主用AI模型ID';
COMMENT ON COLUMN tenants.fallback_model_id IS '租户备用AI模型ID（主模型不可用时切换）';
COMMENT ON COLUMN tenants.max_tokens_per_request IS '单次审核最大输出Token限制';
COMMENT ON COLUMN tenants.temperature IS 'AI生成温度参数（0~1，越低越确定）';
COMMENT ON COLUMN tenants.timeout_seconds IS 'AI请求超时时间（秒）';
COMMENT ON COLUMN tenants.retry_count IS 'AI请求失败重试次数';
