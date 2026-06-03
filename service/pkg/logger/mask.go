package logger

import (
	"regexp"
	"strings"
)

var (
	// sensitiveParamKeys lists query parameter names (lowercase) whose values are masked.
	sensitiveParamKeys = []string{
		"card_number", "bank_account", "bank_card", "phone", "mobile",
		"id_card", "identity", "secret", "token", "api_key", "apikey",
		"password", "passwd", "pwd", "sign", "signature", "private_key",
		"cert", "credential", "ssn", "cvv", "cvc", "pan", "encrypted_data",
		"code", "sms_code", "auth_code",
	}

	// sensitiveParamRe matches any known sensitive query parameter and its value.
	sensitiveParamRe = buildSensitiveParamRe()

	// Card-like number: 13-19 consecutive digits.
	cardLikeRe = regexp.MustCompile(`\b\d{13,19}\b`)

	// Chinese mobile phone: 1[3-9]xxxxxxxxx
	phoneRe = regexp.MustCompile(`\b1[3-9]\d{9}\b`)

	// Chinese ID card: 17 digits + digit/X, or 15 digits (legacy).
	idCardRe = regexp.MustCompile(`\b\d{15}(?:\d{2}[\dXx])?\b`)
)

func buildSensitiveParamRe() *regexp.Regexp {
	// Match sensitive params preceded by &, ?, or at start-of-string.
	keys := strings.Join(sensitiveParamKeys, `|`)
	pattern := `(?i)((?:^|[&?])` + keys + `)=[^&]*`
	return regexp.MustCompile(pattern)
}

// MaskQuery masks sensitive query parameter values in a query string.
// Recognized sensitive parameter names have their values replaced with "****".
// It also scans values for card numbers (Luhn-validated), phone numbers, and
// ID card numbers.
func MaskQuery(rawQuery string) string {
	if rawQuery == "" {
		return rawQuery
	}

	masked := sensitiveParamRe.ReplaceAllString(rawQuery, `${1}=****`)

	if masked != rawQuery {
		return masked
	}

	// No known-sensitive params — check for embedded patterns in values.
	return maskValue(rawQuery)
}

// maskValue applies pattern-based masking to a single value string.
// It detects and masks card numbers, phone numbers, and ID card numbers.
func maskValue(v string) string {
	v = cardLikeRe.ReplaceAllStringFunc(v, func(match string) string {
		if luhnPasses(match) {
			return maskPreserveLast4(match)
		}
		return match
	})
	v = phoneRe.ReplaceAllStringFunc(v, func(match string) string {
		return match[:3] + "****" + match[7:]
	})
	v = idCardRe.ReplaceAllStringFunc(v, func(match string) string {
		return match[:4] + "**********" + match[len(match)-2:]
	})
	return v
}

// maskPreserveLast4 keeps the last 4 characters, masks the rest.
func maskPreserveLast4(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return strings.Repeat("*", len(s)-4) + s[len(s)-4:]
}

// luhnPasses returns true if s passes the Luhn (mod-10) checksum algorithm.
func luhnPasses(s string) bool {
	sum := 0
	double := false
	for i := len(s) - 1; i >= 0; i-- {
		d := int(s[i] - '0')
		if d < 0 || d > 9 {
			return false
		}
		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		double = !double
	}
	return sum%10 == 0
}
