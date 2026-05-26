# Xin API 🚀

[English](./README-en.md) | 中文

<p align="center">
  <strong>集大模型 API 统一管理、全链路追踪、智能分发于一体的高性能网关服务</strong>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go Version">
  <img src="https://img.shields.io/badge/Gin-1.12-00ADD8?style=flat-square" alt="Gin Framework">
  <img src="https://img.shields.io/badge/Vue.js-3.x-42b883?style=flat-square&logo=vue.js&logoColor=white" alt="Vue.js">
  <img src="https://img.shields.io/badge/PostgreSQL-16-336791?style=flat-square&logo=postgresql&logoColor=white" alt="PostgreSQL">
  <img src="https://img.shields.io/badge/Redis-7-DC382D?style=flat-square&logo=redis&logoColor=white" alt="Redis">
  <img src="https://img.shields.io/badge/Nginx-Alpine-009639?style=flat-square&logo=nginx&logoColor=white" alt="Nginx">
  <img src="https://img.shields.io/badge/Docker-Ready-2496ED?style=flat-square&logo=docker&logoColor=white" alt="Docker Ready">
</p>

---

## ✨ 项目简介

Xin API 是一个高性能的大语言模型（LLM）网关服务，无缝兼容 OpenAI、Anthropic Claude、DeepSeek 等主流大模型，实现标准化统一 API 接入。

### 核心特性

- **🎯 统一 API 接入**: 标准化不同大模型的 API 接口，提供统一的 `/v1/chat/completions` 端点
- **🔀 智能分发**: 基于权重的负载均衡，自动将请求分发到最佳渠道
- **🛡️ 熔断保护**: 分布式熔断器，自动隔离故障渠道，防止雪崩效应
- **⚡ 限流控制**: 基于 Redis 的分布式限流器，支持 RPM/TPM 多维度限流
- **👥 多租户管理**: 支持用户、项目组、渠道的多层级管理
- **🔑 API Key 管理**: 灵活的 API Key 分发和吊销机制
- **📊 全链路追踪**: 完整的请求链路追踪和审计日志（可扩展）
- **🌐 Web 管理后台**: 基于 Vue 3 + Tailwind CSS 的现代化管理界面
- **🐳 容器化部署**: 开箱即用的 Docker Compose 部署方案

---

## 🏗️ 技术架构

### 技术栈

| 层级 | 技术 |
|------|------|
| **前端** | Vue 3 + TypeScript + Vite + Tailwind CSS + Radix UI |
| **后端** | Go 1.25 + Gin + GORM + JWT |
| **数据库** | PostgreSQL 16（元数据存储） |
| **缓存** | Redis 7（限流、熔断、会话） |
| **代理** | Nginx（反向代理、静态资源） |
| **部署** | Docker + Docker Compose |

---

## 🚀 快速开始

### 环境要求

- Go 1.25+
- Node.js 22+
- PostgreSQL 16+
- Redis 7+
- Docker & Docker Compose（推荐）

### Docker Compose 一键部署（推荐）

```bash
# 1. 克隆项目
git clone https://github.com/your-org/Xin-api.git
cd Xin-api

# 2. 配置环境变量
cp .env.example .env
# 编辑 .env 修改数据库密码、JWT密钥等

# 3. 启动服务
cd build
./scripts/deploy.sh start

# 或直接使用 docker-compose
# docker-compose up -d --build

# 4. 访问服务
# Web 管理后台: http://localhost
# API 地址: http://localhost/v1
```

### 本地开发部署

#### 1. 启动依赖服务

```bash
# 启动 PostgreSQL 和 Redis
docker run -d --name xin-postgres \
  -e POSTGRES_PASSWORD=123456 \
  -p 5432:5432 \
  postgres:16-alpine

docker run -d --name xin-redis \
  -e REDIS_PASSWORD=123456 \
  -p 6379:6379 \
  redis:7-alpine
```

#### 2. 启动后端服务

```bash
# 配置环境变量
cp .env.example .env

# 安装依赖并运行
go mod download
go run cmd/xin_api/main.go
```

#### 3. 启动前端开发服务

```bash
cd web
npm install
npm run dev
```

前端开发服务器默认运行在 `http://localhost:5173`

---

## 🔧 配置说明

### 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `PROXY_PORT` | `8080` | 后端服务端口 |
| `ADMIN_PORT` | `9090` | 管理面端口 |
| `NGINX_HTTP_PORT` | `80` | Nginx HTTP 端口 |
| `NGINX_HTTPS_PORT` | `443` | Nginx HTTPS 端口 |
| `POSTGRES_USER` | `root` | PostgreSQL 用户名 |
| `POSTGRES_PASSWORD` | `123456` | PostgreSQL 密码 |
| `POSTGRES_DB` | `gateway` | PostgreSQL 数据库名 |
| `POSTGRES_PORT` | `5432` | PostgreSQL 端口 |
| `REDIS_PORT` | `6379` | Redis 端口 |
| `GIN_MODE` | `debug` | 运行模式（debug/release）|
| `POSTGRESQL_DSN` | - | PostgreSQL 连接字符串 |
| `REDIS_HOST` | `127.0.0.1` | Redis 主机 |
| `REDIS_PORT` | `6379` | Redis 端口 |
| `REDIS_PASSWORD` | - | Redis 密码 |
| `JWT_SECRET` | - | JWT 签名密钥 |
| `JWT_EXPIRE` | `24h` | JWT 过期时间 |
| `CB_FAILURE_THRESHOLD` | `5` | 熔断失败阈值 |
| `CB_RECOVERY_INTERVAL` | `60s` | 熔断恢复间隔 |
| `PROXY_REQUEST_TIMEOUT` | `120s` | 上游请求超时 |
| `PROXY_MAX_BODY_MB` | `10` | 最大请求体大小 |
| `DEFAULT_RPM` | `60` | 默认每分钟请求数限制 |
| `DEFAULT_TPM` | `100000` | 默认每分钟 Token 数限制 |

---

## 📸 界面预览

### 管理后台功能

- **📊 仪表盘**: 总览系统状态、请求统计
- **👥 用户管理**: 用户账号、权限管理
- **📁 项目组管理**: 创建和管理项目组，配置渠道
- **🔌 渠道管理**: 配置上游大模型渠道、权重分配
- **🔑 API Key 管理**: 分发和管理 API Key
- **⚙️ 系统设置**: 全局配置

---

## 🐳 生产部署

### Docker Compose 部署

完整的部署配置请参考 [build/README.md](build/README.md)

部署目录结构：

```
build/
├── Dockerfile              # 前后端统一多阶段构建（Node.js + Go + Alpine）
├── Dockerfile.nginx        # Nginx 反向代理配置
├── docker-compose.yml      # 服务编排（PostgreSQL + Redis + App + Nginx）
├── .dockerignore          # Docker 构建忽略文件
├── nginx/                 # Nginx 配置文件
│   ├── nginx.conf
│   └── default.conf
└── scripts/
    └── deploy.sh          # 部署脚本
```

启动命令：

```bash
cd build

# 一键启动所有服务
./scripts/deploy.sh start

# 或构建并启动所有服务
docker-compose up -d --build

# 查看服务状态
docker-compose ps

# 查看应用日志
docker-compose logs -f app

# 停止服务
docker-compose down
```

### 部署脚本命令

```bash
./scripts/deploy.sh start    # 启动所有服务
./scripts/deploy.sh stop     # 停止所有服务
./scripts/deploy.sh restart  # 重启所有服务
./scripts/deploy.sh logs     # 查看所有日志
./scripts/deploy.sh logs app      # 查看应用日志（前后端）
./scripts/deploy.sh logs nginx    # 查看 Nginx 日志
./scripts/deploy.sh logs postgres # 查看数据库日志
./scripts/deploy.sh logs redis    # 查看缓存日志
./scripts/deploy.sh status   # 查看服务状态
./scripts/deploy.sh build    # 重新构建镜像
./scripts/deploy.sh health   # 检查服务健康状态
./scripts/deploy.sh update   # 更新代码并重启服务
```

### 服务架构

```
用户请求 → Nginx (80/443)
    ├── /v1/*  → 后端 API 服务 (app:8080)
    └── /*     → 前端静态文件 (app:8080)

后端依赖：PostgreSQL + Redis
```

### 服务端口

| 服务 | 容器端口 | 主机端口 | 说明 |
|------|---------|---------|------|
| Nginx | 80/443 | 80/443 | Web 入口（统一） |
| Xin-api | 8080 | - | 后端 + 前端静态文件（内部） |
| PostgreSQL | 5432 | 5432 | 数据库 |
| Redis | 6379 | 6379 | 缓存服务 |

### 数据持久化

数据通过 Docker Volumes 自动持久化：

- `postgres_data`: PostgreSQL 数据
- `redis_data`: Redis 数据
- `nginx_logs`: Nginx 日志

### 生产环境建议

1. **修改默认密码**: 务必修改数据库和 Redis 密码
2. **更换 JWT 密钥**: 使用强随机字符串作为 JWT_SECRET
3. **启用 HTTPS**: 配置 Nginx SSL 证书
4. **配置日志轮转**: 防止日志占满磁盘
5. **定期备份**: 定期备份 PostgreSQL 数据

---

## 📄 License

本项目采用 [MIT License](LICENSE) 开源协议。

---

<p align="center">
  <strong>⭐ 如果觉得这个项目对你有帮助，请给个 Star！</strong>
</p>
