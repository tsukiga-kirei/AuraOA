# 构建上下文须为仓库根目录：docker build -f go-service/Dockerfile .
#
# 若 go mod download 访问 proxy.golang.org 超时（国内常见），默认走 goproxy 镜像；可覆盖：
#   docker build --build-arg GOPROXY=https://proxy.golang.org,direct -f go-service/Dockerfile .
# Stage 1: Build
FROM golang:1.25-alpine AS builder

ARG GOPROXY=https://goproxy.cn,https://goproxy.io,https://proxy.golang.org,direct
ARG GOSUMDB=sum.golang.google.cn
ENV GOPROXY=${GOPROXY}
ENV GOSUMDB=${GOSUMDB}

RUN apk add --no-cache git

WORKDIR /src/go-service

COPY go-service/go.mod go-service/go.sum ./
RUN go mod download

COPY go-service/ ./

RUN mkdir -p /out && CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server/main.go

# Stage 2: Runtime
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /out/server .
COPY go-service/config.yaml .
COPY db/migrations /migrations

# 预建日志目录，容器内默认写入此路径（可通过 LOG_DIR 环境变量覆盖）
RUN mkdir -p /app/logs/tenants

ENV MIGRATIONS_PATH=/migrations
# 日志默认配置（可通过环境变量覆盖，对应 config.yaml 的 log.* 字段）
ENV LOG_LEVEL=info
ENV LOG_DIR=/app/logs
ENV LOG_MAX_SIZE_MB=100
ENV LOG_MAX_BACKUPS=5
ENV LOG_COMPRESS=true
ENV LOG_GLOBAL_RETENTION_DAYS=30

EXPOSE 8080

CMD ["./server"]
