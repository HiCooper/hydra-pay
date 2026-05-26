package hydrapay

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// SubscriptionService provides subscription-related API methods.
type SubscriptionService struct {
	client *Client
}

// CreateSubscription creates a new subscription. Pass an optional IdempotencyParams for idempotent creation.
func (s *SubscriptionService) CreateSubscription(params *CreateSubscriptionParams, idem *IdempotencyParams) (*Subscription, error) {
	var sub Subscription
	if err := s.client.do("POST", "/subscriptions", params, idem, &sub); err != nil {
		return nil, err
	}
	return &sub, nil
}

// GetSubscription retrieves a subscription by ID.
func (s *SubscriptionService) GetSubscription(id string) (*Subscription, error) {
	var sub Subscription
	if err := s.client.do("GET", "/subscriptions/"+id, nil, nil, &sub); err != nil {
		return nil, err
	}
	return &sub, nil
}

// ListSubscriptions lists subscriptions, optionally filtered by user_id.
func (s *SubscriptionService) ListSubscriptions(params *ListSubscriptionsParams) ([]Subscription, error) {
	query := url.Values{}
	if params != nil {
		if params.UserID != "" {
			query.Set("user_id", params.UserID)
		}
		if params.Page > 0 {
			query.Set("page", strconv.Itoa(params.Page))
		}
		if params.PageSize > 0 {
			query.Set("page_size", strconv.Itoa(params.PageSize))
		}
	}

	path := "/subscriptions"
	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	// The API returns two shapes:
	//   user_id filter:  {data: {subscriptions: [...]}}
	//   paginated list:  {data: [...]}
	// Try wrapped shape first, fall back to raw array.
	var raw json.RawMessage
	if err := s.client.do("GET", path, nil, nil, &raw); err != nil {
		return nil, err
	}

	// Try wrapped object
	var wrapped struct {
		Subscriptions []Subscription `json:"subscriptions"`
	}
	if err := json.Unmarshal(raw, &wrapped); err == nil && wrapped.Subscriptions != nil {
		return wrapped.Subscriptions, nil
	}

	// Try raw array
	var subs []Subscription
	if err := json.Unmarshal(raw, &subs); err != nil {
		return nil, fmt.Errorf("hydrapay: failed to parse subscriptions: %w", err)
	}
	return subs, nil
}

// CancelSubscription cancels an active or past_due subscription.
func (s *SubscriptionService) CancelSubscription(id string, idem *IdempotencyParams) (*Subscription, error) {
	var sub Subscription
	if err := s.client.do("POST", "/subscriptions/"+id+"/cancel", nil, idem, &sub); err != nil {
		return nil, err
	}
	return &sub, nil
}
