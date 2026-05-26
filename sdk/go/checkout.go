package hydrapay

// CheckoutService provides checkout session API methods.
type CheckoutService struct {
	client *Client
}

// CreateSession creates a new checkout session. Pass an optional IdempotencyParams for idempotent creation.
func (s *CheckoutService) CreateSession(params *CreateCheckoutSessionParams, idem *IdempotencyParams) (*CheckoutSession, error) {
	var session CheckoutSession
	if err := s.client.do("POST", "/checkout/sessions", params, idem, &session); err != nil {
		return nil, err
	}
	return &session, nil
}
