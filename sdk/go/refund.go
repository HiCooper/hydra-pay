package hydrapay

// RefundService provides refund-related API methods.
type RefundService struct {
	client *Client
}

// CreateRefund creates a new refund. Pass an optional IdempotencyParams for idempotent creation.
func (s *RefundService) CreateRefund(params *CreateRefundParams, idem *IdempotencyParams) (*Refund, error) {
	var refund Refund
	if err := s.client.do("POST", "/refunds", params, idem, &refund); err != nil {
		return nil, err
	}
	return &refund, nil
}

// GetRefund retrieves a refund by ID.
func (s *RefundService) GetRefund(id string) (*Refund, error) {
	var refund Refund
	if err := s.client.do("GET", "/refunds/"+id, nil, nil, &refund); err != nil {
		return nil, err
	}
	return &refund, nil
}

// ListPaymentRefunds lists all refunds for a payment.
func (s *RefundService) ListPaymentRefunds(paymentID string) (*RefundList, error) {
	var list RefundList
	if err := s.client.do("GET", "/payments/"+paymentID+"/refunds", nil, nil, &list); err != nil {
		return nil, err
	}
	return &list, nil
}
