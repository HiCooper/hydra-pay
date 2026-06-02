// Package unionpay 是银联全渠道支付系统的 Go SDK。
// 对标 wechatpay-go 的核心架构：Client 负责 HTTP 通信和签名，
// notify.Handler 负责回调验签和解析。
package unionpay

import (
	"context"
	"crypto/rsa"
	"net/http"
	"time"
)

// Client 是银联 API 的 HTTP 客户端，管理网关地址、商户号、RSA 密钥和 HTTP 连接。
// 对标 wechatpay-go 的 core.Client。
type Client struct {
	mchID      string
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	isSandbox  bool
	httpClient *http.Client
}

// Config 包含创建 Client 所需的全部配置。
// 推荐使用 NewClient(ctx, opts...) + Functional Options 来初始化。
type Config struct {
	MchID      string
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
	IsSandbox  bool
}

// NewClient 使用 Functional Options 模式创建 Client。
// 对标 wechatpay-go 的 NewClient(ctx, opts...)。
//
// 示例：
//
//	client, err := unionpay.NewClient(ctx,
//	    unionpay.WithMchID("777290058110048"),
//	    unionpay.WithPrivateKey(privateKey),
//	    unionpay.WithPublicKey(publicKey),
//	    unionpay.WithSandbox(true),
//	)
func NewClient(_ context.Context, opts ...Option) (*Client, error) {
	c := &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	for _, o := range opts {
		if err := o.Apply(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// Service 是银联业务接口的基类，所有子服务（支付、查询、退款）都嵌入此结构。
// 对标 wechatpay-go 的 services.Service。
type Service struct {
	Client *Client
}
