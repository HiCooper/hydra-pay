# Hydra-Pay

统一支付网关——为接入方提供统一的支付 API、托管结算页面和多渠道支付能力。

## 概览

Hydra-Pay 旨在以最低的研发成本，帮助产品快速接入支付宝、微信支付、云闪付、数字人名币 等多种支付渠道。项目包含：后端 `pay-service`（Go）、托管结算页面 `pay-frontend`（React）、以及若干客户端 SDK。

主要特性：
- 统一 API：一次接入，多渠道支持
- 智能路由：按规则和熔断策略选择最优渠道
- 渠道插件化：每个渠道由独立 Adapter 实现
- 托管结算页：可托管在 CDN/Vercel，接入方嵌入少量代码即可使用

详情设计请参见：ARCHITECTURE.md

## 目录结构（简要）

主要目录：

- `service/`：支付后端（Go）
- `pay-frontend/`：托管结算页面（React + Vite）
- `sdk/`：客户端 SDK（iOS/Android/Web/Flutter）
- `docs/`：架构与 API 文档

完整架构说明见 `ARCHITECTURE.md`。

## 快速开始

先决条件：

- Go 1.20+（用于后端）
- Node.js 16+（用于前端）
- PostgreSQL
- Redis

1) 后端（本地运行）

```bash
# 进入后端目录
cd hydra-pay/service

# 设置环境变量（示例）
export DATABASE_URL="postgres://user:pass@localhost:5432/pay_db?sslmode=disable"
export REDIS_ADDR="localhost:6379"
export PORT=8080

# 运行服务
go run ./cmd/server
```

后端入口位于 `service/cmd/server`。

2) 前端（开发模式）

```bash
cd hydra-pay/pay-frontend
npm install
npm run dev
```

托管结算页默认通过路由 `/checkout/{payment_id}` 提供。

3) 迁移与初始化

- 数据库迁移文件位于 `service/migrations/`。
- 若使用容器化或本地数据库，请先创建并运行数据库，然后执行迁移脚本。

## 配置（示例环境变量）

- `DATABASE_URL`：Postgres 连接串
- `REDIS_ADDR`：Redis 地址
- `PORT`：服务监听端口
- `JWT_SECRET`：API 鉴权所用密钥（如适用）

具体配置项请参考 `service/config/config.go`。

## API 概览

部分主要接口（详见文档）：

- `POST /v1/payments/create`：创建支付
- `GET /v1/payments/{payment_id}`：查询支付状态
- `POST /v1/payments/{payment_id}/refund`：申请退款

更多 API 细节请参考 `docs/API.md`。

## 开发者说明

- 渠道适配器位于 `service/internal/channel/`，实现 `ChannelAdapter` 接口以添加新渠道。
- 支付路由器与策略配置支持基于地域、设备、失败率等条件做路由决策。
- 回调（Webhook）统一由 `WebhookManager` 处理，并通过事件分发内部消费。

如果要本地调试渠道回调，建议使用 `ngrok` 或类似工具将本地回调地址暴露到公网。

## SDK

SDK 源码在 `sdk/` 目录下，包含 iOS/Android/Web/Flutter 示例和集成指南。可直接使用托管结算页或调用后端 API 集成到产品中。

## 部署建议

- 后端：以容器化部署（Kubernetes / Docker Compose）为主，连接托管的 Postgres 与 Redis 集群。
- 前端：构建后部署到 CDN（Vercel、Netlify 或对象存储 + CDN）。
- 安全：回调签名校验、IP 白名单、速率限制与熔断策略为必要防护。

## 贡献

欢迎贡献：请基于 issue 与 PR 流程提交变更。开发分支、代码风格和测试用例请参考仓库贡献指南。

## 许可证

本项目遵循仓库根目录的 LICENSE（请参阅 LICENSE 文件）。

## 联系

如需帮助或咨询架构设计，请查看 `docs/` 或在仓库中创建 issue。
