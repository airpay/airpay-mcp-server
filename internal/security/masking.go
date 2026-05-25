package security

import (
	"regexp"
	"strings"
)

var (
	cardNumberRegex = regexp.MustCompile(`\b(\d{4})\d{8,12}(\d{4})\b`)
	tokenRegex      = regexp.MustCompile(`"(access_token|token|privatekey|encdata|checksum)"\s*:\s*"([^"]+)"`)
	emailRegex      = regexp.MustCompile(`([a-zA-Z0-9._%+-]+)@([a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`)
	phoneRegex      = regexp.MustCompile(`\b(\d{2})\d{4,8}(\d{2})\b`)
)

// MaskCardNumber masks middle digits of a card number: 4321XXXXXXXX5678
func MaskCardNumber(cardNumber string) string {
	return cardNumberRegex.ReplaceAllString(cardNumber, "${1}XXXXXXXX${2}")
}

// MaskSensitiveJSON masks sensitive fields in JSON string for logging.
func MaskSensitiveJSON(jsonStr string) string {
	return tokenRegex.ReplaceAllString(jsonStr, `"$1":"***MASKED***"`)
}

// MaskEmail masks email: j***@example.com
func MaskEmail(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 || len(parts[0]) == 0 {
		return email
	}
	return string(parts[0][0]) + "***@" + parts[1]
}

// MaskPhone masks phone number: 99****99
func MaskPhone(phone string) string {
	if len(phone) < 6 {
		return phone
	}
	return phone[:2] + strings.Repeat("*", len(phone)-4) + phone[len(phone)-2:]
}

// MaskResponseForLog masks all sensitive data in a response string for safe logging.
func MaskResponseForLog(response string) string {
	result := MaskSensitiveJSON(response)
	result = cardNumberRegex.ReplaceAllString(result, "${1}XXXXXXXX${2}")
	return result
}
