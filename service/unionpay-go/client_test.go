package unionpay

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"net/url"
	"testing"
)

// generateTestKeyPair 生成 RSA 密钥对用于测试。
func generateTestKeyPair(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}
	return priv, &priv.PublicKey
}

func exportPrivateKeyPEM(key *rsa.PrivateKey) string {
	der := x509.MarshalPKCS1PrivateKey(key)
	return string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: der}))
}

func exportPublicKeyPEM(key *rsa.PublicKey) string {
	der, _ := x509.MarshalPKIXPublicKey(key)
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
}

func signTestParams(priv *rsa.PrivateKey, params map[string]string) string {
	str := BuildSigningString(params)
	hashed := sha256.Sum256([]byte(str))
	sig, _ := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, hashed[:])
	return base64.StdEncoding.EncodeToString(sig)
}

// ---- Tests ----

func TestNewClient(t *testing.T) {
	priv, pub := generateTestKeyPair(t)

	client, err := NewClient(context.Background(),
		WithMchID("123456789012345"),
		WithPrivateKey(priv),
		WithPublicKey(pub),
		WithSandbox(true),
	)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	if client.mchID != "123456789012345" {
		t.Errorf("expected mchID, got %q", client.mchID)
	}
	if !client.isSandbox {
		t.Error("expected sandbox=true")
	}
}

func TestNewClientMissingOptions(t *testing.T) {
	// NewClient with no options should succeed (Options pattern,
	// validation is the caller's responsibility, like wechatpay-go)
	client, err := NewClient(context.Background())
	if err != nil {
		t.Fatalf("NewClient with no options should succeed: %v", err)
	}
	if client == nil {
		t.Fatal("client is nil")
	}
	// WithMchID with empty string should error
	_, err = NewClient(context.Background(), WithMchID(""))
	if err == nil {
		t.Error("expected error for empty MchID")
	}
}

func TestSignAndVerify(t *testing.T) {
	priv, pub := generateTestKeyPair(t)

	client, _ := NewClient(context.Background(),
		WithMchID("123456789012345"),
		WithPrivateKey(priv),
		WithPublicKey(pub),
	)

	params := map[string]string{
		"version":    "5.1.0",
		"txnType":    "01",
		"txnSubType": "07",
		"bizType":    "000201",
		"merId":      "123456789012345",
		"orderId":    "TEST001",
		"txnTime":    "20260602150405",
		"txnAmt":     "100",
	}

	sig, err := client.sign(params)
	if err != nil {
		t.Fatalf("sign failed: %v", err)
	}

	params["signature"] = sig
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}

	if err := client.verifySign(values); err != nil {
		t.Errorf("verifySign failed: %v", err)
	}
}

func TestBuildSigningString(t *testing.T) {
	params := map[string]string{
		"bizType": "000201",
		"orderId": "TEST001",
		"txnAmt":  "100",
		"txnType": "01",
	}

	result := BuildSigningString(params)
	expected := "bizType=000201&orderId=TEST001&txnAmt=100&txnType=01"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestBuildSigningStringSkipEmpty(t *testing.T) {
	params := map[string]string{
		"bizType":   "000201",
		"orderId":   "TEST001",
		"signature": "ignored",
		"emptyKey":  "",
	}

	result := BuildSigningString(params)
	if contains(result, "signature") {
		t.Error("signing string should not contain 'signature'")
	}
	if contains(result, "emptyKey") {
		t.Error("signing string should not contain empty values")
	}
}

func TestNotifyHandlerParse_Success(t *testing.T) {
	priv, pub := generateTestKeyPair(t)

	client, _ := NewClient(context.Background(),
		WithMchID("123456789012345"),
		WithPrivateKey(priv),
		WithPublicKey(pub),
	)

	handler := NewNotifyHandler(client)

	params := map[string]string{
		"version":    "5.1.0",
		"encoding":   "UTF-8",
		"signMethod": "01",
		"txnType":   "01",
		"txnSubType": "07",
		"bizType":    "000201",
		"merId":      "123456789012345",
		"orderId":    "TEST_ORDER_001",
		"queryId":    "20260602150405123456",
		"txnTime":    "20260602150405",
		"txnAmt":     "9900",
		"respCode":   "00",
		"respMsg":    "成功",
		"settleAmt":  "9900",
	}

	sig := signTestParams(priv, params)
	params["signature"] = sig

	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	body := values.Encode()

	result, err := handler.Parse([]byte(body))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if result.OrderID != "TEST_ORDER_001" {
		t.Errorf("expected OrderID 'TEST_ORDER_001', got %q", result.OrderID)
	}
	if result.QueryID != "20260602150405123456" {
		t.Errorf("expected QueryID, got %q", result.QueryID)
	}
	if result.TxnAmt != 9900 {
		t.Errorf("expected TxnAmt 9900, got %d", result.TxnAmt)
	}
}

func TestNotifyHandlerParse_MissingSignature(t *testing.T) {
	_, pub := generateTestKeyPair(t)

	client, _ := NewClient(context.Background(),
		WithMchID("123456789012345"),
		WithPrivateKey(&rsa.PrivateKey{}),
		WithPublicKey(pub),
	)

	handler := NewNotifyHandler(client)

	_, err := handler.Parse([]byte("orderId=TEST001&txnAmt=100&respCode=00"))
	if err == nil {
		t.Error("expected error for missing signature")
	}
}

func TestLoadPrivateKeyPEM(t *testing.T) {
	priv, _ := generateTestKeyPair(t)
	pemStr := exportPrivateKeyPEM(priv)

	loaded, err := LoadPrivateKeyPEM(pemStr, "")
	if err != nil {
		t.Fatalf("LoadPrivateKeyPEM failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("loaded key is nil")
	}
}

func TestLoadPublicKeyPEM(t *testing.T) {
	_, pub := generateTestKeyPair(t)
	pemStr := exportPublicKeyPEM(pub)

	loaded, err := LoadPublicKeyPEM(pemStr, "")
	if err != nil {
		t.Fatalf("LoadPublicKeyPEM failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("loaded key is nil")
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
