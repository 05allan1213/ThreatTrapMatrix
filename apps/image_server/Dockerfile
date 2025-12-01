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

FROM docker:cli AS dockercli

# 使用轻量级的 Alpine Linux 作为最终镜像
FROM alpine

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk add tzdata
# 设置工作目录
WORKDIR /app

# 从dockercli阶段复制Docker二进制文件和依赖
COPY --from=dockercli /usr/local/bin/docker /usr/local/bin/
# 安装Docker依赖
RUN apk add --no-cache ca-certificates iptables ip6tables

# 复制配置文件和证书
COPY settings.yaml /app
# 从构建器镜像中复制可执行文件
COPY --from=builder /app/main /app

# 启动应用程序
CMD ["./main"]

# docker build -t image_server:v1 .
# docker run -itd --name image_server -p 8080:8080 -v /var/run/docker.sock:/var/run/docker.sock image_server:v1