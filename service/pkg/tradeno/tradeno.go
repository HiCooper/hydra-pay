package tradeno

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

// Generate creates a trade number from a channel code (from payment_channels.code).
// Format: YYYYMMDD + code + HHMMSS + 6-random-digits
func Generate(code string) string {
	now := time.Now()
	date := now.Format("20060102")
	timePart := now.Format("150405")

	if code == "" {
		code = "99"
	}

	randomPart := randomDigits(6)

	return fmt.Sprintf("%s%s%s%s", date, code, timePart, randomPart)
}

func randomDigits(n int) string {
	digits := make([]byte, n)
	for i := 0; i < n; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			num = big.NewInt(int64(time.Now().UnixNano() % 10))
		}
		digits[i] = '0' + byte(num.Int64())
	}
	return string(digits)
}
