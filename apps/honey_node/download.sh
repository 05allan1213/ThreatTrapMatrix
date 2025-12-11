#!/bin/bash
set -euo pipefail

# 配置参数
LOG=true
NODE_VERSION="v1.0.1"
MANAGE_IP="192.168.5.130"  # 管理端的ip地址
REMOTE_SERVER="http://${MANAGE_IP}"  # 请替换为实际服务器地址
NODE_IMAGE_ID="9245f46f729f"
NET_WORK="ens33" # 网卡

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # 无颜色

# 日志函数
info() {
    echo -e "${GREEN}INFO: $*${NC}"
}

warning() {
    echo -e "${YELLOW}WARNING: $*${NC}"
}

error() {
    echo -e "${RED}ERROR: $*${NC}" >&2
    exit 1
}

# 检查命令是否存在
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# 检查是否能上网
check_internet() {
    info "检查网络连接..."
    if curl -s --head "http://www.baidu.com" | head -n 1 | grep "HTTP/1.[01] [23].." >/dev/null; then
        return 0
    else
        return 1
    fi
}

# 检测操作系统并安装Docker
install_docker() {
    info "开始安装Docker..."

    # 检测操作系统
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$ID
        VERSION=$VERSION_ID
    else
        error "无法检测操作系统类型，无法自动安装Docker"
    fi

    info "检测到操作系统: $OS $VERSION"

    # 根据操作系统安装Docker
    case $OS in
        ubuntu|debian)
            sudo apt-get update
            sudo apt-get install -y apt-transport-https ca-certificates curl software-properties-common
            curl -fsSL https://download.docker.com/linux/$OS/gpg | sudo apt-key add -
            sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/$OS $(lsb_release -cs) stable"
            sudo apt-get update
            sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
            ;;
        centos|rhel)
            sudo yum install -y yum-utils
            sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
            sudo yum install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
            sudo systemctl start docker
            sudo systemctl enable docker
            ;;
        *)
            error "不支持的操作系统: $OS，无法自动安装Docker"
            ;;
    esac

    # 检查Docker是否安装成功
    if command_exists docker; then
        info "Docker安装成功"
        # 可选：将当前用户添加到docker组，避免使用sudo
        if ! groups $USER | grep &>/dev/null '\bdocker\b'; then
            info "将当前用户添加到docker组..."
            sudo usermod -aG docker $USER
            warning "用户已添加到docker组，请注销并重新登录以使更改生效"
        fi
        return 0
    else
        error "Docker安装失败"
    fi
}

# 下载并导入镜像
load_image() {
    local dockerImage=$1
    local download_url=$2

    info "开始下载${dockerImage}镜像..."
    if curl -H "X-Version: dev" -fSL "$download_url" -o "${dockerImage}.tar.gz"; then
        info "${dockerImage}镜像下载成功..."
        if docker load -i "${dockerImage}.tar.gz"; then
            info "${dockerImage}镜像导入成功"
            return 0
        else
            error "${dockerImage}镜像导入失败"
        fi
    else
        error "${dockerImage}镜像下载失败"
    fi
}

# 检查并获取Node镜像
check_and_get_node_image() {
    local node_image_id="${NODE_IMAGE_ID}"  # 替换为实际的镜像ID
    info "检查Node镜像是否存在..."

    if docker images --format "{{.ID}}" | grep -q "^${node_image_id:0:12}"; then
        info "Node镜像已存在"
        return 0
    else
        info "Node镜像不存在，开始下载..."
        local download_url="${REMOTE_SERVER}/api/honey_server/node_version/download?version=${NODE_VERSION}"
        load_image "node_${NODE_VERSION}" "$download_url"
    fi
}

# 检查并获取Suricata镜像
check_and_get_suricata_image() {
    info "检查Suricata镜像是否存在..."
    if docker images | grep -q "suricata"; then
        info "Suricata镜像已存在"
        return 0
    else
        info "Suricata镜像不存在，开始下载..."
        local download_url="${REMOTE_SERVER}/uploads/docker/suricata_7.0.10.tar.gz"
        load_image "Suricata" "$download_url"
    fi
}

# 检查并获取Filebeat镜像
check_and_get_filebeat_image() {
    info "检查Filebeat镜像是否存在..."
    if docker images | grep -q "filebeat"; then
        info "Filebeat镜像已存在"
        return 0
    else
        info "Filebeat镜像不存在，开始下载..."
        local download_url="${REMOTE_SERVER}/uploads/docker/filebeat_7.5.1.tar.gz"
        load_image "Filebeat" "$download_url"
    fi
}

# 创建目录
create_directory() {
    info "创建工作目录..."
    WORK_DIR="$HOME/honey"
    if mkdir -p "$WORK_DIR"; then
        cd "$WORK_DIR" || error "无法进入$WORK_DIR目录"
        info "工作目录创建成功: $WORK_DIR"
        mkdir -p node_server
        touch node_server/gorm.db
        mkdir -p node_server/logs
        mkdir -p suricata
        mkdir -p suricata/rules
        mkdir -p filebeat
        mkdir -p kafka-certs
    else
        error "无法创建$WORK_DIR目录"
    fi
}

# 生成Node服务器配置文件
generate_node_config() {
    info "生成Node服务器配置文件..."
    cat > "node_server/settings.yaml" << EOF
# Node Server 配置文件
logger:
  format: json
  appName: node_server
  level: info
system:
  grpcManageAddr: "hy.io:50001"
  network: ens33
  uid:
  evePath: /var/log/suricata/eve.json
db:
  db_name: "gorm.db"
  maxIdleConns: 10
  maxOpenConns: 100
  connMaxLifetime: 10000
filterNetworkList:
  - br-
  - docker
  - mc_
  - hy_
mq:
  user: admin
  password: password
  host: hy.io
  port: 5671
  createIpExchangeName: createIpExchange
  deleteIpExchangeName: deleteIpExchangeName
  bindPortExchangeName: bindPortExchangeName
  batchDeployExchangeName: batchDeployExchangeName
  ssl: true
  clientCertificate: mq_cert/client_certificate.pem # 客户端的证书
  clientKey: mq_cert/client_key.pem # 客户端的私钥
  caCertificate: mq_cert/ca_certificate.pem # ca的证书
  alertTopic: alertTopic
  batchDeployStatusTopic: batchDeployStatusTopic
  batchUpdateDeployExchangeName: batchUpdateDeployExchangeName
  batchUpdateDeployStatusTopic: batchUpdateDeployStatusTopic
  batchRemoveDeployExchangeName: batchRemoveDeployExchangeName
  batchRemoveDeployStatusTopic: batchRemoveDeployStatusTopic
EOF
    info "Node配置文件生成成功: node_server/settings.yaml"
}

# 生成suricata的配置文件
generate_suricata_config(){
      cat > "suricata/suricata.yaml" << EOF
%YAML 1.1
---
vars:
  address-groups:
    HOME_NET: "[192.168.0.0/16,10.0.0.0/8,172.16.0.0/12]"
    EXTERNAL_NET: "!\$HOME_NET"

default-log-dir: /var/log/suricata/

outputs:
  - eve-log:
      enabled: yes
      filename: eve.json
      types:
        - alert:
            payload: yes
            http-body: yes
        - http

af-packet:
  - interface: ens33
    cluster-id: 99
    cluster-type: cluster_flow
    defrag: yes

app-layer:
  protocols:
    http:
      enabled: yes

flow:
  memcap: 128mb
  hash-size: 65536
  prealloc: 10000

default-rule-path: /etc/suricata/rules
rule-files:
  - local.rules
EOF
      cat > "suricata/rules/local.rules" << EOF
alert http any any -> any any (msg:"使用 curl 请求"; flow:established,to_server; content:"curl"; http_user_agent; classtype:web-application-attack; sid:600033; priority:10; rev:1; metadata: level 2;)
alert icmp any any -> any any (itype:8; msg:"ping请求检测";  priority: 3;sid:2023040702;metadata: level 1;)
EOF
    info "suricata/rules/local.rules 规则文件生成成功"
}

# 生成filebeat的配置文件
generate_filebeat_config(){
      cat > "filebeat/filebeat.yml" << EOF
filebeat.inputs:
  - type: log
    enabled: true
    paths:
      - /app/ttm/*.log             # 监听所有 .log 文件
      - /app/ttm/**/*.log          # 监听子目录中的所有 .log 文件
    fields:
      log_type: ttm_logs           # 添加自定义字段
    # 多行日志处理（适用于Java堆栈跟踪等）
#    multiline.pattern: '^[[:space:]]+(at|\.{3})[[:space:]]+\b|^Caused by:'
#    multiline.negate: false
#    multiline.match: after
    # JSON解析配置
    json.keys_under_root: true  # 将JSON字段提升到根级别
    json.overwrite_keys: true   # 覆盖同名的已有字段
    json.add_error_key: true    # 解析失败时添加error字段

    # 只保留解析后的JSON字段，移除Filebeat默认添加的冗余字段
    processors:
      - drop_fields:
          fields: [ "log", "@version", "ecs", "input", "agent", "host" ]  # 移除不需要的字段
          ignore_missing: true  # 忽略不存在的字段

# 处理器配置（可选）
processors:
  - add_host_metadata: ~
  - add_docker_metadata: ~
  - drop_fields: # 移除不需要的字段
      fields: [ "log.offset", "input.type", "monitoring", "@metadata", "host", "agent" ]

# Kafka 输出配置
output.kafka:
  enabled: true
  hosts: [ "kafka-broker:9095" ]       # Kafka 地址
  topic: "logs_topic"              # Kafka 主题名称

  ssl.enabled: true
  ssl.certificate_authorities: [ "/usr/share/filebeat/certs/ca.crt" ]
  username: "user1"
  password: "password1"
  sasl.mechanism: "PLAIN"
  security_protocol: "SASL_SSL"  # 明确指定协议类型

  partition.round_robin: # 分区策略
    reachable_only: false
  required_acks: 1               # 确认级别
  compression: gzip              # 压缩提高吞吐量
  max_message_bytes: 1048576     # 最大消息大小 (1MB)
  codec.json:
    pretty: false

# 日志记录设置
logging.level: info
logging.selectors: [ "kafka", "tls" ]  # 聚焦关键模块
logging.to_files: true
logging.files:
  path: /var/log/filebeat
  name: filebeat.log
  keepfiles: 7
EOF

    cat > "kafka-certs/ca.crt" << EOF
-----BEGIN CERTIFICATE-----
MIIDlTCCAn2gAwIBAgIUUCE37H6RmkXSLAqdbLwoJCsnO+IwDQYJKoZIhvcNAQEL
BQAwWjELMAkGA1UEBhMCQ04xEDAOBgNVBAgMB0JlaWppbmcxEDAOBgNVBAcMB0Jl
aWppbmcxFDASBgNVBAoMC1lvdXJDb21wYW55MREwDwYDVQQDDAhrYWZrYS1jYTAe
Fw0yNTEyMDgxNDIzMTJaFw0zNTEyMDYxNDIzMTJaMFoxCzAJBgNVBAYTAkNOMRAw
DgYDVQQIDAdCZWlqaW5nMRAwDgYDVQQHDAdCZWlqaW5nMRQwEgYDVQQKDAtZb3Vy
Q29tcGFueTERMA8GA1UEAwwIa2Fma2EtY2EwggEiMA0GCSqGSIb3DQEBAQUAA4IB
DwAwggEKAoIBAQDkidIODfc1xigU9KhY2bOdIcKV1aLDFbeUaVd9oGmin1bzxXK6
CoJKoZ/dtV7yDqOxnsYWrFly4ic8oQm80yBW+wvNgdifauItQue/nmbC1ZZlsxeZ
Iccm9gGXECNnlr4n7G3Jjs4lo80XFfP/CaJudn0oxK+YZiwiFDfHnXbU4uKWNyB8
gMlqbskzGb6NCpqKghi3AZeMI18PME69PeMfSEojhCGNoBs9oNpI6ffsWJ0AYxL/
P/KiJHdBEzitSIAr2t04QdTfHNCf+uGLJBa7omuVnuZadZR/5Of6uumyfKkCHIIw
7qugWsB3AkQGJLEd0k58prPqEVpsznUqp35BAgMBAAGjUzBRMB0GA1UdDgQWBBTf
42rON3qlOra1R/nmMNTxX3qpMDAfBgNVHSMEGDAWgBTf42rON3qlOra1R/nmMNTx
X3qpMDAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEBCwUAA4IBAQAY9/FjOxwt
+5FtEqnecy9H0n77kN3bX9rpVVL2a27Q+GAIqzrH3Ke1b4lzkBrmHIwc2fw/YHWR
uE9ATMSdxn7gh5t85RFwJEFGC5WzHFftftTUoHM8a+GlwdZiSRkyUILg8E+Pox9N
vbFQQBG6rvka2XLIjZAl5uqwL+cLaUJk+C+gS5stHF9jyqk5a7rsCD+CQPd5qbQL
HE05cGYVVXtviWLBzkr+ruO/jMU49smOJXnz/fWDdiznqxongcF95+DUz//lvA+G
xpInCb38I3da7X/wjnrmvDBxIXF7amSk9kuJI892BhCBCFfHV4eTWCQUvSJgKLok
v1gK3T8NtmxR
-----END CERTIFICATE-----
EOF
    info "filebeat/filebeat.yml 配置文件生成成功"
}

# 生成docker-compose.yaml文件
generate_docker_compose() {
    info "生成docker-compose配置文件..."
    cat > "docker-compose.yaml" << EOF
version: '3'
services:
  suricata:
    image: jasonish/suricata:7.0.10  # 指定具体版本
    network_mode: host  # 必须使用主机模式监听网卡
    cap_add:
      - NET_ADMIN
      - NET_RAW
      - SYS_NICE  # 必需的能力
    volumes:
      - ./suricata:/etc/suricata
      - ./suricata/logs:/var/log/suricata         # 挂载日志目录
    command: -i ${NET_WORK}  # 指定监听的网卡（需替换为实际网卡名）
    restart: always
    environment:
      TZ: Asia/Shanghai
  node_server:
    image: node:${NODE_VERSION}
    network_mode: host
    restart: always
    environment:
      - TZ=Asia/Shanghai
    cap_add:
      - NET_ADMIN
      - NET_RAW
    extra_hosts:
      - "hy.io:${MANAGE_IP}"  # 虚拟主机名映射
    volumes:
      - ./node_server/settings.yaml:/app/settings.yaml
      - ./node_server/gorm.db:/app/gorm.db
      - ./suricata/logs:/var/log/suricata         # 挂载日志目录
      - ./node_server/logs:/app/logs
EOF

    # 如果需要日志收集，添加Filebeat配置
    if [ "$LOG" = true ]; then
        cat >> "docker-compose.yaml" << EOF

  filebeat:
    image: elastic/filebeat:7.5.1
    network_mode: host
    volumes:
      - ./node_server/logs:/app/ttm
      - ./filebeat/filebeat.yml:/usr/share/filebeat/filebeat.yml
      - ./kafka-certs/ca.crt:/usr/share/filebeat/certs/ca.crt:ro
    environment:
      - SSL_CERT_FILE=/usr/share/filebeat/certs/ca.crt
    extra_hosts:
      - "kafka-broker:${MANAGE_IP}"  # 虚拟主机名映射
EOF
    fi

    info "docker-compose配置文件生成成功"
}

# 主函数
main() {
    info "开始部署 Honey Node..."

    # 检查Docker是否存在
    if ! command_exists docker; then
        info "未检测到Docker，需要安装Docker"

        # 检查是否能上网
        if check_internet; then
            info "网络连接正常，将从官方源安装Docker"
            install_docker
        else
            info "无法连接到互联网，将从本地服务器下载Docker"
            if curl -fSL "${REMOTE_SERVER}/download/docker" -o "docker_install.sh"; then
                chmod +x docker_install.sh
                sudo ./docker_install.sh
                rm -f docker_install.sh
            else
                error "从本地服务器下载Docker安装包失败"
            fi
        fi

        # 重新检查Docker
        if ! command_exists docker; then
            error "Docker安装失败，请手动安装后重试"
        fi
    else
        info "已检测到Docker，版本: $(docker --version | awk '{print $3}' | cut -d',' -f1)"
    fi

    # 检查并安装Docker Compose
    if ! command_exists docker compose && ! docker compose version >/dev/null 2>&1; then
        error "未检测到Docker Compose，请安装后重试"
    fi

    # 检查并获取所需镜像
    check_and_get_node_image
    check_and_get_suricata_image

    check_and_get_filebeat_image

    # 创建工作目录
    create_directory

    # 生成配置文件
    generate_node_config
    generate_suricata_config

    if [ "$LOG" = true ]; then
        generate_filebeat_config
    fi

    generate_docker_compose

    info "Node Server部署准备工作完成！"
    docker compose up -d
}

# 启动主函数
main