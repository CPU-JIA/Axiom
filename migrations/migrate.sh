#!/usr/bin/env bash

# =====================================================================================
# 数据库迁移管理脚本
# 几何原本 (Euclid Elements) - Database Migration Manager
#
# 功能:
# - 执行数据库Schema迁移
# - 支持PostgreSQL数据库
# - 迁移版本控制
# - 回滚支持
# =====================================================================================

set -euo pipefail

# 配置变量
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MIGRATIONS_DIR="${SCRIPT_DIR}"
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
NC='\033[0m' # No Color

# 日志函数
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
    exit 1
}

# 检查PostgreSQL连接
check_db_connection() {
    log_info "检查数据库连接..."
    
    if ! command -v psql &> /dev/null; then
        log_error "psql 命令未找到，请安装PostgreSQL客户端"
    fi
    
    export PGPASSWORD="$DB_PASSWORD"
    
    if ! psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -c '\q' 2>/dev/null; then
        log_error "无法连接到PostgreSQL服务器 ($DB_HOST:$DB_PORT)"
    fi
    
    log_success "数据库连接正常"
}

# 创建数据库和迁移表
setup_database() {
    log_info "设置数据库和迁移表..."
    
    export PGPASSWORD="$DB_PASSWORD"
    
    # 检查数据库是否存在，不存在则创建
    if ! psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -lqt | cut -d \| -f 1 | grep -qw "$DB_NAME"; then
        log_info "创建数据库: $DB_NAME"
        psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -c "CREATE DATABASE $DB_NAME;"
    fi
    
    # 创建迁移跟踪表
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "
        CREATE TABLE IF NOT EXISTS schema_migrations (
            version VARCHAR(255) PRIMARY KEY,
            filename VARCHAR(512) NOT NULL,
            description TEXT,
            applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            execution_time_ms INTEGER,
            checksum VARCHAR(64)
        );
        
        CREATE INDEX IF NOT EXISTS idx_schema_migrations_applied_at 
        ON schema_migrations (applied_at DESC);
    " > /dev/null
    
    log_success "数据库设置完成"
}

# 计算文件校验和
calculate_checksum() {
    local file="$1"
    sha256sum "$file" | cut -d' ' -f1
}

# 获取已应用的迁移
get_applied_migrations() {
    export PGPASSWORD="$DB_PASSWORD"
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT version FROM schema_migrations ORDER BY version;" 2>/dev/null | tr -d ' ' | grep -v '^$' || true
}

# 获取待应用的迁移文件
get_pending_migrations() {
    local applied_migrations
    applied_migrations=$(get_applied_migrations)
    
    for migration_file in "$MIGRATIONS_DIR"/*.sql; do
        if [ -f "$migration_file" ]; then
            local filename
            filename=$(basename "$migration_file")
            local version
            version=${filename%%.sql}
            
            if ! echo "$applied_migrations" | grep -q "^$version$"; then
                echo "$migration_file"
            fi
        fi
    done | sort
}

# 执行单个迁移
apply_migration() {
    local migration_file="$1"
    local filename
    filename=$(basename "$migration_file")
    local version
    version=${filename%%.sql}
    local description
    description=$(head -n 20 "$migration_file" | grep -E '^--.*:.*' | head -n 1 | sed 's/^--//' | xargs || echo "No description")
    local checksum
    checksum=$(calculate_checksum "$migration_file")
    
    log_info "应用迁移: $filename"
    
    local start_time
    start_time=$(date +%s%3N)
    
    export PGPASSWORD="$DB_PASSWORD"
    
    # 在事务中执行迁移和记录
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 <<EOF
BEGIN;

-- 执行迁移脚本
\i $migration_file

-- 记录迁移信息
INSERT INTO schema_migrations (version, filename, description, checksum, execution_time_ms)
VALUES ('$version', '$filename', '$description', '$checksum', $(( $(date +%s%3N) - start_time )));

COMMIT;
EOF
    
    local execution_time=$(($(date +%s%3N) - start_time))
    log_success "迁移 $filename 应用成功 (耗时: ${execution_time}ms)"
}

# 执行所有待应用的迁移
migrate_up() {
    log_info "开始数据库迁移..."
    
    local pending_migrations
    pending_migrations=$(get_pending_migrations)
    
    if [ -z "$pending_migrations" ]; then
        log_info "没有待应用的迁移"
        return
    fi
    
    local migration_count
    migration_count=$(echo "$pending_migrations" | wc -l)
    log_info "发现 $migration_count 个待应用的迁移"
    
    while IFS= read -r migration_file; do
        apply_migration "$migration_file"
    done <<< "$pending_migrations"
    
    log_success "所有迁移应用成功！"
}

# 显示迁移状态
migration_status() {
    log_info "数据库迁移状态:"
    echo
    
    export PGPASSWORD="$DB_PASSWORD"
    
    # 检查迁移表是否存在
    if ! psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c '\dt schema_migrations' 2>/dev/null | grep -q schema_migrations; then
        log_warning "迁移跟踪表不存在，请先运行 setup 命令"
        return
    fi
    
    echo "已应用的迁移:"
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "
        SELECT 
            version,
            description,
            applied_at::timestamp(0) as applied_at,
            execution_time_ms || 'ms' as duration
        FROM schema_migrations 
        ORDER BY applied_at DESC
        LIMIT 10;
    " 2>/dev/null || log_error "查询迁移状态失败"
    
    echo
    local pending_migrations
    pending_migrations=$(get_pending_migrations)
    
    if [ -n "$pending_migrations" ]; then
        echo "待应用的迁移:"
        while IFS= read -r migration_file; do
            local filename
            filename=$(basename "$migration_file")
            echo "  - $filename"
        done <<< "$pending_migrations"
    else
        echo "✅ 所有迁移已应用"
    fi
}

# 验证迁移完整性
validate_migrations() {
    log_info "验证迁移文件完整性..."
    
    export PGPASSWORD="$DB_PASSWORD"
    local validation_failed=false
    
    # 检查已应用迁移的校验和
    while IFS='|' read -r version filename checksum; do
        local migration_file="$MIGRATIONS_DIR/$filename"
        
        if [ -f "$migration_file" ]; then
            local current_checksum
            current_checksum=$(calculate_checksum "$migration_file")
            
            if [ "$checksum" != "$current_checksum" ]; then
                log_error "迁移文件 $filename 已被修改 (校验和不匹配)"
                validation_failed=true
            fi
        else
            log_warning "迁移文件 $filename 不存在"
        fi
    done < <(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT version || '|' || filename || '|' || checksum FROM schema_migrations;" 2>/dev/null | grep -v '^$')
    
    if [ "$validation_failed" = true ]; then
        log_error "迁移完整性验证失败"
    else
        log_success "所有迁移文件完整性验证通过"
    fi
}

# 创建新的迁移文件
create_migration() {
    local description="$1"
    
    if [ -z "$description" ]; then
        log_error "请提供迁移描述"
    fi
    
    local timestamp
    timestamp=$(date +%Y%m%d%H%M%S)
    local filename="${timestamp}_${description// /_}.sql"
    local filepath="$MIGRATIONS_DIR/$filename"
    
    cat > "$filepath" << EOF
-- =====================================================================================
-- Migration: $description
-- Created: $(date '+%Y-%m-%d %H:%M:%S')
-- Version: $timestamp
-- =====================================================================================

BEGIN;

-- TODO: Add your migration SQL here

COMMIT;
EOF

    log_success "新迁移文件已创建: $filename"
    echo "文件路径: $filepath"
}

# 显示帮助信息
show_help() {
    cat << EOF
数据库迁移管理工具 - 几何原本 (Euclid Elements)

用法: $0 <command> [options]

命令:
    setup           设置数据库和迁移跟踪表
    up              执行所有待应用的迁移
    status          显示迁移状态
    validate        验证迁移文件完整性
    create <desc>   创建新的迁移文件

环境变量:
    DB_HOST         数据库主机 (默认: localhost)
    DB_PORT         数据库端口 (默认: 5432)
    DB_NAME         数据库名称 (默认: euclid_elements)
    DB_USER         数据库用户 (默认: postgres)
    DB_PASSWORD     数据库密码 (默认: password)

示例:
    $0 setup                    # 初始化数据库和迁移表
    $0 up                       # 执行所有迁移
    $0 status                   # 查看迁移状态
    $0 create add_user_table    # 创建新迁移
    $0 validate                 # 验证迁移完整性

EOF
}

# 主函数
main() {
    local command="${1:-}"
    
    case "$command" in
        "setup")
            check_db_connection
            setup_database
            ;;
        "up")
            check_db_connection
            setup_database
            migrate_up
            ;;
        "status")
            check_db_connection
            migration_status
            ;;
        "validate")
            check_db_connection
            validate_migrations
            ;;
        "create")
            local description="${2:-}"
            create_migration "$description"
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        "")
            log_error "请指定命令。使用 '$0 help' 查看帮助信息"
            ;;
        *)
            log_error "未知命令: $command。使用 '$0 help' 查看帮助信息"
            ;;
    esac
}

# 执行主函数
main "$@"