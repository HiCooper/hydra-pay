#!/bin/bash
# 本地开发用: 启动 sidecar 监控 hydra-pay 日志
# 生产环境应使用 K8s sidecar 或 Docker Compose companion

SIDECAR_DIR="$(cd "$(dirname "$0")/../../sentinel/sidecar" && pwd)"

export SENTINEL_SERVICE_NAME=hydra-pay
export SENTINEL_PLATFORM_URL=http://localhost:8084

# 用法: ./sentinel-sidecar.sh
# pipe 用法: go run ./cmd/server 2>&1 | ./sentinel-sidecar.sh
cd "$SIDECAR_DIR" && go run ./cmd
