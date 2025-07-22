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
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api-gateway ./cmd

# 运行阶段
FROM alpine:latest

# 安装ca-certificates（HTTPS请求所需）
RUN apk --no-cache add ca-certificates tzdata wget

# 创建非root用户
RUN adduser -D -s /bin/sh gateway

# 设置工作目录
WORKDIR /app

# 创建静态文件目录
RUN mkdir -p static

# 从构建阶段复制二进制文件
COPY --from=builder /app/api-gateway .

# 更改所有者为非root用户
RUN chown -R gateway:gateway /app
USER gateway

# 暴露端口
EXPOSE 8000 9090

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8000/health || exit 1

# 运行应用程序
CMD ["./api-gateway"]