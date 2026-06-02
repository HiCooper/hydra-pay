// Package ecny 是数字人民币（e-CNY）受理服务机构 API 的 Go SDK。
// 对标 unionpay-go 的核心架构：Client 负责 HTTP 通信和 SM2 签名，
// NotifyHandler 负责回调验签和解析。
//
// 数字人民币没有公开的标准 API，商户需通过运营机构（银行）或
// 受理服务机构（2.5 层，如银盛支付、易宝支付、中移电商等）接入。
// 本 SDK 默认对接受理服务机构的 JSON API 模式，可通过 WithBaseURL
// 或 WithAgency 配置具体机构。
package ecny

import (
	"context"
	"net/http"
	"time"

	"github.com/tjfoc/gmsm/sm2"
)

// Client 是数字人民币受理服务机构 API 的 HTTP 客户端。
// 对标 unionpay-go 的 Client，但使用 SM2 密钥代替 RSA。
type Client struct {
	mchID       string
	appID       string
	privateKey  *sm2.PrivateKey
	publicKey   *sm2.PublicKey
	isSandbox   bool
	baseURL     string
	httpClient  *http.Client
}

// Service 是数字人民币业务接口的基类，所有子服务都嵌入此结构。
// 对标 unionpay-go 的 Service。
type Service struct {
	Client *Client
}

// NewClient 使用 Functional Options 模式创建 Client。
//
// 示例：
//
//	client, err := ecny.NewClient(ctx,
//	    ecny.WithMchID("商户号"),
//	    ecny.WithAppID("应用ID"),
//	    ecny.WithPrivateKey(privateKey),
//	    ecny.WithPublicKey(publicKey),
//	    ecny.WithSandbox(true),
//	)
func NewClient(_ context.Context, opts ...Option) (*Client, error) {
	c := &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: DefaultProductionBaseURL,
	}
	for _, o := range opts {
		if err := o.Apply(c); err != nil {
			return nil, err
		}
	}
	if c.isSandbox && c.baseURL == DefaultProductionBaseURL {
		c.baseURL = DefaultSandboxBaseURL
	}
	return c, nil
}

// POST /v1/ecny/order/create
func (c *Client) orderCreateURL() string { return c.baseURL + "/v1/ecny/order/create" }

// POST /v1/ecny/order/query
func (c *Client) orderQueryURL() string { return c.baseURL + "/v1/ecny/order/query" }

// POST /v1/ecny/order/refund
func (c *Client) orderRefundURL() string { return c.baseURL + "/v1/ecny/order/refund" }
