package webhook

import (
	"testing"
	"time"
)

func TestSignAndVerify(t *testing.T) {
	secret := "whsec_test_12345"
	body := []byte(`{"event":"payment.success","payment_id":"abc-123"}`)
	ts := time.Now().Unix()

	header := Sign(secret, body, ts)
	if header == "" {
		t.Fatal("expected non-empty signature header")
	}

	if !Verify(secret, body, header, 300) {
		t.Fatal("expected valid signature")
	}
}

func TestVerifyWrongSecret(t *testing.T) {
	secret := "whsec_test_12345"
	body := []byte(`{"event":"payment.success"}`)
	ts := time.Now().Unix()

	header := Sign(secret, body, ts)

	if Verify("wrong_secret", body, header, 300) {
		t.Fatal("expected invalid signature with wrong secret")
	}
}

func TestVerifyTamperedBody(t *testing.T) {
	secret := "whsec_test_12345"
	body := []byte(`{"event":"payment.success"}`)
	ts := time.Now().Unix()

	header := Sign(secret, body, ts)

	if Verify(secret, []byte(`{"event":"payment.failed"}`), header, 300) {
		t.Fatal("expected invalid signature with tampered body")
	}
}

func TestVerifyExpiredTimestamp(t *testing.T) {
	secret := "whsec_test_12345"
	body := []byte(`{"event":"payment.success"}`)
	// Timestamp from 10 minutes ago
	ts := time.Now().Unix() - 600

	header := Sign(secret, body, ts)

	if Verify(secret, body, header, 300) {
		t.Fatal("expected invalid signature with expired timestamp")
	}
}

func TestVerifyEmptyHeader(t *testing.T) {
	if Verify("secret", []byte("body"), "", 300) {
		t.Fatal("expected invalid with empty header")
	}
}

func TestVerifyEmptySecret(t *testing.T) {
	body := []byte(`{"event":"payment.success"}`)
	ts := time.Now().Unix()
	header := Sign("secret", body, ts)

	if Verify("", body, header, 300) {
		t.Fatal("expected invalid with empty secret")
	}
}

func TestParseHeaderInvalid(t *testing.T) {
	cases := []string{
		"",
		"t=123",
		"v1=abc",
		"t=abc,v1=def",
		"t=,v1=",
	}

	for _, c := range cases {
		_, _, ok := parseHeader(c)
		if ok {
			t.Errorf("expected parse failure for %q", c)
		}
	}
}

func TestIdempotentSignatures(t *testing.T) {
	secret := "whsec_test"
	body := []byte(`same body`)
	ts := int64(1716652800)

	h1 := Sign(secret, body, ts)
	h2 := Sign(secret, body, ts)

	if h1 != h2 {
		t.Fatal("expected identical signatures for same inputs")
	}
}
