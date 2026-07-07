DROP INDEX IF EXISTS idx_tenants_embed_token_hash;

ALTER TABLE tenants
    DROP COLUMN IF EXISTS embed_token_rotated_at,
    DROP COLUMN IF EXISTS embed_token_hint,
    DROP COLUMN IF EXISTS embed_token_hash,
    DROP COLUMN IF EXISTS embed_enabled;
