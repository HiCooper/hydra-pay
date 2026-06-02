package unionpay

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// sign 使用商户 RSA 私钥对参数进行 SHA256-RSA 签名，返回 Base64 编码的签名值。
// 对标 wechatpay-go signer.Sign()。
func (c *Client) sign(params map[string]string) (string, error) {
	if c.privateKey == nil {
		return "", fmt.Errorf("private key is nil")
	}
	signingStr := BuildSigningString(params)
	hashed := sha256.Sum256([]byte(signingStr))
	sigBytes, err := rsa.SignPKCS1v15(rand.Reader, c.privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return "", fmt.Errorf("rsa sign failed: %w", err)
	}
	return base64.StdEncoding.EncodeToString(sigBytes), nil
}

// verifySign 使用银联公钥验证回调参数中的签名。
// params 应包含 signature 字段，signature 不会被纳入签名字符串。
// 对标 wechatpay-go verifier.Verify()。
func (c *Client) verifySign(params url.Values) error {
	if c.publicKey == nil {
		return fmt.Errorf("public key is nil, cannot verify signature")
	}

	sigB64 := params.Get("signature")
	if sigB64 == "" {
		return fmt.Errorf("no signature in params")
	}

	sigBytes, err := base64.StdEncoding.DecodeString(sigB64)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	sigParams := make(map[string]string)
	for key, vals := range params {
		if key == "signature" || key == "signMethod" {
			continue
		}
		if len(vals) > 0 {
			sigParams[key] = vals[0]
		}
	}

	signingStr := BuildSigningString(sigParams)
	hashed := sha256.Sum256([]byte(signingStr))

	if err := rsa.VerifyPKCS1v15(c.publicKey, crypto.SHA256, hashed[:], sigBytes); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}
	return nil
}

// BuildSigningString 按 key 字母序拼接参数为 "key=value&key=value&..." 格式。
// 跳过 signature、signMethod 以及空值参数。
func BuildSigningString(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "signature" || k == "signMethod" || params[k] == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(params[k])
	}
	return b.String()
}

// gatewayURL 返回银联网关基础地址。
func (c *Client) gatewayURL() string {
	if c.isSandbox {
		return "https://gateway.test.95516.com/gateway/api"
	}
	return "https://gateway.95516.com/gateway/api"
}

func (c *Client) backTransURL() string  { return c.gatewayURL() + "/backTransReq.do" }
func (c *Client) appTransURL() string   { return c.gatewayURL() + "/appTransReq.do" }
func (c *Client) frontTransURL() string { return c.gatewayURL() + "/frontTransReq.do" }
func (c *Client) queryTransURL() string { return c.gatewayURL() + "/queryTrans.do" }
