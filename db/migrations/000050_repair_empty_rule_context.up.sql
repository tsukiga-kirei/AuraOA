-- 修复历史导入规则把“AI 建议外部关联”误写成“已启用外部关联”的不一致数据。
-- 只有至少存在一个 enabled=true 的挂载时，context_enabled 才能保持为 true。
UPDATE audit_rules AS r
SET context_enabled = false,
    updated_at = NOW()
WHERE r.context_enabled = true
  AND NOT EXISTS (
      SELECT 1
      FROM jsonb_array_elements(COALESCE(r.context_mounts, '[]'::jsonb)) AS mount
      WHERE COALESCE((mount ->> 'enabled')::boolean, false) = true
  );

UPDATE archive_rules AS r
SET context_enabled = false,
    updated_at = NOW()
WHERE r.context_enabled = true
  AND NOT EXISTS (
      SELECT 1
      FROM jsonb_array_elements(COALESCE(r.context_mounts, '[]'::jsonb)) AS mount
      WHERE COALESCE((mount ->> 'enabled')::boolean, false) = true
  );
