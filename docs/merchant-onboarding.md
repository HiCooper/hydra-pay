# 商户自助进件（Merchant Self-Service Onboarding）

## 背景

当前 hydra-pay 支持服务商模式的**支付**（通过 `SubMerchantID` 字段），但缺少商户进件流程。管理员需要手动从支付宝/微信服务商后台获取商户的 PID 和 sub_mchid，再填入 hydra-pay。

本方案实现"路径二"——平台生成进件链接，商户自行完成认证，审核结果异步回调自动更新 App 配置。

## 架构概览

```
Admin 发起进件 → 调用渠道进件 API → 返回签约链接/二维码
                                            ↓
                                   商户扫码自助填写资料 + 人脸认证
                                            ↓
                      渠道异步回调 → 自动更新 App.alipay_pid / wechat_sub_mchid
```

## 新增文件

| 文件 | 说明 |
|------|------|
| `service/internal/model/merchant_onboarding.go` | 进件记录数据模型 |
| `service/internal/repository/onboarding_repo.go` | 进件记录仓库 |
| `service/internal/channel/alipay/onboarding.go` | 支付宝进件实现 |
| `service/internal/channel/wechat/onboarding.go` | 微信支付进件实现 |
| `service/internal/handler/onboarding_callback_handler.go` | 进件回调处理器 |

## 修改文件

| 文件 | 变更 |
|------|------|
| `service/internal/channel/adapter.go` | 新增进件接口类型 + OnboardingProvider 接口 |
| `service/internal/config/config.go` | 新增 OnboardingNotifyURL 配置字段 |
| `service/internal/admin/handler.go` | 新增进件管理 API（发起进件、查询状态、列表） |
| `service/internal/router/router.go` | 注册进件回调路由 + 管理后台进件路由 |
| `service/internal/database/database.go` | 添加 MerchantOnboarding 表迁移 |
| `service/internal/service/payment_service.go` | 导出 getAdapter → GetAdapter |

## 数据模型

### MerchantOnboarding

| 字段 | 类型 | 说明 |
|------|------|------|
| ID | uuid PK | |
| AppID | uuid FK | 关联的应用 |
| Channel | string | alipay / wechat |
| OutRequestNo | string (唯一索引) | 幂等键，防重复提交 |
| ApplymentID | string | 渠道返回的申请单号 |
| Status | string | pending → submitted → auditing → approved / rejected |
| SubMerchantID | string | 审批通过后的 PID / sub_mchid |
| SignURL | string | 商户签约链接 |
| QrCodeURL | string | 签约二维码 |
| RequestData | jsonb | 提交的申请数据 |
| ResponseData | jsonb | 渠道返回的原始数据 |
| CallbackData | jsonb | 回调原始数据 |
| ErrorMessage | string | 拒绝原因 |

## API 端点

### 管理后台（Admin Auth）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/admin/apps/:id/onboard` | 为 App 发起进件 |
| GET | `/api/admin/apps/:id/onboarding` | 查询进件状态（含自动轮询） |
| GET | `/api/admin/onboarding` | 进件列表（支持过滤和分页） |

### 渠道回调（公开）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/v1/onboarding/callback/:channel` | 渠道进件结果异步通知 |

### 进件请求参数

```json
{
  "channel": "wechat",
  "merchant_name": "某某科技有限公司",
  "contact_name": "张三",
  "contact_phone": "13800138000",
  "contact_email": "zhangsan@example.com"
}
```

## 进件流程

### 支付宝

1. Admin 发起进件 → 调用 `ant.merchant.expand.indirect.create`
2. 支付宝返回申请单号 + 签约链接
3. 商户访问链接 → 填写资料（营业执照、法人信息、银行账户）
4. 支付宝审核通过 → 异步通知 `/v1/onboarding/callback/alipay`
5. 系统验签 → 自动更新 `App.alipay_pid`

### 微信支付

1. Admin 发起进件 → 调用 `POST /v3/applyment4sub/applyment/`
2. 微信返回 `applyment_id`（此时尚无签约链接）
3. 轮询查询状态 → 审核通过后获取签约链接
4. 商户法人扫码 → 完成人脸识别
5. 微信异步通知 → `/v1/onboarding/callback/wechat`
6. 系统验证 V3 签名 → 自动更新 `App.wechat_sub_mchid`

## 配置（环境变量）

| 变量 | 说明 |
|------|------|
| `ALIPAY_ONBOARDING_NOTIFY_URL` | 支付宝进件结果回调地址 |
| `WECHAT_ONBOARDING_NOTIFY_URL` | 微信支付进件结果回调地址 |

## 实施顺序

1. 数据模型 + 数据库迁移
2. 接口类型 + 配置
3. 数据访问层（Repository）
4. 渠道实现（Alipay + WeChat）
5. 回调处理器
6. 管理后台 API + 路由
