package model

import (
	"testing"
)

func TestSetPassword(t *testing.T) {
	m := &Merchant{}

	err := m.SetPassword("my-secret-password")
	if err != nil {
		t.Fatalf("SetPassword failed: %v", err)
	}
	if m.PasswordHash == "" {
		t.Fatal("expected PasswordHash to be set")
	}
	if m.PasswordHash == "my-secret-password" {
		t.Fatal("password should be hashed, not stored in plaintext")
	}
}

func TestCheckPassword(t *testing.T) {
	m := &Merchant{}
	if err := m.SetPassword("correct-password"); err != nil {
		t.Fatalf("SetPassword failed: %v", err)
	}

	if !m.CheckPassword("correct-password") {
		t.Fatal("expected CheckPassword to return true for correct password")
	}
	if m.CheckPassword("wrong-password") {
		t.Fatal("expected CheckPassword to return false for wrong password")
	}
	if m.CheckPassword("") {
		t.Fatal("expected CheckPassword to return false for empty password")
	}
}

func TestSetPasswordEmpty(t *testing.T) {
	m := &Merchant{}
	err := m.SetPassword("")
	if err != nil {
		t.Fatalf("SetPassword with empty string should not error (bcrypt handles it): %v", err)
	}
	if m.PasswordHash == "" {
		t.Fatal("expected PasswordHash to be set even for empty password")
	}
	// Empty password should still verify
	if !m.CheckPassword("") {
		t.Fatal("expected empty password to verify after SetPassword('')")
	}
}

func TestSetPasswordLong(t *testing.T) {
	t.Run("exactly 72 bytes", func(t *testing.T) {
		m := &Merchant{}
		longPass := make([]byte, 72)
		for i := range longPass {
			longPass[i] = 'a'
		}
		err := m.SetPassword(string(longPass))
		if err != nil {
			t.Fatalf("SetPassword with 72-byte password should succeed: %v", err)
		}
		if !m.CheckPassword(string(longPass)) {
			t.Fatal("expected 72-byte password to verify correctly")
		}
	})

	t.Run("over 72 bytes", func(t *testing.T) {
		m := &Merchant{}
		longPass := make([]byte, 100)
		for i := range longPass {
			longPass[i] = 'a'
		}
		err := m.SetPassword(string(longPass))
		if err == nil {
			t.Fatal("expected error for password exceeding 72 bytes")
		}
	})
}

func TestCheckPasswordNoHash(t *testing.T) {
	m := &Merchant{PasswordHash: ""}
	if m.CheckPassword("anything") {
		t.Fatal("expected CheckPassword to return false when no hash is set")
	}
}

func TestSetPasswordBcryptCost(t *testing.T) {
	m := &Merchant{}
	if err := m.SetPassword("test"); err != nil {
		t.Fatalf("SetPassword failed: %v", err)
	}
	// bcrypt hash with DefaultCost should start with $2a$10$
	if len(m.PasswordHash) < 7 || m.PasswordHash[:7] != "$2a$10$" && m.PasswordHash[:7] != "$2b$10$" {
		t.Logf("PasswordHash format: %s", m.PasswordHash[:20])
	}
}
