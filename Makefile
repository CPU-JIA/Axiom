# 云平台开发协作平台 - 统一构建文件
.PHONY: help dev-up dev-down build test lint security-scan generate clean db-setup db-migrate db-status db-test

# 默认目标
.DEFAULT_GOAL := help

# 项目配置
PROJECT_NAME := euclid-elements
DOCKER_REGISTRY := your-registry.com
VERSION := $(shell git describe --tags --always --dirty)

# Go相关配置
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
GO_VERSION := 1.21
GOLANGCI_LINT_VERSION := v1.54.2

# 数据库配置
DB_HOST := localhost
DB_PORT := 5432
DB_NAME := euclid_elements
DB_USER := postgres
DB_PASSWORD := password

# 服务列表
SERVICES := iam-service tenant-service project-service git-gateway-service cicd-service notification-service kb-service api-gateway

help: ## 显示帮助信息
	@echo "云平台协作开发平台 - 构建工具"
	@echo ""
	@echo "可用命令："
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## 开发环境管理
dev-up: ## 启动开发环境 (所有基础设施服务)
	@echo "🚀 启动开发环境..."
	docker-compose up -d
	@echo "✅ 开发环境启动完成!"
	@echo "📊 服务状态检查:"
	@make dev-status

dev-down: ## 停止开发环境
	@echo "🛑 停止开发环境..."
	docker-compose down

dev-status: ## 检查开发环境状态
	@echo "📊 开发环境服务状态:"
	@docker-compose ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}"

infra-up: ## 仅启动基础设施 (数据库、缓存等)
	@echo "🔧 启动基础设施服务..."
	docker-compose up -d postgres redis kafka zookeeper minio vault

logs: ## 查看所有服务日志
	docker-compose logs -f

logs-%: ## 查看特定服务日志 (如: make logs-postgres)
	docker-compose logs -f $*

## 代码构建
build: ## 构建所有微服务
	@echo "🔨 构建所有微服务..."
	@for service in $(SERVICES); do \
		echo "构建 $$service..."; \
		cd services/$$service && go build -o bin/$$service ./cmd/main.go; \
		cd ../..; \
	done
	@echo "✅ 构建完成!"

build-%: ## 构建特定服务 (如: make build-iam-service)
	@echo "🔨 构建 $*..."
	@cd services/$* && go build -o bin/$* ./cmd/main.go

docker-build: ## 构建Docker镜像
	@echo "🐳 构建Docker镜像..."
	@for service in $(SERVICES); do \
		echo "构建 $$service 镜像..."; \
		docker build -t $(DOCKER_REGISTRY)/$$service:$(VERSION) services/$$service/; \
	done

docker-push: docker-build ## 推送Docker镜像
	@echo "📤 推送Docker镜像..."
	@for service in $(SERVICES); do \
		docker push $(DOCKER_REGISTRY)/$$service:$(VERSION); \
	done

## 代码质量
lint: ## 运行代码检查
	@echo "🔍 运行代码质量检查..."
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "安装 golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
	fi
	@for service in $(SERVICES); do \
		echo "检查 $$service..."; \
		cd services/$$service && golangci-lint run; \
		cd ../..; \
	done

fmt: ## 格式化代码
	@echo "🎨 格式化Go代码..."
	@find services -name "*.go" -exec gofmt -w {} \;
	@find services -name "*.go" -exec goimports -w {} \;

## 测试
test: ## 运行所有测试
	@echo "🧪 运行单元测试..."
	@for service in $(SERVICES); do \
		echo "测试 $$service..."; \
		cd services/$$service && go test -v -race -coverprofile=coverage.out ./...; \
		cd ../..; \
	done

test-%: ## 运行特定服务测试
	@echo "🧪 测试 $*..."
	@cd services/$* && go test -v -race -coverprofile=coverage.out ./...

test-integration: ## 运行集成测试
	@echo "🔧 运行集成测试..."
	@cd tests/integration && go test -v ./...

test-e2e: dev-up ## 运行端到端测试
	@echo "🎭 运行E2E测试..."
	@cd tests/e2e && npm test

coverage: ## 生成测试覆盖率报告
	@echo "📊 生成测试覆盖率报告..."
	@for service in $(SERVICES); do \
		cd services/$$service && go tool cover -html=coverage.out -o coverage.html; \
		cd ../..; \
	done

## 安全检查
security-scan: ## 运行安全扫描
	@echo "🔒 运行安全扫描..."
	@if ! command -v gosec &> /dev/null; then \
		go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
	fi
	@for service in $(SERVICES); do \
		echo "安全扫描 $$service..."; \
		cd services/$$service && gosec ./...; \
		cd ../..; \
	done

dependency-check: ## 检查依赖漏洞
	@echo "📦 检查Go模块安全漏洞..."
	@for service in $(SERVICES); do \
		echo "检查 $$service 依赖..."; \
		cd services/$$service && go list -json -deps ./... | nancy sleuth; \
		cd ../..; \
	done

## 代码生成
generate: ## 生成代码 (protobuf, 数据库模型等)
	@echo "⚙️ 生成代码..."
	@echo "生成 protobuf..."
	@cd shared/proto && buf generate
	@echo "生成数据库模型..."
	@cd shared/database && sqlc generate
	@echo "✅ 代码生成完成!"

proto-gen: ## 仅生成protobuf代码
	@echo "📡 生成 gRPC protobuf 代码..."
	@cd shared/proto && buf generate

## 数据库操作
db-setup: ## 设置数据库和迁移表
	@echo "🗃️ 设置数据库环境..."
	@export DB_HOST=$(DB_HOST) DB_PORT=$(DB_PORT) DB_NAME=$(DB_NAME) DB_USER=$(DB_USER) DB_PASSWORD=$(DB_PASSWORD) && \
	cd migrations && ./migrate.sh setup

db-migrate: ## 运行数据库迁移
	@echo "🗃️ 运行数据库迁移..."
	@export DB_HOST=$(DB_HOST) DB_PORT=$(DB_PORT) DB_NAME=$(DB_NAME) DB_USER=$(DB_USER) DB_PASSWORD=$(DB_PASSWORD) && \
	cd migrations && ./migrate.sh up

db-status: ## 查看数据库迁移状态
	@echo "📊 检查数据库迁移状态..."
	@export DB_HOST=$(DB_HOST) DB_PORT=$(DB_PORT) DB_NAME=$(DB_NAME) DB_USER=$(DB_USER) DB_PASSWORD=$(DB_PASSWORD) && \
	cd migrations && ./migrate.sh status

db-validate: ## 验证数据库Schema
	@echo "✅ 验证数据库Schema..."
	@export PGPASSWORD=$(DB_PASSWORD) && \
	psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME) -f migrations/validate_schema.sql

db-test: ## 测试数据库连接
	@echo "🔍 测试数据库连接..."
	@export DB_HOST=$(DB_HOST) DB_PORT=$(DB_PORT) DB_NAME=$(DB_NAME) DB_USER=$(DB_USER) DB_PASSWORD=$(DB_PASSWORD) && \
	cd migrations && ./test_db.sh

db-create-migration: ## 创建新的迁移文件 (使用: make db-create-migration DESC="描述")
	@if [ -z "$(DESC)" ]; then echo "❌ 请提供迁移描述: make db-create-migration DESC='添加用户表'"; exit 1; fi
	@cd migrations && ./migrate.sh create "$(DESC)"
	@echo "✅ 迁移文件已创建: $(DESC)"

db-reset: ## 重置数据库 (危险操作!)
	@echo "⚠️  重置数据库 - 将删除所有数据!"
	@read -p "确定要继续吗? [y/N] " confirm && [ "$$confirm" = "y" ] || exit 1
	@export PGPASSWORD=$(DB_PASSWORD) && \
	psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d postgres -c "DROP DATABASE IF EXISTS $(DB_NAME);" && \
	psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d postgres -c "CREATE DATABASE $(DB_NAME);" && \
	make db-migrate

db-backup: ## 备份数据库
	@echo "💾 备份数据库..."
	@export PGPASSWORD=$(DB_PASSWORD) && \
	pg_dump -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME) --clean --if-exists > backups/$(DB_NAME)_$(shell date +%Y%m%d_%H%M%S).sql
	@echo "✅ 数据库备份完成!"

db-restore: ## 恢复数据库 (使用: make db-restore FILE=backup.sql)
	@if [ -z "$(FILE)" ]; then echo "❌ 请提供备份文件: make db-restore FILE=backup.sql"; exit 1; fi
	@echo "🔄 恢复数据库从: $(FILE)"
	@export PGPASSWORD=$(DB_PASSWORD) && \
	psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME) < $(FILE)
	@echo "✅ 数据库恢复完成!"

## 前端相关
web-install: ## 安装前端依赖
	@echo "📦 安装前端依赖..."
	@cd web && npm install

web-build: web-install ## 构建前端应用
	@echo "🏗️ 构建前端应用..."
	@cd web && npm run build

web-dev: web-install ## 启动前端开发服务器
	@echo "🖥️ 启动前端开发服务器..."
	@cd web && npm run dev

web-test: ## 运行前端测试
	@echo "🧪 运行前端测试..."
	@cd web && npm run test

## 部署相关
deploy: ## 快速部署整个平台
	@echo "🚀 部署几何原本云端开发协作平台..."
	@chmod +x deploy.sh
	@./deploy.sh

stop: ## 停止平台服务
	@echo "🛑 停止平台服务..."
	@chmod +x stop.sh
	@./stop.sh

stop-clean: ## 停止并清理资源
	@echo "🧹 停止并清理资源..."
	@chmod +x stop.sh
	@./stop.sh --clean

backup: ## 备份平台数据
	@echo "💾 备份平台数据..."
	@chmod +x stop.sh
	@./stop.sh --backup

full-reset: ## 完全重置平台 (危险操作!)
	@echo "⚠️  完全重置平台..."
	@chmod +x stop.sh
	@./stop.sh --full

k8s-deploy: ## 部署到Kubernetes
	@echo "☸️ 部署到Kubernetes..."
	@cd configs/kubernetes && kubectl apply -f .

helm-install: ## 使用Helm安装
	@echo "⛵ 使用Helm安装..."
	@cd configs/helm && helm install cloud-platform ./cloud-platform

terraform-apply: ## 应用Terraform配置
	@echo "🌍 应用基础设施配置..."
	@cd configs/terraform && terraform apply

## 清理
clean: ## 清理构建产物
	@echo "🧹 清理构建产物..."
	@find . -name "bin" -type d -exec rm -rf {} + 2>/dev/null || true
	@find . -name "coverage.out" -delete 2>/dev/null || true
	@find . -name "coverage.html" -delete 2>/dev/null || true
	@docker system prune -f

clean-data: ## 清理Docker数据卷 (危险操作!)
	@echo "⚠️  清理所有Docker数据卷..."
	@read -p "这将删除所有数据，确认请输入 'yes': " confirm && [ "$$confirm" = "yes" ]
	@docker-compose down -v
	@docker volume prune -f

## 工具安装
install-tools: ## 安装必要的开发工具
	@echo "🛠️ 安装开发工具..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	@go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	@go install github.com/sonatype-nexus-community/nancy@latest
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@go install github.com/kyleconroy/sqlc/cmd/sqlc@latest
	@echo "✅ 开发工具安装完成!"

## 快速命令
quick-start: install-tools dev-up generate ## 快速开始 (安装工具 + 启动环境 + 生成代码)
	@echo "🎉 开发环境已就绪!"
	@echo ""
	@echo "🔗 服务访问地址:"
	@echo "  - PostgreSQL: localhost:5432"
	@echo "  - Redis: localhost:6379"
	@echo "  - Kafka: localhost:9092"
	@echo "  - Gitea: http://localhost:3000"
	@echo "  - MinIO: http://localhost:9001"
	@echo "  - Vault: http://localhost:8200"
	@echo "  - Prometheus: http://localhost:9090"
	@echo "  - Grafana: http://localhost:3001 (admin/admin123)"
	@echo "  - Jaeger: http://localhost:16686"
	@echo ""
	@echo "📚 下一步:"
	@echo "  make build     # 构建服务"
	@echo "  make test      # 运行测试"
	@echo "  make web-dev   # 启动前端"