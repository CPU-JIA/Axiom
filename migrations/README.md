# 数据库Schema实现文档

## 概述

本文档描述了"Axiom (Axiom Platform)"项目的完整数据库Schema实现，基于详细的数据库设计文档V3.1。

## 文件结构

```
migrations/
├── 001_initial_schema.sql    # 初始Schema创建脚本
├── migrate.sh               # 数据库迁移管理脚本
├── .env.example             # 环境配置示例
├── validate_schema.sql      # Schema验证测试脚本
└── README.md               # 本文档
```

## 核心特性

### 1. 多租户架构
- **逻辑隔离**: 每个核心业务表包含`tenant_id`列
- **行级安全策略(RLS)**: 数据库层面强制租户隔离
- **租户感知查询**: 应用层必须设置`app.current_tenant_id`

### 2. UUID v7主键标准
- **时间有序**: 结合全局唯一性与索引友好性
- **高并发友好**: 减少索引页分裂和写放大
- **分布式就绪**: 支持多节点部署

### 3. 审计与软删除
- **审计字段**: `created_at`、`updated_at`自动维护
- **软删除**: 核心表支持`deleted_at`字段
- **完整审计日志**: `audit_logs`表记录所有关键操作

### 4. 高性能索引策略
- **复合索引**: 针对高频查询优化
- **部分唯一索引**: 支持软删除场景的业务键唯一性
- **GIN索引**: JSONB字段查询优化
- **分区支持**: 大表按时间和租户分区

## 表结构概览

### 核心业务表

| 表名 | 用途 | 关键特性 |
|------|------|----------|
| `tenants` | 租户管理 | 多租户根表，订阅套餐关联 |
| `users` | 用户管理 | 全局用户，解耦认证方式 |
| `projects` | 项目管理 | 支持软删除，项目键唯一性 |
| `tasks` | 任务管理 | 自动序号生成，状态可配置 |
| `repositories` | 代码仓库 | Git元数据，项目级隔离 |
| `pull_requests` | 代码评审 | PR工作流，自动编号 |

### 系统支撑表

| 表名 | 用途 | 关键特性 |
|------|------|----------|
| `roles` | 权限管理 | 统一角色模型，租户/项目级 |
| `audit_logs` | 审计跟踪 | 分区表，完整操作记录 |
| `secrets` | 密钥管理 | 信封加密，KMS集成 |
| `notifications` | 消息通知 | 高写入优化，支持分区 |

## 使用指南

### 1. 初始化数据库

```bash
# 设置环境变量
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=euclid_elements
export DB_USER=postgres
export DB_PASSWORD=your_password

# 初始化数据库和迁移表
./migrate.sh setup

# 执行所有迁移
./migrate.sh up
```

### 2. 验证Schema

```bash
# 运行验证脚本
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f validate_schema.sql
```

### 3. 查看迁移状态

```bash
./migrate.sh status
```

## RLS配置示例

在应用层设置租户上下文：

```sql
-- 设置当前租户ID
SET app.current_tenant_id = '123e4567-e89b-12d3-a456-426614174000';

-- 所有查询将自动应用租户隔离
SELECT * FROM projects; -- 只返回当前租户的项目
```

## 性能优化建议

### 1. 连接池配置
```
max_connections: 20
min_connections: 5  
connection_timeout: 30s
idle_timeout: 5min
```

### 2. 查询优化
- 避免`SELECT *`，只查询必要列
- 利用复合索引优化`WHERE`和`ORDER BY`
- 使用`EXPLAIN ANALYZE`分析查询计划

### 3. 分区策略
- `audit_logs`: 按月分区
- `notifications`: 按租户+时间复合分区
- 定期归档旧分区数据

## 安全考量

### 1. 数据加密
- **传输加密**: 强制TLS/SSL连接
- **静态加密**: 启用数据库透明加密
- **应用层加密**: 敏感数据信封加密

### 2. 访问控制
- **最小权限**: 应用用户仅授予必要权限
- **RLS策略**: 数据库层强制多租户隔离
- **审计日志**: 完整操作追踪

### 3. 备份策略
- **全量备份**: 每日凌晨执行
- **增量备份**: 每小时执行WAL归档
- **异地备份**: 自动同步到云存储

## 监控指标

### 1. 性能指标
- 查询响应时间 (<100ms for 95%ile)
- 连接池使用率 (<80%)
- 索引命中率 (>95%)

### 2. 业务指标  
- 租户数据隔离完整性
- 审计日志完整性
- 软删除数据清理状态

## 故障排除

### 常见问题

1. **RLS策略阻止查询**
   ```sql
   -- 检查当前租户设置
   SELECT current_setting('app.current_tenant_id', true);
   
   -- 临时禁用RLS (仅限管理员)
   SET row_security = off;
   ```

2. **迁移执行失败**
   ```bash
   # 检查迁移状态
   ./migrate.sh status
   
   # 验证数据库连接
   psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c '\l'
   ```

3. **性能问题**
   ```sql
   -- 检查慢查询
   SELECT query, mean_time, calls 
   FROM pg_stat_statements 
   ORDER BY mean_time DESC LIMIT 10;
   ```

## 后续规划

### 1. 自动化运维
- [ ] 分区自动创建和清理脚本
- [ ] 数据生命周期管理策略
- [ ] 自动备份验证

### 2. 性能优化
- [ ] 查询计划缓存优化
- [ ] 物化视图用于复杂报表
- [ ] 连接池配置调优

### 3. 安全增强
- [ ] 数据库活动监控
- [ ] 异常访问模式检测
- [ ] 加密密钥轮转策略

## 联系方式

如有问题或建议，请联系开发团队或提交Issue。