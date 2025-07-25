# 构建阶段
FROM golang:1.21-alpine AS builder

WORKDIR /app

# 安装必要的包
RUN apk add --no-cache git ca-certificates tzdata

# 复制go mod文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/

# 运行阶段
FROM alpine:latest

# 安装ca-certificates和tzdata
RUN apk --no-cache add ca-certificates tzdata

# 创建非root用户
RUN addgroup -g 1000 appuser && \
    adduser -D -s /bin/sh -u 1000 -G appuser appuser

WORKDIR /app

# 创建必要的目录
RUN mkdir -p /data/cicd /data/cache && \
    chown -R appuser:appuser /data

# 从构建阶段复制二进制文件
COPY --from=builder /app/main .
COPY --from=builder /app/configs ./configs

# 切换到非root用户
USER appuser

# 暴露端口
EXPOSE 8005

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8005/health || exit 1

# 启动应用
CMD ["./main"]