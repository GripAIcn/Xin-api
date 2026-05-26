# Xin-api Docker 部署方案

## 项目架构分析

### 技术栈

* **后端**: Go 1.25.8 + Gin 框架

* **前端**: Vue 3 + Vite + TypeScript + Tailwind CSS

* **数据库**: PostgreSQL (元数据存储)

* **缓存**: Redis (分布式限流、熔断器)

### 服务依赖

1. **PostgreSQL**: 存储用户、渠道、分组等管理元数据
2. **Redis**: 分布式限流、熔断器状态、会话缓存

### 端口说明

* `8080`: Go 后端 API 服务端口

* `5173`: 前端开发服务器端口（生产环境不需要暴露）

## 部署架构决策

### 是否需要 Nginx？

**建议在生产环境使用 Nginx**，原因如下：

1. **静态资源服务**: 前端构建后的静态文件可以通过 Nginx 高效提供
2. **反向代理**: 统一入口，将 API 请求转发到 Go 服务
3. **负载均衡**: 未来多实例部署时的负载均衡
4. **SSL/TLS 终止**: 处理 HTTPS 证书
5. **缓存优化**: 静态资源缓存策略
6. **安全**: 隐藏后端服务真实端口，提供额外的安全层

### 部署方案

采用 **多阶段构建 + Docker Compose** 方案：

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

## 实施步骤

### 1. 创建 Dockerfile (Go 后端)

* 使用多阶段构建减小镜像体积

* 基于 Alpine Linux 的轻量级镜像

### 2. 创建 Dockerfile (前端构建)

* 构建 Vue 项目为静态文件

* 产物供 Nginx 使用

### 3. 创建 Nginx 配置

* 静态文件服务

* API 反向代理到 Go 服务

* 健康检查端点

### 4. 创建 docker-compose.yml

* 定义所有服务

* 配置网络和数据卷

* 环境变量管理

### 5. 创建部署脚本和文档

* 一键启动脚本

* 环境变量配置说明

* 常见问题排查

## 文件清单

所有 Docker 相关文件存储在 `build/` 目录下：

| 文件                         | 说明            |
| -------------------------- | ------------- |
| `build/Dockerfile`         | Go 后端多阶段构建    |
| `build/docker-compose.yml` | 完整服务编排        |
| `build/nginx/nginx.conf`   | Nginx 配置文件    |
| `build/nginx/default.conf` | Nginx 站点配置    |
| `build/.dockerignore`      | Docker 构建忽略文件 |
| `build/scripts/deploy.sh`  | 部署脚本          |

### 目录结构

```
build/
├── Dockerfile              # Go 后端构建
├── docker-compose.yml      # 服务编排
├── .dockerignore          # 构建忽略文件
├── nginx/
│   ├── nginx.conf         # Nginx 主配置
│   └── default.conf       # 站点配置
└── scripts/
    └── deploy.sh          # 部署脚本
```

## 环境变量配置

需要在生产环境配置的变量：

```env
# 数据库
POSTGRESQL_DSN=host=postgres user=root password=xxx dbname=gateway port=5432 sslmode=disable

# Redis
REDIS_HOST=redis
REDIS_PASSWORD=xxx

# JWT
JWT_SECRET=your-strong-secret-key

# 其他
GIN_MODE=release
```

