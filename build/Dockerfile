# ============================================
# 阶段一：前端构建
# ============================================
FROM node:22-alpine AS frontend-builder

WORKDIR /web

COPY web/package.json web/package-lock.json ./

RUN npm ci --ignore-scripts

COPY web/ .

RUN npm run build

# ============================================
# 阶段二：Go 后端构建
# ============================================
FROM golang:1.25-alpine AS go-builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o xin-api \
    ./cmd/xin_api/main.go

# ============================================
# 阶段三：最终运行镜像
# ============================================
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

ENV TZ=Asia/Shanghai

WORKDIR /app

COPY --from=go-builder /app/xin-api .

COPY --from=frontend-builder /web/dist ./web/dist

EXPOSE 8080

ENTRYPOINT ["./xin-api"]
