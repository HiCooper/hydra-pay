package tradeno

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

var channelCodes = map[string]string{
	"alipay": "00",
	"wechat": "01",
	"stripe": "02",
}

func Generate(channel string) string {
	now := time.Now()
	date := now.Format("20060102")
	timePart := now.Format("150405")

	code, ok := channelCodes[channel]
	if !ok {
		code = "99"
	}

	randomPart := randomDigits(6)

	return fmt.Sprintf("%s%s%s%s", date, code, timePart, randomPart)
}

func ChannelCode(channel string) string {
	if code, ok := channelCodes[channel]; ok {
		return code
	}
	return "99"
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
