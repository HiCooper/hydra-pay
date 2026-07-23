package hydrapay

import "time"

// --- Envelope ---

type apiResponse struct {
	Success    bool        `json:"success"`
	Data       interface{} `json:"data,omitempty"`
	Error      *apiError   `json:"error,omitempty"`
	Pagination *Pagination `json:"pagination,omitempty"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Pagination is included in list endpoint responses.
type Pagination struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// --- Payment ---

// CreatePaymentParams is the request body for creating a payment.
type CreatePaymentParams struct {
	UserID       string                 `json:"user_id"`
	PlanID       string                 `json:"plan_id,omitempty"`
	Amount       int64                  `json:"amount"`
	Currency     string                 `json:"currency,omitempty"`
	Channel      string                 `json:"channel,omitempty"`
	TradeType    string                 `json:"trade_type,omitempty"`
	SuccessURL   string                 `json:"success_url,omitempty"`
	CancelURL    string                 `json:"cancel_url,omitempty"`
	Description  string                 `json:"description,omitempty"`
	OpenID         string               `json:"open_id,omitempty"`
	ChannelAppID   string               `json:"channel_app_id,omitempty"`
	ClientIP       string               `json:"client_ip,omitempty"`
	NotifyURL      string               `json:"notify_url,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Payment represents a created or retrieved payment.
type Payment struct {
	ID          string     `json:"payment_id"`
	TradeNo     string     `json:"trade_no"`
	AppID       string     `json:"app_id,omitempty"`
	UserID      string     `json:"user_id,omitempty"`
	PlanID      string     `json:"plan_id,omitempty"`
	Amount      int64      `json:"amount"`
	Currency    string     `json:"currency"`
	Channel     string     `json:"channel"`
	Status      string     `json:"status"`
	PaymentURL  string     `json:"payment_url,omitempty"`
	QRCodeURL   string     `json:"qr_code_url,omitempty"`
	ExternalID  string     `json:"external_id,omitempty"`
	Description string     `json:"description,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	PaidAt      *time.Time `json:"paid_at,omitempty"`
}

// --- Checkout Session ---

// CreateCheckoutSessionParams is the request body for creating a checkout session.
type CreateCheckoutSessionParams struct {
	Amount      int64  `json:"amount"`
	Currency    string `json:"currency,omitempty"`
	Description string `json:"description,omitempty"`
	SuccessURL  string `json:"success_url,omitempty"`
	CancelURL   string `json:"cancel_url,omitempty"`
}

// CheckoutSession represents a created checkout session.
type CheckoutSession struct {
	ID          string    `json:"id"`
	CheckoutURL string    `json:"checkout_url"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// --- Refund ---

// CreateRefundParams is the request body for creating a refund.
type CreateRefundParams struct {
	TradeNo      string `json:"trade_no"`
	RefundAmount int64  `json:"refund_amount"`
	RefundReason string `json:"refund_reason,omitempty"`
}

// Refund represents a created or retrieved refund.
type Refund struct {
	ID              string    `json:"refund_id"`
	PaymentID       string    `json:"payment_id,omitempty"`
	TradeNo         string    `json:"trade_no"`
	Channel         string    `json:"channel,omitempty"`
	RefundAmount    int64     `json:"refund_amount"`
	RefundReason    string    `json:"refund_reason,omitempty"`
	OutRequestNo    string    `json:"out_request_no,omitempty"`
	Status          string    `json:"status"`
	ChannelRefundID string    `json:"channel_refund_id,omitempty"`
	RefundFee       int64     `json:"refund_fee,omitempty"`
	ErrorMsg        string    `json:"error_msg,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// RefundList contains a payment's refund list.
type RefundList struct {
	PaymentID string   `json:"payment_id"`
	Refunds   []Refund `json:"refunds"`
}

// --- Subscription ---

// CreateSubscriptionParams is the request body for creating a subscription.
type CreateSubscriptionParams struct {
	PlanID string `json:"plan_id"`
	UserID string `json:"user_id"`
}

// Subscription represents a created or retrieved subscription.
type Subscription struct {
	ID                 string    `json:"id"`
	PlanID             string    `json:"plan_id"`
	PlanName           string    `json:"plan_name,omitempty"`
	UserID             string    `json:"user_id"`
	Status             string    `json:"status"`
	CurrentPeriodStart time.Time `json:"current_period_start"`
	CurrentPeriodEnd   time.Time `json:"current_period_end"`
	Amount             int64     `json:"amount,omitempty"`
	Currency           string    `json:"currency,omitempty"`
	Interval           string    `json:"interval,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

// ListSubscriptionsParams are query parameters for listing subscriptions.
type ListSubscriptionsParams struct {
	UserID   string `json:"user_id,omitempty"`
	Page     int    `json:"page,omitempty"`
	PageSize int    `json:"page_size,omitempty"`
}

// --- Idempotency ---

// IdempotencyParams holds an optional idempotency key for mutation requests.
type IdempotencyParams struct {
	Key string
}

// --- Webhook ---

// Event represents a parsed webhook event.
type Event struct {
	Event        string `json:"event"`
	PaymentID    string `json:"payment_id"`
	UserID       string `json:"user_id,omitempty"`
	PlanID       string `json:"plan_id,omitempty"`
	Amount       int64  `json:"amount"`
	Currency     string `json:"currency,omitempty"`
	Status       string `json:"status,omitempty"`
	Channel      string `json:"channel,omitempty"`
	RefundAmount int64  `json:"refund_amount,omitempty"`
	RefundReason string `json:"refund_reason,omitempty"`
	RefundID     string `json:"refund_id,omitempty"`
}
