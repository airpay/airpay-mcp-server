package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// AirpayEncryption handles AES-256-CBC encryption for Airpay API communication.
type AirpayEncryption struct {
	encryptionKey []byte
}

// NewAirpayEncryption creates a new encryption instance from merchant credentials.
// Key derivation: MD5(username + "~:~" + password) → hex-encoded 32-byte AES-256 key.
// MD5 is mandated by the Airpay API protocol spec and cannot be changed unilaterally.
func NewAirpayEncryption(username, password string) *AirpayEncryption {
	hash := md5.Sum([]byte(username + "~:~" + password))
	return &AirpayEncryption{
		encryptionKey: []byte(hex.EncodeToString(hash[:])),
	}
}

// Encrypt encrypts the given data using AES-256-CBC with a random IV.
// Returns: IV (16-char hex) + Base64(encrypted_data)
func (e *AirpayEncryption) Encrypt(data string) (string, error) {
	block, err := aes.NewCipher(e.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("creating cipher: %w", err)
	}

	// Generate 8 random bytes, encode as 16-char hex string for IV
	ivBytes := make([]byte, 8)
	if _, err := rand.Read(ivBytes); err != nil {
		return "", fmt.Errorf("generating IV: %w", err)
	}

	iv := []byte(hex.EncodeToString(ivBytes))
	padded := pkcs5Pad([]byte(data), aes.BlockSize)
	encrypted := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(encrypted, padded)

	return string(iv) + base64.StdEncoding.EncodeToString(encrypted), nil
}

// Decrypt decrypts an Airpay API response.
// Input format: first 16 chars = IV, remainder = Base64-encoded ciphertext
func (e *AirpayEncryption) Decrypt(response string) (json.RawMessage, error) {
	if len(response) < 17 {
		return nil, fmt.Errorf("response too short to contain IV and data")
	}

	iv := []byte(response[:16])
	encryptedData, err := base64.StdEncoding.DecodeString(response[16:])
	if err != nil {
		return nil, fmt.Errorf("decoding base64: %w", err)
	}

	block, err := aes.NewCipher(e.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	if len(encryptedData)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("encrypted data length %d is not a multiple of block size %d", len(encryptedData), aes.BlockSize)
	}

	decrypted := make([]byte, len(encryptedData))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(decrypted, encryptedData)
	unpadded := pkcs5Unpad(decrypted)

	return json.RawMessage(unpadded), nil
}

// keyLen returns the length of the derived encryption key (used in tests to verify key derivation).
func (e *AirpayEncryption) keyLen() int {
	return len(e.encryptionKey)
}

// pkcs5Pad applies PKCS5 padding to the data.
func pkcs5Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padText := make([]byte, padding)
	for i := range padText {
		padText[i] = byte(padding)
	}
	return append(data, padText...)
}

// pkcs5Unpad removes PKCS5 padding from the data.
func pkcs5Unpad(data []byte) []byte {
	if len(data) == 0 {
		return data
	}
	padding := int(data[len(data)-1])
	if padding > len(data) || padding == 0 {
		return data
	}
	for i := len(data) - padding; i < len(data); i++ {
		if data[i] != byte(padding) {
			return data
		}
	}
	return data[:len(data)-padding]
}
