-- fix_notification_body_chinese.sql
-- 将 user_notifications.body 中的英文枚举值替换为中文，并统一顺序为「建议/合规性在前，评分在后」。
-- 执行前建议先 SELECT 确认影响行数，再执行 UPDATE。
-- 幂等：已中文化的行不会被重复处理。

BEGIN;

-- 1. 审核通知：建议字段中文化
--    旧格式可能是「评分 N，建议：approve」或「建议：approve，评分 N」
UPDATE user_notifications
SET body = REGEXP_REPLACE(body, '建议：\s*approve', '建议：通过')
WHERE body ~ '建议：\s*approve';

UPDATE user_notifications
SET body = REGEXP_REPLACE(body, '建议：\s*return', '建议：退回')
WHERE body ~ '建议：\s*return';

UPDATE user_notifications
SET body = REGEXP_REPLACE(body, '建议：\s*review', '建议：人工复核')
WHERE body ~ '建议：\s*review';

-- 2. 归档复盘通知：合规性字段中文化
UPDATE user_notifications
SET body = REGEXP_REPLACE(body, '合规性：\s*partially_compliant', '合规性：部分合规')
WHERE body ~ '合规性：\s*partially_compliant';

UPDATE user_notifications
SET body = REGEXP_REPLACE(body, '合规性：\s*non_compliant', '合规性：不合规')
WHERE body ~ '合规性：\s*non_compliant';

UPDATE user_notifications
SET body = REGEXP_REPLACE(body, '合规性：\s*compliant', '合规性：合规')
WHERE body ~ '合规性：\s*compliant'
  AND body !~ '合规性：\s*(不合规|部分合规)';

-- 3. 统一顺序：「评分 N，建议：X」→「建议：X，评分 N」
UPDATE user_notifications
SET body = REGEXP_REPLACE(body, '评分\s+(\d+)[，,]\s*建议：\s*(.+)', '建议：\2，评分 \1')
WHERE body ~ '评分\s+\d+[，,]\s*建议：';

COMMIT;
