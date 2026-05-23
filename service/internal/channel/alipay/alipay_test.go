package alipay

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"net/url"
	"testing"

	"github.com/smartwalle/nsign"

	"github.com/hydra/pay-service/internal/channel"
	"github.com/hydra/pay-service/internal/config"
	"github.com/hydra/pay-service/internal/model"
)

func generateRSAKeyPair(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	return priv, &priv.PublicKey
}

func pubKeyToPEM(t *testing.T, pub *rsa.PublicKey) string {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		t.Fatalf("failed to marshal public key: %v", err)
	}
	return "-----BEGIN PUBLIC KEY-----\n" + base64.StdEncoding.EncodeToString(der) + "\n-----END PUBLIC KEY-----"
}

func privKeyToPEMString(t *testing.T, priv *rsa.PrivateKey) string {
	t.Helper()
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("failed to marshal private key: %v", err)
	}
	return "-----BEGIN PRIVATE KEY-----\n" + base64.StdEncoding.EncodeToString(der) + "\n-----END PRIVATE KEY-----"
}

// alipaySign creates a valid Alipay-style RSA2 signature for url.Values.
func alipaySign(t *testing.T, values url.Values, priv *rsa.PrivateKey) string {
	t.Helper()
	rsaMethod := nsign.NewRSAMethod(crypto.SHA256, priv, nil)
	signer := nsign.New(nsign.WithMethod(rsaMethod))
	sigBytes, err := signer.SignValues(values,
		nsign.WithIgnore("sign", "sign_type", "alipay_cert_sn"),
	)
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}
	return base64.StdEncoding.EncodeToString(sigBytes)
}

func TestNewAdapter_MissingAppID(t *testing.T) {
	_, err := NewAdapter(&config.AlipayConfig{})
	if err == nil {
		t.Fatal("expected error for missing AppID")
	}
}

func TestNewAdapter_MissingPrivateKey(t *testing.T) {
	_, err := NewAdapter(&config.AlipayConfig{AppID: "test_app"})
	if err == nil {
		t.Fatal("expected error for missing private key")
	}
}

func TestNewAdapter_Success(t *testing.T) {
	priv, _ := generateRSAKeyPair(t)
	pkPEM := privKeyToPEMString(t, priv)

	adapter, err := NewAdapter(&config.AlipayConfig{
		AppID:      "2021000000000001",
		PrivateKey: pkPEM,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if adapter.Name() != model.ChannelAlipay {
		t.Fatalf("expected name %s, got %s", model.ChannelAlipay, adapter.Name())
	}
}

func TestFormatAmount(t *testing.T) {
	tests := []struct {
		cents int64
		want  string
	}{
		{0, "0.00"},
		{1, "0.01"},
		{100, "1.00"},
		{9999, "99.99"},
		{10000, "100.00"},
		{1000000, "10000.00"},
	}
	for _, tt := range tests {
		got := formatAmount(tt.cents)
		if got != tt.want {
			t.Errorf("formatAmount(%d) = %s, want %s", tt.cents, got, tt.want)
		}
	}
}

func TestMapAlipayTradeStatus(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"TRADE_SUCCESS", model.PaymentStatusPaid},
		{"TRADE_FINISHED", model.PaymentStatusPaid},
		{"WAIT_BUYER_PAY", model.PaymentStatusPending},
		{"TRADE_CLOSED", model.PaymentStatusFailed},
		{"UNKNOWN", model.PaymentStatusPending},
	}
	for _, tt := range tests {
		got := mapAlipayTradeStatus(tt.status)
		if got != tt.want {
			t.Errorf("mapAlipayTradeStatus(%s) = %s, want %s", tt.status, got, tt.want)
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
		{"hello world", 5, "hello"},
		{"你好世界", 2, "你好"},
		{"", 5, ""},
	}
	for _, tt := range tests {
		got := truncate(tt.in, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.in, tt.maxLen, got, tt.want)
		}
	}
}

// TestVerifyCallback_ValidSignature generates real RSA keys, signs a mock Alipay
// callback using the same RSA2 algorithm, and verifies the adapter accepts it.
func TestVerifyCallback_ValidSignature(t *testing.T) {
	priv, pub := generateRSAKeyPair(t)

	adapter, err := NewAdapter(&config.AlipayConfig{
		AppID:           "2021000000000001",
		PrivateKey:      privKeyToPEMString(t, priv),
		AlipayPublicKey: pubKeyToPEM(t, pub),
	})
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}

	values := buildMockAlipayCallback(t, priv)

	rawBody := []byte(values.Encode())
	result, err := adapter.VerifyCallback(context.Background(), &channel.CallbackData{
		RawBody: rawBody,
	})
	if err != nil {
		t.Fatalf("VerifyCallback failed: %v", err)
	}

	if result.PaymentID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("wrong payment ID: %s", result.PaymentID)
	}
	if result.ChannelTxID != "2024011522001000000000000001" {
		t.Errorf("wrong channel tx ID: %s", result.ChannelTxID)
	}
	if result.Amount != 100 {
		t.Errorf("wrong amount: %d (want 100 cents)", result.Amount)
	}
	if result.Currency != "CNY" {
		t.Errorf("wrong currency: %s", result.Currency)
	}
}

// TestVerifyCallback_InvalidSignature ensures tampered callbacks are rejected.
func TestVerifyCallback_InvalidSignature(t *testing.T) {
	priv, _ := generateRSAKeyPair(t)
	_, wrongPub := generateRSAKeyPair(t)

	adapter, err := NewAdapter(&config.AlipayConfig{
		AppID:           "2021000000000001",
		PrivateKey:      privKeyToPEMString(t, priv),
		AlipayPublicKey: pubKeyToPEM(t, wrongPub),
	})
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}

	values := buildMockAlipayCallback(t, priv)
	rawBody := []byte(values.Encode())

	_, err = adapter.VerifyCallback(context.Background(), &channel.CallbackData{
		RawBody: rawBody,
	})
	if err == nil {
		t.Fatal("expected signature verification error with wrong public key")
	}
}

// TestVerifyCallback_MissingFields ensures missing required fields are rejected.
func TestVerifyCallback_MissingFields(t *testing.T) {
	priv, pub := generateRSAKeyPair(t)

	adapter, err := NewAdapter(&config.AlipayConfig{
		AppID:           "2021000000000001",
		PrivateKey:      privKeyToPEMString(t, priv),
		AlipayPublicKey: pubKeyToPEM(t, pub),
	})
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}

	// Missing trade_no
	values := url.Values{}
	values.Set("out_trade_no", "test-order-001")
	values.Set("total_amount", "1.00")
	values.Set("trade_status", "TRADE_SUCCESS")
	values.Set("sign_type", "RSA2")
	values.Set("sign", alipaySign(t, values, priv))

	rawBody := []byte(values.Encode())
	_, err = adapter.VerifyCallback(context.Background(), &channel.CallbackData{
		RawBody: rawBody,
	})
	if err == nil {
		t.Fatal("expected error for missing trade_no")
	}
}

// TestCreatePayment_InvalidAmount tests validation.
func TestCreatePayment_InvalidAmount(t *testing.T) {
	priv, _ := generateRSAKeyPair(t)
	adapter, err := NewAdapter(&config.AlipayConfig{
		AppID:      "test",
		PrivateKey: privKeyToPEMString(t, priv),
	})
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}

	_, err = adapter.CreatePayment(context.Background(), &channel.CreatePaymentRequest{
		Amount: 0,
	})
	if err == nil {
		t.Fatal("expected error for zero amount")
	}

	_, err = adapter.CreatePayment(context.Background(), &channel.CreatePaymentRequest{
		Amount:    -1,
		TradeType: "native",
	})
	if err == nil {
		t.Fatal("expected error for negative amount")
	}
}

// buildMockAlipayCallback creates url.Values mimicking a real Alipay async notification.
func buildMockAlipayCallback(t *testing.T, priv *rsa.PrivateKey) url.Values {
	t.Helper()
	values := url.Values{}
	values.Set("gmt_create", "2024-01-15 12:00:00")
	values.Set("charset", "utf-8")
	values.Set("seller_email", "seller@example.com")
	values.Set("subject", "Test Order")
	values.Set("sign_type", "RSA2")
	values.Set("body", "Test payment")
	values.Set("buyer_id", "2088000000000001")
	values.Set("invoice_amount", "1.00")
	values.Set("notify_id", "notify_20240115000001")
	values.Set("fund_bill_list", "[{\"amount\":\"1.00\",\"fundChannel\":\"ALIPAYACCOUNT\"}]")
	values.Set("notify_type", "trade_status_sync")
	values.Set("trade_status", "TRADE_SUCCESS")
	values.Set("receipt_amount", "1.00")
	values.Set("buyer_pay_amount", "1.00")
	values.Set("app_id", "2021000000000001")
	values.Set("point_amount", "0.00")
	values.Set("gmt_payment", "2024-01-15 12:05:00")
	values.Set("notify_time", "2024-01-15 12:05:00")
	values.Set("seller_id", "2088000000000002")
	values.Set("out_trade_no", "550e8400-e29b-41d4-a716-446655440000")
	values.Set("total_amount", "1.00")
	values.Set("trade_no", "2024011522001000000000000001")
	values.Set("version", "1.0")
	values.Set("auth_app_id", "2021000000000001")
	values.Set("buyer_logon_id", "buy***@example.com")

	// Sign after all fields are set
	sig := alipaySign(t, values, priv)
	values.Set("sign", sig)

	return values
}