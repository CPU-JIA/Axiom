-- =====================================================================================
-- 几何原本 (Euclid Elements) - 初始数据库Schema
-- 基于详细的数据库设计文档 V3.1 实现
-- 
-- 功能特性:
-- - 多租户逻辑隔离 + 行级安全策略(RLS)
-- - UUID v7主键标准
-- - 审计字段标准(created_at, updated_at)
-- - 软删除支持(deleted_at)
-- - UTC时间标准
-- - 部分唯一索引支持软删除场景
-- - 信封加密密钥管理
-- - 复合分区策略
-- =====================================================================================

-- 启用必要的扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =====================================================================================
-- 1. 自定义数据类型 (ENUM Definitions)
-- =====================================================================================

-- 租户状态
CREATE TYPE tenants_status_enum AS ENUM ('active', 'suspended', 'pending_deletion');

-- 任务优先级
CREATE TYPE tasks_priority_enum AS ENUM ('low', 'medium', 'high', 'urgent');

-- 任务状态分类
CREATE TYPE task_statuses_category_enum AS ENUM ('todo', 'in_progress', 'done');

-- 拉取请求状态
CREATE TYPE pull_requests_status_enum AS ENUM ('open', 'draft', 'merged', 'closed');

-- 认证提供方
CREATE TYPE auth_provider_enum AS ENUM ('local', 'google', 'github', 'saml');

-- CI/CD 执行器状态
CREATE TYPE runner_status_enum AS ENUM ('online', 'offline', 'disabled');

-- CI/CD 执行记录状态
CREATE TYPE pipeline_run_status_enum AS ENUM ('pending', 'running', 'success', 'failed', 'cancelled');

-- CI/CD 作业状态
CREATE TYPE job_status_enum AS ENUM ('pending', 'running', 'success', 'failed', 'cancelled');

-- 角色作用域
CREATE TYPE role_scope_enum AS ENUM ('tenant', 'project');

-- =====================================================================================
-- 2. 核心：租户、订阅与身份认证 (Tenant, Subscription & Identity)
-- =====================================================================================

-- 订阅套餐表
CREATE TABLE subscription_plans (
    id                UUID         PRIMARY KEY DEFAULT uuid_generate_v7(),
    name              VARCHAR(255) NOT NULL UNIQUE,
    description       TEXT,
    features          JSONB        NOT NULL DEFAULT '{}',
    price_monthly     DECIMAL(10, 2),
    price_annually    DECIMAL(10, 2),
    is_active         BOOLEAN      NOT NULL DEFAULT TRUE,
    display_order     INTEGER
);

COMMENT ON TABLE subscription_plans IS '订阅套餐定义表，包含功能限制和价格';
COMMENT ON COLUMN subscription_plans.features IS 'JSONB格式存储功能限制，如{"max_users": 5, "max_projects": 10, "ci_minutes": 1000}';

-- 租户表
CREATE TABLE tenants (
    id                    UUID                PRIMARY KEY DEFAULT uuid_generate_v7(),
    name                  VARCHAR(255)        NOT NULL,
    domain                VARCHAR(255)        UNIQUE,
    subscription_plan_id  UUID                REFERENCES subscription_plans(id),
    data_residency_region VARCHAR(50)         NOT NULL,
    status                tenants_status_enum NOT NULL DEFAULT 'active',
    created_at            TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ         NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE tenants IS '租户表，多租户系统的核心隔离单位';
COMMENT ON COLUMN tenants.domain IS '租户自定义子域名，可选';
COMMENT ON COLUMN tenants.data_residency_region IS '数据存储地理区域，满足数据主权要求';

-- 用户表
CREATE TABLE users (
    id            UUID          PRIMARY KEY DEFAULT uuid_generate_v7(),
    email         VARCHAR(255)  NOT NULL UNIQUE,
    full_name     VARCHAR(255),
    avatar_url    VARCHAR(1024),
    last_login_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE users IS '全局用户表，存储核心用户画像，与认证方式解耦';
COMMENT ON COLUMN users.email IS '主邮箱，全局唯一标识';

-- 用户认证表
CREATE TABLE user_authentications (
    id               UUID                NOT NULL PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id          UUID                NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider         auth_provider_enum  NOT NULL,
    provider_user_id VARCHAR(255),
    credentials      JSONB               NOT NULL DEFAULT '{}',
    created_at       TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    
    -- 确保一个第三方账号只关联一个平台用户
    CONSTRAINT unique_provider_user UNIQUE(provider, provider_user_id)
);

COMMENT ON TABLE user_authentications IS '用户多种登录凭证存储表';
COMMENT ON COLUMN user_authentications.credentials IS '存储凭证，如密码哈希或OAuth token';

-- 统一角色表
CREATE TABLE roles (
    id            UUID                NOT NULL PRIMARY KEY DEFAULT uuid_generate_v7(),
    scope         role_scope_enum     NOT NULL,
    tenant_id     UUID                REFERENCES tenants(id) ON DELETE CASCADE,
    name          VARCHAR(255)        NOT NULL,
    description   TEXT,
    is_predefined BOOLEAN             NOT NULL DEFAULT FALSE,
    permissions   JSONB               NOT NULL DEFAULT '[]',
    created_at    TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    
    -- 确保角色名在同一租户、同一作用域内唯一
    CONSTRAINT unique_role_per_tenant_scope UNIQUE(tenant_id, scope, name)
);

COMMENT ON TABLE roles IS '统一角色定义表，通过scope字段区分租户级和项目级角色';
COMMENT ON COLUMN roles.permissions IS 'JSONB数组存储权限列表，如["project:create", "task:delete"]';

-- 租户成员关系表
CREATE TABLE tenant_members (
    tenant_id UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id   UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id   UUID        NOT NULL REFERENCES roles(id),
    status    VARCHAR(20) NOT NULL DEFAULT 'active',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    PRIMARY KEY (tenant_id, user_id)
);

COMMENT ON TABLE tenant_members IS '用户在租户中的成员身份和租户级角色关系表';

-- =====================================================================================
-- 3. 项目与任务管理 (Project & Task)
-- =====================================================================================

-- 项目表
CREATE TABLE projects (
    id          UUID         NOT NULL PRIMARY KEY DEFAULT uuid_generate_v7(),
    tenant_id   UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    key         VARCHAR(10)  NOT NULL,
    description TEXT,
    manager_id  UUID         REFERENCES users(id),
    status      VARCHAR(20)  NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

COMMENT ON TABLE projects IS '项目表，支持软删除';
COMMENT ON COLUMN projects.key IS '项目键，如PROJ，在租户内的活跃项目中唯一';

-- 项目成员关系表
CREATE TABLE project_members (
    project_id UUID        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id    UUID        NOT NULL REFERENCES roles(id),
    added_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    added_by   UUID        REFERENCES users(id),
    
    PRIMARY KEY (project_id, user_id)
);

COMMENT ON TABLE project_members IS '用户在项目中的成员身份和项目级角色关系表';

-- 任务状态表
CREATE TABLE task_statuses (
    id            UUID                        NOT NULL PRIMARY KEY DEFAULT uuid_generate_v7(),
    tenant_id     UUID                        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name          VARCHAR(50)                 NOT NULL,
    category      task_statuses_category_enum NOT NULL,
    display_order INTEGER                     NOT NULL,
    
    CONSTRAINT unique_status_per_tenant UNIQUE(tenant_id, name)
);

COMMENT ON TABLE task_statuses IS '自定义任务状态表，支持工作流配置';

-- 任务表
CREATE TABLE tasks (
    id             UUID                NOT NULL PRIMARY KEY DEFAULT uuid_generate_v7(),
    project_id     UUID                NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    task_number    BIGINT              NOT NULL,
    title          VARCHAR(512)        NOT NULL,
    description    TEXT,
    status_id      UUID                REFERENCES task_statuses(id),
    assignee_id    UUID                REFERENCES users(id),
    creator_id     UUID                NOT NULL REFERENCES users(id),
    parent_task_id UUID                REFERENCES tasks(id),
    due_date       DATE,
    priority       tasks_priority_enum NOT NULL DEFAULT 'medium',
    created_at     TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_task_number_per_project UNIQUE(project_id, task_number)
);

COMMENT ON TABLE tasks IS '任务表，支持子任务和自定义状态';
COMMENT ON COLUMN tasks.task_number IS '项目内任务序号，通过SEQUENCE生成保证唯一性和连续性';

-- =====================================================================================
-- 4. 代码与CI/CD (Code & CI/CD)
-- =====================================================================================

-- 代码仓库表
CREATE TABLE repositories (
    id             UUID         NOT NULL PRIMARY KEY DEFAULT uuid_generate_v7(),
    project_id     UUID         NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name           VARCHAR(255) NOT NULL,
    description    TEXT,
    visibility     VARCHAR(20)  NOT NULL DEFAULT 'private',
    default_branch VARCHAR(255) NOT NULL DEFAULT 'main',
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_repo_per_project UNIQUE(project_id, name)
);

COMMENT ON TABLE repositories IS '代码仓库元数据表，实际Git数据由专用Git服务管理';

-- 拉取请求表
CREATE TABLE pull_requests (
    id            UUID                      NOT NULL PRIMARY KEY DEFAULT uuid_generate_v7(),
    repository_id UUID                      NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    pr_number     BIGINT                    NOT NULL,
    title         VARCHAR(512)              NOT NULL,
    description   TEXT,
    source_branch VARCHAR(255)              NOT NULL,
    target_branch VARCHAR(255)              NOT NULL,
    status        pull_requests_status_enum NOT NULL DEFAULT 'open',
    creator_id    UUID                      NOT NULL REFERENCES users(id),
    created_at    TIMESTAMPTZ               NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ               NOT NULL DEFAULT NOW(),
    merged_at     TIMESTAMPTZ,
    
    CONSTRAINT unique_pr_number_per_repo UNIQUE(repository_id, pr_number)
);

COMMENT ON TABLE pull_requests IS '拉取请求表，记录代码评审流程';

-- CI/CD 执行器表
CREATE TABLE runners (
    id              UUID                NOT NULL PRIMARY KEY DEFAULT uuid_generate_v7(),
    tenant_id       UUID                NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name            VARCHAR(255)        NOT NULL,
    tags            JSONB               DEFAULT '[]',
    status          runner_status_enum  NOT NULL DEFAULT 'offline',
    last_contact_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ         NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE runners IS 'CI/CD执行器表';
COMMENT ON COLUMN runners.tags IS 'JSON数组存储标签，用于作业匹配';

-- CI/CD 流水线表
CREATE TABLE pipelines (
    id                   UUID         NOT NULL PRIMARY KEY DEFAULT uuid_generate_v7(),
    repository_id        UUID         NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    name                 VARCHAR(255) NOT NULL,
    definition_file_path VARCHAR(512) NOT NULL,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE pipelines IS 'CI/CD流水线定义表';

-- CI/CD 流水线执行记录表
CREATE TABLE pipeline_runs (
    id           UUID                     NOT NULL PRIMARY KEY DEFAULT uuid_generate_v7(),
    pipeline_id  UUID                     NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
    trigger_type VARCHAR(50)              NOT NULL,
    trigger_by   UUID                     REFERENCES users(id),
    commit_sha   VARCHAR(40)              NOT NULL,
    branch       VARCHAR(255),
    status       pipeline_run_status_enum NOT NULL DEFAULT 'pending',
    started_at   TIMESTAMPTZ,
    finished_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ              NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE pipeline_runs IS 'CI/CD流水线执行记录表';

-- CI/CD 作业表
CREATE TABLE jobs (
    id              UUID            NOT NULL PRIMARY KEY DEFAULT uuid_generate_v7(),
    pipeline_run_id UUID            NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
    name            VARCHAR(255)    NOT NULL,
    status          job_status_enum NOT NULL DEFAULT 'pending',
    runner_id       UUID            REFERENCES runners(id),
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE jobs IS 'CI/CD作业表';

-- =====================================================================================
-- 5. 知识与协作 (Knowledge & Collaboration)
-- =====================================================================================

-- 文档表
CREATE TABLE documents (
    id         UUID         NOT NULL PRIMARY KEY DEFAULT uuid_generate_v7(),
    project_id UUID         NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    title      VARCHAR(512) NOT NULL,
    content    TEXT,
    creator_id UUID         NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE documents IS '项目文档表，内容使用Markdown格式';

-- 评论表 (多态关联)
CREATE TABLE comments (
    id                 UUID         NOT NULL PRIMARY KEY DEFAULT uuid_generate_v7(),
    tenant_id          UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    author_id          UUID         NOT NULL REFERENCES users(id),
    content            TEXT         NOT NULL,
    parent_entity_type VARCHAR(50)  NOT NULL,
    parent_entity_id   UUID         NOT NULL,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE comments IS '多态评论表，支持对任务、PR、文档等实体的评论';
COMMENT ON COLUMN comments.parent_entity_type IS '被评论实体类型：task, pull_request, document等';

-- =====================================================================================
-- 6. 系统与审计 (System & Auditing)
-- =====================================================================================

-- 系统配置表
CREATE TABLE system_settings (
    key         VARCHAR(255) NOT NULL PRIMARY KEY,
    value       JSONB        NOT NULL,
    description TEXT,
    is_public   BOOLEAN      NOT NULL DEFAULT FALSE,
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE system_settings IS '系统级配置表，存储功能开关和默认设置';
COMMENT ON COLUMN system_settings.is_public IS '是否可被前端无鉴权读取，如平台名称、Logo等';

-- 通知表
CREATE TABLE notifications (
    id           UUID          NOT NULL PRIMARY KEY DEFAULT uuid_generate_v7(),
    recipient_id UUID          NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id    UUID          NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    message      TEXT          NOT NULL,
    link         VARCHAR(1024),
    is_read      BOOLEAN       NOT NULL DEFAULT FALSE,
    read_at      TIMESTAMPTZ,
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE notifications IS '用户通知表，设计为高性能写入，适合分区';

-- 密钥存储表 (信封加密)
CREATE TABLE secrets (
    id            UUID         NOT NULL PRIMARY KEY DEFAULT uuid_generate_v7(),
    owner_type    VARCHAR(50)  NOT NULL,
    owner_id      UUID         NOT NULL,
    name          VARCHAR(255) NOT NULL,
    value_encrypted BYTEA      NOT NULL,
    kek_ref       VARCHAR(512) NOT NULL,
    dek_encrypted BYTEA        NOT NULL,
    
    CONSTRAINT unique_secret_per_owner UNIQUE(owner_type, owner_id, name)
);

COMMENT ON TABLE secrets IS 'CI/CD等场景的加密敏感信息存储表，采用信封加密';
COMMENT ON COLUMN secrets.kek_ref IS '主密钥(KEK)的引用ID，由KMS管理';
COMMENT ON COLUMN secrets.dek_encrypted IS '被KEK加密后的数据加密密钥(DEK)';

-- 审计日志表 (分区表)
CREATE TABLE audit_logs (
    id                 BIGSERIAL    NOT NULL,
    tenant_id          UUID         NOT NULL REFERENCES tenants(id),
    user_id            UUID         REFERENCES users(id),
    impersonator_id    UUID         REFERENCES users(id),
    action             VARCHAR(100) NOT NULL,
    target_entity_type VARCHAR(50),
    target_entity_id   UUID,
    details            JSONB,
    client_ip          INET,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW()
) PARTITION BY RANGE (created_at);

COMMENT ON TABLE audit_logs IS '审计日志表，记录所有关键操作，按时间分区';
COMMENT ON COLUMN audit_logs.impersonator_id IS '模拟登录者ID，支持管理员代理操作审计';

-- 创建审计日志分区 (按月分区)
CREATE TABLE audit_logs_y2024m07 PARTITION OF audit_logs
    FOR VALUES FROM ('2024-07-01') TO ('2024-08-01');
CREATE TABLE audit_logs_y2024m08 PARTITION OF audit_logs
    FOR VALUES FROM ('2024-08-01') TO ('2024-09-01');
CREATE TABLE audit_logs_y2024m09 PARTITION OF audit_logs
    FOR VALUES FROM ('2024-09-01') TO ('2024-10-01');
CREATE TABLE audit_logs_y2024m10 PARTITION OF audit_logs
    FOR VALUES FROM ('2024-10-01') TO ('2024-11-01');
CREATE TABLE audit_logs_y2024m11 PARTITION OF audit_logs
    FOR VALUES FROM ('2024-11-01') TO ('2024-12-01');
CREATE TABLE audit_logs_y2024m12 PARTITION OF audit_logs
    FOR VALUES FROM ('2024-12-01') TO ('2025-01-01');

-- =====================================================================================
-- 7. 索引策略 (Indexing Strategy)
-- =====================================================================================

-- 部分唯一索引：支持软删除的项目键唯一性
CREATE UNIQUE INDEX idx_unique_active_project_key 
ON projects (tenant_id, key) 
WHERE (deleted_at IS NULL);
COMMENT ON INDEX idx_unique_active_project_key IS '确保活跃项目的键在租户内唯一，支持软删除';

-- 高频查询优化索引
CREATE INDEX idx_tasks_project_status ON tasks (project_id, status_id);
COMMENT ON INDEX idx_tasks_project_status IS '支持项目看板页按状态快速过滤任务';

CREATE INDEX idx_tasks_assignee ON tasks (assignee_id) WHERE assignee_id IS NOT NULL;
COMMENT ON INDEX idx_tasks_assignee IS '支持个人任务列表查询';

CREATE INDEX idx_comments_parent ON comments (parent_entity_type, parent_entity_id);
COMMENT ON INDEX idx_comments_parent IS '支持快速查询特定实体的评论';

CREATE INDEX idx_pull_requests_status ON pull_requests (repository_id, status);
COMMENT ON INDEX idx_pull_requests_status IS '支持仓库PR列表按状态过滤';

CREATE INDEX idx_notifications_recipient ON notifications (recipient_id, is_read, created_at DESC);
COMMENT ON INDEX idx_notifications_recipient IS '支持用户通知列表查询和未读统计';

CREATE INDEX idx_audit_logs_tenant_time ON audit_logs (tenant_id, created_at DESC);
COMMENT ON INDEX idx_audit_logs_tenant_time IS '支持租户审计日志查询';

CREATE INDEX idx_audit_logs_user ON audit_logs (user_id, created_at DESC) WHERE user_id IS NOT NULL;
COMMENT ON INDEX idx_audit_logs_user IS '支持用户操作历史查询';

-- JSONB字段GIN索引
CREATE INDEX idx_roles_permissions ON roles USING GIN (permissions);
COMMENT ON INDEX idx_roles_permissions IS '支持基于权限的角色查询';

CREATE INDEX idx_subscription_plans_features ON subscription_plans USING GIN (features);
COMMENT ON INDEX idx_subscription_plans_features IS '支持基于功能特性的套餐查询';

-- =====================================================================================
-- 8. 行级安全策略 (Row-Level Security - RLS)
-- =====================================================================================

-- 为包含tenant_id的核心表启用RLS
ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;
ALTER TABLE projects ENABLE ROW LEVEL SECURITY;
ALTER TABLE tasks ENABLE ROW LEVEL SECURITY;
ALTER TABLE repositories ENABLE ROW LEVEL SECURITY;
ALTER TABLE pull_requests ENABLE ROW LEVEL SECURITY;
ALTER TABLE documents ENABLE ROW LEVEL SECURITY;
ALTER TABLE comments ENABLE ROW LEVEL SECURITY;
ALTER TABLE task_statuses ENABLE ROW LEVEL SECURITY;
ALTER TABLE runners ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;

-- 租户隔离策略
CREATE POLICY tenant_isolation_policy ON tenants
FOR ALL
USING (id = current_setting('app.current_tenant_id', TRUE)::uuid);

CREATE POLICY tenant_isolation_policy ON projects
FOR ALL
USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::uuid);

CREATE POLICY tenant_isolation_policy ON tasks
FOR ALL
USING (project_id IN (
    SELECT id FROM projects 
    WHERE tenant_id = current_setting('app.current_tenant_id', TRUE)::uuid
));

CREATE POLICY tenant_isolation_policy ON repositories
FOR ALL
USING (project_id IN (
    SELECT id FROM projects 
    WHERE tenant_id = current_setting('app.current_tenant_id', TRUE)::uuid
));

CREATE POLICY tenant_isolation_policy ON pull_requests
FOR ALL
USING (repository_id IN (
    SELECT r.id FROM repositories r
    JOIN projects p ON r.project_id = p.id
    WHERE p.tenant_id = current_setting('app.current_tenant_id', TRUE)::uuid
));

CREATE POLICY tenant_isolation_policy ON documents
FOR ALL
USING (project_id IN (
    SELECT id FROM projects 
    WHERE tenant_id = current_setting('app.current_tenant_id', TRUE)::uuid
));

CREATE POLICY tenant_isolation_policy ON comments
FOR ALL
USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::uuid);

CREATE POLICY tenant_isolation_policy ON task_statuses
FOR ALL
USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::uuid);

CREATE POLICY tenant_isolation_policy ON runners
FOR ALL
USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::uuid);

CREATE POLICY tenant_isolation_policy ON notifications
FOR ALL
USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::uuid);

CREATE POLICY tenant_isolation_policy ON audit_logs
FOR ALL
USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::uuid);

-- =====================================================================================
-- 9. 触发器与函数 (Triggers & Functions)
-- =====================================================================================

-- 自动更新updated_at时间戳的函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 为需要updated_at自动更新的表创建触发器
CREATE TRIGGER update_tenants_updated_at BEFORE UPDATE ON tenants FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();
CREATE TRIGGER update_roles_updated_at BEFORE UPDATE ON roles FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();
CREATE TRIGGER update_projects_updated_at BEFORE UPDATE ON projects FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();
CREATE TRIGGER update_tasks_updated_at BEFORE UPDATE ON tasks FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();
CREATE TRIGGER update_repositories_updated_at BEFORE UPDATE ON repositories FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();
CREATE TRIGGER update_pull_requests_updated_at BEFORE UPDATE ON pull_requests FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();
CREATE TRIGGER update_documents_updated_at BEFORE UPDATE ON documents FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();
CREATE TRIGGER update_comments_updated_at BEFORE UPDATE ON comments FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();
CREATE TRIGGER update_pipelines_updated_at BEFORE UPDATE ON pipelines FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();
CREATE TRIGGER update_system_settings_updated_at BEFORE UPDATE ON system_settings FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();

-- 角色作用域验证函数
CREATE OR REPLACE FUNCTION validate_role_scope() RETURNS TRIGGER AS $$
BEGIN
    -- 验证tenant_members中的role_id必须是tenant scope
    IF TG_TABLE_NAME = 'tenant_members' THEN
        IF NOT EXISTS (
            SELECT 1 FROM roles 
            WHERE id = NEW.role_id AND scope = 'tenant'
        ) THEN
            RAISE EXCEPTION 'Role must have tenant scope for tenant membership';
        END IF;
    END IF;
    
    -- 验证project_members中的role_id必须是project scope
    IF TG_TABLE_NAME = 'project_members' THEN
        IF NOT EXISTS (
            SELECT 1 FROM roles 
            WHERE id = NEW.role_id AND scope = 'project'
        ) THEN
            RAISE EXCEPTION 'Role must have project scope for project membership';
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 创建角色作用域验证触发器
CREATE TRIGGER validate_tenant_member_role_scope
    BEFORE INSERT OR UPDATE ON tenant_members
    FOR EACH ROW EXECUTE FUNCTION validate_role_scope();

CREATE TRIGGER validate_project_member_role_scope
    BEFORE INSERT OR UPDATE ON project_members
    FOR EACH ROW EXECUTE FUNCTION validate_role_scope();

-- 任务序号生成函数
CREATE OR REPLACE FUNCTION get_next_task_number(p_project_id UUID) RETURNS BIGINT AS $$
DECLARE
    sequence_name TEXT;
    next_number BIGINT;
BEGIN
    sequence_name := 'task_number_seq_for_project_' || replace(p_project_id::text, '-', '_');
    
    -- 如果序列不存在则创建
    IF NOT EXISTS (SELECT 1 FROM pg_sequences WHERE schemaname = 'public' AND sequencename = sequence_name) THEN
        EXECUTE format('CREATE SEQUENCE %I', sequence_name);
    END IF;
    
    -- 获取下一个序号
    EXECUTE format('SELECT nextval(%L)', sequence_name) INTO next_number;
    
    RETURN next_number;
END;
$$ LANGUAGE plpgsql;

-- 自动为新任务分配序号的触发器
CREATE OR REPLACE FUNCTION assign_task_number() RETURNS TRIGGER AS $$
BEGIN
    NEW.task_number := get_next_task_number(NEW.project_id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER assign_task_number_trigger
    BEFORE INSERT ON tasks
    FOR EACH ROW EXECUTE FUNCTION assign_task_number();

-- PR序号生成函数（类似任务序号）
CREATE OR REPLACE FUNCTION get_next_pr_number(p_repository_id UUID) RETURNS BIGINT AS $$
DECLARE
    sequence_name TEXT;
    next_number BIGINT;
BEGIN
    sequence_name := 'pr_number_seq_for_repo_' || replace(p_repository_id::text, '-', '_');
    
    -- 如果序列不存在则创建
    IF NOT EXISTS (SELECT 1 FROM pg_sequences WHERE schemaname = 'public' AND sequencename = sequence_name) THEN
        EXECUTE format('CREATE SEQUENCE %I', sequence_name);
    END IF;
    
    -- 获取下一个序号
    EXECUTE format('SELECT nextval(%L)', sequence_name) INTO next_number;
    
    RETURN next_number;
END;
$$ LANGUAGE plpgsql;

-- 自动为新PR分配序号的触发器
CREATE OR REPLACE FUNCTION assign_pr_number() RETURNS TRIGGER AS $$
BEGIN
    NEW.pr_number := get_next_pr_number(NEW.repository_id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER assign_pr_number_trigger
    BEFORE INSERT ON pull_requests
    FOR EACH ROW EXECUTE FUNCTION assign_pr_number();

-- =====================================================================================
-- 10. 初始数据 (Initial Data)
-- =====================================================================================

-- 创建默认订阅套餐
INSERT INTO subscription_plans (name, description, features, price_monthly, price_annually, display_order) VALUES
('Free', '免费套餐，适合个人开发者', '{"max_users": 5, "max_projects": 3, "ci_minutes": 500, "storage_gb": 1}', 0.00, 0.00, 1),
('Pro', '专业套餐，适合小团队', '{"max_users": 25, "max_projects": 20, "ci_minutes": 3000, "storage_gb": 10}', 29.99, 299.99, 2),
('Enterprise', '企业套餐，无限制使用', '{"max_users": -1, "max_projects": -1, "ci_minutes": -1, "storage_gb": 100}', 99.99, 999.99, 3);

-- 创建预定义角色 (平台级)
INSERT INTO roles (scope, tenant_id, name, description, is_predefined, permissions) VALUES
-- 租户级角色
('tenant', NULL, 'Owner', '租户所有者，拥有所有权限', TRUE, '["tenant:*", "project:*", "user:*"]'),
('tenant', NULL, 'Admin', '租户管理员，拥有管理权限', TRUE, '["tenant:manage", "project:*", "user:invite"]'),
('tenant', NULL, 'Member', '租户普通成员', TRUE, '["project:view", "task:view"]'),

-- 项目级角色
('project', NULL, 'Maintainer', '项目维护者，拥有项目管理权限', TRUE, '["project:manage", "repository:*", "pipeline:*"]'),
('project', NULL, 'Developer', '开发者，可提交代码和创建PR', TRUE, '["repository:write", "task:*", "pipeline:view"]'),
('project', NULL, 'Viewer', '观察者，只读权限', TRUE, '["project:view", "task:view", "repository:read"]');

-- 创建系统配置
INSERT INTO system_settings (key, value, description, is_public) VALUES
('platform.name', '"几何原本 (Euclid Elements)"', '平台名称', TRUE),
('platform.version', '"1.0.0"', '平台版本', TRUE),
('feature.ai.enabled', 'true', '是否启用AI功能', FALSE),
('defaults.tenant.trial_days', '30', '新租户试用天数', FALSE),
('security.session.timeout', '7200', '会话超时时间(秒)', FALSE),
('notification.email.enabled', 'true', '是否启用邮件通知', FALSE);

-- =====================================================================================
-- Schema创建完成
-- 版本: V1.0
-- 创建时间: 2024-07-22
-- 基于: 详细的数据库设计文档 V3.1
-- =====================================================================================