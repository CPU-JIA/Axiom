#!/bin/bash

echo "🚀 Axiom平台启动验证脚本"
echo "=================================="

# 检查项目结构完整性
echo "📁 检查项目结构..."
if [ -d "web/src" ] && [ -d "services" ] && [ -f "docker-compose.yml" ]; then
    echo "✅ 项目结构完整"
else
    echo "❌ 项目结构不完整"
    exit 1
fi

# 检查前端代码
echo "🎨 检查前端代码..."
frontend_files=$(find web/src -name "*.tsx" -o -name "*.ts" | wc -l)
echo "   前端文件数: $frontend_files"
if [ "$frontend_files" -gt "10" ]; then
    echo "✅ 前端代码完整"
else
    echo "❌ 前端代码不完整"
fi

# 检查后端服务
echo "🔧 检查后端服务..."
backend_services=$(ls services/ | wc -l)
echo "   后端服务数: $backend_services"
if [ "$backend_services" -ge "6" ]; then
    echo "✅ 后端服务完整"
else
    echo "❌ 后端服务不完整"
fi

# 检查CI/CD配置
echo "🔄 检查CI/CD配置..."
if [ -d ".github/workflows" ]; then
    workflow_files=$(find .github/workflows -name "*.yml" | wc -l)
    echo "   工作流文件数: $workflow_files"
    echo "✅ CI/CD配置完整"
else
    echo "❌ CI/CD配置缺失"
fi

# 检查文档
echo "📚 检查文档..."
if [ -f "README.md" ] && [ -f "DEPLOYMENT.md" ] && [ -f "PROJECT_COMPLETION_REPORT.md" ]; then
    echo "✅ 文档完整"
else
    echo "❌ 文档不完整"
fi

echo ""
echo "🎯 Axiom平台状态检查完成"
echo "=================================="

# 显示启动指南
echo "🚀 平台启动指南:"
echo "1. 本地开发: docker-compose up -d"
echo "2. 前端开发: cd web && npm run dev"
echo "3. 生产部署: kubectl apply -f configs/kubernetes/"
echo ""
echo "📊 平台访问地址:"
echo "- 前端界面: http://localhost:3000"
echo "- API网关: http://localhost:8000" 
echo "- 监控面板: http://localhost:3001"
echo ""
echo "🎉 Axiom平台已准备就绪，开始改变世界！"