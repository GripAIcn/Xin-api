# Xin-api Docker Deployment Guide

## Table of Contents

- [Directory Structure](#directory-structure)
- [Quick Start](#quick-start)
  - [1. Environment Preparation](#1-environment-preparation)
  - [2. Configure Environment Variables](#2-configure-environment-variables)
  - [3. Start Services](#3-start-services)
  - [4. Access Services](#4-access-services)
- [Deployment Script Usage](#deployment-script-usage)
- [Service Architecture](#service-architecture)
  - [Port Mapping](#port-mapping)
- [Data Persistence](#data-persistence)
- [Production Deployment Recommendations](#production-deployment-recommendations)
  - [1. Security Configuration](#1-security-configuration)
  - [2. Performance Optimization](#2-performance-optimization)
  - [3. Log Management](#3-log-management)
  - [4. Backup Strategy](#4-backup-strategy)
- [Troubleshooting](#troubleshooting)
  - [View Service Logs](#view-service-logs)
  - [Common Issues](#common-issues)
- [Update and Upgrade](#update-and-upgrade)
- [Uninstall and Cleanup](#uninstall-and-cleanup)
- [Support](#support)

## Directory Structure

```
build/
├── Dockerfile              # Go backend multi-stage build (includes frontend static files)
├── Dockerfile.nginx        # Nginx reverse proxy configuration
├── docker-compose.yml      # Service orchestration
├── .dockerignore          # Docker build ignore files
├── nginx/
│   ├── nginx.conf         # Nginx main configuration
│   └── default.conf       # Site configuration
├── scripts/
│   └── deploy.sh          # Deployment script
└── README.md              # This file
web/                        # Frontend source directory (integrated into backend build)
├── src/
├── public/
└── package.json
```

## Quick Start

### 1. Environment Preparation

Ensure you have installed:
- Docker >= 20.10
- Docker Compose >= 2.0

### 2. Configure Environment Variables

```bash
# Copy environment variable template
cp ../.env.example .env

# Edit .env file, modify the following key configurations
vim .env
```

**Key Configuration Items:**

```env
# Database configuration (be sure to change password in production)
POSTGRES_USER=root
POSTGRES_PASSWORD=your-strong-password
POSTGRES_DB=gateway

# Redis configuration
REDIS_PASSWORD=your-redis-password

# JWT secret (be sure to change to a random strong password in production)
JWT_SECRET=your-jwt-secret-key-min-32-chars

# Run mode
GIN_MODE=release
```

### 3. Start Services

Start all services with one command using the deployment script:

```bash
# Enter build directory
cd build

# Start all services
./scripts/deploy.sh start
```

Or use Docker Compose commands:

```bash
docker-compose up -d --build
```

### 4. Access Services

- **Web Interface**: http://localhost
- **API Address**: http://localhost/v1

## Deployment Script Usage

```bash
./scripts/deploy.sh [command]

Commands:
  start    Start all services
  stop     Stop all services
  restart  Restart all services
  logs     View logs (optional: specify service name)
  status   View service status
  build    Rebuild images
  reset    Complete reset (delete all data)
  update   Update code and restart services
  help     Show help information
```

### Common Command Examples

```bash
# Start services
./scripts/deploy.sh start

# View all service status
./scripts/deploy.sh status

# View application logs
./scripts/deploy.sh logs app

# View Nginx logs in real-time
./scripts/deploy.sh logs -f nginx

# Restart services
./scripts/deploy.sh restart

# Stop services
./scripts/deploy.sh stop

# Update to latest version
./scripts/deploy.sh update
```

## Service Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Docker Compose                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │    Nginx     │  │  Xin-api     │  │  PostgreSQL  │       │
│  │   (80/443)   │  │   (8080)     │  │    (5432)    │       │
│  └──────┬───────┘  └──────────────┘  └──────────────┘       │
│         │                                                    │
│         │  Static files                                      │
│         │  API proxy                                         │
│         ▼                                                    │
│  ┌──────────────┐                                            │
│  │    Redis     │                                            │
│  │    (6379)    │                                            │
│  └──────────────┘                                            │
└─────────────────────────────────────────────────────────────┘
```

### Port Mapping

| Service | Container Port | Host Port | Description |
|---------|---------------|-----------|-------------|
| Nginx | 80/443 | 80/443 | Web entry point |
| Xin-api | 8080 | 8080 | API service |
| PostgreSQL | 5432 | 5432 | Database |
| Redis | 6379 | 6379 | Cache service |

## Data Persistence

Data is persisted through Docker Volumes:

- `postgres_data`: PostgreSQL data
- `redis_data`: Redis data
- `nginx_logs`: Nginx logs

View data volumes:

```bash
docker volume ls
docker volume inspect build_postgres_data
```

## Production Deployment Recommendations

### 1. Security Configuration

- **Change default passwords**: Be sure to modify `POSTGRES_PASSWORD`, `REDIS_PASSWORD`, `JWT_SECRET`
- **Close port exposure**: In production, only expose Nginx ports 80/443
- **Enable HTTPS**: Configure SSL certificates

### 2. Performance Optimization

Edit `docker-compose.yml` to adjust resource limits:

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

### 3. Log Management

Configure log rotation:

```yaml
services:
  app:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

### 4. Backup Strategy

Regularly backup PostgreSQL data:

```bash
# Backup database
docker exec xin-api-postgres pg_dump -U root gateway > backup_$(date +%Y%m%d).sql

# Restore database
docker exec -i xin-api-postgres psql -U root gateway < backup_20240101.sql
```

## Troubleshooting

### View Service Logs

```bash
# View all service logs
docker-compose logs

# View specific service logs
docker-compose logs app
docker-compose logs postgres
docker-compose logs redis
docker-compose logs nginx

# Real-time log tracking
docker-compose logs -f app
```

### Common Issues

#### 1. Port Occupied

```bash
# Check port usage
netstat -tlnp | grep 80

# Modify port mapping in docker-compose.yml
```

#### 2. Database Connection Failed

```bash
# Check PostgreSQL status
docker-compose ps postgres
docker-compose logs postgres

# Enter database container
docker exec -it xin-api-postgres psql -U root -d gateway
```

#### 3. Application Startup Failed

```bash
# Check application logs
docker-compose logs app

# Check environment variables
docker exec xin-api-app env
```

## Update and Upgrade

```bash
# 1. Pull latest code
git pull

# 2. Rebuild images
docker-compose build --no-cache

# 3. Restart services
docker-compose down
docker-compose up -d
```

## Uninstall and Cleanup

```bash
# Stop and remove containers
docker-compose down

# Delete data volumes (caution!)
docker-compose down -v

# Delete images
docker rmi build-app
```

## Support

If you have any questions, please check the project documentation or submit an Issue.
