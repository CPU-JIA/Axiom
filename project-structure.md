# 项目代码仓库结构

## 整体架构
采用 **Monorepo** 模式，所有微服务统一管理，便于依赖管理和原子化提交。

```
cloud-platform/
├── README.md                    # 项目总体说明
├── .gitignore                   # Git忽略规则
├── docker-compose.yml           # 本地开发环境
├── .github/                     # GitHub Actions CI/CD
│   └── workflows/
│       ├── ci.yml              # 持续集成
│       ├── deploy.yml          # 部署流程
│       └── security-scan.yml   # 安全扫描
├── docs/                        # 项目文档
│   ├── api/                    # API文档
│   ├── deployment/             # 部署文档
│   └── architecture/           # 架构图
├── configs/                     # 配置文件
│   ├── kubernetes/             # K8s部署配置
│   ├── helm/                   # Helm Charts
│   ├── terraform/              # 基础设施即代码
│   └── monitoring/             # 监控配置
├── services/                    # 微服务代码
│   ├── iam-service/            # 身份认证服务
│   │   ├── cmd/
│   │   ├── internal/
│   │   ├── pkg/
│   │   ├── migrations/         # 数据库迁移
│   │   ├── Dockerfile
│   │   └── go.mod
│   ├── tenant-service/         # 租户管理服务
│   ├── project-service/        # 项目任务服务
│   ├── git-gateway-service/    # Git网关服务
│   ├── cicd-service/           # CI/CD服务
│   ├── notification-service/   # 通知服务
│   ├── kb-service/             # 知识库服务
│   └── api-gateway/            # API网关
├── web/                         # 前端代码
│   ├── packages/
│   │   ├── ui-components/      # UI组件库
│   │   ├── shared/             # 共享工具
│   │   └── app/                # 主应用
│   ├── package.json
│   └── lerna.json              # Monorepo管理
├── shared/                      # 共享代码
│   ├── proto/                  # gRPC协议定义
│   ├── database/               # 数据库Schema
│   ├── events/                 # 事件定义
│   └── security/               # 安全工具
├── tests/                       # 端到端测试
│   ├── e2e/
│   ├── integration/
│   └── performance/
├── tools/                       # 开发工具
│   ├── codegen/                # 代码生成
│   ├── migration/              # 数据库迁移工具
│   └── scripts/                # 构建脚本
└── Makefile                     # 统一构建命令
```

## 技术栈选择

### 后端技术栈
- **语言**: Go 1.21+
- **框架**: Gin (HTTP), gRPC (服务间通信)
- **数据库**: PostgreSQL 14+ (主库), Redis 7+ (缓存)
- **消息队列**: Apache Kafka
- **容器化**: Docker, Kubernetes
- **监控**: Prometheus, Grafana, Jaeger

### 前端技术栈
- **框架**: React 18 + TypeScript
- **UI库**: Ant Design Pro
- **状态管理**: Redux Toolkit + RTK Query
- **构建工具**: Vite
- **测试**: Jest + React Testing Library

### DevOps & 基础设施
- **容器编排**: Kubernetes
- **CI/CD**: GitHub Actions + Tekton
- **基础设施**: Terraform + Helm
- **密钥管理**: HashiCorp Vault
- **服务网格**: Istio (可选)

## 开发环境设置

### 前置要求
- Go 1.21+
- Node.js 18+
- Docker & Docker Compose
- kubectl & helm
- PostgreSQL 14+
- Redis 7+

### 本地启动命令
```bash
# 启动基础设施
make infra-up

# 启动所有服务
make dev-up

# 运行测试
make test

# 代码生成
make generate
```

## Git工作流

遵循 **Git Flow** 简化版：
- `main`: 生产分支，只接受release/hotfix合并
- `develop`: 开发分支，集成测试
- `feature/*`: 功能分支
- `release/*`: 发布准备分支
- `hotfix/*`: 紧急修复分支

### 提交规范
使用 Conventional Commits:
- `feat(auth): add multi-factor authentication`
- `fix(api): resolve JWT token expiration issue`
- `docs(readme): update installation instructions`