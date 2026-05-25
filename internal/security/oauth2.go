package security

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// OAuth2TokenManager manages the lifecycle of Airpay OAuth2 access tokens.
// It handles automatic refresh 60 seconds before expiry.
type OAuth2TokenManager struct {
	mu           sync.RWMutex
	accessToken  string
	expiresAt    time.Time
	clientID     string
	clientSecret string
	merchantID   string
	tokenURL     string
	encryption   *AirpayEncryption
	privateKey   string
	httpClient   *http.Client
}

// oauth2Response represents the decrypted OAuth2 response from Airpay.
type oauth2Response struct {
	StatusCode   string `json:"status_code"`
	ResponseCode string `json:"response_code"`
	Status       string `json:"status"`
	Message      string `json:"message"`
	Data         struct {
		AccessToken string      `json:"access_token"`
		ExpiresIn   json.Number `json:"expires_in"`
		Scope       *string     `json:"scope"`
	} `json:"data"`
}

// NewOAuth2TokenManager creates a new token manager.
func NewOAuth2TokenManager(
	clientID, clientSecret, merchantID, tokenURL string,
	encryption *AirpayEncryption,
	privateKey string,
) *OAuth2TokenManager {
	return &OAuth2TokenManager{
		clientID:     clientID,
		clientSecret: clientSecret,
		merchantID:   merchantID,
		tokenURL:     tokenURL,
		encryption:   encryption,
		privateKey:   privateKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetToken returns a valid access token, refreshing if necessary.
// It refreshes proactively 60 seconds before expiry.
func (tm *OAuth2TokenManager) GetToken() (string, error) {
	tm.mu.RLock()
	if tm.accessToken != "" && time.Now().Before(tm.expiresAt.Add(-60*time.Second)) {
		token := tm.accessToken
		tm.mu.RUnlock()
		return token, nil
	}
	tm.mu.RUnlock()

	return tm.refresh()
}

// refresh acquires a new access token from the Airpay OAuth2 endpoint.
func (tm *OAuth2TokenManager) refresh() (string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Double-check after acquiring write lock
	if tm.accessToken != "" && time.Now().Before(tm.expiresAt.Add(-60*time.Second)) {
		return tm.accessToken, nil
	}

	log.Println("[oauth2] Refreshing access token...")

	// Build the OAuth2 request payload
	data := map[string]string{
		"client_id":     tm.clientID,
		"client_secret": tm.clientSecret,
		"merchant_id":   tm.merchantID,
		"grant_type":    "client_credentials",
	}

	// Encrypt the payload
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshaling OAuth2 request: %w", err)
	}

	encdata, err := tm.encryption.Encrypt(string(jsonData))
	if err != nil {
		return "", fmt.Errorf("encrypting OAuth2 request: %w", err)
	}

	// Generate checksum
	checksum := GenerateChecksum(data)

	// Build form data
	formData := url.Values{
		"merchant_id": {tm.merchantID},
		"encdata":     {encdata},
		"checksum":    {checksum},
	}

	// Send the request
	resp, err := tm.httpClient.Post(tm.tokenURL, "application/x-www-form-urlencoded", strings.NewReader(formData.Encode()))
	if err != nil {
		return "", fmt.Errorf("sending OAuth2 request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading OAuth2 response: %w", err)
	}

	// Parse the outer response to extract encrypted "response" field
	var outerResp struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(body, &outerResp); err != nil {
		return "", fmt.Errorf("parsing OAuth2 outer response: %w (body: %s)", err, string(body))
	}

	if outerResp.Response == "" {
		return "", fmt.Errorf("empty response field in OAuth2 response: %s", string(body))
	}

	// Decrypt the response
	decrypted, err := tm.encryption.Decrypt(outerResp.Response)
	if err != nil {
		return "", fmt.Errorf("decrypting OAuth2 response: %w", err)
	}

	// Parse the decrypted response
	var tokenResp oauth2Response
	if err := json.Unmarshal(decrypted, &tokenResp); err != nil {
		return "", fmt.Errorf("parsing decrypted OAuth2 response: %w", err)
	}

	if tokenResp.Status != "success" {
		return "", fmt.Errorf("OAuth2 token request failed: %s (code: %s)", tokenResp.Message, tokenResp.ResponseCode)
	}

	if tokenResp.Data.AccessToken == "" {
		return "", fmt.Errorf("empty access token in OAuth2 response")
	}

	// Parse expiry duration
	expiresIn, err := tokenResp.Data.ExpiresIn.Int64()
	if err != nil {
		expiresIn = 300 // Default 5 minutes
	}

	tm.accessToken = tokenResp.Data.AccessToken
	tm.expiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)

	log.Printf("[oauth2] Token refreshed, expires in %d seconds", expiresIn)
	return tm.accessToken, nil
}

// ForceRefresh forces a token refresh regardless of current token state.
func (tm *OAuth2TokenManager) ForceRefresh() (string, error) {
	tm.mu.Lock()
	tm.accessToken = ""
	tm.expiresAt = time.Time{}
	tm.mu.Unlock()

	return tm.refresh()
}
