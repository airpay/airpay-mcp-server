package security

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// GeneratePrivateKey computes the SHA-256 private key for Airpay authentication.
// Formula: SHA-256(secret + "@" + username + ":|:" + password)
func GeneratePrivateKey(secret, username, password string) string {
	input := fmt.Sprintf("%s@%s:|:%s", secret, username, password)
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}
