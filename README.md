# ThreatTrapMatrix（威胁诱捕矩阵）

ThreatTrapMatrix 是一个“集中管控 + 分布式诱捕节点”的威胁诱捕/欺骗防御平台：  
在服务端统一管理诱捕网络、诱捕 IP、端口转发与镜像（诱捕服务），在诱捕节点侧落地执行（创建/删除诱捕 IP、绑定端口、运行 IDS），并将告警与日志回传到平台进行检索与可视化。

---

## 1. 核心能力

- **诱捕节点管理（Node）**
    - 节点注册、在线状态维护（心跳/资源检测）
    - 子网扫描（发现资产/空闲 IP），支撑诱捕 IP 规划
- **诱捕 IP 管理（Honey IP）**
    - 创建/删除诱捕 IP（节点侧落地）
    - 诱捕 IP 的端口绑定/转发（将攻击流量导向指定诱捕服务）
    - 与节点建立双向 gRPC 通信，接收节点执行结果与状态回调
- **诱捕服务（镜像）管理**
    - 镜像服务负责管理/拉起诱捕服务容器与虚拟网络（Docker 网络隔离）
- **告警与日志**
    - 节点侧 Suricata 检测攻击行为
    - 服务端告警服务汇总、入库/索引，提供检索与管理
    - ELK + Kafka + Filebeat/Logstash：采集平台与节点日志，统一检索与可视化
- **矩阵部署服务**
    - 批量部署与删除：支持 C 段级别的一键部署，通过任务拆分与队列调度避免节点过载
    - 大子网优化：针对 /16 等大子网进行部署规划
---

## 2. 架构总览
![架构图](架构图.png)
平台分为三块：

### 2.1 Web/管理入口
- `honey_web`：前端静态站点，由 Nginx 承载（80/443）
- `ws_server`：WebSocket 服务，用于向 Web 推送实时状态/任务进度

### 2.2 服务端（控制面）
服务端由多个 Go 微服务组成：
- `honey_server`：核心控制服务（HTTP + gRPC），负责节点与诱捕资源编排（gRPC 默认监听 `:50001`）
- `matrix_server`：矩阵/资源管理（网络、任务、编排等）
- `image_server`：镜像与虚拟网络管理（依赖 Docker socket）
- `alert_server`：告警服务（对接 ES 索引、告警管理）
- 基础设施：MySQL、Redis、RabbitMQ、Elasticsearch
- 日志侧（可选增强）：Kafka + Filebeat + Logstash + Kibana

消息/数据通路：
- **RabbitMQ**：服务端向节点下发部署任务/变更任务；节点回传执行结果与告警事件（服务端内部通常走 5672，节点可走 TLS 5671）
- **gRPC**：节点注册、资源上报、命令下发（双向流）、状态回调、端口转发通道（Tunnel）

### 2.3 诱捕节点（数据面）
- `node_server`（仓库内为 `apps/honey_node`）：节点执行器
    - 接收服务端任务：子网扫描、创建/删除诱捕 IP、端口绑定等
    - 本地 SQLite 存储（gorm.db）
- `suricata`：入侵检测/告警产生（eve.json）
- `filebeat`：采集节点日志/IDS 日志，发送到 Kafka（可选）

---

## 3. 目录结构

- `apps/`
    - `honey_server/`：核心控制服务（HTTP + gRPC）
    - `matrix_server/`：矩阵管理服务（HTTP）
    - `image_server/`：镜像/虚拟网络服务（HTTP，连接 Docker）
    - `alert_server/`：告警服务（HTTP，写 ES）
    - `ws_server/`：WebSocket 服务（推送）
    - `honey_node/`：诱捕节点（node_server）与节点侧部署文件
- `deploy/`
    - `docker-compose.yaml`：服务端一键部署（MySQL/Redis/RabbitMQ/ES/Nginx + 各服务）
    - `*/settings.yaml`：各服务端配置
- `elk/`
    - `docker-compose.yaml`：Kafka/Filebeat/Logstash/Kibana（依赖服务端创建的 external 网络）

---

## 4. 快速开始

### 4.1 前置依赖
- Docker / Docker Compose
- 推荐 Linux 主机（建议以 root 用户启动）

### 4.2 启动服务端基础栈 + 平台服务
在服务端机器上：

```bash
cd deploy
chmod 0777 es/data
docker compose -f docker-compose.yaml up -d
```

执行完，使用浏览器访问管理ip，如果看到web页面，则服务启动成功。
然后创建一个用户

```bash
docker exec -it deploy-honey_server-1 ./main -m user -t create
# 选择角色，输入用户名和密码
```

其中分布式日志系统的web界面在`http://管理ip:5601`

### 4.3 启动日志可视化（可选）

```bash
cd elk
docker compose -f docker-compose.yaml up -d
 ```
该 compose 包含 Kafka(SSL/SASL)、Filebeat、Logstash、Kibana

### 4.4 节点部署

```bash
cd apps/honey_node
docker build -t node:v1.0.1 .
docker save -o node_v1.0.1.tar node:v1.0.1
gzip node_v1.0.1.tar
```

然后把这个镜像下载下来，在管理web界面的节点版本上传，再从节点添加处添加节点，把命令放到节点终端上执行即可。