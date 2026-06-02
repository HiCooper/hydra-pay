package ecny

import (
	"encoding/pem"
	"fmt"
	"os"

	gmx509 "github.com/tjfoc/gmsm/x509"

	"github.com/tjfoc/gmsm/sm2"
)

// LoadSM2PrivateKeyPEM 从 PEM 字符串或文件路径加载 SM2 私钥。
// 支持 PKCS8 格式（SM2 OID 1.2.156.10197.1.301）和 gmsm 库的专用格式。
//
// 调用规则：优先使用 pemStr；如果为空且 path 不为空，则从文件读取。
func LoadSM2PrivateKeyPEM(pemStr, path string) (*sm2.PrivateKey, error) {
	if pemStr == "" && path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("ecny-go: failed to read private key file: %w", err)
		}
		pemStr = string(data)
	}
	if pemStr == "" {
		return nil, fmt.Errorf("ecny-go: no private key provided")
	}

	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("ecny-go: failed to decode PEM block containing private key")
	}

	// Use gmsm/x509 which understands SM2 OIDs
	key, err := gmx509.ReadPrivateKeyFromPem([]byte(pemStr), nil)
	if err == nil && key != nil {
		return key, nil
	}

	// Fallback: try parsing unencrypted PKCS8
	key, err = gmx509.ParsePKCS8UnecryptedPrivateKey(block.Bytes)
	if err == nil && key != nil {
		return key, nil
	}

	return nil, fmt.Errorf("ecny-go: failed to parse SM2 private key: %w", err)
}

// LoadSM2PublicKeyPEM 从 PEM 字符串或文件路径加载 SM2 公钥。
// 支持 X.509 SubjectPublicKeyInfo (-----BEGIN PUBLIC KEY-----) 和
// X.509 证书 (-----BEGIN CERTIFICATE-----) 两种格式。
//
// 调用规则：优先使用 pemStr；如果为空且 path 不为空，则从文件读取。
func LoadSM2PublicKeyPEM(pemStr, path string) (*sm2.PublicKey, error) {
	if pemStr == "" && path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("ecny-go: failed to read public key file: %w", err)
		}
		pemStr = string(data)
	}
	if pemStr == "" {
		return nil, fmt.Errorf("ecny-go: no public key provided")
	}

	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("ecny-go: failed to decode PEM block containing public key")
	}

	// Use gmsm/x509 which understands SM2 OIDs
	key, err := gmx509.ReadPublicKeyFromPem([]byte(pemStr))
	if err == nil && key != nil {
		return key, nil
	}

	// Try parsing as DER
	key, err = gmx509.ParseSm2PublicKey(block.Bytes)
	if err == nil && key != nil {
		return key, nil
	}

	return nil, fmt.Errorf("ecny-go: failed to parse SM2 public key: %w", err)
}
