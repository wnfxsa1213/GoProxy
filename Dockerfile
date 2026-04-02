# 构建阶段（使用完整 Debian 镜像，内置 gcc，避免 alpine apk 问题）
FROM golang:1.25 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o proxy-pool .

# 运行阶段（使用轻量 debian-slim）
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates tzdata && \
    rm -rf /var/lib/apt/lists/*

ENV TZ=Asia/Shanghai

WORKDIR /app
COPY --from=builder /app/proxy-pool .

EXPOSE 7776 7777 7778 7779 7780 7781 7782

CMD ["./proxy-pool"]
