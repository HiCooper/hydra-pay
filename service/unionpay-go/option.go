package unionpay

import (
	"crypto/rsa"
	"fmt"
)

// Option 是 Client 的配置选项接口，对标 wechatpay-go 的 core.ClientOption。
type Option interface {
	Apply(c *Client) error
}

// ---- Functional Options ----

type withMchID struct{ mchID string }

func (o withMchID) Apply(c *Client) error {
	if o.mchID == "" {
		return fmt.Errorf("unionpay-go: MchID is required")
	}
	c.mchID = o.mchID
	return nil
}

// WithMchID 设置银联商户号（15位数字）。
func WithMchID(id string) Option { return withMchID{id} }

type withPrivateKey struct{ key *rsa.PrivateKey }

func (o withPrivateKey) Apply(c *Client) error {
	if o.key == nil {
		return fmt.Errorf("unionpay-go: PrivateKey is required")
	}
	c.privateKey = o.key
	return nil
}

// WithPrivateKey 设置商户 RSA 私钥，用于签名请求。
func WithPrivateKey(key *rsa.PrivateKey) Option { return withPrivateKey{key} }

type withPublicKey struct{ key *rsa.PublicKey }

func (o withPublicKey) Apply(c *Client) error {
	c.publicKey = o.key
	return nil
}

// WithPublicKey 设置银联 RSA 公钥，用于验签回调和响应。
// 可选：不设置则无法使用回调验证功能。
func WithPublicKey(key *rsa.PublicKey) Option { return withPublicKey{key} }

type withSandbox struct{ sandbox bool }

func (o withSandbox) Apply(c *Client) error {
	c.isSandbox = o.sandbox
	return nil
}

// WithSandbox 设置是否使用测试网关。
func WithSandbox(v bool) Option { return withSandbox{v} }
