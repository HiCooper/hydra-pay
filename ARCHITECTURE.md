# Hydra-Pay 架构设计文档

> 统一支付网关 - 自建 Stripe

## 一、产品定位

Hydra-Pay 是一套自建的统一支付网关（类比 Stripe），旨在帮助接入方以极低研发成本快速接入多种支付渠道。

### 核心价值

| 维度 | 描述 |
|------|------|
| **统一支付入口** | 一次接入，连接支付宝、微信支付、Stripe、Apple IAP 等多种渠道 |
| **降低对接复杂度** | 接入方无需了解各渠道对接细节，通过统一 API 即可完成支付 |
| **支付成功率优化** | 内置智能路由、失败重试、熔断降级等策略 |
| **托管结算页面** | 提供托管结算页面，接入方一行代码即可完成支付功能 |

### 目标用户

- **移动应用开发者**：需要应用内支付的 iOS/Android/Flutter 应用
- **Web 产品**：需要网页端支付的 SaaS 产品
- **游戏开发者**：需要多渠道支付的休闲游戏
- **出海产品**：需要 Stripe 等国际支付渠道

---

## 二、系统架构

### 2.1 高层架构图

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          客户端 SDK                                     │
│   iOS SDK  ·  Android SDK  ·  Web SDK  ·  Flutter SDK  ·  Unity SDK    │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                    ┌───────────────┴───────────────┐
                    ▼                               ▼
        ┌───────────────────┐           ┌───────────────────┐
        │   Full Hosted      │           │   Embedded SDK    │
        │   (托管模式)        │           │   (嵌入模式)       │
        └───────────────────┘           └───────────────────┘
                    │                               │
                    ▼                               ▼
        ┌─────────────────────────────────────────────────────────────┐
        │                    Hydra API Gateway                          │
        │              (统一入口 / 路由 / 鉴权 / 限流)                   │
        └─────────────────────────────────────────────────────────────┘
                                    │
                    ┌───────────────┴───────────────┐
                    ▼                               ▼
        ┌───────────────────┐           ┌───────────────────┐
        │   pay-frontend    │           │   pay-service     │
        │   (托管前端)       │           │   (后端服务)       │
        └───────────────────┘           └───────────────────┘
                    │                               │
                    ▼                               ▼
        ┌───────────────────┐           ┌───────────────────┐
        │   Redis           │           │   pay-db         │
        │   (会话缓存)       │           │   PostgreSQL      │
        └───────────────────┘           └───────────────────┘
                                    │
                                    ▼
                        ┌───────────────────┐
                        │  Payment Router   │
                        │  (支付路由)        │
                        └───────────────────┘
                                    │
                    ┌───────────────┴───────────────┐
                    ▼                               ▼
        ┌───────────────────┐           ┌───────────────────┐
        │ Channel Adapter   │           │ Channel Adapter   │
        │ (支付宝)          │           │ (微信支付)         │
        └───────────────────┘           └───────────────────┘
                    │                               │
                    ▼                               ▼
        ┌───────────────────┐           ┌───────────────────┐
        │ Channel Adapter   │           │ Channel Adapter   │
        │ (Stripe)         │           │ (Apple IAP)       │
        └───────────────────┘           └───────────────────┘
                                    │
                                    ▼
                        ┌───────────────────┐
                        │ Payment Channels  │
                        │ 支付宝 / 微信 / Stripe / Apple IAP / Google Play │
                        └───────────────────┘
```

### 2.2 组件说明

| 组件 | 职责 | 技术选型 |
|------|------|----------|
| **pay-frontend** | 托管结算页面渲染 | React + Vite, 部署在 Vercel/CDN |
| **pay-service** | 支付业务逻辑 | Go (推荐) |
| **payment-router** | 智能路由选择 | 内置策略引擎 |
| **channel-adapters** | 渠道适配器（插件化） | 每个渠道独立实现 |
| **pay-db** | 持久化存储 | PostgreSQL |
| **Redis** | 队列/缓存 | Redis Cluster |

---

## 三、工程结构

```
hydra-pay/
├── pay-service/                  # 后端服务 (Go)
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── api/                  # HTTP API 层
│   │   │   ├── handler/
│   │   │   ├── middleware/
│   │   │   └── router.go
│   │   ├── service/
│   │   │   ├── payment/
│   │   │   ├── router/
│   │   │   ├── ledger/
│   │   │   ├── risk/
│   │   │   └── webhook/
│   │   ├── domain/
│   │   │   ├── order.go
│   │   │   ├── transaction.go
│   │   │   └── channel.go
│   │   ├── channel/
│   │   │   ├── adapter.go
│   │   │   ├── alipay/
│   │   │   ├── wechat/
│   │   │   ├── stripe/
│   │   │   ├── apple_iap/
│   │   │   └── google_billing/
│   │   └── repository/
│   ├── pkg/errors/
│   ├── migrations/
│   ├── config/config.go
│   └── go.mod
│
├── pay-frontend/
│   └── src/
│       ├── pages/CheckoutPage.tsx
│       ├── components/
│       └── sdk/
│
├── pay-sdk/
│   ├── ios/
│   ├── android/
│   ├── web/
│   └── flutter/
│
├── docs/
│   ├── ARCHITECTURE.md
│   ├── API.md
│   └── CHANNEL_INTEGRATION.md
│
└── docker/
```

---

## 四、核心功能模块

### 4.1 Payment Router（支付路由）

智能选择最优支付渠道。

```go
type PaymentRouter interface {
    Route(ctx context.Context, req *PaymentRequest) (*RouteResult, error)
    Fallback(ctx context.Context, originalChannel string, reason string) (*RouteResult, error)
}

type RoutingRule struct {
    Condition RoutingCondition `yaml:"condition"`
    Channel   string           `yaml:"channel"`
    Priority  int              `yaml:"priority"`
}
```

#### 路由策略示例

```yaml
routing:
  default_channel: alipay
  rules:
    - condition:
        region: ["CN"]
      channel: alipay
      priority: 100
    - condition:
        region: ["US", "EU", "GB"]
      channel: stripe
      priority: 100
    - condition:
        device: "apple"
        app_type: "subscription"
      channel: apple_iap
      priority: 100
  circuit_breaker:
    alipay:
      error_threshold: 5
      timeout: 30s
```

### 4.2 Channel Adapters（渠道适配器）

每个支付渠道对应一个 Adapter，插件化设计。

```go
type ChannelAdapter interface {
    CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*CreatePaymentResponse, error)
    GetPaymentStatus(ctx context.Context, channelTxID string) (*PaymentStatus, error)
    ConfirmPayment(ctx context.Context, channelTxID string) (*ConfirmResponse, error)
    Refund(ctx context.Context, req *RefundRequest) (*RefundResponse, error)
    VerifySignature(ctx context.Context, notification []byte, signature string) error
    GetChannelName() string
}
```

#### 渠道适配器实现列表

| 渠道 | 说明 |
|------|------|
| **Alipay** | 扫码支付、App 支付、JSAPI 支付 |
| **WeChat Pay** | 扫码支付、JSAPI 支付、小程序支付 |
| **Stripe** | Checkout、Payment Intent、Subscription |
| **Apple IAP** | 应用内购买、订阅验证 |
| **Google Billing** | 应用内购买、订阅验证 |

### 4.3 Ledger Service（账务系统）

管理订单、交易流水、对账清算。

```go
type LedgerService interface {
    CreateOrder(ctx context.Context, req *CreateOrderRequest) (*Order, error)
    UpdateOrderStatus(ctx context.Context, orderID string, status OrderStatus) error
    RecordTransaction(ctx context.Context, tx *Transaction) error
    Reconcile(ctx context.Context, date time.Time) (*ReconcileResult, error)
    Refund(ctx context.Context, req *RefundRequest) (*Refund, error)
}

type Order struct {
    ID                 string    `json:"id"`
    AppID              string    `json:"app_id"`
    UserID             string    `json:"user_id"`
    Amount             int64     `json:"amount"`
    Currency           string    `json:"currency"`
    Channel            string    `json:"channel"`
    Status             string    `json:"status"`
    ProductID          string    `json:"product_id"`
    PaywallSessionID   string    `json:"paywall_session_id"`
    CreatedAt          time.Time `json:"created_at"`
    PaidAt             *time.Time `json:"paid_at"`
}
```

### 4.4 Webhook Manager（回调处理）

统一处理各渠道的异步回调。

```go
type WebhookManager interface {
    RegisterCallback(channel string, callbackURL string) error
    HandleNotification(ctx context.Context, channel string, notification []byte, signature string) (*WebhookEvent, error)
    DispatchEvent(ctx context.Context, event *WebhookEvent) error
    RetryFailedEvents(ctx context.Context) error
}
```

---

## 五、API 设计

### 5.1 创建支付

```
POST /v1/payments/create
```

Request:
```json
{
  "app_id": "app_xxxx",
  "user_id": "user_xxxx",
  "amount": 9900,
  "currency": "CNY",
  "channel": "alipay",
  "product_id": "plan_monthly",
  "product_name": "Premium Monthly",
  "success_url": "https://yourapp.com/payment/success",
  "cancel_url": "https://yourapp.com/payment/cancel",
  "metadata": {}
}
```

Response:
```json
{
  "payment_id": "pay_xxxx",
  "channel": "alipay",
  "amount": 9900,
  "currency": "CNY",
  "status": "pending",
  "payment_url": "https://pay.hydra.io/checkout/pay_xxxx",
  "qr_code": "https://qr.alipay.com/xxxxx",
  "expires_at": "2026-05-18T15:30:00Z"
}
```

### 5.2 查询支付状态

```
GET /v1/payments/{payment_id}
```

### 5.3 申请退款

```
POST /v1/payments/{payment_id}/refund
```

---

## 六、托管结算页面

### 页面路由

| 路由 | 说明 |
|------|------|
| `/checkout/{payment_id}` | 全托管结算页面 |
| `/checkout/{payment_id}/success` | 支付成功页 |

### 页面流程

```
Hydra-Wall 重定向用户到 Hydra-Pay
    │
    ▼
用户选择支付渠道
    │
    ├──────────────────────────────────────┐
    ▼                                      ▼
选择支付宝                              选择微信支付
    │                                      │
    ▼                                      ▼
扫码支付                                扫码支付
    │                                      │
    ▼                                      ▼
渠道回调通知                            更新订单状态
    │                                      │
    ▼                                      ▼
Hydra-Pay 通知 Hydra-Wall               用户被重定向到 success_url
```

---

## 七、数据模型

### 数据库 Schema

```sql
CREATE TABLE orders (
    id VARCHAR(32) PRIMARY KEY,
    app_id VARCHAR(32) NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    amount INT NOT NULL,
    currency VARCHAR(3) DEFAULT 'CNY',
    channel VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    product_id VARCHAR(64),
    paywall_session_id VARCHAR(64),
    metadata JSONB DEFAULT '{}',
    expires_at TIMESTAMP,
    paid_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE transactions (
    id VARCHAR(32) PRIMARY KEY,
    order_id VARCHAR(32) NOT NULL,
    channel VARCHAR(32) NOT NULL,
    channel_transaction_id VARCHAR(128),
    amount INT NOT NULL,
    fee INT DEFAULT 0,
    net_amount INT,
    type VARCHAR(32) NOT NULL DEFAULT 'payment',
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE refunds (
    id VARCHAR(32) PRIMARY KEY,
    order_id VARCHAR(32) NOT NULL,
    transaction_id VARCHAR(32) NOT NULL,
    amount INT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    channel_refund_id VARCHAR(128),
    processed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

---

## 八、客户端 SDK

### 全托管模式

```typescript
import { HydraPay } from '@hydra/pay-sdk';
const pay = new HydraPay({ appId: 'your_app_id' });

await pay.redirectToCheckout({
  paymentId: 'pay_xxxx',
  successUrl: 'https://yourapp.com/payment/success'
});
```

### 嵌入式 SDK

```typescript
await pay.mountPaymentForm({
  container: '#payment-container',
  paymentId: 'pay_xxxx',
  onPaymentComplete: (result) => console.log('Done:', result)
});
```

### API 模式

```typescript
const payment = await pay.api.createPayment({
  amount: 9900,
  channel: 'alipay',
  productId: 'premium_monthly'
});
// 接入方自行调起支付
```

---

## 九、安全设计

### 签名验证

每个渠道的回调都需要验证签名：

- **支付宝**: RSA2 签名验证
- **微信支付**: HMAC-SHA256 签名验证
- **Stripe**: Webhook 签名验证 (v1)
- **Apple/Google**: Receipt Validation

### 敏感数据加密

使用 AES-256-GCM 加密敏感字段，密钥从 KMS 获取。

### 熔断机制

```go
type CircuitBreaker struct {
    state        CircuitState
    failures     int
    threshold    int
    timeout      time.Duration
}

const (
    StateClosed CircuitState = iota
    StateOpen
    StateHalfOpen
)
```

---

## 十、部署架构

### 生产环境架构

```
                          ┌─────────────────┐
                          │   Cloudflare   │
                          └────────┬────────┘
                                   │
                    ┌──────────────┴──────────────┐
                    ▼                             ▼
          ┌─────────────────┐          ┌─────────────────┐
          │  pay-frontend   │          │   API Gateway   │
          │  (Vercel/CDN)    │          │   (Kong/Envoy) │
          └─────────────────┘          └────────┬────────┘
                                               │
                    ┌──────────────────────────┴─────────────────┐
                    ▼                                              ▼
          ┌─────────────────┐                            ┌─────────────────┐
          │  pay-service    │◄───────────────────────────►│  pay-service    │
          └────────┬────────┘                            └────────┬────────┘
                   │                                              │
         ┌─────────┴─────────┐                        ┌──────────┴──────────┐
         ▼                   ▼                        ▼                     ▼
┌───────────────┐   ┌───────────────┐        ┌───────────────┐    ┌───────────────┐
│  PostgreSQL   │   │     Redis     │        │  PostgreSQL   │    │     Redis     │
└───────────────┘   └───────────────┘        └───────────────┘    └───────────────┘
                           │
                           ▼
                  ┌─────────────────┐
                  │  Channel APIs  │
                  │  支付宝/微信等   │
                  └─────────────────┘
```

---

## 十一、监控指标

| 指标 | 描述 | 告警阈值 |
|------|------|----------|
| payment_success_rate | 支付成功率 | < 95% |
| refund_rate | 退款率 | > 5% |
| channel_availability | 渠道可用性 | < 99% |
| webhook_delivery_rate | Webhook 投递率 | < 99.5% |
| api_latency_p99 | API P99 延迟 | > 300ms |

---

## 十二、路线图

### Phase 1: MVP (4-6 周)

- [ ] 基础 pay-service
- [ ] Web SDK
- [ ] 托管结算页面
- [ ] 支付宝 + 微信支付渠道

### Phase 2: 多渠道 (4 周)

- [ ] Stripe 渠道
- [ ] Apple IAP + Google Billing
- [ ] 支付路由 + 熔断

### Phase 3: 高级功能 (4 周)

- [ ] 风控系统
- [ ] 账务对账
- [ ] 管理后台
- [ ] 多平台 SDK

### Phase 4: Enterprise (4 周)

- [ ] 自定义域名
- [ ] 白标功能
- [ ] 私有化部署
