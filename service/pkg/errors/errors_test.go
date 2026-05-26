package errors

import (
	"fmt"
	"testing"
)

func TestNew(t *testing.T) {
	err := New(ValidationError, "amount must be positive")

	if err.Code != ValidationError {
		t.Errorf("expected code %s, got %s", ValidationError, err.Code)
	}
	if err.Message != "amount must be positive" {
		t.Errorf("expected message, got %s", err.Message)
	}
	if err.Err != nil {
		t.Errorf("expected nil cause, got %v", err.Err)
	}
}

func TestWrap(t *testing.T) {
	cause := fmt.Errorf("db connection refused")
	err := Wrap(InternalError, "failed to save", cause)

	if err.Code != InternalError {
		t.Errorf("expected code %s, got %s", InternalError, err.Code)
	}
	if err.Message != "failed to save" {
		t.Errorf("expected message, got %s", err.Message)
	}
	if err.Err != cause {
		t.Errorf("expected cause to be preserved")
	}
}

func TestUnwrap(t *testing.T) {
	cause := fmt.Errorf("original error")
	err := Wrap("CODE", "msg", cause)

	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Errorf("Unwrap should return the cause")
	}
}

func TestErrorString(t *testing.T) {
	err := New(NotFound, "payment not found")
	s := err.Error()

	if s != "NOT_FOUND: payment not found" {
		t.Errorf("expected 'NOT_FOUND: payment not found', got '%s'", s)
	}
}

func TestErrorStringWithCause(t *testing.T) {
	cause := fmt.Errorf("record not found")
	err := Wrap(NotFound, "payment not found", cause)
	s := err.Error()

	expected := "NOT_FOUND: payment not found: record not found"
	if s != expected {
		t.Errorf("expected '%s', got '%s'", expected, s)
	}
}

func TestConstants(t *testing.T) {
	// Verify all error code constants are non-empty and consistent
	codes := []string{
		ValidationError,
		NotFound,
		InternalError,
		Unauthorized,
		PaymentFailed,
		ChannelError,
		DuplicatePayment,
		InvalidSignature,
	}
	for _, code := range codes {
		if code == "" {
			t.Errorf("empty error code constant")
		}
	}
}
