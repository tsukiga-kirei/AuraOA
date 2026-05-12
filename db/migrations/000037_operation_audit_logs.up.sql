CREATE TABLE IF NOT EXISTS operation_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    tenant_id UUID NULL,
    method VARCHAR(16) NOT NULL,
    path TEXT NOT NULL,
    status_code INT NOT NULL,
    latency_ms INT NOT NULL,
    client_ip VARCHAR(64) DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_operation_audit_logs_created_at ON operation_audit_logs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_operation_audit_logs_user_id ON operation_audit_logs (user_id);
CREATE INDEX IF NOT EXISTS idx_operation_audit_logs_tenant_id ON operation_audit_logs (tenant_id);

COMMENT ON TABLE operation_audit_logs IS '用户 HTTP 操作审计，受 system.enable_audit_trail 开关控制写入';

COMMENT ON COLUMN operation_audit_logs.id IS '主键 UUID';
COMMENT ON COLUMN operation_audit_logs.user_id IS '执行操作的用户 ID（JWT sub）';
COMMENT ON COLUMN operation_audit_logs.tenant_id IS '租户上下文中的租户 ID，系统管理员未带 tenant_id 查询参数时可为空';
COMMENT ON COLUMN operation_audit_logs.method IS 'HTTP 方法（GET/POST/PUT/DELETE 等）';
COMMENT ON COLUMN operation_audit_logs.path IS '请求 URL 路径（不含 query string）';
COMMENT ON COLUMN operation_audit_logs.status_code IS 'HTTP 响应状态码';
COMMENT ON COLUMN operation_audit_logs.latency_ms IS '请求处理耗时（毫秒，自中间件进入至 c.Next 返回）';
COMMENT ON COLUMN operation_audit_logs.client_ip IS '客户端 IP（经反向代理时依赖 Gin Forwarded 配置）';
COMMENT ON COLUMN operation_audit_logs.created_at IS '审计记录写入时间';
