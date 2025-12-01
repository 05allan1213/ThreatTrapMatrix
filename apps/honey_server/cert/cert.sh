#!/bin/bash

# 确保使用bash执行
if [ -z "$BASH_VERSION" ]; then
    echo "请使用bash执行此脚本: bash $0"
    exit 1
fi

# 设置证书有效期（天）
DAYS=3650

# 清理旧证书文件
rm -f ca.key ca.crt ca.srl
rm -f server.key server.csr server.crt
rm -f client.key client.csr client.crt

# 生成CA私钥和自签名证书
openssl genrsa -out ca.key 2048
openssl req -new -x509 -key ca.key -out ca.crt -days $DAYS \
    -subj "/C=CN/ST=Beijing/L=Beijing/O=MyOrg/OU=IT/CN=MyCA"

# 生成服务端私钥和证书
openssl genrsa -out server.key 2048
openssl req -new -key server.key -out server.csr \
    -subj "/C=CN/ST=Beijing/L=Beijing/O=MyOrg/OU=IT/CN=localhost"

# 创建临时扩展文件
echo "subjectAltName=DNS:localhost,DNS:example.com,IP:127.0.0.1,IP:192.168.56.1,IP:192.168.5.1,IP:192.168.5.130" > server.ext

openssl x509 -req -sha256 -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
    -out server.crt -days $DAYS -extfile server.ext

rm -f server.ext

# 生成客户端私钥和证书
openssl genrsa -out client.key 2048
openssl req -new -key client.key -out client.csr \
    -subj "/C=CN/ST=Beijing/L=Beijing/O=MyOrg/OU=IT/CN=MyClient"
openssl x509 -req -sha256 -in client.csr -CA ca.crt -CAkey ca.key \
    -out client.crt -days $DAYS

# 设置文件权限
chmod 600 *.key

# 显示生成结果
echo "证书生成完成:"
ls -l *.key *.crt *.csr