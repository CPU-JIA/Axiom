#!/bin/bash

# CI/CD服务API测试脚本

BASE_URL="http://localhost:8005"
API_URL="$BASE_URL/api/v1"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 测试用的JWT Token（开发环境用）
JWT_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMTExMTExMTEtMjIyMi0zMzMzLTQ0NDQtNTU1NTU1NTU1NTU1IiwidGVuYW50X2lkIjoiMTExMTExMTEtMjIyMi0zMzMzLTQ0NDQtNTU1NTU1NTU1NTU1IiwiZW1haWwiOiJ0ZXN0QGV4YW1wbGUuY29tIiwicm9sZSI6ImFkbWluIiwiaXNzIjoiZXVjbGlkLWVsZW1lbnRzIiwiZXhwIjo5OTk5OTk5OTk5fQ.example"

# 测试项目ID
PROJECT_ID="01234567-89ab-cdef-0123-456789abcdef"

print_section() {
    echo -e "\n${BLUE}==================== $1 ====================${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

# 检查服务健康状态
check_health() {
    print_section "健康检查"
    
    print_info "检查服务状态..."
    response=$(curl -s -w "\n%{http_code}" "$BASE_URL/health")
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n -1)
    
    if [[ $http_code -eq 200 ]]; then
        print_success "服务健康检查通过"
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
    else
        print_error "服务健康检查失败 (HTTP $http_code)"
        echo "$body"
        exit 1
    fi
    
    print_info "检查存活探针..."
    response=$(curl -s "$BASE_URL/health/live")
    echo "$response" | jq '.' 2>/dev/null || echo "$response"
    
    print_info "检查就绪探针..."
    response=$(curl -s "$BASE_URL/health/ready")
    echo "$response" | jq '.' 2>/dev/null || echo "$response"
}

# 测试流水线CRUD
test_pipeline_crud() {
    print_section "流水线CRUD测试"
    
    # 创建流水线
    print_info "创建流水线..."
    create_response=$(curl -s -X POST "$API_URL/pipelines" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -H "Content-Type: application/json" \
        -d '{
            "project_id": "'$PROJECT_ID'",
            "name": "测试流水线",
            "description": "API测试用流水线",
            "config": {
                "workspace": "default",
                "timeout": 1800,
                "service_account": "default"
            },
            "triggers": [
                {
                    "type": "webhook",
                    "conditions": {"branch": "main"},
                    "enabled": true
                }
            ],
            "variables": {
                "ENV": "test",
                "DEBUG": "true"
            },
            "tasks": [
                {
                    "name": "构建任务",
                    "description": "编译和构建应用",
                    "type": "build",
                    "image": "golang:1.21-alpine",
                    "command": ["go"],
                    "args": ["build", "-o", "app", "./cmd/"],
                    "env": {
                        "CGO_ENABLED": "0"
                    },
                    "order": 1,
                    "timeout": 600,
                    "retries": 2
                },
                {
                    "name": "测试任务", 
                    "description": "运行单元测试",
                    "type": "test",
                    "image": "golang:1.21-alpine",
                    "command": ["go"],
                    "args": ["test", "./..."],
                    "depends_on": ["构建任务"],
                    "order": 2,
                    "timeout": 300,
                    "retries": 1
                }
            ]
        }')
    
    if echo "$create_response" | jq -e '.success' > /dev/null 2>&1; then
        pipeline_id=$(echo "$create_response" | jq -r '.data.id')
        print_success "流水线创建成功: $pipeline_id"
    else
        print_error "流水线创建失败"
        echo "$create_response" | jq '.' 2>/dev/null || echo "$create_response"
        return 1
    fi
    
    # 获取流水线详情
    print_info "获取流水线详情..."
    get_response=$(curl -s "$API_URL/pipelines/$pipeline_id" \
        -H "Authorization: Bearer $JWT_TOKEN")
    
    if echo "$get_response" | jq -e '.success' > /dev/null 2>&1; then
        print_success "流水线详情获取成功"
        echo "$get_response" | jq '.data.name, .data.status' 2>/dev/null
    else
        print_error "获取流水线详情失败"
        echo "$get_response"
    fi
    
    # 列表查询流水线
    print_info "查询流水线列表..."
    list_response=$(curl -s "$API_URL/pipelines?project_id=$PROJECT_ID&limit=5" \
        -H "Authorization: Bearer $JWT_TOKEN")
        
    if echo "$list_response" | jq -e '.success' > /dev/null 2>&1; then
        count=$(echo "$list_response" | jq '.data.pagination.total')
        print_success "流水线列表查询成功，共 $count 个流水线"
    else
        print_error "流水线列表查询失败"
        echo "$list_response"
    fi
    
    # 触发流水线执行
    print_info "触发流水线执行..."
    trigger_response=$(curl -s -X POST "$API_URL/pipelines/$pipeline_id/trigger" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -H "Content-Type: application/json" \
        -d '{
            "trigger_type": "manual",
            "parameters": {
                "branch": "main",
                "commit_id": "abc123"
            }
        }')
    
    if echo "$trigger_response" | jq -e '.success' > /dev/null 2>&1; then
        run_id=$(echo "$trigger_response" | jq -r '.data.id')
        print_success "流水线触发成功: $run_id"
        
        # 获取运行详情
        print_info "获取运行详情..."
        sleep 1
        run_response=$(curl -s "$API_URL/pipeline-runs/$run_id" \
            -H "Authorization: Bearer $JWT_TOKEN")
        
        if echo "$run_response" | jq -e '.success' > /dev/null 2>&1; then
            status=$(echo "$run_response" | jq -r '.data.status')
            print_success "运行状态: $status"
        fi
    else
        print_error "流水线触发失败"
        echo "$trigger_response"
    fi
    
    export PIPELINE_ID=$pipeline_id
}

# 测试缓存功能
test_cache() {
    print_section "构建缓存测试"
    
    # 模拟存储缓存
    print_info "存储构建缓存..."
    cache_response=$(curl -s -X POST "$API_URL/cache" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -H "Content-Type: application/json" \
        -d '{
            "project_id": "'$PROJECT_ID'",
            "key": "test-cache-key",
            "source_path": "/tmp/test-file",
            "metadata": {
                "type": "build",
                "version": "1.0.0"
            },
            "ttl_hours": 24
        }')
    
    if echo "$cache_response" | jq -e '.success' > /dev/null 2>&1; then
        print_success "缓存存储请求已提交"
    else
        print_error "缓存存储失败"
        echo "$cache_response"
    fi
    
    # 查询缓存列表
    print_info "查询缓存列表..."
    list_response=$(curl -s "$API_URL/cache?project_id=$PROJECT_ID" \
        -H "Authorization: Bearer $JWT_TOKEN")
        
    if echo "$list_response" | jq -e '.success' > /dev/null 2>&1; then
        count=$(echo "$list_response" | jq '.data.pagination.total')
        print_success "缓存列表查询成功，共 $count 个缓存"
    else
        print_error "缓存列表查询失败"  
        echo "$list_response"
    fi
    
    # 获取缓存统计
    print_info "获取缓存统计..."
    stats_response=$(curl -s "$API_URL/cache/statistics?project_id=$PROJECT_ID" \
        -H "Authorization: Bearer $JWT_TOKEN")
        
    if echo "$stats_response" | jq -e '.success' > /dev/null 2>&1; then
        total_caches=$(echo "$stats_response" | jq '.data.total_caches')
        total_size=$(echo "$stats_response" | jq '.data.total_size')
        print_success "缓存统计: $total_caches 个缓存，总大小 $total_size 字节"
    else
        print_error "缓存统计查询失败"
        echo "$stats_response"
    fi
}

# 测试统计API
test_statistics() {
    print_section "统计信息测试"
    
    # 流水线统计
    print_info "获取流水线统计..."
    pipeline_stats=$(curl -s "$API_URL/pipelines/statistics?project_id=$PROJECT_ID" \
        -H "Authorization: Bearer $JWT_TOKEN")
        
    if echo "$pipeline_stats" | jq -e '.success' > /dev/null 2>&1; then
        total=$(echo "$pipeline_stats" | jq '.data.total_pipelines')
        active=$(echo "$pipeline_stats" | jq '.data.active_pipelines')
        print_success "流水线统计: 总计 $total 个，活跃 $active 个"
    else
        print_error "流水线统计获取失败"
        echo "$pipeline_stats"
    fi
    
    # 运行统计
    print_info "获取运行统计..."
    run_stats=$(curl -s "$API_URL/pipeline-runs/statistics?project_id=$PROJECT_ID" \
        -H "Authorization: Bearer $JWT_TOKEN")
        
    if echo "$run_stats" | jq -e '.success' > /dev/null 2>&1; then
        total_runs=$(echo "$run_stats" | jq '.data.total_runs')
        success_rate=$(echo "$run_stats" | jq '.data.success_rate')
        print_success "运行统计: 总计 $total_runs 次运行，成功率 $success_rate%"
    else
        print_error "运行统计获取失败"
        echo "$run_stats"
    fi
}

# 清理测试数据
cleanup() {
    if [[ -n "$PIPELINE_ID" ]]; then
        print_section "清理测试数据"
        print_info "删除测试流水线..."
        
        delete_response=$(curl -s -X DELETE "$API_URL/pipelines/$PIPELINE_ID" \
            -H "Authorization: Bearer $JWT_TOKEN")
            
        if echo "$delete_response" | jq -e '.success' > /dev/null 2>&1; then
            print_success "测试流水线已删除"
        else
            print_warning "测试流水线删除失败，请手动清理"
        fi
    fi
}

# 主测试流程
main() {
    echo -e "${BLUE}🧪 CI/CD服务API测试${NC}"
    echo -e "${BLUE}测试地址: $BASE_URL${NC}\n"
    
    # 检查依赖
    if ! command -v curl &> /dev/null; then
        print_error "curl命令不存在，请安装curl"
        exit 1
    fi
    
    if ! command -v jq &> /dev/null; then
        print_warning "jq命令不存在，输出格式可能不美观"
    fi
    
    # 执行测试
    check_health
    test_pipeline_crud
    test_cache
    test_statistics
    
    # 清理
    cleanup
    
    print_section "测试完成"
    print_success "所有API测试已完成！"
}

# 脚本入口
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi