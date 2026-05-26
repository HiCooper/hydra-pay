package model

import (
	"strings"
	"testing"
)

func TestGenerateAPIKey(t *testing.T) {
	key := GenerateAPIKey()
	if key == "" {
		t.Fatal("expected non-empty API key")
	}
	if !strings.HasPrefix(key, "sk_") {
		t.Fatalf("expected API key to start with 'sk_', got: %s", key)
	}
	// "sk_" prefix + 48 hex chars (24 bytes)
	if len(key) != 3+48 {
		t.Fatalf("expected API key length 51, got %d: %s", len(key), key)
	}
}

func TestGenerateAPIKeyUniqueness(t *testing.T) {
	keys := make(map[string]bool)
	for i := 0; i < 100; i++ {
		key := GenerateAPIKey()
		if keys[key] {
			t.Fatalf("duplicate API key generated: %s", key)
		}
		keys[key] = true
	}
}

func TestGenerateAPIKeyFormat(t *testing.T) {
	key := GenerateAPIKey()
	hexPart := key[3:] // strip "sk_"
	// hex part should only contain hex characters
	for _, c := range hexPart {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Fatalf("unexpected character in hex part: %c in %s", c, key)
		}
	}
}
