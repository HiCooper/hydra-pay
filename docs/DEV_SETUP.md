# Hydra-Pay 本地开发快速启动指南

本文档提供两种启动本地开发环境的方式。推荐使用 **Docker Compose 方案**（最简单、最可靠）。

## 🚀 方案 A：Docker Compose 一键启动（推荐）

### 前置要求

- Docker 20.10+
- Docker Compose 2.0+
- 至少 4GB RAM 可用
- 不超过 10GB 磁盘空间

### 启动步骤

1. **启动所有服务**

```bash
# 在项目根目录运行
docker-compose up -d

# 查看启动日志
docker-compose logs -f pay-service
```

2. **验证服务启动成功**

```bash
# 检查所有容器状态
docker-compose ps

# 应该看到所有服务都是 Up 状态
# CONTAINER ID   IMAGE                  STATUS
# ...            hydra-pay-postgres     Up (healthy)
# ...            hydra-pay-redis        Up (healthy)
# ...            hydra-pay-service      Up (healthy)
# ...            hydra-pay-admin        Up
# ...            hydra-pay-frontend     Up
# ...            hydra-pay-portal       Up
# ...            hydra-pay-prometheus   Up
# ...            hydra-pay-grafana      Up
```

3. **访问应用**

| 应用 | 地址 | 说明 |
|------|------|------|
| 后端 API | http://localhost:8080 | RESTful API 端点 |
| 管理后台 | http://localhost:5173 | 商户管理后台 |
| 支付页面 | http://localhost:5174 | 托管结算页面 |
| 商户门户 | http://localhost:5175 | 商户自助服务 |
| Prometheus | http://localhost:9090 | 指标展示 |
| Grafana | http://localhost:3000 | 仪表板 (账号: admin/admin) |

### 验证检查表

运行以下命令逐一验证：

```bash
# 1. 检查后端健康状态
curl http://localhost:8080/health
# 预期返回: {"status":"ok","checks":{"database":"ok"}}

# 2. 检查 Prometheus 指标
curl http://localhost:8080/metrics | head -20

# 3. 创建测试支付
curl -X POST http://localhost:8080/v1/payments/create \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk_test_xxxx" \
  -d '{
    "amount": 9900,
    "currency": "CNY",
    "channel": "alipay",
    "product_name": "Test Product"
  }'

# 4. 查询支付状态
curl http://localhost:8080/v1/payments/{payment_id} \
  -H "Authorization: Bearer sk_test_xxxx"
```

### 常见问题

**Q: 容器启动失败或不健康**

```bash
# 查看详细日志
docker-compose logs pay-service

# 完全重启
docker-compose down -v
docker-compose up -d
```

**Q: 端口已被占用**

编辑 `docker-compose.yml`，修改 ports 部分，例如：
```yaml
ports:
  - "8081:8080"  # 改为 8081
```

**Q: 数据库迁移失败**

```bash
# 手动连接数据库
docker-compose exec postgres psql -U hydra -d hydra_pay

# 或重建容器
docker-compose down -v postgres
docker-compose up -d postgres
```

### 停止和清理

```bash
# 停止所有服务（保留数据）
docker-compose stop

# 停止并删除所有容器（保留卷）
docker-compose down

# 完全清理（包括数据库）
docker-compose down -v
```

---

## 🛠️ 方案 B：本地开发环境（手工启动）

### 前置要求

- Go 1.20+
- Node.js 16+
- PostgreSQL 14+
- Redis 6+

### 启动步骤

1. **启动外部服务**

```bash
# PostgreSQL (macOS with Homebrew)
brew services start postgresql

# 创建数据库（如未存在）
createdb hydra_pay

# Redis (macOS with Homebrew)
brew services start redis
```

2. **启动后端服务**

```bash
cd hydra-pay/service

# 复制环境变量模板
cp .env.example .env

# （可选）修改 .env 文件中的配置

# 运行后端
go run ./cmd/server

# 预期输出: [GIN] listening on :8080
```

3. **启动前端服务** (新终端窗口)

```bash
# 管理后台
cd hydra-pay/admin
npm install
npm run dev
# 访问: http://localhost:5173

# 支付页面（另一个新终端）
cd hydra-pay/pay-frontend
npm install
npm run dev
# 访问: http://localhost:5174

# 商户门户（另一个新终端）
cd hydra-pay/portal
npm install
npm run dev
# 访问: http://localhost:5175
```

### 环境变量配置

编辑 `service/.env`：

```bash
# 服务器
PORT=8080
GIN_MODE=debug
LOG_LEVEL=debug

# 数据库
DATABASE_URL=postgres://hydra:hydra_secret@localhost:5432/hydra_pay?sslmode=disable

# Redis
REDIS_ADDR=localhost:6379

# 支付宝（获取真实凭证或使用沙箱凭证）
ALIPAY_APP_ID=your_alipay_app_id
ALIPAY_PRIVATE_KEY=your_alipay_private_key
ALIPAY_PUBLIC_KEY=your_alipay_public_key

# 微信支付
WECHAT_MCH_ID=your_mch_id
WECHAT_API_V3_KEY=your_api_v3_key
WECHAT_CERT_PATH=./certs/wechat_cert.pem
WECHAT_KEY_PATH=./certs/wechat_key.pem
```

---

## 🧪 测试验证

### 运行单元测试

```bash
cd service

# 运行所有单元测试
go test ./...

# 运行特定包的测试
go test ./internal/service -v

# 运行集成测试（需要 Docker）
go test -tags integration ./internal/integration -v

# 生成覆盖率报告
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### 运行集成测试

```bash
cd service

# 仅运行超时处理测试
go test ./internal/service -run TestSyncExpiredOrders -v

# 运行所有集成测试（跳过 short 模式）
go test ./internal/integration -v
```

---

## 📊 监控和调试

### Prometheus 指标

访问 http://localhost:9090 查询指标：

```promql
# 支付成功率
rate(payments_created_total{status="success"}[5m]) / rate(payments_created_total[5m])

# API 响应时间 P99
histogram_quantile(0.99, http_request_duration_seconds)

# 错误率
rate(http_requests_total{status=~"5.."}[5m])
```

### Grafana 仪表板

1. 访问 http://localhost:3000 (admin/admin)
2. 配置 Prometheus 数据源：http://prometheus:9090
3. 导入或创建仪表板查看实时指标

### 查看日志

```bash
# 后端日志
docker-compose logs -f pay-service

# 所有服务日志
docker-compose logs -f

# 特定时间段
docker-compose logs --since 10m pay-service
```

---

## 🔄 常见工作流

### 开发新功能

```bash
# 1. 启动环境
docker-compose up -d

# 2. 开发并测试
# 编辑代码 → go run ./cmd/server

# 3. 运行测试
go test ./internal/service -v

# 4. 验证 API
curl http://localhost:8080/v1/payments/create ...

# 5. 查看监控
# 打开 Grafana: http://localhost:3000
```

### 调试 webhook 回调

```bash
# 使用 ngrok 将本地服务暴露到公网
ngrok http 8080

# 配置回调 URL
export NOTIFY_URL=https://xxxx.ngrok.io/v1/webhooks/alipay

# 在支付宝测试平台发送回调
# Hydra-Pay 会收到并处理回调
```

### 数据库操作

```bash
# 连接数据库
docker-compose exec postgres psql -U hydra -d hydra_pay

# 查看表结构
\dt

# 查询支付记录
SELECT * FROM payments LIMIT 10;

# 导出数据
pg_dump -U hydra hydra_pay > backup.sql

# 导入数据
psql -U hydra hydra_pay < backup.sql
```

---

## 🚨 故障排查

### 服务无法启动

1. **检查端口占用**
```bash
# macOS
lsof -i :8080
kill -9 <PID>

# Linux
netstat -tulpn | grep 8080
```

2. **检查依赖服务**
```bash
docker-compose ps
docker-compose logs

# 重启一个服务
docker-compose restart postgres
```

### 数据库连接错误

```bash
# 验证连接字符串
psql "postgres://hydra:hydra_secret@localhost:5432/hydra_pay?sslmode=disable"

# 检查 PostgreSQL 状态
docker-compose logs postgres
```

### 支付创建失败

```bash
# 查看详细日志
docker-compose logs -f pay-service | grep "payment_id\|error"

# 检查配置和 API Key
curl http://localhost:8080/v1/health

# 验证请求格式
# API Key 需要在 Authorization header 中
```

---

## 📚 有用的命令

```bash
# 查看所有容器
docker-compose ps

# 查看实时日志
docker-compose logs -f [service]

# 进入容器 shell
docker-compose exec [service] sh

# 重启服务
docker-compose restart [service]

# 查看容器网络
docker network inspect hydra-pay_hydra-network

# 清理未使用的资源
docker system prune -a

# 导出/导入镜像
docker save hydra-pay-service -o image.tar
docker load -i image.tar
```

---

## 🆘 需要帮助？

如果遇到问题，请提供：

1. 运行的命令
2. 完整的错误信息
3. 系统信息（OS、Docker 版本）
4. 相关日志输出

在 GitHub 上创建 Issue 或联系开发团队。
