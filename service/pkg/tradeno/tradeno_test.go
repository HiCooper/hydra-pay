package tradeno

import (
	"strings"
	"testing"
)

func TestGenerateFormat(t *testing.T) {
	no := Generate("00")

	// Format: YYYYMMDD + channel code(2) + HHMMSS + random(6)
	// Total: 8 + 2 + 6 + 6 = 22 characters
	if len(no) != 22 {
		t.Errorf("expected length 22, got %d: %s", len(no), no)
	}

	// Code "00" should appear at positions 8-10
	if no[8:10] != "00" {
		t.Errorf("expected code '00' in position 8-10, got %s", no[8:10])
	}
}

func TestGenerateCodes(t *testing.T) {
	tests := []struct {
		code string
	}{
		{"00"},
		{"01"},
		{"03"},
	}

	for _, tt := range tests {
		no := Generate(tt.code)
		if no[8:10] != tt.code {
			t.Errorf("code %s: expected in position 8-10, got %s", tt.code, no[8:10])
		}
	}
}

func TestGenerateEmptyCode(t *testing.T) {
	no := Generate("")
	// Empty code falls back to "99"
	if no[8:10] != "99" {
		t.Errorf("expected code '99' for empty code, got %s", no[8:10])
	}
}

func TestGenerateUniqueness(t *testing.T) {
	// Generate 10 trade numbers and verify all are unique
	seen := make(map[string]bool)
	for i := 0; i < 10; i++ {
		no := Generate("00")
		if seen[no] {
			t.Errorf("duplicate trade number generated: %s", no)
		}
		seen[no] = true
	}
}

func TestGenerateContainsTimestamp(t *testing.T) {
	no := Generate("00")
	// Positions 0-8 should be numeric (YYYYMMDD date)
	datePart := no[0:8]
	for _, c := range datePart {
		if c < '0' || c > '9' {
			t.Errorf("expected numeric date part, got %s", datePart)
			break
		}
	}
	// Positions 10-16 should be numeric (HHMMSS time)
	timePart := no[10:16]
	for _, c := range timePart {
		if c < '0' || c > '9' {
			t.Errorf("expected numeric time part, got %s", timePart)
			break
		}
	}
	// Positions 16-22 should be random digits
	randomPart := no[16:22]
	if !strings.ContainsFunc(randomPart, func(r rune) bool { return r >= '0' && r <= '9' }) {
		t.Errorf("expected numeric random part, got %s", randomPart)
	}
}
