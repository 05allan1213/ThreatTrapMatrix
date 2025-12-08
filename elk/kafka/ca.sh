#!/bin/bash

# 创建证书目录（与 Docker 挂载目录一致）
mkdir -p kafka-certs && cd kafka-certs

# 生成 CA 证书
openssl genrsa -out ca.key 2048
openssl req -new -x509 -key ca.key -out ca.crt -days 3650 \
  -subj "/C=CN/ST=Beijing/L=Beijing/O=YourCompany/CN=kafka-ca"

# 生成 Broker 证书（含 SAN 扩展）
cat > san.cnf <<EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no

[req_distinguished_name]
C = CN
ST = Beijing
L = Beijing
O = YourCompany
CN = kafka-broker

[v3_req]
subjectAltName = @alt_names

[alt_names]
IP.1 = 192.168.5.130
DNS.1 = kafka-broker
IP.2 = 10.4.0.20
DNS.2 = kafka-broker
EOF

openssl genrsa -out kafka.server.key 2048
openssl req -new -key kafka.server.key -out kafka.server.csr -config san.cnf
openssl x509 -req -in kafka.server.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
  -out kafka.server.crt -days 3650 -extensions v3_req -extfile san.cnf

# 生成 JKS 格式证书（Bitnami 镜像专用）
openssl pkcs12 -export \
  -in kafka.server.crt -inkey kafka.server.key \
  -name kafka-ssl -out kafka.keystore.p12 \
  -password pass:yourpassword -certfile ca.crt

keytool -importkeystore \
  -srckeystore kafka.keystore.p12 -srcstoretype pkcs12 \
  -destkeystore kafka.keystore.jks -deststoretype JKS \
  -alias kafka-ssl -srcstorepass yourpassword -deststorepass yourpassword

# 生成 Truststore
keytool -keystore kafka.truststore.jks \
  -alias ca-cert -import -file ca.crt \
  -storepass yourpassword -noprompt

echo "证书已生成到 kafka-certs 目录，请挂载到容器的 /bitnami/kafka/config/certs"