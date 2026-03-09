-- 000010_user_personal_configs.down.sql
-- 回滚：删除用户仪表板偏好表、用户个人配置表

DROP TABLE IF EXISTS user_dashboard_prefs;
DROP TABLE IF EXISTS user_personal_configs;
