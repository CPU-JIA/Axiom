#!/usr/bin/env bash

# =====================================================================================
# 数据库连接测试脚本
# 几何原本 (Euclid Elements) - Database Connection Test
# =====================================================================================

set -euo pipefail

# 默认配置
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-euclid_elements}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-password}"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*"
}

# 测试PostgreSQL连接
test_connection() {
    log_info "测试数据库连接..."
    log_info "主机: $DB_HOST:$DB_PORT"
    log_info "数据库: $DB_NAME"
    log_info "用户: $DB_USER"
    echo
    
    export PGPASSWORD="$DB_PASSWORD"
    
    # 测试基础连接
    if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -c 'SELECT version();' 2>/dev/null; then
        log_success "PostgreSQL连接测试成功"
    else
        log_error "PostgreSQL连接失败"
        exit 1
    fi
    
    # 测试目标数据库
    if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c 'SELECT NOW();' 2>/dev/null; then
        log_success "目标数据库连接成功"
    else
        log_warning "目标数据库 '$DB_NAME' 不存在或无法连接"
        log_info "可以运行 './migrate.sh setup' 创建数据库"
    fi
}

# 测试Schema状态
test_schema() {
    log_info "检查Schema状态..."
    
    export PGPASSWORD="$DB_PASSWORD"
    
    # 检查表数量
    local table_count
    table_count=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "
        SELECT COUNT(*) FROM information_schema.tables 
        WHERE table_schema = 'public' AND table_type = 'BASE TABLE';
    " 2>/dev/null | tr -d ' ' || echo "0")
    
    log_info "发现 $table_count 个数据表"
    
    if [ "$table_count" -gt "20" ]; then
        log_success "Schema已初始化"
        
        # 检查关键表
        local key_tables=("tenants" "users" "projects" "tasks" "roles")
        for table in "${key_tables[@]}"; do
            if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "\dt $table" 2>/dev/null | grep -q "$table"; then
                log_success "✓ $table 表存在"
            else
                log_warning "✗ $table 表不存在"
            fi
        done
    else
        log_warning "Schema未初始化或不完整"
        log_info "运行 './migrate.sh up' 初始化Schema"
    fi
}

# 测试基本CRUD操作
test_crud() {
    log_info "测试基本CRUD操作..."
    
    export PGPASSWORD="$DB_PASSWORD"
    
    # 测试读取系统配置
    local settings_count
    settings_count=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "
        SELECT COUNT(*) FROM system_settings;
    " 2>/dev/null | tr -d ' ' || echo "0")
    
    if [ "$settings_count" -gt "0" ]; then
        log_success "系统配置读取正常 ($settings_count 项)"
    else
        log_warning "系统配置为空或表不存在"
    fi
    
    # 测试订阅套餐
    local plans_count
    plans_count=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "
        SELECT COUNT(*) FROM subscription_plans;
    " 2>/dev/null | tr -d ' ' || echo "0")
    
    if [ "$plans_count" -gt "0" ]; then
        log_success "订阅套餐数据正常 ($plans_count 个)"
    else
        log_warning "订阅套餐数据为空"
    fi
}

# 性能基准测试
test_performance() {
    log_info "执行基准性能测试..."
    
    export PGPASSWORD="$DB_PASSWORD"
    
    # 简单查询性能测试
    local start_time end_time
    start_time=$(date +%s%3N)
    
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "
        SELECT COUNT(*) FROM information_schema.tables;
        SELECT COUNT(*) FROM information_schema.columns;
        SELECT COUNT(*) FROM pg_indexes WHERE schemaname = 'public';
    " > /dev/null 2>&1
    
    end_time=$(date +%s%3N)
    local duration=$((end_time - start_time))
    
    if [ "$duration" -lt "1000" ]; then
        log_success "基准查询性能良好 (${duration}ms)"
    else
        log_warning "基准查询性能较慢 (${duration}ms)"
    fi
}

# 主函数
main() {
    echo "数据库连接测试 - 几何原本 (Euclid Elements)"
    echo "========================================================"
    echo
    
    test_connection
    echo
    
    if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c '\q' 2>/dev/null; then
        test_schema
        echo
        test_crud
        echo
        test_performance
    fi
    
    echo
    log_success "数据库测试完成！"
}

main "$@"