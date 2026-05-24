#!/bin/bash
set -e

cd "$(dirname "$0")"

echo "==> 清理端口 8082 残留进程 ..."
lsof -ti :8082 | xargs kill -9 2>/dev/null; true

echo "==> 编译 pay-service ..."
go build -o /tmp/pay-server ./cmd/server/

echo "==> 加载环境变量 ..."
set -a
source .env
set +a

echo "==> 启动 pay-service (http://localhost:8082) ..."
/tmp/pay-server
