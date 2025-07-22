-- CI/CD服务数据库初始化脚本

-- 启用UUID扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 创建数据库用户（如果不存在）
DO
$do$
BEGIN
   IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'cicd_user') THEN
      CREATE ROLE cicd_user LOGIN PASSWORD 'cicd_password123';
   END IF;
END
$do$;

-- 授予权限
GRANT ALL PRIVILEGES ON DATABASE euclid_elements TO cicd_user;
GRANT ALL ON SCHEMA public TO cicd_user;

-- 创建项目表（如果不存在，用于外键引用）
CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) DEFAULT 'active',
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_projects_tenant_id ON projects(tenant_id);
CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status);
CREATE INDEX IF NOT EXISTS idx_projects_deleted_at ON projects(deleted_at);

-- 插入示例项目数据（用于测试）
INSERT INTO projects (id, tenant_id, name, description, status) 
VALUES 
    ('01234567-89ab-cdef-0123-456789abcdef', '11111111-2222-3333-4444-555555555555', '示例项目', 'CI/CD服务测试项目', 'active'),
    ('fedcba98-7654-3210-fedc-ba9876543210', '11111111-2222-3333-4444-555555555555', '演示项目', 'CI/CD演示用项目', 'active')
ON CONFLICT (id) DO NOTHING;

-- 创建函数：自动更新updated_at字段
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 为projects表创建触发器
DROP TRIGGER IF EXISTS update_projects_updated_at ON projects;
CREATE TRIGGER update_projects_updated_at
    BEFORE UPDATE ON projects
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 打印初始化完成信息
DO $$
BEGIN
    RAISE NOTICE '✅ CI/CD服务数据库初始化完成';
    RAISE NOTICE '📊 已创建基础表结构和示例数据';
    RAISE NOTICE '🔑 用户: cicd_user';
    RAISE NOTICE '🏗️ 应用将自动创建其他必要的表';
END $$;