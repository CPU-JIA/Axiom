# 租户管理服务 (Tenant Service)

## 概述

租户管理服务是多租户SaaS架构的核心组件，负责管理租户生命周期、成员权限、资源配额和审计日志等功能。

## 功能特性

- ✅ 租户生命周期管理（创建、更新、激活、暂停）
- ✅ 成员邀请和权限管理（基于角色的访问控制 RBAC）
- ✅ 资源配额管理（成员数、项目数、存储空间）
- ✅ 多计划类型支持（免费版、专业版、企业版等）
- ✅ 审计日志记录
- ✅ 租户隔离和安全控制
- ✅ RESTful API设计
- ✅ JWT认证集成
- ✅ Redis缓存支持
- ✅ 健康检查和监控
- ✅ 优雅关闭机制
- ✅ Docker容器化

## 技术栈

- **语言**: Go 1.21+
- **Web框架**: Gin
- **数据库**: PostgreSQL + GORM
- **缓存**: Redis
- **认证**: JWT
- **配置管理**: Viper
- **日志**: Go slog
- **容器化**: Docker

## 项目结构

```
services/tenant-service/
├── cmd/                    # 应用程序入口
├── internal/              # 内部代码
│   ├── config/           # 配置管理
│   ├── database/         # 数据库连接和迁移
│   ├── handlers/         # HTTP处理器
│   ├── middleware/       # 中间件
│   ├── models/           # 数据模型
│   ├── routes/           # 路由配置
│   └── services/         # 业务逻辑
├── pkg/                   # 可复用包
│   ├── logger/           # 日志接口
│   └── utils/            # 工具函数
├── Dockerfile            # Docker构建文件
├── go.mod               # Go模块定义
└── test.http            # API测试文件
```

## 数据模型

### 租户 (Tenant)
- 基本信息：名称、显示名称、描述、Logo
- 状态管理：活跃、暂停、非活跃等
- 计划类型：免费版、专业版、企业版
- 资源限制：最大成员数、项目数、存储配额
- 自定义域名支持

### 租户成员 (TenantMember)  
- 用户与租户的关联关系
- 角色权限：所有者、管理员、维护者、开发者、访客
- 状态管理：活跃、非活跃、暂停、待激活
- 加入时间和权限记录

### 租户邀请 (TenantInvitation)
- 邮箱邀请系统
- 邀请令牌和过期时间
- 邀请状态跟踪
- 角色预设和权限定义

### 审计日志 (TenantAuditLog)
- 操作记录和追踪
- 用户行为审计
- IP地址和用户代理记录
- 资源变更历史

## 角色权限体系

### 角色层次（从高到低）
1. **Owner（所有者）** - 完全控制权限
2. **Admin（管理员）** - 管理成员和设置
3. **Maintainer（维护者）** - 项目维护权限  
4. **Developer（开发者）** - 开发和协作权限
5. **Guest（访客）** - 只读访问权限

### 权限映射
- 创建/删除租户：Owner
- 邀请/移除成员：Admin+
- 更新租户设置：Admin+  
- 查看成员列表：Developer+
- 查看租户信息：Guest+

## 快速启动

### 1. 环境准备

确保已安装：
- Go 1.21+
- PostgreSQL 13+
- Redis 6+
- Docker (可选)

### 2. 环境变量配置

```bash
export DATABASE_URL="postgres://developer:dev_password_2024@localhost:5432/cloud_platform?sslmode=disable"
export REDIS_URL="redis://localhost:6379"
export JWT_SECRET="your-jwt-secret-key"
export INTERNAL_SECRET="your-internal-secret"
export TENANT_ENVIRONMENT="development"
export TENANT_PORT="8002"
```

### 3. 运行服务

```bash
# 开发模式
cd services/tenant-service
go run ./cmd

# 或编译后运行
go build ./cmd
./cmd
```

### 4. Docker部署

```bash
# 构建镜像
docker build -t tenant-service .

# 运行容器
docker run -p 8002:8002 \
  -e DATABASE_URL="postgres://user:pass@host:5432/db" \
  -e REDIS_URL="redis://host:6379" \
  -e JWT_SECRET="your-jwt-secret" \
  tenant-service
```

## API端点

### 健康检查
```
GET  /health           # 健康状态
GET  /health/ready     # 就绪检查
GET  /health/live      # 存活检查
```

### 租户管理（需要JWT认证）
```
POST /api/v1/tenants                     # 创建租户
GET  /api/v1/tenants/my                  # 获取我的租户列表
GET  /api/v1/tenants/{id}                # 获取租户信息
PUT  /api/v1/tenants/{id}                # 更新租户信息（需要Admin+）
```

### 成员管理（需要租户权限）
```
GET  /api/v1/tenants/{id}/members        # 获取成员列表
POST /api/v1/tenants/{id}/members/invite # 邀请成员（需要Admin+）
```

### 待实现端点
```
GET    /api/v1/tenants/{id}/invitations     # 获取邀请列表
DELETE /api/v1/tenants/{id}/invitations/{id} # 取消邀请
DELETE /api/v1/tenants/{id}/members/{id}     # 移除成员
PUT    /api/v1/tenants/{id}/members/{id}/role # 更新成员角色
GET    /api/v1/tenants/{id}/audit           # 获取审计日志
GET    /api/v1/tenants/{id}/settings        # 获取租户设置
PUT    /api/v1/tenants/{id}/settings        # 更新租户设置
```

## 使用示例

### 创建租户
```bash
curl -X POST http://localhost:8002/api/v1/tenants \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "name": "my-awesome-team",
    "display_name": "My Awesome Team",
    "description": "This is our development team workspace",
    "plan_type": "pro"
  }'
```

### 邀请成员
```bash
curl -X POST http://localhost:8002/api/v1/tenants/TENANT_ID/members/invite \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "email": "newmember@example.com",
    "role": "developer",
    "message": "Welcome to our team!"
  }'
```

### 获取租户信息
```bash
curl -X GET http://localhost:8002/api/v1/tenants/TENANT_ID \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

## 配置说明

### 主要配置项
- `PORT`: 服务端口（默认8002）
- `DATABASE_URL`: PostgreSQL连接URL
- `REDIS_URL`: Redis连接URL
- `JWT_SECRET`: JWT签名密钥（与IAM服务保持一致）
- `INTERNAL_SECRET`: 内部API密钥
- `LOG_LEVEL`: 日志级别（debug/info/warn/error）

### 租户配置
- `DEFAULT_MAX_MEMBERS`: 默认最大成员数（默认50）
- `DEFAULT_MAX_PROJECTS`: 默认最大项目数（默认10）
- `DEFAULT_STORAGE_QUOTA`: 默认存储配额GB（默认100）
- `MAX_TENANTS_PER_USER`: 每用户最大租户数（默认5）

### 通知配置  
- `KAFKA_BROKERS`: Kafka代理列表
- `NOTIFICATION_TOPIC`: 通知主题
- `TENANT_EVENT_TOPIC`: 租户事件主题

## 架构设计

### 多租户隔离
- 行级安全策略（Row Level Security）
- 租户ID强制校验
- 成员权限分离
- 资源配额限制

### 缓存策略
- 租户信息缓存
- 成员权限缓存  
- 统计数据缓存
- 邀请令牌缓存

### 事件通知
- 租户创建/更新事件
- 成员加入/离开事件
- 权限变更事件
- 配额超限告警

## 开发指南

### 添加新角色
1. 在`models/tenant.go`中添加角色常量
2. 更新`middleware/auth.go`中的角色层次映射
3. 在相关API端点中添加权限检查

### 扩展租户功能
1. 更新数据模型和迁移脚本
2. 在服务层添加业务逻辑
3. 创建对应的API处理器
4. 更新路由配置

### 集成外部服务
1. 在`config.go`中添加服务配置
2. 创建客户端连接和健康检查
3. 在服务层中调用外部API
4. 处理错误和重试逻辑

## 监控和运维

### 健康检查端点
- `/health`: 基本服务状态
- `/health/ready`: 依赖服务就绪状态
- `/health/live`: 服务存活状态

### 关键指标
- 租户创建/更新速率
- 成员邀请成功率
- API响应时间
- 数据库连接池状态
- Redis缓存命中率

### 日志监控
- 结构化JSON日志
- 请求追踪ID
- 错误和异常记录
- 性能指标记录

## 安全考虑

### 认证授权
- JWT令牌验证
- 租户权限隔离
- 角色权限检查
- 内部API保护

### 数据安全
- 敏感信息加密
- SQL注入防护
- XSS攻击防护
- CSRF令牌保护

### 访问控制
- IP白名单（可选）
- 请求频率限制
- 资源配额控制
- 审计日志记录

## 故障排除

### 常见问题
1. **数据库连接失败**: 检查DATABASE_URL配置
2. **Redis连接失败**: 检查REDIS_URL配置
3. **JWT验证失败**: 确认JWT_SECRET与IAM服务一致
4. **权限被拒绝**: 检查用户角色和租户成员关系

### 诊断方法
```bash
# 查看服务健康状态
curl http://localhost:8002/health/ready

# 查看日志
docker logs tenant-service

# 检查数据库连接
psql $DATABASE_URL -c "SELECT 1"

# 检查Redis连接
redis-cli -u $REDIS_URL ping
```

这个租户管理服务为整个多租户架构提供了坚实的基础，支持灵活的权限管理和资源隔离，下一步可以继续实现项目管理服务和API网关。