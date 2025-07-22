# 构建阶段
FROM golang:1.21-alpine AS builder

# 安装必要的包
RUN apk add --no-cache git ca-certificates tzdata

# 设置工作目录
WORKDIR /app

# 复制go.mod和go.sum
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用程序
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o git-gateway-service ./cmd

# 运行阶段
FROM alpine:latest

# 安装ca-certificates和git（HTTPS请求和Git操作所需）
RUN apk --no-cache add ca-certificates tzdata git openssh

# 创建非root用户
RUN adduser -D -s /bin/sh appuser

# 创建数据目录
RUN mkdir -p /data/repositories /data/lfs && chown -R appuser:appuser /data

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/git-gateway-service .
COPY --from=builder /app/config.yaml .

# 更改所有者为非root用户
RUN chown -R appuser:appuser /app
USER appuser

# 暴露端口
EXPOSE 8004

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8004/health || exit 1

# 运行应用程序
CMD ["./git-gateway-service"]