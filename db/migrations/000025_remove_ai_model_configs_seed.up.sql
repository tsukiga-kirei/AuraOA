-- 000025_remove_ai_model_configs_seed.up.sql
-- 移除历史版本中 000005 写入的 ai_model_configs 预置行；新库已不再 INSERT 这些行。

WITH seed_rows AS (
    SELECT id
    FROM ai_model_configs
    WHERE (provider, model_name) IN (
        ('xinference', 'Qwen2.5-72B'),
        ('xinference', 'Qwen2.5-32B'),
        ('aliyun_bailian', 'qwen-plus'),
        ('aliyun_bailian', 'qwen-max'),
        ('xinference', 'DeepSeek-V3'),
        ('deepseek', 'deepseek-chat'),
        ('ollama', 'qwen2.5:14b')
    )
)
UPDATE tenant_llm_message_logs t
SET model_config_id = NULL
WHERE t.model_config_id IN (SELECT id FROM seed_rows);

DELETE FROM ai_model_configs
WHERE (provider, model_name) IN (
    ('xinference', 'Qwen2.5-72B'),
    ('xinference', 'Qwen2.5-32B'),
    ('aliyun_bailian', 'qwen-plus'),
    ('aliyun_bailian', 'qwen-max'),
    ('xinference', 'DeepSeek-V3'),
    ('deepseek', 'deepseek-chat'),
    ('ollama', 'qwen2.5:14b')
);
