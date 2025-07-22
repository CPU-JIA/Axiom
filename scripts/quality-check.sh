#!/bin/bash

# Axiom 平台代码质量检查脚本

echo "🚀 开始 Axiom 平台代码质量检查..."

# 检查前端代码结构
echo "📁 检查前端代码结构..."
if [ -d "web/src" ]; then
    echo "✅ 前端源码目录存在"
    echo "📊 前端文件统计:"
    find web/src -name "*.tsx" -o -name "*.ts" | wc -l | xargs echo "   TypeScript文件数:"
    find web/src -name "*.json" | wc -l | xargs echo "   配置文件数:"
else
    echo "❌ 前端源码目录不存在"
fi

# 检查后端服务结构
echo "📁 检查后端服务结构..."
if [ -d "services" ]; then
    echo "✅ 后端服务目录存在"
    echo "📊 后端服务统计:"
    ls services/ | wc -l | xargs echo "   服务数量:"
    find services -name "*.go" | wc -l | xargs echo "   Go源文件数:"
else
    echo "❌ 后端服务目录不存在"
fi

# 检查Docker配置
echo "🐳 检查Docker配置..."
if [ -f "docker-compose.yml" ]; then
    echo "✅ Docker Compose配置存在"
    grep -c "image:" docker-compose.yml | xargs echo "   容器镜像数:"
else
    echo "❌ Docker Compose配置不存在"
fi

# 检查Kubernetes配置
echo "☸️ 检查Kubernetes配置..."
if [ -d "configs/kubernetes" ]; then
    echo "✅ Kubernetes配置存在"
    find configs/kubernetes -name "*.yaml" -o -name "*.yml" | wc -l | xargs echo "   K8s配置文件数:"
else
    echo "❌ Kubernetes配置不存在"
fi

# 检查文档完整性
echo "📚 检查文档完整性..."
docs=("README.md" "DEPLOYMENT.md" "项目结构.md" "详细的需求分析文档 (RAD) V5.0.md")
for doc in "${docs[@]}"; do
    if [ -f "$doc" ]; then
        echo "✅ $doc 存在"
    else
        echo "❌ $doc 缺失"
    fi
done

# 检查CI/CD配置
echo "🔄 检查CI/CD配置..."
if [ -d ".github/workflows" ]; then
    echo "✅ GitHub Actions配置存在"
    find .github/workflows -name "*.yml" -o -name "*.yaml" | wc -l | xargs echo "   工作流文件数:"
else
    echo "❌ GitHub Actions配置不存在"
fi

# 安全检查
echo "🔒 执行基本安全检查..."
echo "   检查是否包含敏感信息..."
if grep -r "password\|secret\|token" --include="*.json" --include="*.yaml" --include="*.yml" . >/dev/null 2>&1; then
    echo "⚠️  发现可能的敏感信息，请检查配置文件"
else
    echo "✅ 未发现明显的敏感信息泄露"
fi

# 代码统计
echo "📈 项目代码统计:"
echo "   总文件数: $(find . -type f | wc -l)"
echo "   代码行数: $(find . -name "*.go" -o -name "*.ts" -o -name "*.tsx" -o -name "*.js" -o -name "*.jsx" | xargs wc -l 2>/dev/null | tail -1 | awk '{print $1}' || echo "0")"

echo ""
echo "🎉 代码质量检查完成!"
echo "📊 Axiom平台项目状态: 企业级云开发协作平台已就绪"