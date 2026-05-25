package security

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
	"time"
)

// GenerateChecksum computes SHA-256 checksum from sorted parameter values.
// Steps: sort keys alphabetically → concatenate values → append date (YYYY-MM-DD) → SHA-256
func GenerateChecksum(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var values strings.Builder
	for _, k := range keys {
		values.WriteString(params[k])
	}
	values.WriteString(time.Now().Format("2006-01-02"))

	hash := sha256.Sum256([]byte(values.String()))
	return hex.EncodeToString(hash[:])
}

// GenerateChecksumWithDate computes SHA-256 checksum using a specific date string.
func GenerateChecksumWithDate(params map[string]string, date string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var values strings.Builder
	for _, k := range keys {
		values.WriteString(params[k])
	}
	values.WriteString(date)

	hash := sha256.Sum256([]byte(values.String()))
	return hex.EncodeToString(hash[:])
}
