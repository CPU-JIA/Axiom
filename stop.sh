#!/bin/bash

# 几何原本云端开发协作平台 - 停止脚本
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

# 显示帮助信息
show_help() {
    echo "使用方法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  --full    完全停止并删除所有容器和数据卷"
    echo "  --clean   停止容器并删除未使用的镜像"
    echo "  --backup  在停止前备份数据"
    echo "  --help    显示此帮助信息"
    echo ""
    echo "默认行为: 仅停止容器，保留数据"
}

# 备份数据
backup_data() {
    log_info "备份数据..."
    
    backup_dir="backups/$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$backup_dir"
    
    # 备份数据库
    if docker-compose ps | grep -q "postgres.*Up"; then
        log_info "备份PostgreSQL数据库..."
        docker-compose exec -T postgres pg_dumpall -U postgres > "$backup_dir/postgres_backup.sql"
        log_success "数据库备份完成: $backup_dir/postgres_backup.sql"
    fi
    
    # 备份Redis数据
    if docker-compose ps | grep -q "redis.*Up"; then
        log_info "备份Redis数据..."
        docker-compose exec -T redis redis-cli BGSAVE
        docker cp cloud-platform-cache:/data/dump.rdb "$backup_dir/redis_dump.rdb" 2>/dev/null || true
        log_success "Redis备份完成: $backup_dir/redis_dump.rdb"
    fi
    
    # 备份配置文件
    log_info "备份配置文件..."
    cp -r configs "$backup_dir/" 2>/dev/null || true
    
    log_success "数据备份完成: $backup_dir"
}

# 停止服务
stop_services() {
    log_info "停止所有服务..."
    
    if [ -f "docker-compose.yml" ]; then
        docker-compose down
        log_success "所有容器已停止"
    else
        log_error "未找到 docker-compose.yml 文件"
        return 1
    fi
}

# 完全清理
full_cleanup() {
    log_warning "执行完全清理，这将删除所有数据！"
    read -p "确定要继续吗？(y/N): " -n 1 -r
    echo
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        log_info "停止并删除所有容器和数据卷..."
        docker-compose down -v --remove-orphans
        
        # 删除相关镜像
        log_info "删除应用镜像..."
        docker images | grep "cloud-platform" | awk '{print $3}' | xargs -r docker rmi -f
        
        log_success "完全清理完成"
    else
        log_info "取消完全清理"
    fi
}

# 清理未使用的资源
clean_unused() {
    log_info "清理未使用的Docker资源..."
    
    # 删除未使用的镜像
    docker image prune -f
    
    # 删除未使用的容器
    docker container prune -f
    
    # 删除未使用的网络
    docker network prune -f
    
    # 删除未使用的数据卷（谨慎使用）
    read -p "是否删除未使用的数据卷？这可能会删除重要数据 (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        docker volume prune -f
    fi
    
    log_success "清理完成"
}

# 显示状态信息
show_status() {
    echo ""
    log_info "当前系统状态:"
    
    # 显示容器状态
    if command -v docker-compose &> /dev/null && [ -f "docker-compose.yml" ]; then
        echo ""
        echo -e "${BLUE}====== 容器状态 ======${NC}"
        docker-compose ps
    fi
    
    # 显示磁盘使用情况
    echo ""
    echo -e "${BLUE}====== Docker磁盘使用 ======${NC}"
    docker system df
    
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
    echo "                          停止脚本 v1.0.0"
    echo -e "${NC}"
    
    # 处理命令行参数
    case "${1:-}" in
        --help|-h)
            show_help
            exit 0
            ;;
        --full)
            backup_data
            full_cleanup
            ;;
        --clean)
            stop_services
            clean_unused
            ;;
        --backup)
            backup_data
            stop_services
            ;;
        *)
            stop_services
            ;;
    esac
    
    show_status
    
    echo ""
    log_success "操作完成！"
    echo ""
    echo -e "${BLUE}如需重新启动平台，请运行: ${GREEN}./deploy.sh${NC}"
    echo ""
}

# 错误处理
trap 'log_error "停止过程中发生错误"; exit 1' ERR

# 运行主函数
main "$@"