package logger

import (
	"testing"
)

func TestMaskQuerySensitiveParam(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  string
	}{
		{"card_number masked", "card_number=6222021234567890", "card_number=****"},
		{"phone masked", "phone=13800138000", "phone=****"},
		{"secret masked", "secret=abc123&foo=bar", "secret=****&foo=bar"},
		{"token masked", "amount=100&token=sk_live_secret", "amount=100&token=****"},
		{"multiple params", "auth_code=xyz&password=secret&amount=100", "auth_code=****&password=****&amount=100"},
		{"code masked", "code=123456&name=test", "code=****&name=test"},
		{"no sensitive params", "amount=100&currency=CNY", "amount=100&currency=CNY"},
		{"empty string", "", ""},
		{"not a query string", "just-a-plain-string", "just-a-plain-string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaskQuery(tt.in); got != tt.out {
				t.Errorf("MaskQuery(%q) = %q, want %q", tt.in, got, tt.out)
			}
		})
	}
}

func TestMaskValueCardLike(t *testing.T) {
	// Valid Luhn card numbers should be masked.
	// 6222021234567890 is not Luhn-valid, so let's use known test numbers.
	// Visa test: 4111111111111111 (valid Luhn)
	in := "card=4111111111111111"
	got := maskValue(in)
	if got == in {
		t.Errorf("expected masking for valid Luhn card number, got %q", got)
	}
}

func TestMaskValuePhone(t *testing.T) {
	in := "phone=13800138000 extra"
	got := maskValue(in)
	expected := "phone=138****8000 extra"
	if got != expected {
		t.Errorf("maskValue(%q) = %q, want %q", in, got, expected)
	}
}

func TestLuhnPasses(t *testing.T) {
	tests := []struct {
		num  string
		pass bool
	}{
		{"4111111111111111", true},  // Visa test number
		{"5500000000000004", true},  // Mastercard test number
		{"1234567890123456", false}, // random invalid
		{"0000", true},              // mathematically valid
		{"", true},                  // empty sum is 0
	}
	for _, tt := range tests {
		if got := luhnPasses(tt.num); got != tt.pass {
			t.Errorf("luhnPasses(%q) = %v, want %v", tt.num, got, tt.pass)
		}
	}
}
