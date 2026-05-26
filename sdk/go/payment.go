package hydrapay

// PaymentService provides payment-related API methods.
type PaymentService struct {
	client *Client
}

// CreatePayment creates a new payment. Pass an optional IdempotencyParams for idempotent creation.
func (s *PaymentService) CreatePayment(params *CreatePaymentParams, idem *IdempotencyParams) (*Payment, error) {
	var payment Payment
	if err := s.client.do("POST", "/payments/create", params, idem, &payment); err != nil {
		return nil, err
	}
	return &payment, nil
}

// GetPayment retrieves a payment by ID.
func (s *PaymentService) GetPayment(id string) (*Payment, error) {
	var payment Payment
	if err := s.client.do("GET", "/payments/"+id, nil, nil, &payment); err != nil {
		return nil, err
	}
	return &payment, nil
}
