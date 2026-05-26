#!/bin/bash

# Xin-api Docker 部署脚本
# 使用方法: ./deploy.sh [start|stop|restart|logs|status|build]

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BUILD_DIR="$(dirname "$SCRIPT_DIR")"
PROJECT_DIR="$(dirname "$BUILD_DIR")"

# 切换到 build 目录
cd "$BUILD_DIR"

# 打印带颜色的消息
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# 检查环境
check_env() {
    if ! command -v docker &> /dev/null; then
        print_error "Docker 未安装，请先安装 Docker"
        exit 1
    fi

    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        print_error "Docker Compose 未安装，请先安装 Docker Compose"
        exit 1
    fi

    if [ ! -f ".env" ]; then
        print_warn ".env 文件不存在，将使用默认配置"
        print_info "建议复制 .env.example 到 .env 并根据实际情况修改"
    fi
}

# 使用 docker compose 命令
docker_compose() {
    if command -v docker-compose &> /dev/null; then
        docker-compose "$@"
    else
        docker compose "$@"
    fi
}

# 启动服务
start() {
    print_info "正在启动 Xin-api 服务..."
    check_env
    
    print_step "1/3 启动数据库和缓存服务..."
    docker_compose up -d postgres redis
    
    print_step "2/3 等待数据库就绪..."
    sleep 5
    
    print_step "3/3 启动所有服务..."
    docker_compose up -d --build
    
    print_info "服务启动完成！"
    print_info "访问地址: http://localhost"
    print_info "API 地址: http://localhost/v1"
}

# 停止服务
stop() {
    print_info "正在停止 Xin-api 服务..."
    docker_compose down
    print_info "服务已停止"
}

# 重启服务
restart() {
    print_info "正在重启 Xin-api 服务..."
    docker_compose restart
    print_info "服务重启完成"
}

# 查看日志
logs() {
    if [ -z "$2" ]; then
        docker_compose logs -f
    else
        docker_compose logs -f "$2"
    fi
}

# 查看状态
status() {
    docker_compose ps
}

# 构建镜像
build() {
    print_info "正在构建 Docker 镜像..."
    docker_compose build --no-cache
    print_info "镜像构建完成"
}

# 查看指定服务日志
logs_service() {
    if [ -z "$2" ]; then
        print_error "请指定服务名称: app, nginx, postgres, redis"
        exit 1
    fi
    docker_compose logs -f "$2"
}

# 完全重置（删除数据卷）
reset() {
    print_warn "警告: 此操作将删除所有数据，包括数据库数据！"
    read -p "确定要继续吗? (y/N): " confirm
    if [[ $confirm == [yY] || $confirm == [yY][eE][sS] ]]; then
        print_info "正在停止服务并删除数据..."
        docker_compose down -v
        print_info "数据已清除，可以使用 ./deploy.sh start 重新启动"
    else
        print_info "操作已取消"
    fi
}

# 更新服务（拉取最新代码并重启）
update() {
    print_info "正在更新服务..."
    cd "$PROJECT_DIR"
    git pull
    cd "$BUILD_DIR"
    docker_compose down
    docker_compose build --no-cache
    docker_compose up -d
    print_info "服务更新完成"
}

# 健康检查
health() {
    print_info "检查服务健康状态..."
    echo ""
    
    services=("postgres" "redis" "app" "nginx")
    for service in "${services[@]}"; do
        if docker_compose ps "$service" | grep -q "healthy"; then
            echo -e "${GREEN}✓${NC} $service: 健康"
        elif docker_compose ps "$service" | grep -q "Up"; then
            echo -e "${YELLOW}○${NC} $service: 运行中 (健康检查中)"
        else
            echo -e "${RED}✗${NC} $service: 异常"
        fi
    done
}

# 显示帮助信息
help() {
    echo "Xin-api Docker 部署脚本"
    echo ""
    echo "用法: ./deploy.sh [命令]"
    echo ""
    echo "命令:"
    echo "  start    启动所有服务"
    echo "  stop     停止所有服务"
    echo "  restart  重启所有服务"
    echo "  logs     查看所有日志 (可选: 指定服务名如 app, frontend, nginx)"
    echo "  status   查看服务状态"
    echo "  build    重新构建镜像"
    echo "  health   检查服务健康状态"
    echo "  reset    完全重置（删除所有数据）"
    echo "  update   更新代码并重启服务"
    echo "  help     显示帮助信息"
    echo ""
    echo "示例:"
    echo "  ./deploy.sh start          # 启动服务"
    echo "  ./deploy.sh logs app       # 查看应用日志"
    echo "  ./deploy.sh health         # 检查服务健康状态"
    echo "  ./deploy.sh status         # 查看服务状态"
}

# 主逻辑
case "${1:-help}" in
    start)
        start
        ;;
    stop)
        stop
        ;;
    restart)
        restart
        ;;
    logs)
        if [ -z "$2" ]; then
            logs "$@"
        else
            logs_service "$@"
        fi
        ;;
    status)
        status
        ;;
    build)
        build
        ;;
    health)
        health
        ;;
    reset)
        reset
        ;;
    update)
        update
        ;;
    help|--help|-h)
        help
        ;;
    *)
        print_error "未知命令: $1"
        help
        exit 1
        ;;
esac
