package ecny

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sort"
	"strings"

	"github.com/tjfoc/gmsm/sm3"
)

// sign 使用商户 SM2 私钥对参数进行 SM3-SM2 签名，返回 Base64 编码的签名值。
// SM2 签名流程：签名字符串 → SM3 哈希 → SM2 Sign（通过私钥方法）。
// 对标 unionpay-go sign()，但用 SM2 替代 RSA。
func (c *Client) sign(params map[string]string) (string, error) {
	if c.privateKey == nil {
		return "", fmt.Errorf("ecny-go: private key is nil")
	}
	signingStr := BuildSigningString(params)
	sm3Hash := sm3.Sm3Sum([]byte(signingStr))
	// SM2 Sign — 私钥方法内部调用 sm2.SignWithSM2
	sig, err := c.privateKey.Sign(rand.Reader, sm3Hash, nil)
	if err != nil {
		return "", fmt.Errorf("ecny-go: sm2 sign failed: %w", err)
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

// verifySign 使用受理机构 SM2 公钥验证签名字符串的签名。
// signingStr 是不含签名的排序后参数串，signB64 是 Base64 编码的签名值。
func (c *Client) verifySign(signingStr, signB64 string) error {
	if c.publicKey == nil {
		return fmt.Errorf("ecny-go: public key is nil, cannot verify signature")
	}
	if signB64 == "" {
		return fmt.Errorf("ecny-go: empty signature")
	}
	sigBytes, err := base64.StdEncoding.DecodeString(signB64)
	if err != nil {
		return fmt.Errorf("ecny-go: failed to decode signature: %w", err)
	}
	sm3Hash := sm3.Sm3Sum([]byte(signingStr))
	if !c.publicKey.Verify(sm3Hash, sigBytes) {
		return fmt.Errorf("ecny-go: sm2 signature verification failed")
	}
	return nil
}

// BuildSigningString 按 key 字母序拼接参数为 "key=value&key=value&..." 格式。
// 跳过 signature、signMethod、sign 以及空值参数。
// 对标 unionpay-go BuildSigningString。
func BuildSigningString(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "signature" || k == "signMethod" || k == "sign" || params[k] == "" {
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

// SM3Digest 计算字节数据的 SM3 摘要，返回 32 字节哈希。
func SM3Digest(data []byte) []byte {
	return sm3.Sm3Sum(data)
}

// SM3DigestBase64 计算字节数据的 SM3 摘要并返回 Base64 编码。
func SM3DigestBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(sm3.Sm3Sum(data))
}
