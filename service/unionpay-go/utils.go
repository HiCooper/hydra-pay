package unionpay

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

// LoadPrivateKeyPEM 从 PEM 字符串或文件路径加载 RSA 私钥。
// 支持 PKCS1 和 PKCS8 两种格式。
// 对标 wechatpay-go utils.LoadPrivateKey()。
//
// 调用规则：优先使用 pemStr；如果为空且 path 不为空，则从文件读取。
func LoadPrivateKeyPEM(pemStr, path string) (*rsa.PrivateKey, error) {
	if pemStr == "" && path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key file: %w", err)
		}
		pemStr = string(data)
	}
	if pemStr == "" {
		return nil, fmt.Errorf("no private key provided")
	}

	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block containing private key")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key (tried PKCS8 and PKCS1): %w", err)
		}
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("key is not an RSA private key, got %T", key)
	}
	return rsaKey, nil
}

// LoadPublicKeyPEM 从 PEM 字符串或文件路径加载 RSA 公钥。
// 支持 X.509 SubjectPublicKeyInfo (-----BEGIN PUBLIC KEY-----) 和
// X.509 证书 (-----BEGIN CERTIFICATE-----) 两种格式。
// 对标 wechatpay-go 的证书加载逻辑。
//
// 调用规则：优先使用 pemStr；如果为空且 path 不为空，则从文件读取。
func LoadPublicKeyPEM(pemStr, path string) (*rsa.PublicKey, error) {
	if pemStr == "" && path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read public key file: %w", err)
		}
		pemStr = string(data)
	}
	if pemStr == "" {
		return nil, fmt.Errorf("no public key provided")
	}

	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block containing public key")
	}

	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		cert, certErr := x509.ParseCertificate(block.Bytes)
		if certErr != nil {
			return nil, fmt.Errorf("failed to parse public key (tried PKIX and X509 cert): %w", err)
		}
		rsaKey, ok := cert.PublicKey.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("certificate public key is not RSA")
		}
		return rsaKey, nil
	}

	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("key is not an RSA public key, got %T", key)
	}
	return rsaKey, nil
}
