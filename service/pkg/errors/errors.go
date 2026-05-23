package errors

import "fmt"

type AppError struct {
	Code    string
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error { return e.Err }

func New(code, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

func Wrap(code, message string, err error) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}

const (
	ValidationError       = "VALIDATION_ERROR"
	NotFound             = "NOT_FOUND"
	InternalError        = "INTERNAL_ERROR"
	Unauthorized         = "UNAUTHORIZED"
	PaymentFailed        = "PAYMENT_FAILED"
	ChannelError         = "CHANNEL_ERROR"
	DuplicatePayment     = "DUPLICATE_PAYMENT"
	InvalidSignature     = "INVALID_SIGNATURE"
)
