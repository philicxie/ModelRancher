# Backend Dockerfile - 本地CPU执行版本
FROM golang:1.21-alpine AS builder

WORKDIR /app

# 复制依赖文件
COPY go.mod go.sum* ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 编译
RUN CGO_ENABLED=0 GOOS=linux go build -o ml-platform ./cmd/server

# 运行镜像
FROM alpine:latest

WORKDIR /app

# 安装必要的工具
RUN apk add --no-cache docker-cli bash curl

# 复制二进制文件
COPY --from=builder /app/ml-platform .

# 创建数据目录
RUN mkdir -p /app/data /app/output

ENV PORT=8080
ENV DATA_DIR=/app/data
ENV OUTPUT_DIR=/app/output

EXPOSE 8080

# 注意：此容器需要挂载宿主机的Docker socket来执行Docker命令
# 运行命令: docker run -v /var/run/docker.sock:/var/run/docker.sock ml-platform-backend

CMD ["./ml-platform"]