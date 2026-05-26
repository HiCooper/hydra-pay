package hydrapay

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// WebhookService provides webhook verification utilities.
// It is stateless and does not require a Client instance.
type WebhookService struct{}

// VerifySignature checks an X-HydraPay-Signature header against the webhook secret and raw body.
// It rejects timestamps older than toleranceSeconds (use 300 for 5 minutes).
func (w *WebhookService) VerifySignature(payload []byte, signatureHeader string, secret string, toleranceSeconds int64) bool {
	if secret == "" || signatureHeader == "" {
		return false
	}

	timestamp, sig, ok := parseSignatureHeader(signatureHeader)
	if !ok {
		return false
	}

	// Replay protection
	if abs(time.Now().Unix()-timestamp) > toleranceSeconds {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(fmt.Sprintf("%d.", timestamp)))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(sig), []byte(expected))
}

// ParseEvent parses a webhook payload into an Event.
func (w *WebhookService) ParseEvent(payload []byte) (*Event, error) {
	var event Event
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("hydrapay: failed to parse webhook event: %w", err)
	}
	return &event, nil
}

// parseSignatureHeader extracts the timestamp and v1 signature.
// Format: "t=1234567890,v1=abcdef..."
func parseSignatureHeader(header string) (timestamp int64, sig string, ok bool) {
	tIdx := strings.Index(header, "t=")
	vIdx := strings.Index(header, ",v1=")
	if tIdx < 0 || vIdx < 0 || vIdx <= tIdx {
		return 0, "", false
	}

	ts, err := strconv.ParseInt(header[tIdx+2:vIdx], 10, 64)
	if err != nil {
		return 0, "", false
	}

	signature := header[vIdx+4:]
	if signature == "" {
		return 0, "", false
	}

	return ts, signature, true
}

func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
