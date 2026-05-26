package hydrapay

import (
	"fmt"
	"testing"
	"time"
)

func TestVerifySignatureValid(t *testing.T) {
	secret := "whsec_test_12345"
	payload := []byte(`{"event":"payment.success","payment_id":"p_001"}`)

	// Sign manually
	now := time.Now().Unix()
	sig := signForTest(secret, payload, now)
	header := fmt.Sprintf("t=%d,v1=%s", now, sig)

	w := &WebhookService{}
	if !w.VerifySignature(payload, header, secret, 300) {
		t.Error("expected valid signature to pass")
	}
}

func TestVerifySignatureExpired(t *testing.T) {
	secret := "whsec_test_12345"
	payload := []byte(`{"event":"payment.success"}`)

	// Timestamp from 10 minutes ago
	old := time.Now().Unix() - 600
	sig := signForTest(secret, payload, old)
	header := fmt.Sprintf("t=%d,v1=%s", old, sig)

	w := &WebhookService{}
	if w.VerifySignature(payload, header, secret, 300) {
		t.Error("expected expired signature to be rejected")
	}
}

func TestVerifySignatureWrongSecret(t *testing.T) {
	payload := []byte(`{"event":"payment.success"}`)
	now := time.Now().Unix()
	sig := signForTest("correct_secret", payload, now)
	header := fmt.Sprintf("t=%d,v1=%s", now, sig)

	w := &WebhookService{}
	if w.VerifySignature(payload, header, "wrong_secret", 300) {
		t.Error("expected wrong secret to fail")
	}
}

func TestVerifySignatureTamperedBody(t *testing.T) {
	secret := "whsec_test_12345"
	now := time.Now().Unix()
	sig := signForTest(secret, []byte(`{"event":"payment.success"}`), now)
	header := fmt.Sprintf("t=%d,v1=%s", now, sig)

	// Tampered body
	w := &WebhookService{}
	if w.VerifySignature([]byte(`{"event":"payment.failed"}`), header, secret, 300) {
		t.Error("expected tampered body to fail")
	}
}

func TestVerifySignatureEmptyInputs(t *testing.T) {
	w := &WebhookService{}
	if w.VerifySignature([]byte(`{}`), "", "secret", 300) {
		t.Error("expected empty header to fail")
	}
	if w.VerifySignature([]byte(`{}`), "t=1,v1=abc", "", 300) {
		t.Error("expected empty secret to fail")
	}
}

func TestParseEvent(t *testing.T) {
	w := &WebhookService{}
	payload := []byte(`{"event":"payment.success","payment_id":"p_001","amount":29900,"currency":"CNY","channel":"alipay"}`)

	event, err := w.ParseEvent(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Event != "payment.success" {
		t.Errorf("expected event 'payment.success', got %s", event.Event)
	}
	if event.PaymentID != "p_001" {
		t.Errorf("expected payment_id 'p_001', got %s", event.PaymentID)
	}
	if event.Amount != 29900 {
		t.Errorf("expected amount 29900, got %d", event.Amount)
	}
}

func TestParseEventRefund(t *testing.T) {
	w := &WebhookService{}
	payload := []byte(`{"event":"payment.refunded","payment_id":"p_001","refund_id":"r_001","refund_amount":10000,"refund_reason":"customer request"}`)

	event, err := w.ParseEvent(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Event != "payment.refunded" {
		t.Errorf("expected event 'payment.refunded', got %s", event.Event)
	}
	if event.RefundAmount != 10000 {
		t.Errorf("expected refund_amount 10000, got %d", event.RefundAmount)
	}
}

func TestParseEventInvalidJSON(t *testing.T) {
	w := &WebhookService{}
	_, err := w.ParseEvent([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseSignatureHeader(t *testing.T) {
	ts, sig, ok := parseSignatureHeader("t=1700000000,v1=abcdef123456")
	if !ok {
		t.Fatal("expected parse to succeed")
	}
	if ts != 1700000000 {
		t.Errorf("expected ts 1700000000, got %d", ts)
	}
	if sig != "abcdef123456" {
		t.Errorf("expected sig 'abcdef123456', got '%s'", sig)
	}
}

func TestParseSignatureHeaderInvalid(t *testing.T) {
	cases := []string{
		"",
		"t=abc,v1=def",
		"v1=def",
		"t=1",
		"t=1,v1=",
	}
	for _, h := range cases {
		_, _, ok := parseSignatureHeader(h)
		if ok {
			t.Errorf("expected failure for header '%s'", h)
		}
	}
}
