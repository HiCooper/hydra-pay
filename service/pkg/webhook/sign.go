package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Sign generates an X-HydraPay-Signature header value using HMAC-SHA256.
// Scheme: t={unix_timestamp},v1={hex(hmac-sha256(secret, "timestamp.body"))}
func Sign(secret string, body []byte, timestamp int64) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(fmt.Sprintf("%d.", timestamp)))
	mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("t=%d,v1=%s", timestamp, sig)
}

// Verify checks an X-HydraPay-Signature header value against the secret and body.
// toleranceSeconds is the maximum age of the timestamp (use 300 for 5 minutes).
func Verify(secret string, body []byte, headerValue string, toleranceSeconds int64) bool {
	if secret == "" || headerValue == "" {
		return false
	}

	timestamp, sig, ok := parseHeader(headerValue)
	if !ok {
		return false
	}

	// Reject expired timestamps (prevent replay attacks)
	now := time.Now().Unix()
	if abs(now-timestamp) > toleranceSeconds {
		return false
	}

	// Recompute and compare
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(fmt.Sprintf("%d.", timestamp)))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(sig), []byte(expected))
}

// parseHeader extracts the timestamp and v1 signature from a header value.
// Format: "t=1234567890,v1=abcdef..."
func parseHeader(headerValue string) (timestamp int64, sig string, ok bool) {
	// Find timestamp part
	tIdx := strings.Index(headerValue, "t=")
	vIdx := strings.Index(headerValue, ",v1=")
	if tIdx < 0 || vIdx < 0 || vIdx <= tIdx {
		return 0, "", false
	}

	ts, err := strconv.ParseInt(headerValue[tIdx+2:vIdx], 10, 64)
	if err != nil {
		return 0, "", false
	}

	signature := headerValue[vIdx+4:]
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
