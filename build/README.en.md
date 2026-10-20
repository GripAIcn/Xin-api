# Xin-api Docker Deployment Guide

## Table of Contents

- [Directory Structure](#directory-structure)
- [Quick Start](#quick-start)
- [Architecture and Ports](#architecture-and-ports)
- [Environment Variables](#environment-variables)
- [Data Persistence](#data-persistence)
- [Day-to-Day Operations](#day-to-day-operations)
- [Backup and Restore](#backup-and-restore)
- [Upgrade and Rollback](#upgrade-and-rollback)
- [Production Recommendations](#production-recommendations)
- [Troubleshooting](#troubleshooting)

## Directory Structure

```
build/
├── Dockerfile                   # Go backend + frontend multi-stage build
├── Dockerfile.nginx             # Nginx reverse proxy image
├── docker-compose.yml           # Four services (postgres / redis / app / nginx)
├── .dockerignore                # Ignore rules for the build/ context
├── nginx/
│   ├── nginx.conf               # Nginx main configuration
│   └── default.conf.template    # Site template (PROXY_PORT injected at startup)
├── README.md                    # Chinese version
└── README.en.md                 # This file
```

> The application image is built from the **repository root**, so build exclusions come from the root `.dockerignore`.

## Quick Start

### 1. Requirements

- Docker Engine >= 24.0
- Docker Compose v2 (`docker compose`, not the legacy `docker-compose`)
- At least 2 GB of RAM recommended (`vite build` is memory-hungry)

### 2. Configure Environment Variables

```bash
cd build
cp ../.env.example .env
```

Change at least these three values before starting:

```env
POSTGRES_PASSWORD=your-strong-password
REDIS_PASSWORD=your-redis-password
JWT_SECRET=your-random-secret-at-least-32-chars   # openssl rand -base64 48
```

### 3. Start

```bash
# Run from the build directory
docker compose up -d --build
```

The first build runs `npm ci` + `vite build`, then compiles Go, then assembles the runtime image.
`app` waits for healthy PostgreSQL / Redis, and `nginx` waits for a healthy `app`.

### 4. Verify

```bash
docker compose ps          # all four services should report healthy
curl http://localhost/     # returns the landing page HTML
```

Open `http://localhost` for the web console. The default administrator is `admin / 12345678` — **change it immediately after the first login**.

### 5. Stop

```bash
docker compose down        # stops and removes containers, keeps volumes
```

## Architecture and Ports

```
Browser / SDK
    │
    ▼
┌───────────────┐        ┌──────────────────────────┐
│    nginx      │──────▶ │  app (Go, non-root user)  │
│  80 / 443     │        │  container port 8080      │
└───────────────┘        └───────┬──────────┬────────┘
                                 │          │
                          ┌──────▼───┐  ┌───▼──────┐
                          │ postgres │  │  redis   │
                          │   5432   │  │  6379    │
                          └──────────┘  └──────────┘
```

| Service | Container port | Host port | Notes |
|---------|---------------|-----------|-------|
| nginx | 80 / 443 | `${NGINX_HTTP_PORT:-80}` / `${NGINX_HTTPS_PORT:-443}` | Only public entry point |
| app | `${PROXY_PORT:-8080}` | not published | Internal only, declared with `expose` |
| postgres | 5432 | `127.0.0.1:${POSTGRES_PORT:-5432}` | Bound to loopback for local debugging |
| redis | 6379 | `127.0.0.1:${REDIS_PORT:-6379}` | Bound to loopback for local debugging |

> Port 443 is mapped but no TLS listener is configured yet. To enable HTTPS, mount certificates and add a `listen 443 ssl;` server block to `default.conf.template`.

Exposed API surfaces:

| Path | Protocol |
|------|----------|
| `/v1/chat/completions` | OpenAI Chat Completions |
| `/v1/responses` | OpenAI Responses |
| `/v1beta/models/{model}:generateContent` | Gemini |

`proxy_buffering` is disabled for both `/v1/` and `/v1beta/`, so SSE chunks are streamed straight through.

## Environment Variables

See [.env.example](../.env.example) for the full list. Common entries:

| Variable | Default | Description |
|----------|---------|-------------|
| `PROXY_PORT` | `8080` | Backend listen port; also drives the Nginx upstream |
| `NGINX_HTTP_PORT` / `NGINX_HTTPS_PORT` | `80` / `443` | Published Nginx ports |
| `GIN_MODE` | `release` | `release` or `debug` |
| `GATEWAY_NODE` | empty (hostname) | Written to `request_logs.gateway_node` |
| `POSTGRES_USER` / `POSTGRES_PASSWORD` / `POSTGRES_DB` | `root` / `123456` / `gateway` | Applied by the postgres image on first init |
| `REDIS_PASSWORD` / `REDIS_DB` | `123456` / `0` | Redis credentials and database index |
| `JWT_SECRET` / `JWT_EXPIRE` | built-in default / `24h` | Admin API signing key and lifetime |
| `CB_FAILURE_THRESHOLD` / `CB_RECOVERY_INTERVAL` | `5` / `60s` | Circuit breaker threshold and probe interval |
| `CACHE_CHANNEL_EXPIRE` | `300s` | Channel cache pool TTL |
| `PROXY_REQUEST_TIMEOUT` | `120s` | Upstream first-byte (response header) timeout / non-streaming body read limit |
| `PROXY_MAX_BODY_MB` | `10` | Request body limit (Nginx allows 50 MB) |
| `PROXY_STREAM_KEEPALIVE_INTERVAL` | `15s` | Downstream SSE idle keepalive interval (`0` disables) |
| `PROXY_STREAM_IDLE_TIMEOUT` | `300s` | Upstream stream idle limit; stream is ended on timeout (`0` disables) |
| `LOG_REQUEST_BODY` / `LOG_RESPONSE_BODY` | `false` | Persist raw request/response bodies |
| `USD_CNY_RATE` | `7.2` | Settlement rate for USD-priced models |

> After editing ports or credentials in `.env`, run `docker compose up -d` to recreate containers.
> Changing `PROXY_PORT` alone requires no Nginx edit — the template picks up the new upstream port.

## Data Persistence

| Volume | Mount | Content |
|--------|-------|---------|
| `xin-api_postgres_data` | `/var/lib/postgresql/data` | All business data |
| `xin-api_redis_data` | `/data` | Redis AOF files |
| `xin-api_nginx_logs` | `/var/log/nginx` | Access and error logs |

```bash
docker volume ls | grep xin-api
docker volume inspect xin-api_postgres_data
```

## Day-to-Day Operations

```bash
# Status including health check results
docker compose ps

# Follow logs
docker compose logs -f app
docker compose logs -f nginx

# Rebuild and restart only the application
docker compose up -d --build app

# Restart a single service
docker compose restart nginx

# Shell into the application container
docker compose exec app sh

# Inspect effective environment variables
docker compose exec app env
```

Logs use `json-file` rotation (10 MB per file, 3 files); no logrotate setup is needed.

## Backup and Restore

```bash
# Backup (run from the build directory)
docker compose exec -T postgres pg_dump -U root -d gateway \
  > backup_$(date +%Y%m%d_%H%M%S).sql

# Restore
docker compose exec -T postgres psql -U root -d gateway < backup_20260101_030000.sql
```

The user comes from `POSTGRES_USER` in `.env`; replace `root` if you changed it.

## Upgrade and Rollback

```bash
# 1. Pull new code
git pull

# 2. Rebuild and update (PostgreSQL / Redis containers stay untouched)
docker compose up -d --build

# 3. Remove dangling images
docker image prune -f
```

To roll back, pin a fixed tag in `docker-compose.yml` (e.g. `image: xin-api:20260101`) and run `docker compose up -d`.

## Production Recommendations

1. **Rotate every default credential**: `POSTGRES_PASSWORD`, `REDIS_PASSWORD`, `JWT_SECRET`, and the admin password after first login.
2. **Expose only 80/443**: PostgreSQL and Redis are already bound to `127.0.0.1`; delete their `ports` blocks to remove host access entirely.
3. **Enable HTTPS**: mount certificates into the `nginx` container, add a 443 server block, and redirect HTTP to HTTPS.
4. **Limit resources** (`deploy.resources` only applies under `docker compose --compatibility`; `mem_limit` / `cpus` works in plain Compose):

   ```yaml
   services:
     app:
       mem_limit: 2g
       cpus: 2.0
   ```

5. **Use managed databases**: point `POSTGRES_HOST` / `REDIS_HOST` at external instances and drop the bundled services.
6. **Schedule backups**: put the `pg_dump` command in cron / Task Scheduler and ship the dumps to object storage.

## Troubleshooting

### 1. Port already in use

```bash
# Linux / macOS
ss -tlnp | grep -E ':(80|443|5432|6379)\b'
# Windows PowerShell
Get-NetTCPConnection -LocalPort 80,443 -State Listen
```

Change `NGINX_HTTP_PORT` / `POSTGRES_PORT` / `REDIS_PORT` in `.env` and run `docker compose up -d` again.

### 2. Application stays unhealthy

```bash
docker compose logs --tail=200 app
docker compose exec app wget -q --spider http://127.0.0.1:8080/ && echo OK
```

Usual cause: the database password no longer matches the credential stored in an existing volume (PostgreSQL only applies `POSTGRES_PASSWORD` to an empty data directory). Restore the original password, or wipe the volume with `docker compose down -v` (destroys data).

### 3. Nginx returns 502

```bash
docker compose logs --tail=100 nginx
docker compose ps app
```

Confirm `app` is healthy and that `PROXY_PORT` in `.env` matches the actual listen port.

### 4. Streaming responses arrive all at once

Verify the request hits the `/v1/` or `/v1beta/` prefix — only those locations disable `proxy_buffering`. Also make sure no CDN or extra gateway is buffering the response.

### 5. Frontend returns 404 or a blank page

`web/dist` is produced during the image build; if it is missing the frontend build failed:

```bash
docker compose build --no-cache app
docker compose exec app ls -l web/dist
```

### 6. Full rebuild

```bash
docker compose down
docker compose build --no-cache
docker compose up -d
```
