#!/bin/bash

# 几何原本云端开发协作平台 - 快速部署脚本
# 作者: JIA
# 版本: 1.0.0

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查系统要求
check_requirements() {
    log_info "检查系统要求..."
    
    # 检查Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装，请先安装 Docker"
        exit 1
    fi
    
    # 检查Docker Compose
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose 未安装，请先安装 Docker Compose"
        exit 1
    fi
    
    # 检查可用内存
    available_memory=$(free -m | awk 'NR==2{printf "%.0f", $7/1024}')
    if [ "$available_memory" -lt 4 ]; then
        log_warning "可用内存少于 4GB，可能影响性能"
    fi
    
    log_success "系统要求检查通过"
}

# 创建必要的目录和文件
setup_directories() {
    log_info "创建必要的目录结构..."
    
    # 创建数据目录
    mkdir -p data/{postgres,redis,minio,grafana,prometheus,elasticsearch}
    mkdir -p logs/{api-gateway,iam,tenant,project,cicd,git-gateway}
    mkdir -p configs/{monitoring,nginx,ssl}
    
    # 设置权限
    chmod -R 755 data
    chmod -R 755 logs
    
    log_success "目录结构创建完成"
}

# 生成配置文件
generate_configs() {
    log_info "生成配置文件..."
    
    # 生成 Prometheus 配置
    if [ ! -f "configs/monitoring/prometheus.yml" ]; then
        cat > configs/monitoring/prometheus.yml << 'EOF'
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  # - "first_rules.yml"
  # - "second_rules.yml"

scrape_configs:
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']

  - job_name: 'api-gateway'
    static_configs:
      - targets: ['api-gateway:8080']
    metrics_path: '/metrics'
    scrape_interval: 10s

  - job_name: 'iam-service'
    static_configs:
      - targets: ['iam-service:8080']
    metrics_path: '/metrics'
    scrape_interval: 10s

  - job_name: 'tenant-service'
    static_configs:
      - targets: ['tenant-service:8080']
    metrics_path: '/metrics'
    scrape_interval: 10s

  - job_name: 'project-service'
    static_configs:
      - targets: ['project-service:8080']
    metrics_path: '/metrics'
    scrape_interval: 10s

  - job_name: 'cicd-service'
    static_configs:
      - targets: ['cicd-service:8080']
    metrics_path: '/metrics'
    scrape_interval: 10s

  - job_name: 'git-gateway-service'
    static_configs:
      - targets: ['git-gateway-service:8080']
    metrics_path: '/metrics'
    scrape_interval: 10s
EOF
    fi
    
    log_success "配置文件生成完成"
}

# 构建和启动服务
start_services() {
    log_info "构建并启动服务..."
    
    # 拉取最新镜像
    log_info "拉取基础镜像..."
    docker-compose pull postgres redis gitea minio vault prometheus grafana jaeger elasticsearch
    
    # 构建应用镜像
    log_info "构建应用镜像..."
    docker-compose build --parallel
    
    # 启动基础设施服务
    log_info "启动基础设施服务..."
    docker-compose up -d postgres redis vault minio
    
    # 等待数据库启动
    log_info "等待数据库启动..."
    until docker-compose exec -T postgres pg_isready -U postgres -d euclid_elements; do
        log_info "等待数据库启动..."
        sleep 5
    done
    
    # 运行数据库迁移
    log_info "运行数据库迁移..."
    docker-compose exec -T postgres psql -U postgres -d euclid_elements -f /docker-entrypoint-initdb.d/001_initial_schema.sql
    
    # 启动应用服务
    log_info "启动应用服务..."
    docker-compose up -d api-gateway iam-service tenant-service project-service cicd-service git-gateway-service
    
    # 启动前端和监控服务
    log_info "启动前端和监控服务..."
    docker-compose up -d web gitea prometheus grafana jaeger elasticsearch
    
    log_success "所有服务已启动"
}

# 健康检查
health_check() {
    log_info "执行健康检查..."
    
    # 检查服务状态
    services=("postgres" "redis" "api-gateway" "iam-service" "tenant-service" "project-service" "cicd-service" "git-gateway-service" "web")
    
    for service in "${services[@]}"; do
        if docker-compose ps | grep -q "${service}.*Up"; then
            log_success "$service 服务运行正常"
        else
            log_error "$service 服务启动失败"
            return 1
        fi
    done
    
    # 检查Web服务访问
    log_info "检查Web服务访问..."
    if curl -f -s http://localhost:3000/health > /dev/null; then
        log_success "Web服务访问正常"
    else
        log_warning "Web服务可能尚未完全启动，请稍后访问"
    fi
    
    log_success "健康检查完成"
}

# 显示访问信息
show_access_info() {
    echo ""
    log_success "🎉 几何原本云端开发协作平台部署成功！"
    echo ""
    echo -e "${BLUE}====== 服务访问地址 ======${NC}"
    echo -e "🌐 主平台:        ${GREEN}http://localhost:3000${NC}"
    echo -e "📊 监控面板:      ${GREEN}http://localhost:3001${NC} (admin/admin123)"
    echo -e "🔧 Git服务:       ${GREEN}http://localhost:3000${NC}"
    echo -e "📈 指标查询:      ${GREEN}http://localhost:9090${NC}"
    echo -e "🔍 链路追踪:      ${GREEN}http://localhost:16686${NC}"
    echo -e "🗂️ 对象存储:      ${GREEN}http://localhost:9001${NC} (minioadmin/minioadmin123)"
    echo -e "🔐 密钥管理:      ${GREEN}http://localhost:8200${NC} (Token: dev-root-token)"
    echo ""
    echo -e "${BLUE}====== API端点 ======${NC}"
    echo -e "🚪 API网关:       ${GREEN}http://localhost:8080${NC}"
    echo -e "👤 身份认证:      ${GREEN}http://localhost:8081${NC}"
    echo -e "🏢 租户管理:      ${GREEN}http://localhost:8082${NC}"
    echo -e "📋 项目管理:      ${GREEN}http://localhost:8083${NC}"
    echo -e "🔄 CI/CD:         ${GREEN}http://localhost:8084${NC}"
    echo -e "📊 Git网关:       ${GREEN}http://localhost:8085${NC}"
    echo ""
    echo -e "${YELLOW}====== 管理命令 ======${NC}"
    echo -e "查看服务状态: ${GREEN}docker-compose ps${NC}"
    echo -e "查看服务日志: ${GREEN}docker-compose logs [service_name]${NC}"
    echo -e "停止所有服务: ${GREEN}docker-compose down${NC}"
    echo -e "重启特定服务: ${GREEN}docker-compose restart [service_name]${NC}"
    echo ""
    echo -e "${BLUE}开始您的云端开发之旅吧！🚀${NC}"
    echo ""
}

# 主函数
main() {
    echo -e "${BLUE}"
    echo "  _____ _   _  ____ _     ___ ____    _____ _     _____ __  __ _____ _   _ _____ ____"
    echo " | ____| | | |/ ___| |   |_ _|  _ \  | ____| |   | ____|  \/  | ____| \ | |_   _/ ___|"
    echo " |  _| | | | | |   | |    | || | | | |  _| | |   |  _| | |\/| |  _| |  \| | | | \___ \\"
    echo " | |___| |_| | |___| |___ | || |_| | | |___| |___| |___| |  | | |___| |\  | | |  ___) |"
    echo " |_____|\___/ \____|_____|___|____/  |_____|_____|_____|_|  |_|_____|_| \_| |_| |____/"
    echo ""
    echo "                    几何原本 - 云端开发协作平台"
    echo "                          快速部署脚本 v1.0.0"
    echo -e "${NC}"
    
    # 执行部署步骤
    check_requirements
    setup_directories
    generate_configs
    start_services
    
    # 等待服务完全启动
    log_info "等待服务完全启动..."
    sleep 30
    
    health_check
    show_access_info
}

# 错误处理
trap 'log_error "部署过程中发生错误，请检查日志"; exit 1' ERR

# 运行主函数
main "$@"