# ==========================================
# 阶段 1: 镜像编译构建环境 (Builder)
# ==========================================
FROM golang:1.24-alpine AS builder

WORKDIR /build

# 安装基础构建依赖
RUN apk add --no-cache git ca-certificates tzdata

# 预先复制依赖声明文件以利用 Docker 缓存层
COPY go.mod go.sum ./
RUN go mod download

# 复制 Go 后端全量源码
COPY . .

# 编译核心 API 服务二进制 (禁用 CGO，构建完全静态链接的极简二进制)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /build/bin/api-server ./cmd/api

# 编译初始题库注入工具 (Seed)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /build/bin/seed ./cmd/seed

# 编译 E2E 集成测试探针工具 (用于集群内冒烟探活)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /build/bin/e2e ./cmd/e2e

# ==========================================
# 阶段 2: 极简生产运行时环境 (Runtime)
# ==========================================
FROM alpine:3.20

WORKDIR /app

# 安装根证书、curl 与时区数据
RUN apk add --no-cache ca-certificates tzdata curl \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone

# 创建非 root 运行用户，增强容器安全
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# 从构建阶段拷贝预编译的二进制文件
COPY --from=builder /build/bin/api-server /app/api-server
COPY --from=builder /build/bin/seed /app/seed
COPY --from=builder /build/bin/e2e /app/e2e

# 拷贝默认配置与示例
COPY configs/ /app/configs/

# 赋予可执行权限并切换所有权
RUN chown -R appuser:appgroup /app && chmod +x /app/api-server /app/seed /app/e2e

USER appuser

# 暴露后端 API 服务端口
EXPOSE 8080

# 容器原生健康检查
HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://127.0.0.1:8080/health || exit 1

# 默认启动核心 API 服务
ENTRYPOINT ["/app/api-server"]
