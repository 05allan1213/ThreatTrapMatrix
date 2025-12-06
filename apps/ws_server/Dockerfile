# 使用带有静态链接库的 Go 镜像作为构建器
FROM golang:alpine AS builder

# 设置环境变量，禁用 CGO，确保静态编译
ENV CGO_ENABLED=1
ENV GOPROXY=https://goproxy.cn,direct
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk add --no-cache build-base
# 设置工作目录
WORKDIR /app
ADD . .
RUN go mod tidy
RUN go build -o main

# 使用轻量级的 Alpine Linux 作为最终镜像
FROM alpine

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk add tzdata
# 设置工作目录
WORKDIR /app

# 复制配置文件和证书
COPY settings.yaml /app
# 从构建器镜像中复制可执行文件
COPY --from=builder /app/main /app

# 启动应用程序
CMD ["./main"]

# docker build -t ws_server:v1.0.1 .