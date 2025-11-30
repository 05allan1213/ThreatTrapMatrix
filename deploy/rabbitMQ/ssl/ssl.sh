openssl genrsa -out ca_key.pem 2048
openssl req -x509 -new -nodes -key ca_key.pem -days 3650 -out ca_certificate.pem -subj "/CN=MyCA"
openssl genrsa -out server_key.pem 2048

# 1. 创建配置文件 ssl.conf
cat > ssl.conf <<EOF
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name

[req_distinguished_name]

[v3_req]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
subjectAltName = @alt_names

[alt_names]
IP.1 = 192.168.5.130
DNS.1 = example.com
EOF

# 2. 生成服务器证书请求（包含SAN）
openssl req -new -key server_key.pem -out server.csr -subj "/CN=rabbitmq-server" -config ssl.conf

# 3. 使用CA签名（包含SAN扩展）
openssl x509 -req -in server.csr \
    -CA ca_certificate.pem \
    -CAkey ca_key.pem \
    -CAcreateserial \
    -out server_certificate.pem \
    -days 365 \
    -extensions v3_req \
    -extfile ssl.conf

openssl genrsa -out client_key.pem 2048
openssl req -new -key client_key.pem -out client.csr -subj "/CN=rabbitmq-client"
openssl x509 -req -in client.csr -CA ca_certificate.pem -CAkey ca_key.pem -CAcreateserial -out client_certificate.pem -days 365