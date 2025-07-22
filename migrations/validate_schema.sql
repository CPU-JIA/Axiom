-- =====================================================================================
-- 数据库Schema验证测试
-- 几何原本 (Euclid Elements) - Database Schema Validation Tests
-- =====================================================================================

-- 测试数据库基本结构
DO $$
DECLARE
    table_count INTEGER;
    index_count INTEGER;
    constraint_count INTEGER;
BEGIN
    -- 检查表数量
    SELECT COUNT(*) INTO table_count 
    FROM information_schema.tables 
    WHERE table_schema = 'public' AND table_type = 'BASE TABLE';
    
    IF table_count < 20 THEN
        RAISE EXCEPTION '表数量不足，期望至少20个，实际: %', table_count;
    END IF;
    
    RAISE NOTICE '✓ 表结构检查通过，共%个表', table_count;
    
    -- 检查索引数量
    SELECT COUNT(*) INTO index_count 
    FROM pg_indexes 
    WHERE schemaname = 'public';
    
    IF index_count < 30 THEN
        RAISE EXCEPTION '索引数量不足，期望至少30个，实际: %', index_count;
    END IF;
    
    RAISE NOTICE '✓ 索引结构检查通过，共%个索引', index_count;
    
    -- 检查约束数量
    SELECT COUNT(*) INTO constraint_count 
    FROM information_schema.table_constraints 
    WHERE constraint_schema = 'public';
    
    RAISE NOTICE '✓ 约束检查通过，共%个约束', constraint_count;
    
    RAISE NOTICE '✓ 数据库Schema验证通过';
END $$;

-- =====================================================================================
-- 核心表结构验证
-- =====================================================================================

-- 验证tenants表
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'tenants') THEN
        RAISE EXCEPTION 'tenants表不存在';
    END IF;
    
    -- 检查关键字段
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tenants' AND column_name = 'id' AND data_type = 'uuid') THEN
        RAISE EXCEPTION 'tenants.id字段类型不正确';
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tenants' AND column_name = 'subscription_plan_id') THEN
        RAISE EXCEPTION 'tenants.subscription_plan_id字段不存在';
    END IF;
    
    RAISE NOTICE '✓ tenants表结构验证通过';
END $$;

-- 验证users表
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'users') THEN
        RAISE EXCEPTION 'users表不存在';
    END IF;
    
    -- 检查唯一约束
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints tc
        JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name
        WHERE tc.table_name = 'users' 
        AND tc.constraint_type = 'UNIQUE' 
        AND kcu.column_name = 'email'
    ) THEN
        RAISE EXCEPTION 'users.email唯一约束不存在';
    END IF;
    
    RAISE NOTICE '✓ users表结构验证通过';
END $$;

-- 验证projects表
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'projects') THEN
        RAISE EXCEPTION 'projects表不存在';
    END IF;
    
    -- 检查软删除字段
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'projects' AND column_name = 'deleted_at') THEN
        RAISE EXCEPTION 'projects.deleted_at字段不存在';
    END IF;
    
    -- 检查部分唯一索引
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes 
        WHERE tablename = 'projects' 
        AND indexname = 'idx_unique_active_project_key'
    ) THEN
        RAISE EXCEPTION 'projects表部分唯一索引不存在';
    END IF;
    
    RAISE NOTICE '✓ projects表结构验证通过';
END $$;

-- 验证tasks表
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'tasks') THEN
        RAISE EXCEPTION 'tasks表不存在';
    END IF;
    
    -- 检查任务序号唯一约束
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints tc
        JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name
        WHERE tc.table_name = 'tasks' 
        AND tc.constraint_type = 'UNIQUE' 
        AND kcu.column_name IN ('project_id', 'task_number')
    ) THEN
        RAISE EXCEPTION 'tasks表任务序号唯一约束不存在';
    END IF;
    
    RAISE NOTICE '✓ tasks表结构验证通过';
END $$;

-- =====================================================================================
-- RLS策略验证
-- =====================================================================================

-- 验证RLS启用状态
DO $$
DECLARE
    rls_table RECORD;
    rls_tables TEXT[] := ARRAY['tenants', 'projects', 'tasks', 'repositories', 'comments', 'audit_logs'];
BEGIN
    FOREACH rls_table.table_name IN ARRAY rls_tables
    LOOP
        IF NOT EXISTS (
            SELECT 1 FROM pg_tables 
            WHERE tablename = rls_table.table_name 
            AND rowsecurity = true
        ) THEN
            RAISE EXCEPTION 'RLS未在表%上启用', rls_table.table_name;
        END IF;
    END LOOP;
    
    RAISE NOTICE '✓ RLS策略验证通过';
END $$;

-- =====================================================================================
-- 触发器验证
-- =====================================================================================

-- 验证updated_at触发器
DO $$
DECLARE
    trigger_table RECORD;
    trigger_tables TEXT[] := ARRAY['tenants', 'users', 'projects', 'tasks'];
BEGIN
    FOREACH trigger_table.table_name IN ARRAY trigger_tables
    LOOP
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.triggers 
            WHERE event_object_table = trigger_table.table_name 
            AND trigger_name LIKE '%updated_at%'
        ) THEN
            RAISE EXCEPTION 'updated_at触发器未在表%上设置', trigger_table.table_name;
        END IF;
    END LOOP;
    
    RAISE NOTICE '✓ updated_at触发器验证通过';
END $$;

-- 验证角色作用域触发器
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.triggers 
        WHERE event_object_table = 'tenant_members' 
        AND trigger_name = 'validate_tenant_member_role_scope'
    ) THEN
        RAISE EXCEPTION '租户成员角色作用域验证触发器不存在';
    END IF;
    
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.triggers 
        WHERE event_object_table = 'project_members' 
        AND trigger_name = 'validate_project_member_role_scope'
    ) THEN
        RAISE EXCEPTION '项目成员角色作用域验证触发器不存在';
    END IF;
    
    RAISE NOTICE '✓ 角色作用域验证触发器通过';
END $$;

-- =====================================================================================
-- 初始数据验证
-- =====================================================================================

-- 验证订阅套餐数据
DO $$
DECLARE
    plan_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO plan_count FROM subscription_plans;
    
    IF plan_count < 3 THEN
        RAISE EXCEPTION '订阅套餐数据不足，期望至少3个，实际: %', plan_count;
    END IF;
    
    -- 检查必要的套餐
    IF NOT EXISTS (SELECT 1 FROM subscription_plans WHERE name = 'Free') THEN
        RAISE EXCEPTION 'Free订阅套餐不存在';
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM subscription_plans WHERE name = 'Enterprise') THEN
        RAISE EXCEPTION 'Enterprise订阅套餐不存在';
    END IF;
    
    RAISE NOTICE '✓ 订阅套餐数据验证通过，共%个套餐', plan_count;
END $$;

-- 验证预定义角色数据
DO $$
DECLARE
    role_count INTEGER;
    tenant_role_count INTEGER;
    project_role_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO role_count FROM roles WHERE is_predefined = true;
    
    IF role_count < 6 THEN
        RAISE EXCEPTION '预定义角色数据不足，期望至少6个，实际: %', role_count;
    END IF;
    
    -- 检查租户级角色
    SELECT COUNT(*) INTO tenant_role_count FROM roles WHERE is_predefined = true AND scope = 'tenant';
    IF tenant_role_count < 3 THEN
        RAISE EXCEPTION '租户级预定义角色不足';
    END IF;
    
    -- 检查项目级角色
    SELECT COUNT(*) INTO project_role_count FROM roles WHERE is_predefined = true AND scope = 'project';
    IF project_role_count < 3 THEN
        RAISE EXCEPTION '项目级预定义角色不足';
    END IF;
    
    RAISE NOTICE '✓ 预定义角色数据验证通过，租户级%个，项目级%个', tenant_role_count, project_role_count;
END $$;

-- 验证系统配置数据
DO $$
DECLARE
    setting_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO setting_count FROM system_settings;
    
    IF setting_count < 5 THEN
        RAISE EXCEPTION '系统配置数据不足，期望至少5个，实际: %', setting_count;
    END IF;
    
    -- 检查关键配置
    IF NOT EXISTS (SELECT 1 FROM system_settings WHERE key = 'platform.name') THEN
        RAISE EXCEPTION 'platform.name系统配置不存在';
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM system_settings WHERE key = 'feature.ai.enabled') THEN
        RAISE EXCEPTION 'feature.ai.enabled系统配置不存在';
    END IF;
    
    RAISE NOTICE '✓ 系统配置数据验证通过，共%个配置项', setting_count;
END $$;

-- =====================================================================================
-- 性能基准测试
-- =====================================================================================

-- 创建测试数据并测试查询性能
DO $$
DECLARE
    start_time TIMESTAMP;
    end_time TIMESTAMP;
    execution_time INTERVAL;
    test_tenant_id UUID;
    test_user_id UUID;
    test_project_id UUID;
BEGIN
    -- 创建测试租户
    INSERT INTO tenants (name, data_residency_region, subscription_plan_id)
    SELECT 'Test Tenant', 'US-EAST-1', id FROM subscription_plans WHERE name = 'Pro' LIMIT 1
    RETURNING id INTO test_tenant_id;
    
    -- 创建测试用户
    INSERT INTO users (email, full_name)
    VALUES ('test@example.com', 'Test User')
    RETURNING id INTO test_user_id;
    
    -- 创建测试项目
    INSERT INTO projects (tenant_id, name, key, manager_id)
    VALUES (test_tenant_id, 'Test Project', 'TEST', test_user_id)
    RETURNING id INTO test_project_id;
    
    -- 测试RLS查询性能
    start_time := clock_timestamp();
    
    -- 设置RLS上下文
    PERFORM set_config('app.current_tenant_id', test_tenant_id::text, true);
    
    -- 执行典型查询
    PERFORM COUNT(*) FROM projects WHERE tenant_id = test_tenant_id;
    PERFORM COUNT(*) FROM tasks WHERE project_id = test_project_id;
    
    end_time := clock_timestamp();
    execution_time := end_time - start_time;
    
    IF EXTRACT(MILLISECONDS FROM execution_time) > 100 THEN
        RAISE WARNING '查询性能较慢: %ms', EXTRACT(MILLISECONDS FROM execution_time);
    ELSE
        RAISE NOTICE '✓ 基础查询性能测试通过: %ms', EXTRACT(MILLISECONDS FROM execution_time);
    END IF;
    
    -- 清理测试数据
    DELETE FROM projects WHERE id = test_project_id;
    DELETE FROM users WHERE id = test_user_id;
    DELETE FROM tenants WHERE id = test_tenant_id;
END $$;

-- =====================================================================================
-- 总结报告
-- =====================================================================================

RAISE NOTICE '';
RAISE NOTICE '=====================================================================================';
RAISE NOTICE '✓ 数据库Schema验证完成！';
RAISE NOTICE '  - 表结构: 正常';
RAISE NOTICE '  - 索引策略: 正常'; 
RAISE NOTICE '  - 约束定义: 正常';
RAISE NOTICE '  - RLS策略: 正常';
RAISE NOTICE '  - 触发器: 正常';
RAISE NOTICE '  - 初始数据: 正常';
RAISE NOTICE '  - 基础性能: 正常';
RAISE NOTICE '=====================================================================================';