-- 租户级 OA 嵌入配置：每租户独立密钥，运行时通过 token 反查租户
ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS embed_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS embed_token_hash VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS embed_token_hint VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS embed_token_rotated_at TIMESTAMPTZ;

CREATE UNIQUE INDEX IF NOT EXISTS idx_tenants_embed_token_hash
    ON tenants (embed_token_hash)
    WHERE embed_token_hash <> '';
