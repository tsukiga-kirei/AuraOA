-- 000003_org_structure.up.sql
-- 创建部门表、组织角色表、组织成员表、成员角色关联表

-- ============================================================
-- departments — 部门树形结构表
-- ============================================================
CREATE TABLE departments (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),                    -- 主键UUID
    tenant_id  UUID         NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,      -- 所属租户ID
    name       VARCHAR(200) NOT NULL,                                                 -- 部门名称
    parent_id  UUID         REFERENCES departments (id) ON DELETE SET NULL,          -- 父级部门ID（NULL表示顶级部门）
    manager    VARCHAR(100),                                                          -- 部门负责人姓名
    sort_order INT          NOT NULL DEFAULT 0,                                       -- 同级部门排序权重（越小越靠前）
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),                                   -- 创建时间
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now()                                    -- 最后更新时间
);

CREATE INDEX idx_departments_tenant_id ON departments (tenant_id);
CREATE INDEX idx_departments_parent_id ON departments (parent_id);

-- ============================================================
-- org_roles — 租户内组织角色表
-- ============================================================
CREATE TABLE org_roles (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),              -- 主键UUID
    tenant_id        UUID         NOT NULL REFERENCES tenants (id) ON DELETE CASCADE, -- 所属租户ID
    name             VARCHAR(100) NOT NULL,                                            -- 角色名称
    description      TEXT,                                                             -- 角色描述
    page_permissions JSONB        NOT NULL DEFAULT '[]',                               -- 页面权限列表（JSON数组，存储可访问的页面路由）
    is_system        BOOLEAN      NOT NULL DEFAULT FALSE,                              -- 是否为系统内置角色（内置角色不可删除）
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),                              -- 创建时间
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now()                               -- 最后更新时间
);

CREATE INDEX idx_org_roles_tenant_id ON org_roles (tenant_id);

-- ============================================================
-- org_members — 用户与部门归属关系表
-- ============================================================
CREATE TABLE org_members (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),                     -- 主键UUID
    tenant_id     UUID        NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,       -- 所属租户ID
    user_id       UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,         -- 关联用户ID
    department_id UUID        NOT NULL REFERENCES departments (id) ON DELETE RESTRICT,  -- 所属部门ID（删除部门前须先迁移成员）
    position      VARCHAR(100),                                                          -- 职位名称
    status        VARCHAR(20) NOT NULL DEFAULT 'active',                                -- 成员状态：active/inactive
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),                                    -- 创建时间
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()                                     -- 最后更新时间
);

CREATE INDEX idx_org_members_tenant_id     ON org_members (tenant_id);
CREATE INDEX idx_org_members_user_id       ON org_members (user_id);
CREATE INDEX idx_org_members_department_id ON org_members (department_id);
CREATE UNIQUE INDEX idx_org_members_tenant_user ON org_members (tenant_id, user_id); -- 同一租户内用户唯一

-- ============================================================
-- org_member_roles — 组织成员与角色多对多关联表
-- ============================================================
CREATE TABLE org_member_roles (
    org_member_id UUID NOT NULL REFERENCES org_members (id) ON DELETE CASCADE, -- 组织成员ID
    org_role_id   UUID NOT NULL REFERENCES org_roles (id) ON DELETE CASCADE,   -- 组织角色ID
    PRIMARY KEY (org_member_id, org_role_id)
);

CREATE INDEX idx_org_member_roles_org_member_id ON org_member_roles (org_member_id);
CREATE INDEX idx_org_member_roles_org_role_id   ON org_member_roles (org_role_id);

-- ============================================================
-- 数据库注释（中文）
-- ============================================================
COMMENT ON TABLE departments IS '部门树形结构表';
COMMENT ON COLUMN departments.id IS '主键UUID';
COMMENT ON COLUMN departments.tenant_id IS '所属租户ID';
COMMENT ON COLUMN departments.name IS '部门名称';
COMMENT ON COLUMN departments.parent_id IS '父级部门ID（NULL表示顶级部门）';
COMMENT ON COLUMN departments.manager IS '部门负责人姓名';
COMMENT ON COLUMN departments.sort_order IS '同级部门排序权重（越小越靠前）';
COMMENT ON COLUMN departments.created_at IS '创建时间';
COMMENT ON COLUMN departments.updated_at IS '最后更新时间';

COMMENT ON TABLE org_roles IS '租户内组织角色表';
COMMENT ON COLUMN org_roles.id IS '主键UUID';
COMMENT ON COLUMN org_roles.tenant_id IS '所属租户ID';
COMMENT ON COLUMN org_roles.name IS '角色名称';
COMMENT ON COLUMN org_roles.description IS '角色描述';
COMMENT ON COLUMN org_roles.page_permissions IS '页面权限列表（JSON数组，存储可访问的页面路由）';
COMMENT ON COLUMN org_roles.is_system IS '是否为系统内置角色（内置角色不可删除）';
COMMENT ON COLUMN org_roles.created_at IS '创建时间';
COMMENT ON COLUMN org_roles.updated_at IS '最后更新时间';

COMMENT ON TABLE org_members IS '用户与部门归属关系表';
COMMENT ON COLUMN org_members.id IS '主键UUID';
COMMENT ON COLUMN org_members.tenant_id IS '所属租户ID';
COMMENT ON COLUMN org_members.user_id IS '关联用户ID';
COMMENT ON COLUMN org_members.department_id IS '所属部门ID（删除部门前须先迁移成员）';
COMMENT ON COLUMN org_members.position IS '职位名称';
COMMENT ON COLUMN org_members.status IS '成员状态：active/inactive';
COMMENT ON COLUMN org_members.created_at IS '创建时间';
COMMENT ON COLUMN org_members.updated_at IS '最后更新时间';

COMMENT ON TABLE org_member_roles IS '组织成员与角色多对多关联表';
COMMENT ON COLUMN org_member_roles.org_member_id IS '组织成员ID';
COMMENT ON COLUMN org_member_roles.org_role_id IS '组织角色ID';
