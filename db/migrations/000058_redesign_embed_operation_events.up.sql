-- 000058：重构 OA 保存/提交事件协议，并持久化首次新建流程的 requestid 解析状态

CREATE TABLE IF NOT EXISTS embed_refresh_events (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_id            VARCHAR(120) NOT NULL,
    action              VARCHAR(30) NOT NULL,
    process_id          VARCHAR(100) NOT NULL DEFAULT '',
    workflow_id         VARCHAR(100) NOT NULL DEFAULT '',
    oa_belong_user_id   VARCHAR(100) NOT NULL DEFAULT '',
    oa_current_user_id  VARCHAR(100) NOT NULL DEFAULT '',
    baseline_request_id BIGINT NOT NULL DEFAULT 0,
    status              VARCHAR(20) NOT NULL DEFAULT 'pending',
    attempt             INTEGER NOT NULL DEFAULT 0,
    next_attempt_at     TIMESTAMPTZ,
    last_error          TEXT NOT NULL DEFAULT '',
    received_at         TIMESTAMPTZ NOT NULL,
    resolved_at         TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_ere_tenant_event UNIQUE (tenant_id, event_id),
    CONSTRAINT chk_ere_action CHECK (action IN ('save_requested', 'submit_requested')),
    CONSTRAINT chk_ere_status CHECK (status IN ('pending', 'scheduled', 'expired', 'ambiguous', 'failed'))
);

CREATE INDEX IF NOT EXISTS idx_ere_pending
    ON embed_refresh_events(status, next_attempt_at);
CREATE INDEX IF NOT EXISTS idx_ere_tenant_process
    ON embed_refresh_events(tenant_id, process_id, received_at DESC);

COMMENT ON TABLE embed_refresh_events IS
    'OA 保存/提交事件及首次新建流程 requestid 解析记录';
COMMENT ON COLUMN embed_refresh_events.action IS
    '事件动作：save_requested=点击保存，submit_requested=点击提交';
COMMENT ON COLUMN embed_refresh_events.baseline_request_id IS
    'OA 操作放行前读取的 workflow_requestbase.requestid 高水位';

UPDATE audit_logs
SET trigger_detail = 'legacy_operation'
WHERE trigger_detail IN ('save', 'submit', 'save_or_submit', 'save_complete');

UPDATE process_summary_logs
SET trigger_detail = 'legacy_operation'
WHERE trigger_detail IN ('save', 'submit', 'save_or_submit', 'save_complete');

COMMENT ON COLUMN audit_logs.trigger_detail IS
    '嵌入审核详细触发来源：manual、visible_open、save_requested、submit_requested、scheduled_scan、legacy_operation、legacy_auto';
COMMENT ON COLUMN process_summary_logs.trigger_detail IS
    '详细触发来源：manual、visible_open、save_requested、submit_requested、scheduled_scan、legacy_operation、legacy';
