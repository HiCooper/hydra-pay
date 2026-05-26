package tradeno

import (
	"strings"
	"testing"
)

func TestGenerateFormat(t *testing.T) {
	no := Generate("alipay")

	// Format: YYYYMMDD + channel code(2) + HHMMSS + random(6)
	// Total: 8 + 2 + 6 + 6 = 22 characters
	if len(no) != 22 {
		t.Errorf("expected length 22, got %d: %s", len(no), no)
	}

	// Channel code for alipay is "00"
	if !strings.Contains(no, "00") {
		t.Errorf("expected alipay channel code '00' in %s", no)
	}
}

func TestGenerateChannelCodes(t *testing.T) {
	tests := []struct {
		channel string
		code    string
	}{
		{"alipay", "00"},
		{"wechat", "01"},
		{"stripe", "02"},
	}

	for _, tt := range tests {
		no := Generate(tt.channel)
		if !strings.Contains(no[8:10], tt.code) {
			t.Errorf("channel %s: expected code %s in position 8-10, got %s", tt.channel, tt.code, no[8:10])
		}
	}
}

func TestGenerateUnknownChannel(t *testing.T) {
	no := Generate("paypal")
	// Unknown channel gets code "99"
	if no[8:10] != "99" {
		t.Errorf("expected code '99' for unknown channel, got %s", no[8:10])
	}
}

func TestGenerateUniqueness(t *testing.T) {
	// Generate 10 trade numbers and verify all are unique
	seen := make(map[string]bool)
	for i := 0; i < 10; i++ {
		no := Generate("alipay")
		if seen[no] {
			t.Errorf("duplicate trade number generated: %s", no)
		}
		seen[no] = true
	}
}

func TestChannelCode(t *testing.T) {
	if c := ChannelCode("alipay"); c != "00" {
		t.Errorf("expected '00', got %s", c)
	}
	if c := ChannelCode("wechat"); c != "01" {
		t.Errorf("expected '01', got %s", c)
	}
	if c := ChannelCode("unknown"); c != "99" {
		t.Errorf("expected '99' for unknown, got %s", c)
	}
}
