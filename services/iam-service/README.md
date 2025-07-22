# IAM服务快速开始指南

## 概述

本IAM服务是一个完整的身份认证和访问管理系统，采用微服务架构，支持多租户、JWT认证、MFA等企业级功能。

## 功能特性

- ✅ 用户注册和登录
- ✅ JWT令牌认证（访问令牌 + 刷新令牌）
- ✅ 多因素认证（MFA）支持
- ✅ 密码重置和邮箱验证
- ✅ 用户资料管理
- ✅ 头像上传
- ✅ 安全中间件（CORS、安全头、请求ID等）
- ✅ 结构化日志
- ✅ 优雅关闭
- ✅ 健康检查端点
- ✅ 内部服务间认证
- ✅ Docker容器化

## 技术栈

- **语言**: Go 1.21+
- **Web框架**: Gin
- **数据库**: PostgreSQL + GORM
- **缓存**: Redis
- **认证**: JWT + bcrypt
- **配置管理**: Viper
- **日志**: Go slog
- **容器化**: Docker

## 项目结构

```
services/iam-service/
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

## 快速启动

### 1. 环境准备

确保已安装：
- Go 1.21+
- PostgreSQL 13+
- Redis 6+
- Docker (可选)

### 2. 数据库设置

```sql
-- 创建数据库
CREATE DATABASE cloud_platform;

-- 创建用户
CREATE USER developer WITH PASSWORD 'dev_password_2024';
GRANT ALL PRIVILEGES ON DATABASE cloud_platform TO developer;
```

### 3. 环境变量配置

```bash
export DATABASE_URL="postgres://developer:dev_password_2024@localhost:5432/cloud_platform?sslmode=disable"
export REDIS_URL="redis://localhost:6379"
export JWT_SECRET="your-jwt-secret-key"
export INTERNAL_SECRET="your-internal-secret"
export IAM_ENVIRONMENT="development"
export IAM_PORT="8080"
```

### 4. 运行服务

```bash
# 开发模式
cd services/iam-service
go run ./cmd

# 或编译后运行
go build ./cmd
./cmd
```

### 5. Docker部署

```bash
# 构建镜像
docker build -t iam-service .

# 运行容器
docker run -p 8080:8080 \
  -e DATABASE_URL="postgres://user:pass@host:5432/db" \
  -e REDIS_URL="redis://host:6379" \
  -e JWT_SECRET="your-jwt-secret" \
  iam-service
```

## API端点

### 健康检查
```
GET  /health           # 健康状态
GET  /health/ready     # 就绪检查
GET  /health/live      # 存活检查
```

### 认证相关（无需Token）
```
POST /api/v1/auth/register           # 用户注册
POST /api/v1/auth/login              # 用户登录
POST /api/v1/auth/refresh            # 刷新Token
POST /api/v1/auth/forgot-password    # 忘记密码
POST /api/v1/auth/reset-password     # 重置密码
POST /api/v1/auth/verify-email       # 验证邮箱
POST /api/v1/auth/resend-verification # 重发验证邮件
```

### 用户管理（需要JWT认证）
```
GET  /api/v1/users/profile        # 获取用户资料
PUT  /api/v1/users/profile        # 更新用户资料
POST /api/v1/users/change-password # 修改密码
POST /api/v1/users/upload-avatar  # 上传头像
GET  /api/v1/users                # 获取用户列表（管理员）
GET  /api/v1/users/:id            # 获取指定用户（管理员）
```

### MFA管理（需要JWT认证）
```
POST /api/v1/mfa/setup                    # 设置MFA
POST /api/v1/mfa/verify                   # 验证MFA
DELETE /api/v1/mfa/disable                # 禁用MFA
GET  /api/v1/mfa/backup-codes             # 获取备份码
POST /api/v1/mfa/backup-codes/regenerate  # 重新生成备份码
```

### 内部API（服务间调用）
```
POST /api/v1/internal/introspect      # Token验证
POST /api/v1/internal/switch-tenant   # 切换租户
```

## 使用示例

### 用户注册
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123",
    "display_name": "测试用户",
    "tenant_name": "测试租户"
  }'
```

### 用户登录
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }'
```

### 获取用户资料（需要Token）
```bash
curl -X GET http://localhost:8080/api/v1/users/profile \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

## 配置说明

服务支持通过配置文件和环境变量进行配置：

### 主要配置项
- `PORT`: 服务端口（默认8080）
- `DATABASE_URL`: PostgreSQL连接URL
- `REDIS_URL`: Redis连接URL
- `JWT_SECRET`: JWT签名密钥
- `INTERNAL_SECRET`: 内部API密钥
- `LOG_LEVEL`: 日志级别（debug/info/warn/error）

### 安全配置
- `PASSWORD_MIN_LENGTH`: 密码最小长度（默认8）
- `MAX_LOGIN_ATTEMPTS`: 最大登录尝试次数（默认5）
- `LOCKOUT_DURATION`: 账户锁定时长（默认15分钟）

### Token配置
- `ACCESS_TOKEN_EXPIRY`: 访问令牌有效期（默认60分钟）
- `REFRESH_TOKEN_EXPIRY`: 刷新令牌有效期（默认30天）

## 开发指南

### 添加新端点
1. 在`internal/handlers/`中添加处理器方法
2. 在`internal/routes/routes.go`中注册路由
3. 在`test.http`中添加测试用例

### 数据库迁移
服务启动时会自动执行数据库迁移，创建所需的表结构和索引。

### 日志记录
使用结构化日志，支持JSON格式输出：
```go
log.Info("User registered", "user_id", userID, "email", email)
log.Error("Database error", "error", err, "query", sql)
```

## 监控和运维

### 健康检查
- `/health`: 基本健康状态
- `/health/ready`: 依赖服务就绪状态
- `/health/live`: 服务存活状态

### 日志监控
所有请求都会记录访问日志，包含：
- 请求方法和路径
- 响应状态码
- 处理时间
- 客户端IP
- 用户代理
- 请求ID

## 安全特性

- 密码使用bcrypt加密
- JWT令牌包含过期时间
- 支持令牌刷新机制
- 登录失败次数限制
- 安全HTTP头设置
- CORS跨域配置
- 内部API认证保护

## 故障排除

### 常见问题
1. **数据库连接失败**: 检查DATABASE_URL配置
2. **Redis连接失败**: 检查REDIS_URL配置
3. **JWT验证失败**: 检查JWT_SECRET配置
4. **端口被占用**: 修改PORT环境变量

### 日志分析
查看应用日志获取详细错误信息：
```bash
# 开发环境
tail -f /dev/stdout

# 生产环境（Docker）
docker logs -f container_name
```

这个IAM服务为整个云协作开发平台提供了坚实的身份认证基础，下一步可以开始实现租户管理服务和API网关。