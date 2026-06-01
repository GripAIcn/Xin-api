# Xin-api Docker 部署指南

## 目录

- [目录结构](#目录结构)
- [快速开始](#快速开始)
  - [1. 环境准备](#1-环境准备)
  - [2. 配置环境变量](#2-配置环境变量)
  - [3. 启动服务](#3-启动服务)
  - [4. 访问服务](#4-访问服务)
- [部署脚本使用](#部署脚本使用)
- [服务架构](#服务架构)
  - [端口映射](#端口映射)
- [数据持久化](#数据持久化)
- [生产环境部署建议](#生产环境部署建议)
  - [1. 安全配置](#1-安全配置)
  - [2. 性能优化](#2-性能优化)
  - [3. 日志管理](#3-日志管理)
  - [4. 备份策略](#4-备份策略)
- [故障排查](#故障排查)
  - [查看服务日志](#查看服务日志)
  - [常见问题](#常见问题)
- [更新升级](#更新升级)
- [卸载清理](#卸载清理)
- [联系支持](#联系支持)

## 目录结构

```
build/
├── Dockerfile              # Go 后端多阶段构建（包含前端静态文件）
├── Dockerfile.nginx        # Nginx 反向代理配置
├── docker-compose.yml      # 服务编排
├── .dockerignore          # Docker 构建忽略文件
├── nginx/
│   ├── nginx.conf         # Nginx 主配置
│   └── default.conf       # 站点配置
├── scripts/
│   └── deploy.sh          # 部署脚本
└── README.md              # 本文件
web/                        # 前端源码目录（已集成到后端构建）
├── src/
├── public/
└── package.json
```

## 快速开始

### 1. 环境准备

确保已安装：
- Docker >= 20.10
- Docker Compose >= 2.0

### 2. 配置环境变量

```bash
# 复制环境变量模板
cp ../.env.example .env

# 编辑 .env 文件，修改以下关键配置
vim .env
```

**关键配置项：**

```env
# 数据库配置（生产环境务必修改密码）
POSTGRES_USER=root
POSTGRES_PASSWORD=your-strong-password
POSTGRES_DB=gateway

# Redis 配置
REDIS_PASSWORD=your-redis-password

# JWT 密钥（生产环境务必修改为随机强密码）
JWT_SECRET=your-jwt-secret-key-min-32-chars

# 运行模式
GIN_MODE=release
```

### 3. 启动服务

使用部署脚本一键启动：

```bash
# 进入 build 目录
cd build

# 启动所有服务
./scripts/deploy.sh start
```

或使用 Docker Compose 命令：

```bash
docker-compose up -d --build
```

### 4. 访问服务

- **Web 界面**: http://localhost
- **API 地址**: http://localhost/v1

## 部署脚本使用

```bash
./scripts/deploy.sh [命令]

命令:
  start    启动所有服务
  stop     停止所有服务
  restart  重启所有服务
  logs     查看日志 (可选: 指定服务名)
  status   查看服务状态
  build    重新构建镜像
  reset    完全重置（删除所有数据）
  update   更新代码并重启服务
  help     显示帮助信息
```

### 常用命令示例

```bash
# 启动服务
./scripts/deploy.sh start

# 查看所有服务状态
./scripts/deploy.sh status

# 查看应用日志
./scripts/deploy.sh logs app

# 实时查看 Nginx 日志
./scripts/deploy.sh logs -f nginx

# 重启服务
./scripts/deploy.sh restart

# 停止服务
./scripts/deploy.sh stop

# 更新到最新版本
./scripts/deploy.sh update
```

## 服务架构

```
┌─────────────────────────────────────────────────────────────┐
│                         Docker Compose                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │    Nginx     │  │  Xin-api     │  │  PostgreSQL  │       │
│  │   (80/443)   │  │   (8080)     │  │    (5432)    │       │
│  └──────┬───────┘  └──────────────┘  └──────────────┘       │
│         │                                                    │
│         │  静态文件                                          │
│         │  API 代理                                          │
│         ▼                                                    │
│  ┌──────────────┐                                            │
│  │    Redis     │                                            │
│  │    (6379)    │                                            │
│  └──────────────┘                                            │
└─────────────────────────────────────────────────────────────┘
```

### 端口映射

| 服务 | 容器端口 | 主机端口 | 说明 |
|------|---------|---------|------|
| Nginx | 80/443 | 80/443 | Web 入口 |
| Xin-api | 8080 | 8080 | API 服务 |
| PostgreSQL | 5432 | 5432 | 数据库 |
| Redis | 6379 | 6379 | 缓存服务 |

## 数据持久化

数据通过 Docker Volumes 持久化存储：

- `postgres_data`: PostgreSQL 数据
- `redis_data`: Redis 数据
- `nginx_logs`: Nginx 日志

查看数据卷：

```bash
docker volume ls
docker volume inspect build_postgres_data
```

## 生产环境部署建议

### 1. 安全配置

- **修改默认密码**: 务必修改 `POSTGRES_PASSWORD`、`REDIS_PASSWORD`、`JWT_SECRET`
- **关闭端口暴露**: 生产环境建议只暴露 Nginx 的 80/443 端口
- **启用 HTTPS**: 配置 SSL 证书

### 2. 性能优化

编辑 `docker-compose.yml` 调整资源限制：

```yaml
services:
  app:
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 2G
        reservations:
          cpus: '0.5'
          memory: 512M
```

### 3. 日志管理

配置日志轮转：

```yaml
services:
  app:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

### 4. 备份策略

定期备份 PostgreSQL 数据：

```bash
# 备份数据库
docker exec xin-api-postgres pg_dump -U root gateway > backup_$(date +%Y%m%d).sql

# 恢复数据库
docker exec -i xin-api-postgres psql -U root gateway < backup_20240101.sql
```

## 故障排查

### 查看服务日志

```bash
# 查看所有服务日志
docker-compose logs

# 查看特定服务日志
docker-compose logs app
docker-compose logs postgres
docker-compose logs redis
docker-compose logs nginx

# 实时跟踪日志
docker-compose logs -f app
```

### 常见问题

#### 1. 端口被占用

```bash
# 检查端口占用
netstat -tlnp | grep 80

# 修改 docker-compose.yml 中的端口映射
```

#### 2. 数据库连接失败

```bash
# 检查 PostgreSQL 状态
docker-compose ps postgres
docker-compose logs postgres

# 进入数据库容器
docker exec -it xin-api-postgres psql -U root -d gateway
```

#### 3. 应用启动失败

```bash
# 检查应用日志
docker-compose logs app

# 检查环境变量
docker exec xin-api-app env
```

## 更新升级

```bash
# 1. 拉取最新代码
git pull

# 2. 重新构建镜像
docker-compose build --no-cache

# 3. 重启服务
docker-compose down
docker-compose up -d
```

## 卸载清理

```bash
# 停止并删除容器
docker-compose down

# 删除数据卷（谨慎操作！）
docker-compose down -v

# 删除镜像
docker rmi build-app
```

## 联系支持

如有问题，请查看项目文档或提交 Issue。
