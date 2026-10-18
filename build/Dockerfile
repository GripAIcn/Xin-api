# ============================================
# 阶段一：前端构建（Vite 多入口：index.html + app.html）
# ============================================
FROM node:22-alpine AS frontend-builder

ARG NPM_REGISTRY=https://registry.npmmirror.com
# 站点正式地址：预渲染产物中的 canonical / og / sitemap / robots 全部由它拼接
ARG SITE_URL=https://api.gripai.cn

WORKDIR /web

COPY web/package.json web/package-lock.json ./

RUN npm ci --no-audit --no-fund --registry=${NPM_REGISTRY}

COPY web/ ./

RUN SITE_URL=${SITE_URL} npm run build

# ============================================
# 阶段二：Go 后端构建
# ============================================
FROM golang:1.25-alpine AS go-builder

ARG TARGETOS
ARG TARGETARCH
ARG GOPROXY=https://goproxy.cn,direct

ENV CGO_ENABLED=0 \
    GOPROXY=${GOPROXY}

WORKDIR /app

# 先只拷贝依赖清单，最大化利用构建缓存
COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -buildvcs=false \
    -ldflags="-s -w" \
    -o /out/xin-api \
    ./cmd/xin_api

# ============================================
# 阶段三：最终运行镜像
# ============================================
FROM alpine:3.21

LABEL org.opencontainers.image.title="xin-api" \
      org.opencontainers.image.description="AI API Gateway (OpenAI / Anthropic / Gemini compatible)" \
      org.opencontainers.image.licenses="MIT"

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -g 10001 -S xinapi \
    && adduser -u 10001 -S -G xinapi -h /app xinapi

ENV TZ=Asia/Shanghai \
    GIN_MODE=release \
    PROXY_PORT=8080

WORKDIR /app

COPY --from=go-builder /out/xin-api ./xin-api
COPY --from=frontend-builder --chown=xinapi:xinapi /web/dist ./web/dist

USER xinapi

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD wget -q --spider "http://127.0.0.1:${PROXY_PORT}/" || exit 1

STOPSIGNAL SIGTERM

ENTRYPOINT ["./xin-api"]
