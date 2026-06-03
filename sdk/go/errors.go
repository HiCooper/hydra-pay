package hydrapay

import "fmt"

// Standard error codes returned by the HydraPay API.
const (
	ErrValidation       = "VALIDATION_ERROR"
	ErrNotFound         = "NOT_FOUND"
	ErrInternal         = "INTERNAL_ERROR"
	ErrUnauthorized     = "UNAUTHORIZED"
	ErrPaymentFailed    = "PAYMENT_FAILED"
	ErrChannelError     = "CHANNEL_ERROR"
	ErrDuplicatePayment = "DUPLICATE_PAYMENT"
	ErrInvalidSignature = "INVALID_SIGNATURE"
)

// APIError represents an error returned by the HydraPay API.
type APIError struct {
	Code       string
	Message    string
	StatusCode int
}

func (e *APIError) Error() string {
	return fmt.Sprintf("hydrapay: [%d] %s: %s", e.StatusCode, e.Code, e.Message)
}
