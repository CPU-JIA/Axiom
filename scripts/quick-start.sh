#!/bin/bash

echo "🚀 Axiom平台一键启动脚本"
echo "=========================="

# 检查Docker是否运行
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker未运行，请先启动Docker Desktop"
    exit 1
fi

echo "✅ Docker运行正常"

# 检查项目是否存在
if [ ! -d "web" ] || [ ! -d "services" ]; then
    echo "📥 项目不存在，正在克隆..."
    git clone https://github.com/CPU-JIA/Axiom.git
    cd Axiom
fi

echo "📁 项目目录确认"

# 启动服务
echo "🚀 正在启动Axiom平台..."
docker-compose up -d

# 等待服务启动
echo "⏳ 等待服务启动..."
sleep 30

# 检查服务状态
echo "📊 检查服务状态..."
docker-compose ps

echo ""
echo "🎉 Axiom平台启动完成！"
echo "=========================="
echo "🌐 访问地址："
echo "  主平台: http://localhost:3000"
echo "  API网关: http://localhost:8000"
echo "  监控面板: http://localhost:3001"
echo ""
echo "🔑 默认登录:"
echo "  用户名: admin"
echo "  密码: admin123"
echo ""
echo "📚 更多信息请查看: ACCESS_GUIDE.md"