# Xin API 🚀

[中文](./README.md) | English

<p align="center">
  <strong>A high-performance gateway for unified LLM API management, full-link tracing, and smart distribution</strong>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go Version">
  <img src="https://img.shields.io/badge/Gin-1.12-00ADD8?style=flat-square" alt="Gin Framework">
  <img src="https://img.shields.io/badge/React-19-61DAFB?style=flat-square&logo=react&logoColor=white" alt="React">
  <img src="https://img.shields.io/badge/PostgreSQL-16-336791?style=flat-square&logo=postgresql&logoColor=white" alt="PostgreSQL">
  <img src="https://img.shields.io/badge/Redis-7-DC382D?style=flat-square&logo=redis&logoColor=white" alt="Redis">
  <img src="https://img.shields.io/badge/Nginx-Alpine-009639?style=flat-square&logo=nginx&logoColor=white" alt="Nginx">
  <img src="https://img.shields.io/badge/Docker-Ready-2496ED?style=flat-square&logo=docker&logoColor=white" alt="Docker Ready">
</p>

---

## ✨ Introduction

Xin API is a high-performance Large Language Model (LLM) gateway service that seamlessly integrates with mainstream models like OpenAI, Anthropic Claude, DeepSeek, Gemini, and more — providing unified API access, channel routing, billing, and observability.

### Core Features

- **🎯 Unified API Gateway**: Expose `/v1/chat/completions`, `/v1/responses`, and `/v1beta/models/{model}:generateContent` as inbound endpoints, standardizing different LLM APIs behind a unified gateway
- **🔀 Smart Distribution**: Route by priority plus weighted round-robin within the same priority; retry automatically and exclude failed channels
- **🛡️ Circuit Breaker**: Redis-backed distributed passive circuit breaker that isolates a channel after consecutive failures and probes for recovery at a configurable interval
- **⚡ Rate Limiting (Reserved)**: A Redis token-bucket limiter middleware is implemented but not yet mounted on the data plane; `rpm_limit` / `tpm_limit` are currently inactive
- **👥 Unified Channel Management**: Manage upstream provider channels with weights, priorities, and status, including per-channel model management (create/update/delete, enable/disable, fetch models from upstream)
- **🗂️ Model Management**: Aggregate available models across channels by logical model name, with cross-channel bulk create, edit, and enable/disable
- **🔁 Model Name Mapping**: Register the upstream model name per "channel × external model name"; rewrite the outgoing request to the upstream name and restore the external name in responses
- **🧩 Multi-Protocol Adaptation**: Accept OpenAI Chat, OpenAI Responses, and native Gemini inbound protocols; adapt outbound requests per channel provider (OpenAI / DeepSeek / Anthropic / Gemini)
- **💰 Billing & Settlement**: Price plans support USD / CNY pricing, settle in CNY using the configured exchange rate, and record balance changes with exchange-rate snapshots
- **🔑 API Key Management**: Flexible API key distribution and revocation
- **🧪 Channel Health Check**: One-click manual channel testing, supporting single/batch testing of multi-model connectivity
- **📋 Full-Link Logging**: Request logs, billing logs, and audit logs, with optional request/response payload persistence (disabled by default)
- **📈 Channel Metrics Aggregation**: 60-second windows for success rate, TTFT, latency, TPOT, RPM/TPM, with a 7-day backfill on startup
- **🌐 Web Admin Dashboard**: Modern management UI built with React 19 + Tailwind CSS
- **🐳 Containerized Deployment**: Out-of-the-box Docker Compose deployment

---

## 🏗️ Architecture

### Tech Stack

| Layer | Technology |
|-------|------------|
| **Frontend** | React 19 + TypeScript + Vite + Tailwind CSS + Radix UI + Zustand |
| **Backend** | Go 1.25 + Gin 1.12 + GORM + JWT |
| **Database** | PostgreSQL 16 (metadata; tables created via AutoMigrate on startup) |
| **Cache** | Redis 7 (channel query cache, circuit breaker state) |
| **Proxy** | Nginx (reverse proxy, SSE passthrough) |
| **Deployment** | Docker + Docker Compose |

---

## 🚀 Quick Start

### Prerequisites

- Go 1.25+
- Node.js 22+ (frontend `engines`: `^20.19.0 || >=22.12.0`)
- PostgreSQL 16+
- Redis 7+
- Docker & Docker Compose (recommended)

### Docker Compose One-Click Deployment (Recommended)

```bash
# 1. Clone the repository
git clone https://github.com/GripAIcn/Xin-api.git
cd Xin-api

# 2. Configure environment variables (Compose reads build/.env)
cp .env.example build/.env
# Edit build/.env to set database password, Redis password, JWT secret, etc.

# 3. Build and start services
cd build
docker compose up -d --build

# 4. Access the services
# Web Dashboard: http://localhost
# API Endpoint:  http://localhost/v1
```

On first startup the backend automatically creates the database, runs GORM `AutoMigrate`, and creates the default administrator account `admin / 12345678`. Change the password immediately after signing in.

### Local Development Setup

#### 1. Start Dependencies

```bash
# Start PostgreSQL and Redis
docker run -d --name xin-postgres \
  -e POSTGRES_PASSWORD=123456 \
  -p 5432:5432 \
  postgres:16-alpine

docker run -d --name xin-redis \
  -e REDIS_PASSWORD=123456 \
  -p 6379:6379 \
  redis:7-alpine
```

#### 2. Start Backend

```bash
# Configure environment (must match the PostgreSQL / Redis credentials above)
cp .env.example .env

# Install dependencies and run
go mod download
go run ./cmd/xin_api/main.go
```

#### 3. Start Frontend Dev Server

```bash
cd web
npm install
npm run dev
```

The frontend dev server runs on `http://localhost:5173` by default; CORS to the backend is preconfigured for development.

---

## 🔧 Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PROXY_PORT` | `8080` | Backend service port |
| `GATEWAY_NODE` | Hostname | Gateway instance identifier, stored in `request_logs.gateway_node` |
| `GIN_MODE` | `release` (`.env.example`) | Gin run mode (debug/release) |
| `LOG_LEVEL` | `info` | Reserved; not read by the current code |
| `POSTGRES_HOST` | `127.0.0.1` | PostgreSQL host |
| `POSTGRES_USER` | `root` | PostgreSQL username |
| `POSTGRES_PASSWORD` | empty | PostgreSQL password (Compose default `123456`) |
| `POSTGRES_DB` | `gateway` | Database name; created automatically if missing |
| `POSTGRES_PORT` | `5432` | PostgreSQL port |
| `REDIS_HOST` | `127.0.0.1` | Redis host |
| `REDIS_PORT` | `6379` | Redis port |
| `REDIS_PASSWORD` | empty | Redis password (Compose default `123456`) |
| `REDIS_DB` | `0` | Redis database number |
| `REDIS_POOL_SIZE` | `20` | Redis connection pool size |
| `JWT_SECRET` | `xin-api-default-secret-key-2026` | JWT signing secret (must change in production) |
| `JWT_EXPIRE` | `24h` | JWT expiration time |
| `CB_FAILURE_THRESHOLD` | `5` | Consecutive failure threshold for the circuit breaker |
| `CB_RECOVERY_INTERVAL` | `60s` | Circuit breaker recovery probe interval |
| `CACHE_CHANNEL_EXPIRE` | `300s` | Channel cache expiration time |
| `LOG_REQUEST_BODY` | `false` | Persist request bodies to `request_payloads` |
| `LOG_RESPONSE_BODY` | `false` | Persist response bodies (streaming responses capped at 1 MiB) |
| `PROXY_REQUEST_TIMEOUT` | `120s` | Upstream request timeout |
| `PROXY_MAX_BODY_MB` | `10` | Max request body size (MB) |
| `USD_CNY_RATE` | `7.2` | Initial USD→CNY settlement rate (written to `exchange_rates` only once) |
| `NGINX_HTTP_PORT` | `80` | Nginx HTTP port (Docker Compose only) |
| `NGINX_HTTPS_PORT` | `443` | Nginx HTTPS port (Docker Compose only) |
| `DEFAULT_RPM` | `60` | Reserved; currently inactive |
| `DEFAULT_TPM` | `100000` | Reserved; currently inactive |

---

## 🌐 Live Demo

Try Xin API Admin Dashboard at [http://api.gripai.cn/](http://api.gripai.cn/)

- **Username**: admin
- **Password**: 12345678

---

## 📸 Screenshots

### Landing Page

![Landing Page](screenshot/landing.png)

### Dashboard

![Dashboard](screenshot/dashboard.png)

### Admin Dashboard Features

- **📊 Dashboard**: Overview of requests / success rate / token usage / cost / average latency (today, last 7 days, last 30 days), with request, token, and cost trends plus model, channel, and error rankings
- **🔑 API Key Management**: Distribute, list, and revoke the current user's API keys
- **📋 Request Logs**: Filter by time, user, model, channel, status, and log type; inspect billing details for each request
- **🧾 Billing Logs**: Merged view of consumption / recharge / refund records with pagination and multi-dimensional filters
- **👤 Account & Balance**: View profile and balance; update email/phone and password
- **👥 User Management** (admin): Create accounts, enable/disable, reset passwords, and recharge users
- **🔌 Channel Management** (admin): Configure upstream LLM channels with priority and weight, one-click channel health testing, and per-channel model management (create/update/delete, enable/disable, fetch models from upstream)
- **🗂️ Model Management** (admin): View all channel models aggregated by logical model name; cross-channel create/edit and enable/disable
- **📈 Channel Monitoring** (admin): Aggregated success rate, TTFT, latency, TPOT, RPM/TPM and trend charts by channel / model (aggregated every minute, with a 7-day backfill on startup)
- **💰 Pricing Management** (admin): Maintain price plans (USD/CNY pricing) and model bindings, and configure the USD→CNY settlement rate

---

## 🐳 Production Deployment

### Docker Compose Deployment

For complete deployment configuration, see [build/README.md](build/README.md)

Deployment directory structure:

```
build/
├── Dockerfile                   # Unified multi-stage build (Node.js + Go + Alpine)
├── Dockerfile.nginx             # Nginx reverse proxy image
├── docker-compose.yml           # Service orchestration (PostgreSQL + Redis + App + Nginx)
├── .dockerignore                # Ignore rules for the build/ context
├── nginx/                       # Nginx configuration files
│   ├── nginx.conf
│   └── default.conf.template    # Site template (PROXY_PORT injected at startup)
└── README.md                    # Deployment guide
```

Start commands:

```bash
cd build

# Prepare environment variables (first deployment)
cp ../.env.example .env

# Build and start all services
docker compose up -d --build

# Check service status
docker compose ps

# View application logs
docker compose logs -f app

# Stop services
docker compose down
```

### Service Architecture

```
User Request → Nginx (80/443)
    ├── /v1/*      → Backend API Service (app:8080)
    ├── /v1beta/*  → Gemini-compatible API (app:8080)
    └── /*         → Frontend Static Files (app:8080, served by the backend)

Backend Dependencies: PostgreSQL + Redis
```

### Service Ports

| Service | Container Port | Host Port | Description |
|---------|---------------|-----------|-------------|
| Nginx | 80/443 | 80/443 | Web Entry (Unified) |
| Xin-api | 8080 | - | Backend + Frontend Static Files (Internal) |
| PostgreSQL | 5432 | 127.0.0.1:5432 | Database (loopback only) |
| Redis | 6379 | 127.0.0.1:6379 | Cache Service (loopback only) |

### Data Persistence

Data is automatically persisted via Docker Volumes:

- `xin-api_postgres_data`: PostgreSQL data
- `xin-api_redis_data`: Redis data (AOF)
- `xin-api_nginx_logs`: Nginx logs

### Production Recommendations

1. **Change default passwords**: Update database and Redis passwords, and change the default administrator password
2. **Rotate JWT secret**: Use a strong random string for `JWT_SECRET`
3. **Enable HTTPS**: Configure Nginx SSL certificates
4. **Limit exposed ports**: In production, expose only Nginx ports 80/443
5. **Configure log rotation**: Prevent logs from filling disk space
6. **Regular backups**: Schedule regular PostgreSQL backups

---

## 📄 License

This project is licensed under the [MIT License](LICENSE).

---

<p align="center">
  <strong>⭐ If you find this project helpful, please give it a Star!</strong>
</p>

## Note

This project is released under the MIT license. **In addition**, attribution and a link to this project must be retained at the bottom of the page. If you do not wish to include attribution, prior authorization must be obtained.

The same applies to derivative projects based on this project.

According to the MIT license, users bear the risk and responsibility of using this project, which is unrelated to the developers of this open-source project.
