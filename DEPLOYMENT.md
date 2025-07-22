# Axiom 平台部署指南

## 🚀 快速部署

### 前置要求

- **Docker** 20.10+
- **Docker Compose** 2.0+
- **Kubernetes** 1.24+ (生产环境)
- **Helm** 3.8+ (生产环境)
- **Git** 2.30+

### 本地开发环境

```bash
# 1. 克隆项目
git clone https://github.com/CPU-JIA/Axiom.git
cd Axiom

# 2. 启动完整环境 (推荐)
make quick-start

# 3. 分步启动 (可选)
make build        # 构建所有服务
make start        # 启动服务
make web-dev      # 启动前端开发服务器
```

### 🐳 Docker 部署

#### 单机部署

```bash
# 使用预构建镜像快速部署
docker-compose -f docker-compose.yml up -d

# 访问应用
open http://localhost:3000
```

#### 自定义构建部署

```bash
# 构建并启动
docker-compose -f docker-compose.build.yml up --build -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f
```

### ☸️ Kubernetes 部署

#### 使用 Helm (推荐)

```bash
# 1. 添加 Helm 仓库
helm repo add axiom https://cpu-jia.github.io/axiom-helm-charts
helm repo update

# 2. 创建命名空间
kubectl create namespace axiom-system

# 3. 部署应用
helm install axiom axiom/axiom \
  --namespace axiom-system \
  --values configs/helm/values.prod.yaml \
  --wait

# 4. 检查部署状态
kubectl get pods -n axiom-system
kubectl get services -n axiom-system
```

#### 使用原生 YAML

```bash
# 应用所有配置
kubectl apply -f configs/kubernetes/

# 检查部署状态
kubectl get deployments,services,ingress -n axiom-system
```

### 🏗️ 生产环境部署

#### 1. 基础设施准备

```bash
# 使用 Terraform 创建云资源
cd configs/terraform
terraform init
terraform plan -var-file="prod.tfvars"
terraform apply -var-file="prod.tfvars"
```

#### 2. 数据库初始化

```bash
# PostgreSQL 初始化
kubectl exec -it postgres-0 -n axiom-system -- \
  psql -U postgres -d axiom_db -f /docker-entrypoint-initdb.d/init.sql

# Redis 配置
kubectl apply -f configs/redis/redis-cluster.yaml
```

#### 3. 证书和域名配置

```bash
# 安装 cert-manager
helm install cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --set installCRDs=true

# 配置 Ingress 和 SSL
kubectl apply -f configs/kubernetes/ingress/
```

#### 4. 监控和日志

```bash
# Prometheus + Grafana
helm install monitoring prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace \
  --values configs/monitoring/values.yaml

# ELK Stack
helm install elk elastic/elasticsearch \
  --namespace logging \
  --create-namespace
```

## 🔧 配置说明

### 环境变量

创建 `.env` 文件：

```bash
# 数据库配置
DB_HOST=localhost
DB_PORT=5432
DB_NAME=axiom_db
DB_USER=axiom_user
DB_PASSWORD=your_secure_password

# Redis 配置
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=your_redis_password

# JWT 配置
JWT_SECRET=your_jwt_secret_key
JWT_EXPIRES_IN=24h

# 邮件配置
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your_email@gmail.com
SMTP_PASS=your_app_password

# 对象存储配置
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin123
```

### 服务端口配置

| 服务 | 端口 | 描述 |
|------|------|------|
| Web Frontend | 3000 | React 应用 |
| API Gateway | 8000 | 统一网关 |
| IAM Service | 8001 | 身份认证 |
| Tenant Service | 8002 | 租户管理 |
| Project Service | 8003 | 项目管理 |
| CI/CD Service | 8004 | 持续集成 |
| Git Service | 8005 | Git 管理 |
| PostgreSQL | 5432 | 主数据库 |
| Redis | 6379 | 缓存数据库 |
| MinIO | 9000 | 对象存储 |

## 🚦 健康检查

### 服务健康状态

```bash
# 检查所有服务健康状态
curl http://localhost:8000/health

# 检查具体服务
curl http://localhost:8001/iam/health
curl http://localhost:8002/tenant/health
curl http://localhost:8003/project/health
```

### 监控端点

```bash
# Prometheus 指标
curl http://localhost:8000/metrics

# 应用性能监控
curl http://localhost:8000/debug/pprof/
```

## 🔄 更新部署

### Docker 环境更新

```bash
# 拉取最新镜像
docker-compose pull

# 重新启动服务
docker-compose up -d
```

### Kubernetes 环境更新

```bash
# Helm 更新
helm upgrade axiom axiom/axiom \
  --namespace axiom-system \
  --values configs/helm/values.prod.yaml

# 滚动更新单个服务
kubectl rollout restart deployment/api-gateway -n axiom-system
```

## 🛡️ 安全配置

### SSL/TLS 配置

```bash
# 生成自签名证书 (开发环境)
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout tls.key -out tls.crt

# Kubernetes 密钥
kubectl create secret tls axiom-tls \
  --key tls.key \
  --cert tls.crt \
  -n axiom-system
```

### 防火墙配置

```bash
# 开放必要端口
ufw allow 80/tcp
ufw allow 443/tcp
ufw allow 22/tcp
ufw enable
```

## 📊 性能调优

### 数据库优化

```sql
-- PostgreSQL 性能配置
ALTER SYSTEM SET shared_buffers = '256MB';
ALTER SYSTEM SET effective_cache_size = '1GB';
ALTER SYSTEM SET maintenance_work_mem = '64MB';
SELECT pg_reload_conf();
```

### Redis 配置优化

```bash
# Redis 内存优化
redis-cli CONFIG SET maxmemory 512mb
redis-cli CONFIG SET maxmemory-policy allkeys-lru
```

## 🔍 故障排查

### 常见问题

#### 1. 服务无法启动

```bash
# 查看服务日志
docker-compose logs service-name
kubectl logs -f deployment/service-name -n axiom-system

# 检查资源使用
kubectl top pods -n axiom-system
```

#### 2. 数据库连接失败

```bash
# 测试数据库连接
kubectl exec -it postgres-0 -n axiom-system -- \
  psql -U axiom_user -d axiom_db -c "SELECT 1;"
```

#### 3. 前端无法访问后端

```bash
# 检查网络连接
kubectl exec -it frontend-pod -n axiom-system -- \
  wget -qO- http://api-gateway:8000/health
```

### 日志收集

```bash
# 收集所有服务日志
kubectl logs -l app=axiom --all-containers=true -n axiom-system > axiom-logs.txt

# 实时监控日志
kubectl logs -f -l app=axiom --all-containers=true -n axiom-system
```

## 📈 监控告警

### Grafana 仪表板

访问 `http://your-domain/grafana` 导入以下仪表板：

- **应用性能监控**: `configs/grafana/dashboards/app-performance.json`
- **基础设施监控**: `configs/grafana/dashboards/infrastructure.json`
- **业务指标监控**: `configs/grafana/dashboards/business-metrics.json`

### 告警规则

```yaml
# Prometheus 告警规则
groups:
- name: axiom.rules
  rules:
  - alert: HighErrorRate
    expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.1
    for: 5m
    annotations:
      summary: "High error rate detected"
```

## 🚀 部署最佳实践

1. **蓝绿部署**: 使用 Kubernetes 的 Deployment 策略实现零停机部署
2. **健康检查**: 确保所有服务都配置了适当的健康检查端点
3. **资源限制**: 为每个容器设置合理的 CPU 和内存限制
4. **备份策略**: 定期备份数据库和配置文件
5. **安全更新**: 定期更新基础镜像和依赖包
6. **监控覆盖**: 覆盖应用、基础设施和业务指标监控

---

**技术支持**: 如有部署问题，请提交 [GitHub Issue](https://github.com/CPU-JIA/Axiom/issues)