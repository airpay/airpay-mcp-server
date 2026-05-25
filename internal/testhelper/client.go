// Package testhelper provides shared test utilities for tool handler tests.
package testhelper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/airpay/airpay-mcp-server/internal/client"
	"github.com/airpay/airpay-mcp-server/internal/security"
)

const (
	TestUsername     = "testuser"
	TestPassword     = "testpass"
	TestMerchantID   = "12345"
	TestClientID     = "cid"
	TestClientSecret = "csec"
	TestPrivateKey   = "testprivkey"
)

// NewTestEncryption returns an AirpayEncryption with test credentials.
func NewTestEncryption() *security.AirpayEncryption {
	return security.NewAirpayEncryption(TestUsername, TestPassword)
}

// BuildEncryptedAPIResponse returns a `{"response":"<encrypted>"}` JSON body.
func BuildEncryptedAPIResponse(enc *security.AirpayEncryption, resp client.APIResponse) ([]byte, error) {
	inner, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}
	encrypted, err := enc.Encrypt(string(inner))
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]string{"response": encrypted})
}

// buildEncryptedTokenResponse returns the encrypted OAuth2 token envelope.
func buildEncryptedTokenResponse(enc *security.AirpayEncryption, accessToken string, expiresIn int) ([]byte, error) {
	inner := map[string]any{
		"status_code":   "200",
		"response_code": "00",
		"status":        "success",
		"message":       "Token issued",
		"data": map[string]any{
			"access_token": accessToken,
			"expires_in":   fmt.Sprintf("%d", expiresIn),
			"scope":        nil,
		},
	}
	innerJSON, err := json.Marshal(inner)
	if err != nil {
		return nil, err
	}
	encrypted, err := enc.Encrypt(string(innerJSON))
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]string{"response": encrypted})
}

// NewTestClientWithServer creates an AirpayClient wired to a combined mock
// server. The mock server handles:
//   - POST /oauth2  → returns a valid encrypted token
//   - all other paths → calls apiHandler
//
// Returns the client, the mock server base URL, and a cleanup func.
func NewTestClientWithServer(
	t *testing.T,
	apiHandler http.HandlerFunc,
) (*client.AirpayClient, string, func()) {
	t.Helper()

	enc := NewTestEncryption()
	privateKey := security.GeneratePrivateKey(TestUsername, TestPassword, TestMerchantID)

	tokenBody, err := buildEncryptedTokenResponse(enc, "tok-test", 3600)
	if err != nil {
		t.Fatalf("building token response: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth2" {
			w.Header().Set("Content-Type", "application/json")
			w.Write(tokenBody)
			return
		}
		if apiHandler != nil {
			apiHandler(w, r)
		} else {
			http.NotFound(w, r)
		}
	}))

	tokenMgr := security.NewOAuth2TokenManager(
		TestClientID, TestClientSecret, TestMerchantID,
		srv.URL+"/oauth2",
		enc,
		privateKey,
	)

	apiClient := client.NewAirpayClient(enc, tokenMgr, privateKey, TestMerchantID)
	return apiClient, srv.URL, srv.Close
}
