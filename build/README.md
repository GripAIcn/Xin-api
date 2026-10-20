# Xin-api Docker 部署指南

## 目录

- [目录结构](#目录结构)
- [快速开始](#快速开始)
- [服务架构与端口](#服务架构与端口)
- [环境变量](#环境变量)
- [数据持久化](#数据持久化)
- [日常运维](#日常运维)
- [备份与恢复](#备份与恢复)
- [升级与回滚](#升级与回滚)
- [生产环境建议](#生产环境建议)
- [故障排查](#故障排查)

## 目录结构

```
build/
├── Dockerfile                   # Go 后端 + 前端多阶段构建
├── Dockerfile.nginx             # Nginx 反向代理镜像
├── docker-compose.yml           # 四服务编排（postgres / redis / app / nginx）
├── .dockerignore                # 部署目录构建忽略规则
├── nginx/
│   ├── nginx.conf               # Nginx 主配置
│   └── default.conf.template    # 站点配置模板（启动时注入 PROXY_PORT）
├── README.md                    # 本文件
└── README.en.md                 # 英文版
```

> 应用镜像的构建上下文是**仓库根目录**，因此构建缓存与排除规则由根目录的 `.dockerignore` 决定。

## 快速开始

### 1. 环境要求

- Docker Engine >= 24.0
- Docker Compose v2（`docker compose` 子命令，不再需要 `docker-compose`）
- 可用内存建议 >= 2 GB（前端 `vite build` 较吃内存）

### 2. 配置环境变量

```bash
cd build
cp ../.env.example .env
```

启动前至少修改以下三项：

```env
POSTGRES_PASSWORD=your-strong-password
REDIS_PASSWORD=your-redis-password
JWT_SECRET=your-random-secret-at-least-32-chars   # openssl rand -base64 48
```

### 3. 启动

```bash
# 在 build 目录执行
docker compose up -d --build
```

首次构建会依次完成：前端 `npm ci` + `vite build` → Go 编译 → 打包运行镜像。
`app` 会等待 PostgreSQL / Redis 健康检查通过后再启动，`nginx` 会等待 `app` 健康后再启动。

### 4. 验证

```bash
docker compose ps          # 四个服务均应为 healthy
curl http://localhost/     # 返回落地页 HTML
```

访问 `http://localhost` 进入 Web 界面，默认管理员账号为 `admin / 12345678`，**登录后请立即修改密码**。

### 5. 停止

```bash
docker compose down        # 停止并删除容器，保留数据卷
```

## 服务架构与端口

```
浏览器 / SDK
    │
    ▼
┌───────────────┐        ┌──────────────────────────┐
│    nginx      │──────▶ │  app（Go，非 root 运行）  │
│  80 / 443     │        │  容器内 8080             │
└───────────────┘        └───────┬──────────┬───────┘
                                 │          │
                          ┌──────▼───┐  ┌───▼──────┐
                          │ postgres │  │  redis   │
                          │   5432   │  │  6379    │
                          └──────────┘  └──────────┘
```

| 服务 | 容器端口 | 主机端口 | 说明 |
|------|---------|---------|------|
| nginx | 80 / 443 | `${NGINX_HTTP_PORT:-80}` / `${NGINX_HTTPS_PORT:-443}` | 唯一对外入口 |
| app | `${PROXY_PORT:-8080}` | 不映射 | 仅集群内部访问，`expose` |
| postgres | 5432 | `127.0.0.1:${POSTGRES_PORT:-5432}` | 仅绑定回环地址，供本机调试 |
| redis | 6379 | `127.0.0.1:${REDIS_PORT:-6379}` | 仅绑定回环地址，供本机调试 |

> 443 端口已预留映射，但当前 Nginx 未配置 TLS 监听。需要 HTTPS 时请挂载证书并在 `default.conf.template` 中增加 `listen 443 ssl;` 的 server 块。

对外暴露的接口：

| 路径 | 协议 |
|------|------|
| `/v1/chat/completions` | OpenAI Chat Completions |
| `/v1/responses` | OpenAI Responses |
| `/v1beta/models/{model}:generateContent` | Gemini |

`/v1/` 与 `/v1beta/` 均已关闭 `proxy_buffering`，SSE 流式响应会逐块透传，不会被 Nginx 攒批。

## 环境变量

完整清单见仓库根目录 [.env.example](../.env.example)。常用项：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `PROXY_PORT` | `8080` | 后端监听端口，同时驱动 Nginx upstream 端口 |
| `NGINX_HTTP_PORT` / `NGINX_HTTPS_PORT` | `80` / `443` | Nginx 对外端口 |
| `GIN_MODE` | `release` | `release` 或 `debug` |
| `GATEWAY_NODE` | 空（取 hostname） | 写入 `request_logs.gateway_node` |
| `POSTGRES_USER` / `POSTGRES_PASSWORD` / `POSTGRES_DB` | `root` / `123456` / `gateway` | 首次启动时由 postgres 镜像初始化 |
| `REDIS_PASSWORD` / `REDIS_DB` | `123456` / `0` | Redis 密码与库号 |
| `JWT_SECRET` / `JWT_EXPIRE` | 内置默认 / `24h` | Admin API 签名密钥与有效期 |
| `CB_FAILURE_THRESHOLD` / `CB_RECOVERY_INTERVAL` | `5` / `60s` | 熔断阈值与恢复探测间隔 |
| `CACHE_CHANNEL_EXPIRE` | `300s` | 渠道缓存池过期时间 |
| `PROXY_REQUEST_TIMEOUT` | `120s` | 上游首字节（响应头）超时 / 非流式读取总超时 |
| `PROXY_MAX_BODY_MB` | `10` | 请求体大小上限（Nginx 侧为 50 MB） |
| `PROXY_STREAM_KEEPALIVE_INTERVAL` | `15s` | 下游 SSE 空闲保活间隔（`0` 禁用） |
| `PROXY_STREAM_IDLE_TIMEOUT` | `300s` | 流式上游静默上限，超时主动结束流（`0` 禁用） |
| `LOG_REQUEST_BODY` / `LOG_RESPONSE_BODY` | `false` | 是否落库请求/响应原文 |
| `USD_CNY_RATE` | `7.2` | 美元计价模型的结算汇率 |

> 修改 `.env` 中端口、密码等值后，执行 `docker compose up -d` 让容器重建生效。
> 单独修改 `PROXY_PORT` 时无需改 Nginx 配置，模板会自动渲染新的 upstream 端口。

## 数据持久化

| 卷名 | 挂载点 | 内容 |
|------|--------|------|
| `xin-api_postgres_data` | `/var/lib/postgresql/data` | 全部业务数据 |
| `xin-api_redis_data` | `/data` | Redis AOF 持久化文件 |
| `xin-api_nginx_logs` | `/var/log/nginx` | 访问日志 / 错误日志 |

```bash
docker volume ls | grep xin-api
docker volume inspect xin-api_postgres_data
```

## 日常运维

```bash
# 查看状态（含健康检查结果）
docker compose ps

# 跟踪日志
docker compose logs -f app
docker compose logs -f nginx

# 只重建并重启应用
docker compose up -d --build app

# 重启单个服务
docker compose restart nginx

# 进入应用容器排查
docker compose exec app sh

# 查看实际生效的环境变量
docker compose exec app env
```

日志已统一配置为 `json-file` 轮转（单文件 10 MB，保留 3 份），无需额外配置 logrotate。

## 备份与恢复

```bash
# 备份（在 build 目录执行）
docker compose exec -T postgres pg_dump -U root -d gateway \
  > backup_$(date +%Y%m%d_%H%M%S).sql

# 恢复
docker compose exec -T postgres psql -U root -d gateway < backup_20260101_030000.sql
```

数据库账号名取自 `.env` 中的 `POSTGRES_USER`，如已修改请同步替换命令中的 `root`。

## 升级与回滚

```bash
# 1. 拉取新代码
git pull

# 2. 重新构建并滚动更新（PostgreSQL / Redis 容器不会重建，数据保留）
docker compose up -d --build

# 3. 清理悬空镜像
docker image prune -f
```

回滚到上一个镜像时，可将 `docker-compose.yml` 中的 `image:` 改为固定 tag（如 `xin-api:20260101`），再执行 `docker compose up -d`。

## 生产环境建议

1. **改掉所有默认凭据**：`POSTGRES_PASSWORD`、`REDIS_PASSWORD`、`JWT_SECRET`，并在首次登录后修改管理员密码。
2. **仅暴露 80/443**：compose 已将数据库与 Redis 绑定到 `127.0.0.1`；如需完全移除此映射，删除对应 `ports` 段落。
3. **启用 HTTPS**：挂载证书到 `nginx` 容器，增加 443 server 块并配置 HTTP 跳转。
4. **限制资源**（compose v2 非 Swarm 模式下使用 `deploy.resources` 仅在 `docker compose --compatibility` 时生效，也可改用 `mem_limit` / `cpus`）：

   ```yaml
   services:
     app:
       mem_limit: 2g
       cpus: 2.0
   ```

5. **接入外部数据库**：将 `POSTGRES_HOST` / `REDIS_HOST` 指向托管实例，并从 compose 中移除对应服务。
6. **定期备份**：把上面的 `pg_dump` 命令加入 cron 或系统计划任务，并将备份文件转存到对象存储。

## 故障排查

### 1. 端口被占用

```bash
# Linux / macOS
ss -tlnp | grep -E ':(80|443|5432|6379)\b'
# Windows PowerShell
Get-NetTCPConnection -LocalPort 80,443 -State Listen
```

在 `.env` 中修改 `NGINX_HTTP_PORT` / `POSTGRES_PORT` / `REDIS_PORT` 后重新 `docker compose up -d`。

### 2. 应用一直 unhealthy

```bash
docker compose logs --tail=200 app
docker compose exec app wget -q --spider http://127.0.0.1:8080/ && echo OK
```

常见原因：数据库密码与已有数据卷中的初始化密码不一致（PostgreSQL 只在空数据目录时应用 `POSTGRES_PASSWORD`）。
此时需改回原密码，或删除卷后重建：`docker compose down -v`（会清空数据）。

### 3. Nginx 502 Bad Gateway

```bash
docker compose logs --tail=100 nginx
docker compose ps app
```

确认 `app` 为 healthy，且 `.env` 中的 `PROXY_PORT` 与实际监听端口一致。

### 4. 流式响应一次性返回

检查请求是否命中 `/v1/` 或 `/v1beta/` 前缀。只有这两个 location 关闭了 `proxy_buffering`；同时确认客户端未经过额外的 CDN / 网关缓冲。

### 5. 前端页面 404 或白屏

镜像内 `web/dist` 由构建阶段生成，若缺失说明前端构建失败：

```bash
docker compose build --no-cache app
docker compose exec app ls -l web/dist
```

### 6. 完全重建

```bash
docker compose down
docker compose build --no-cache
docker compose up -d
```
