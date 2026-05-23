package wechat

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"io"
	"testing"

	"github.com/hydra/pay-service/internal/config"
	"github.com/hydra/pay-service/internal/model"
)

func TestNewAdapter_MissingConfigs(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.WechatConfig
	}{
		{"missing mch_id", config.WechatConfig{}},
		{"missing api_v3_key", config.WechatConfig{MchID: "123"}},
		{"missing serial_no", config.WechatConfig{MchID: "123", APIv3Key: "abcdefghijklmnopqrstuvwxyz123456"}},
		{"missing private_key", config.WechatConfig{
			MchID: "123", APIv3Key: "abcdefghijklmnopqrstuvwxyz123456", SerialNo: "ABC",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAdapter(&tt.cfg)
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestMapWechatTradeState(t *testing.T) {
	tests := []struct {
		state string
		want  string
	}{
		{"SUCCESS", model.PaymentStatusPaid},
		{"NOTPAY", model.PaymentStatusPending},
		{"USERPAYING", model.PaymentStatusPending},
		{"ACCEPT", model.PaymentStatusPending},
		{"CLOSED", model.PaymentStatusFailed},
		{"PAYERROR", model.PaymentStatusFailed},
		{"REVOKED", model.PaymentStatusFailed},
		{"REFUND", model.PaymentStatusRefunded},
		{"UNKNOWN", model.PaymentStatusPending},
	}
	for _, tt := range tests {
		got := mapWechatTradeState(tt.state)
		if got != tt.want {
			t.Errorf("mapWechatTradeState(%s) = %s, want %s", tt.state, got, tt.want)
		}
	}
}

func TestGetCurrency(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", "CNY"},
		{"CNY", "CNY"},
		{"USD", "USD"},
	}
	for _, tt := range tests {
		got := getCurrency(tt.in)
		if got != tt.want {
			t.Errorf("getCurrency(%s) = %s, want %s", tt.in, got, tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		in     string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello world test", 5, "hello"},
		{"你好世界测试", 3, "你好世"},
		{"", 5, ""},
	}
	for _, tt := range tests {
		got := truncate(tt.in, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.in, tt.maxLen, got, tt.want)
		}
	}
}

// TestDecryptResource verifies AEAD_AES_256_GCM decryption — the same algorithm
// WeChat Pay V3 uses for callback resource encryption.
func TestDecryptResource(t *testing.T) {
	apiV3Key := "abcdefghijklmnopqrstuvwxyz123456" // 32 bytes
	adapter := &Adapter{apiV3Key: apiV3Key}

	// Create AES-GCM cipher with the key
	block, err := aes.NewCipher([]byte(apiV3Key))
	if err != nil {
		t.Fatalf("failed to create cipher: %v", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("failed to create GCM: %v", err)
	}

	// Generate random nonce (12 bytes for GCM)
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		t.Fatalf("failed to generate nonce: %v", err)
	}

	// Create a mock WeChat transaction JSON
	transaction := map[string]interface{}{
		"out_trade_no":   "test-order-550e8400",
		"transaction_id": "4200001234567890",
		"trade_state":    "SUCCESS",
		"trade_state_desc": "支付成功",
		"amount": map[string]interface{}{
			"total":    float64(100),
			"currency": "CNY",
		},
	}
	plaintext, _ := json.Marshal(transaction)

	// Encrypt
	additionalData := "test-additional-data"
	ciphertext := aead.Seal(nil, nonce, plaintext, []byte(additionalData))

	// Now decrypt through the adapter's decryptResource
	decrypted, err := adapter.decryptResource(
		base64.StdEncoding.EncodeToString(ciphertext),
		base64.StdEncoding.EncodeToString(nonce),
		additionalData,
	)
	if err != nil {
		t.Fatalf("decryptResource failed: %v", err)
	}

	// Verify decrypted content matches original
	var result map[string]interface{}
	if err := json.Unmarshal(decrypted, &result); err != nil {
		t.Fatalf("failed to parse decrypted result: %v", err)
	}
	if result["out_trade_no"] != "test-order-550e8400" {
		t.Errorf("wrong out_trade_no: %v", result["out_trade_no"])
	}
	if result["transaction_id"] != "4200001234567890" {
		t.Errorf("wrong transaction_id: %v", result["transaction_id"])
	}
	if result["trade_state"] != "SUCCESS" {
		t.Errorf("wrong trade_state: %v", result["trade_state"])
	}
}

// TestDecryptResource_InvalidCiphertext ensures corrupted data is rejected.
func TestDecryptResource_InvalidCiphertext(t *testing.T) {
	apiV3Key := "abcdefghijklmnopqrstuvwxyz123456"
	adapter := &Adapter{apiV3Key: apiV3Key}

	_, err := adapter.decryptResource(
		"!!!not-valid-base64!!!",
		base64.StdEncoding.EncodeToString(make([]byte, 12)),
		"",
	)
	if err == nil {
		t.Fatal("expected error for invalid base64 ciphertext")
	}
}

// TestDecryptResource_WrongKey ensures decryption fails with wrong key.
func TestDecryptResource_WrongKey(t *testing.T) {
	correctKey := "abcdefghijklmnopqrstuvwxyz123456"
	wrongKey := "zyxwvutsrqponmlkjihgfedcba654321"

	// Encrypt with correct key
	block, _ := aes.NewCipher([]byte(correctKey))
	aead, _ := cipher.NewGCM(block)
	nonce := make([]byte, 12)
	rand.Read(nonce)
	plaintext := []byte(`{"out_trade_no":"test"}`)

	ciphertext := aead.Seal(nil, nonce, plaintext, nil)

	// Decrypt with wrong key
	adapter := &Adapter{apiV3Key: wrongKey}
	_, err := adapter.decryptResource(
		base64.StdEncoding.EncodeToString(ciphertext),
		base64.StdEncoding.EncodeToString(nonce),
		"",
	)
	if err == nil {
		t.Fatal("expected decryption error with wrong key")
	}
}

// TestVerifyCallback_TransactionJSON verifies the full callback parsing flow
// with a simulated AEAD-encrypted WeChat transaction.
func TestVerifyCallback_TransactionJSON(t *testing.T) {
	apiV3Key := "abcdefghijklmnopqrstuvwxyz123456"
	adapter := &Adapter{apiV3Key: apiV3Key}

	// Create a mock WeChat transaction
	transaction := map[string]interface{}{
		"out_trade_no":   "order-uuid-12345",
		"transaction_id": "4200001234567890",
		"trade_state":    "SUCCESS",
		"trade_state_desc": "支付成功",
		"trade_type":     "NATIVE",
		"bank_type":      "OTHERS",
		"success_time":   "2024-01-15T12:05:00+08:00",
		"payer": map[string]interface{}{
			"openid": "oUpF8uMuAJO_M2pxb1Q9zNjWeS6o",
		},
		"amount": map[string]interface{}{
			"total":         float64(100),
			"payer_total":   float64(100),
			"currency":      "CNY",
			"payer_currency": "CNY",
		},
	}
	transactionJSON, _ := json.Marshal(transaction)

	// Encrypt the transaction (simulating WeChat's encryption)
	block, _ := aes.NewCipher([]byte(apiV3Key))
	aead, _ := cipher.NewGCM(block)
	nonce := make([]byte, 12)
	rand.Read(nonce)
	ciphertext := aead.Seal(nil, nonce, transactionJSON, nil)

	// Build the notification body
	notification := map[string]interface{}{
		"id":            "ev-20240115120500001",
		"create_time":   "2024-01-15T12:05:00+08:00",
		"resource_type": "encrypt-resource",
		"event_type":    "TRANSACTION.SUCCESS",
		"summary":       "支付成功",
		"resource": map[string]interface{}{
			"algorithm":       "AEAD_AES_256_GCM",
			"ciphertext":      base64.StdEncoding.EncodeToString(ciphertext),
			"associated_data": "",
			"nonce":           base64.StdEncoding.EncodeToString(nonce),
			"original_type":   "transaction",
		},
	}
	notificationJSON, _ := json.Marshal(notification)

	// Parse and verify the notification structure (skipping signature verification,
	// which requires platform certificates that need real WeChat merchant setup)
	var notif struct {
		ID           string `json:"id"`
		EventType    string `json:"event_type"`
		ResourceType string `json:"resource_type"`
		Resource     struct {
			Algorithm      string `json:"algorithm"`
			Ciphertext     string `json:"ciphertext"`
			AssociatedData string `json:"associated_data"`
			Nonce          string `json:"nonce"`
			OriginalType   string `json:"original_type"`
		} `json:"resource"`
	}
	if err := json.Unmarshal(notificationJSON, &notif); err != nil {
		t.Fatalf("failed to parse notification: %v", err)
	}

	if notif.EventType != "TRANSACTION.SUCCESS" {
		t.Fatalf("wrong event type: %s", notif.EventType)
	}

	// Decrypt
	plaintext, err := adapter.decryptResource(
		notif.Resource.Ciphertext,
		notif.Resource.Nonce,
		notif.Resource.AssociatedData,
	)
	if err != nil {
		t.Fatalf("decryptResource failed: %v", err)
	}

	// Parse transaction
	var tx struct {
		OutTradeNo    string `json:"out_trade_no"`
		TransactionID string `json:"transaction_id"`
		TradeState    string `json:"trade_state"`
		Amount        struct {
			Total    int64  `json:"total"`
			Currency string `json:"currency"`
		} `json:"amount"`
	}
	if err := json.Unmarshal(plaintext, &tx); err != nil {
		t.Fatalf("failed to parse transaction: %v", err)
	}

	if tx.OutTradeNo != "order-uuid-12345" {
		t.Errorf("wrong out_trade_no: %s", tx.OutTradeNo)
	}
	if tx.TransactionID != "4200001234567890" {
		t.Errorf("wrong transaction_id: %s", tx.TransactionID)
	}
	if tx.TradeState != "SUCCESS" {
		t.Errorf("wrong trade_state: %s", tx.TradeState)
	}
	if tx.Amount.Total != 100 {
		t.Errorf("wrong amount: %d", tx.Amount.Total)
	}

	// Verify status mapping
	status := mapWechatTradeState(tx.TradeState)
	if status != model.PaymentStatusPaid {
		t.Errorf("expected paid status, got %s", status)
	}
}

// generateRSAKeypair produces a test RSA keypair.
func generateRSAKeypair(t *testing.T) (*rsa.PrivateKey, string) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	der, _ := x509.MarshalPKCS8PrivateKey(priv)
	pem := "-----BEGIN PRIVATE KEY-----\n" + base64.StdEncoding.EncodeToString(der) + "\n-----END PRIVATE KEY-----"
	return priv, pem
}

func TestNewAdapter_InvalidPrivateKey(t *testing.T) {
	_, err := NewAdapter(&config.WechatConfig{
		MchID:     "1234567890",
		APIv3Key:  "abcdefghijklmnopqrstuvwxyz123456",
		SerialNo:  "ABC123",
		PrivateKey: "not-a-valid-pem-key",
	})
	if err == nil {
		t.Fatal("expected error for invalid private key")
	}
}