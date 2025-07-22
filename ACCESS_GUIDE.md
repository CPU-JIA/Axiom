# 🚀 Axiom平台本地访问指南

## 方式一：Docker Compose 一键启动 (推荐)

### 前置要求
- Docker Desktop 已安装并运行
- Git 已安装
- 8GB+ 可用内存

### 启动步骤
```bash
# 1. 克隆项目
git clone https://github.com/CPU-JIA/Axiom.git
cd Axiom

# 2. 一键启动所有服务
docker-compose up -d

# 3. 等待启动完成 (约2-3分钟)
docker-compose ps

# 4. 访问平台
echo "🌐 前端界面: http://localhost:3000"
echo "🔧 API网关: http://localhost:8000"
echo "📊 监控面板: http://localhost:3001"
```

## 方式二：前端开发模式启动

### 前置要求
- Node.js 18+ 已安装
- npm 已安装

### 启动步骤
```bash
# 1. 进入前端目录
cd web

# 2. 安装依赖
npm install --legacy-peer-deps

# 3. 启动开发服务器
npm run dev

# 4. 访问地址
# 前端开发服务器: http://localhost:3000
```

## 方式三：完整开发环境

### 启动所有服务
```bash
# 后端服务 (需要Go 1.21+)
# 启动各个微服务...

# 前端服务
cd web && npm run dev

# 数据库服务
docker run -d --name axiom-postgres \
  -e POSTGRES_DB=axiom_db \
  -e POSTGRES_USER=axiom_user \
  -e POSTGRES_PASSWORD=axiom_pass \
  -p 5432:5432 postgres:15

# Redis缓存
docker run -d --name axiom-redis \
  -p 6379:6379 redis:7-alpine
```

## 🌐 访问地址一览

| 服务 | 地址 | 说明 |
|------|------|------|
| **主平台** | http://localhost:3000 | 核心Web界面 |
| **API网关** | http://localhost:8000 | 后端API入口 |
| **监控面板** | http://localhost:3001 | Grafana监控 |
| **数据库** | localhost:5432 | PostgreSQL |
| **缓存** | localhost:6379 | Redis |
| **对象存储** | http://localhost:9001 | MinIO控制台 |

## 🔧 故障排查

### 常见问题
1. **端口冲突**: 修改docker-compose.yml中的端口映射
2. **内存不足**: 确保至少8GB可用内存
3. **Docker未启动**: 启动Docker Desktop
4. **防火墙阻拦**: 允许相关端口通过防火墙

### 检查命令
```bash
# 检查Docker状态
docker --version
docker-compose --version

# 检查服务运行状态
docker-compose ps

# 查看服务日志
docker-compose logs -f

# 重启所有服务
docker-compose restart
```

## 🎯 首次访问建议

1. **访问主界面**: http://localhost:3000
2. **查看登录页面**: 体验品牌展示和动画效果
3. **测试功能模块**: 
   - 项目管理
   - 任务看板
   - 用户设置
   - 团队协作
4. **监控系统状态**: http://localhost:3001 (admin/admin123)

## 📱 移动端访问

使用手机浏览器访问: http://[您的IP地址]:3000
例如: http://192.168.1.100:3000

---

**快速开始**: 
```bash
git clone https://github.com/CPU-JIA/Axiom.git && cd Axiom && docker-compose up -d
```

**享受您的Axiom平台体验！** 🚀