package hydrapay

import "fmt"

// APIError represents an error returned by the HydraPay API.
type APIError struct {
	Code       string
	Message    string
	StatusCode int
}

func (e *APIError) Error() string {
	return fmt.Sprintf("hydrapay: [%d] %s: %s", e.StatusCode, e.Code, e.Message)
}
