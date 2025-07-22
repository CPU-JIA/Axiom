# CI/CD服务

🚀 Axiom（Axiom Platform）企业级智能开发协作平台的CI/CD服务，基于Tekton Pipelines实现的云原生持续集成/持续部署系统。

## ✨ 功能特性

### 🔧 核心功能
- **流水线管理**: 完整的CI/CD流水线创建、编辑、执行、监控
- **任务编排**: 支持复杂的任务依赖关系和并行执行
- **构建缓存**: 智能缓存机制，提升构建效率
- **多租户隔离**: 基于租户的资源隔离和权限管理
- **事件驱动**: 基于Webhook、Git事件的自动触发

### 🎯 技术特性
- **云原生**: 基于Kubernetes和Tekton Pipelines
- **高可用**: 支持分布式部署和故障恢复
- **可观测性**: 完整的日志、监控、追踪体系
- **安全性**: JWT认证、RBAC权限、数据加密
- **扩展性**: 插件化架构，支持自定义任务类型

## 🏗️ 架构设计

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Web Frontend   │    │   API Gateway    │    │  Other Services  │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          └──────────────────────┼──────────────────────┘
                                 │
                    ┌─────────────────┐
                    │  CI/CD Service   │
                    └─────────┬───────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│   PostgreSQL    │  │  Tekton Engine   │  │   File Storage   │
│   (Metadata)    │  │  (K8s Pipelines) │  │  (Logs/Cache)   │
└─────────────────┘  └─────────────────┘  └─────────────────┘
```

### 核心组件

- **Pipeline Service**: 流水线管理服务
- **Pipeline Run Service**: 流水线运行管理
- **Tekton Service**: Kubernetes/Tekton集成
- **Cache Service**: 构建缓存管理
- **Notification Service**: 通知和事件处理

## 🚀 快速开始

### 前置要求

- Go 1.21+
- PostgreSQL 12+
- Kubernetes 1.24+ (可选)
- Tekton Pipelines v0.50+ (可选)
- Docker & Docker Compose

### 本地开发

1. **克隆代码**
   ```bash
   cd services/cicd-service
   ```

2. **安装依赖**
   ```bash
   go mod download
   ```

3. **启动依赖服务**
   ```bash
   docker-compose up -d postgres redis
   ```

4. **配置环境变量**
   ```bash
   export DB_HOST=localhost
   export DB_PASSWORD=password123
   export JWT_SECRET=your-development-secret
   ```

5. **运行服务**
   ```bash
   go run cmd/main.go
   ```

6. **测试API**
   ```bash
   chmod +x scripts/test-api.sh
   ./scripts/test-api.sh
   ```

### Docker部署

1. **构建镜像**
   ```bash
   docker build -t euclid/cicd-service:latest .
   ```

2. **使用Docker Compose**
   ```bash
   docker-compose up -d
   ```

3. **检查服务状态**
   ```bash
   curl http://localhost:8005/health
   ```

## 📚 API文档

### 健康检查

- `GET /health` - 服务健康状态
- `GET /health/live` - 存活探针  
- `GET /health/ready` - 就绪探针

### 流水线管理

- `POST /api/v1/pipelines` - 创建流水线
- `GET /api/v1/pipelines` - 流水线列表
- `GET /api/v1/pipelines/{id}` - 流水线详情
- `PUT /api/v1/pipelines/{id}` - 更新流水线
- `DELETE /api/v1/pipelines/{id}` - 删除流水线
- `POST /api/v1/pipelines/{id}/trigger` - 触发执行

### 流水线运行

- `POST /api/v1/pipeline-runs` - 创建运行
- `GET /api/v1/pipeline-runs` - 运行列表  
- `GET /api/v1/pipeline-runs/{id}` - 运行详情
- `POST /api/v1/pipeline-runs/{id}/cancel` - 取消运行
- `POST /api/v1/pipeline-runs/{id}/retry` - 重试运行

### 构建缓存

- `POST /api/v1/cache` - 存储缓存
- `GET /api/v1/cache` - 缓存列表
- `GET /api/v1/cache/{key}` - 检索缓存
- `DELETE /api/v1/cache/{id}` - 删除缓存
- `GET /api/v1/cache/statistics` - 缓存统计

## ⚙️ 配置说明

### 环境变量

| 变量名 | 描述 | 默认值 |
|--------|------|--------|
| `ENVIRONMENT` | 运行环境 | `development` |
| `PORT` | 服务端口 | `8005` |
| `DB_HOST` | 数据库主机 | `localhost` |
| `DB_PORT` | 数据库端口 | `5432` |
| `DB_USER` | 数据库用户 | `postgres` |
| `DB_PASSWORD` | 数据库密码 | - |
| `DB_NAME` | 数据库名称 | `euclid_elements` |
| `JWT_SECRET` | JWT密钥 | - |
| `K8S_IN_CLUSTER` | 集群内运行 | `false` |
| `K8S_NAMESPACE` | K8s命名空间 | `cicd` |
| `TEKTON_NAMESPACE` | Tekton命名空间 | `tekton-pipelines` |

### 配置文件

参考 `configs/config.yaml` 了解完整配置选项。

## 🔧 开发指南

### 项目结构

```
cicd-service/
├── cmd/                    # 应用入口
├── internal/              # 内部代码
│   ├── config/           # 配置管理
│   ├── handlers/         # HTTP处理器
│   ├── middleware/       # 中间件
│   ├── models/          # 数据模型
│   ├── routes/          # 路由配置
│   └── services/        # 业务逻辑
├── configs/              # 配置文件
├── scripts/             # 脚本工具
├── Dockerfile           # Docker配置
├── docker-compose.yml   # 本地开发环境
└── go.mod              # Go模块
```

### 代码规范

- 使用 `gofmt` 格式化代码
- 遵循 Go 官方代码规范
- 函数和结构体必须有注释
- 错误处理要完整
- 使用结构化日志

### 数据模型

核心实体关系：

```
Project (项目) 1:N Pipeline (流水线)
Pipeline 1:N Task (任务)  
Pipeline 1:N PipelineRun (运行)
PipelineRun 1:N TaskRun (任务运行)
Project 1:N BuildCache (构建缓存)
```

## 🚢 部署运维

### Kubernetes部署

1. **创建命名空间和RBAC**
   ```bash
   kubectl create namespace cicd
   kubectl apply -f k8s/rbac.yaml
   ```

2. **部署应用**
   ```bash
   kubectl apply -f k8s/deployment.yaml
   kubectl apply -f k8s/service.yaml
   ```

3. **配置Ingress**
   ```bash
   kubectl apply -f k8s/ingress.yaml
   ```

### 监控告警

- **健康检查**: `/health` 端点
- **指标收集**: Prometheus格式
- **日志聚合**: 结构化JSON日志
- **链路追踪**: OpenTelemetry支持

### 备份恢复

- **数据备份**: PostgreSQL定期备份
- **缓存备份**: 构建缓存文件备份  
- **配置备份**: ConfigMap和Secret备份

## 🧪 测试

### 单元测试
```bash
go test ./... -v
```

### 集成测试
```bash
docker-compose -f docker-compose.test.yml up --abort-on-container-exit
```

### API测试
```bash
./scripts/test-api.sh
```

### 性能测试
```bash
./scripts/load-test.sh
```

## 📊 监控指标

### 应用指标
- `cicd_pipelines_total` - 流水线总数
- `cicd_pipeline_runs_total` - 运行总数
- `cicd_pipeline_runs_duration_seconds` - 运行时长
- `cicd_cache_hits_total` - 缓存命中数
- `cicd_cache_size_bytes` - 缓存大小

### 系统指标
- CPU使用率
- 内存使用率
- 磁盘使用率
- 网络I/O
- 数据库连接数

## 🔍 故障排查

### 常见问题

1. **服务无法启动**
   - 检查数据库连接
   - 确认端口未被占用
   - 验证环境变量配置

2. **Tekton连接失败**
   - 检查Kubernetes配置
   - 确认Tekton安装状态
   - 验证RBAC权限

3. **流水线运行失败**
   - 查看Pipeline和Task日志
   - 检查镜像拉取权限
   - 验证资源配额限制

### 日志查看

```bash
# 服务日志
kubectl logs -f deployment/cicd-service

# Tekton日志
kubectl logs -f -l app=tekton-pipelines-controller -n tekton-pipelines

# 数据库日志
kubectl logs -f deployment/postgres
```

## 🤝 贡献指南

1. Fork项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建Pull Request

## 📝 更新日志

### v1.0.0 (2024-01-XX)
- ✨ 初始版本发布
- 🔧 基础流水线管理功能
- 🚀 Tekton集成支持
- 💾 构建缓存机制
- 🔐 JWT认证授权
- 📊 监控和健康检查

## 📄 许可证

本项目采用 [MIT许可证](LICENSE)

## 📞 联系方式

- 项目地址: [GitHub](https://github.com/axiom/cicd-service)
- 问题反馈: [Issues](https://github.com/axiom/cicd-service/issues)
- 文档站点: [Documentation](https://docs.axiom.com/cicd)

---

🌟 **Axiom - 企业级智能开发协作平台** 🌟