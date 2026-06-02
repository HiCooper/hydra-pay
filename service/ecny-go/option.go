package ecny

import (
	"fmt"

	"github.com/tjfoc/gmsm/sm2"
)

// Option 是 Client 的配置选项接口，对标 unionpay-go 的 Option。
type Option interface {
	Apply(c *Client) error
}

// ---- Functional Options ----

type withMchID struct{ mchID string }

func (o withMchID) Apply(c *Client) error {
	if o.mchID == "" {
		return fmt.Errorf("ecny-go: MchID is required")
	}
	c.mchID = o.mchID
	return nil
}

// WithMchID 设置商户号。
func WithMchID(id string) Option { return withMchID{id} }

type withAppID struct{ appID string }

func (o withAppID) Apply(c *Client) error {
	c.appID = o.appID
	return nil
}

// WithAppID 设置商户应用 ID。
func WithAppID(id string) Option { return withAppID{id} }

type withPrivateKey struct{ key *sm2.PrivateKey }

func (o withPrivateKey) Apply(c *Client) error {
	if o.key == nil {
		return fmt.Errorf("ecny-go: PrivateKey is required")
	}
	c.privateKey = o.key
	return nil
}

// WithPrivateKey 设置商户 SM2 私钥，用于签名请求。
func WithPrivateKey(key *sm2.PrivateKey) Option { return withPrivateKey{key} }

type withPublicKey struct{ key *sm2.PublicKey }

func (o withPublicKey) Apply(c *Client) error {
	c.publicKey = o.key
	return nil
}

// WithPublicKey 设置受理机构 SM2 公钥，用于验签回调和响应。
// 可选：不设置则无法使用回调验证功能。
func WithPublicKey(key *sm2.PublicKey) Option { return withPublicKey{key} }

type withSandbox struct{ sandbox bool }

func (o withSandbox) Apply(c *Client) error {
	c.isSandbox = o.sandbox
	return nil
}

// WithSandbox 设置是否使用测试环境。
func WithSandbox(v bool) Option { return withSandbox{v} }

type withBaseURL struct{ url string }

func (o withBaseURL) Apply(c *Client) error {
	if o.url != "" {
		c.baseURL = o.url
	}
	return nil
}

// WithBaseURL 设置 API 网关基础地址，用于对接非默认受理机构。
func WithBaseURL(url string) Option { return withBaseURL{url} }
