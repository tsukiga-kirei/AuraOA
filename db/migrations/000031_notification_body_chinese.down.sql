-- 000031_notification_body_chinese.down.sql
-- 回滚：将中文枚举值还原为英文（尽力而为，无法完美还原顺序）

BEGIN;

-- 建议字段还原
UPDATE user_notifications SET body = REGEXP_REPLACE(body, '建议：\s*通过', '建议：approve') WHERE body ~ '建议：\s*通过';
UPDATE user_notifications SET body = REGEXP_REPLACE(body, '建议：\s*退回', '建议：return') WHERE body ~ '建议：\s*退回';
UPDATE user_notifications SET body = REGEXP_REPLACE(body, '建议：\s*人工复核', '建议：review') WHERE body ~ '建议：\s*人工复核';

-- 合规性字段还原
UPDATE user_notifications SET body = REGEXP_REPLACE(body, '合规性：\s*部分合规', '合规性：partially_compliant') WHERE body ~ '合规性：\s*部分合规';
UPDATE user_notifications SET body = REGEXP_REPLACE(body, '合规性：\s*不合规', '合规性：non_compliant') WHERE body ~ '合规性：\s*不合规';
UPDATE user_notifications SET body = REGEXP_REPLACE(body, '合规性：\s*合规', '合规性：compliant') WHERE body ~ '合规性：\s*合规' AND body !~ '合规性：\s*(non_compliant|partially_compliant)';

COMMIT;
