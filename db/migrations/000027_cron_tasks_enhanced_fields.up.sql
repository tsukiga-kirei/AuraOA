-- 000027_cron_tasks_enhanced_fields.up.sql
-- 为定时任务添加流程过滤、日期范围及执行追踪字段；补充 SMTP 密码及发件人配置

ALTER TABLE cron_tasks ADD COLUMN workflow_ids JSONB DEFAULT '[]';
ALTER TABLE cron_tasks ADD COLUMN date_range INT DEFAULT 30; -- 默认 30 天
ALTER TABLE cron_tasks ADD COLUMN current_log_id UUID; -- 指向当前正在运行的日志 ID

-- 补充 SMTP 系统设置
INSERT INTO system_configs (key, value, remark) VALUES 
    ('system.smtp_password', '', 'SMTP 认证密码/授权码（建议加密存储）'),
    ('system.smtp_sender', '', '邮件发件人显示地址')
ON CONFLICT (key) DO NOTHING;

COMMENT ON COLUMN cron_tasks.workflow_ids IS '筛选流程 ID 列表（多选）';
COMMENT ON COLUMN cron_tasks.date_range IS '筛选日期范围（天数，如 30/90/365）';
COMMENT ON COLUMN cron_tasks.current_log_id IS '当前正在运行的执行日志 ID';
