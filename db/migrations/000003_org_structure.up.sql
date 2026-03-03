-- 000003_org_structure.up.sql
-- Create departments, org_roles, org_members, and org_member_roles tables

-- ============================================================
-- departments
-- ============================================================
CREATE TABLE departments (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID         NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    name       VARCHAR(200) NOT NULL,
    parent_id  UUID         REFERENCES departments (id) ON DELETE SET NULL,
    manager    VARCHAR(100),
    sort_order INT          NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_departments_tenant_id ON departments (tenant_id);
CREATE INDEX idx_departments_parent_id ON departments (parent_id);

-- ============================================================
-- org_roles
-- ============================================================
CREATE TABLE org_roles (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID         NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    name             VARCHAR(100) NOT NULL,
    description      TEXT,
    page_permissions JSONB        NOT NULL DEFAULT '[]',
    is_system        BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_org_roles_tenant_id ON org_roles (tenant_id);

-- ============================================================
-- org_members
-- ============================================================
CREATE TABLE org_members (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID        NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    user_id       UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    department_id UUID        NOT NULL REFERENCES departments (id) ON DELETE RESTRICT,
    position      VARCHAR(100),
    status        VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_org_members_tenant_id     ON org_members (tenant_id);
CREATE INDEX idx_org_members_user_id       ON org_members (user_id);
CREATE INDEX idx_org_members_department_id ON org_members (department_id);
CREATE UNIQUE INDEX idx_org_members_tenant_user ON org_members (tenant_id, user_id);

-- ============================================================
-- org_member_roles (join table: org_members <-> org_roles)
-- ============================================================
CREATE TABLE org_member_roles (
    org_member_id UUID NOT NULL REFERENCES org_members (id) ON DELETE CASCADE,
    org_role_id   UUID NOT NULL REFERENCES org_roles (id) ON DELETE CASCADE,
    PRIMARY KEY (org_member_id, org_role_id)
);

CREATE INDEX idx_org_member_roles_org_member_id ON org_member_roles (org_member_id);
CREATE INDEX idx_org_member_roles_org_role_id   ON org_member_roles (org_role_id);
