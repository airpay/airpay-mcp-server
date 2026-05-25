package security

import (
	"encoding/json"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	enc := NewAirpayEncryption("testuser", "testpass")

	testData := `{"orderid":"ORD123","amount":"100.00","buyer_email":"test@example.com"}`

	encrypted, err := enc.Encrypt(testData)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if len(encrypted) < 17 {
		t.Fatalf("Encrypted data too short: %d chars", len(encrypted))
	}

	// First 16 chars should be hex IV
	iv := encrypted[:16]
	for _, c := range iv {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Fatalf("IV contains non-hex character: %c", c)
		}
	}

	decrypted, err := enc.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if string(decrypted) != testData {
		t.Fatalf("Round-trip mismatch:\n  got:  %s\n  want: %s", string(decrypted), testData)
	}
}

func TestEncryptDecryptJSON(t *testing.T) {
	enc := NewAirpayEncryption("merchant_user", "merchant_pass")

	payload := map[string]string{
		"client_id":     "cid123",
		"client_secret": "csecret456",
		"grant_type":    "client_credentials",
		"merchant_id":   "12345",
	}
	jsonData, _ := json.Marshal(payload)

	encrypted, err := enc.Encrypt(string(jsonData))
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := enc.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(decrypted, &result); err != nil {
		t.Fatalf("Unmarshal decrypted JSON failed: %v", err)
	}

	for k, v := range payload {
		if result[k] != v {
			t.Errorf("Key %q: got %q, want %q", k, result[k], v)
		}
	}
}

func TestEncryptionKeyDerivation(t *testing.T) {
	// MD5("testuser~:~testpass") produces a 32-char hex key (16 bytes → 32 hex chars).
	enc := NewAirpayEncryption("testuser", "testpass")
	if enc.keyLen() != 32 {
		t.Fatalf("Key length: got %d, want 32", enc.keyLen())
	}
}

func TestGenerateChecksum(t *testing.T) {
	params := map[string]string{
		"amount":  "100.00",
		"orderid": "ORD123",
	}
	checksum := GenerateChecksumWithDate(params, "2024-01-15")
	if len(checksum) != 64 { // SHA-256 hex = 64 chars
		t.Fatalf("Checksum length: got %d, want 64", len(checksum))
	}

	// Same input should produce same output
	checksum2 := GenerateChecksumWithDate(params, "2024-01-15")
	if checksum != checksum2 {
		t.Fatal("Checksum not deterministic")
	}

	// Different date should produce different checksum
	checksum3 := GenerateChecksumWithDate(params, "2024-01-16")
	if checksum == checksum3 {
		t.Fatal("Checksum should differ with different date")
	}
}

func TestGeneratePrivateKey(t *testing.T) {
	pk := GeneratePrivateKey("mysecret", "myuser", "mypass")
	if len(pk) != 64 {
		t.Fatalf("Private key length: got %d, want 64", len(pk))
	}

	// Deterministic
	pk2 := GeneratePrivateKey("mysecret", "myuser", "mypass")
	if pk != pk2 {
		t.Fatal("Private key not deterministic")
	}
}

func TestPKCS5PadUnpad(t *testing.T) {
	data := []byte("hello")
	padded := pkcs5Pad(data, 16)
	if len(padded)%16 != 0 {
		t.Fatalf("Padded length %d not multiple of 16", len(padded))
	}

	unpadded := pkcs5Unpad(padded)
	if string(unpadded) != "hello" {
		t.Fatalf("Unpad mismatch: got %q, want %q", string(unpadded), "hello")
	}
}

func TestMaskCardNumber(t *testing.T) {
	masked := MaskCardNumber("4622941234563713")
	if masked != "4622XXXXXXXX3713" {
		t.Fatalf("MaskCardNumber: got %q, want %q", masked, "4622XXXXXXXX3713")
	}
}

func TestMaskEmail(t *testing.T) {
	masked := MaskEmail("john@example.com")
	if masked != "j***@example.com" {
		t.Fatalf("MaskEmail: got %q, want %q", masked, "j***@example.com")
	}
}

func TestMaskPhone(t *testing.T) {
	masked := MaskPhone("9876543210")
	if masked != "98******10" {
		t.Fatalf("MaskPhone: got %q, want %q", masked, "98******10")
	}
}
