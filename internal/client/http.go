package client

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/airpay/airpay-mcp-server/internal/security"
)

// AirpayClient is the HTTP client for communicating with Airpay APIs.
// It transparently handles encryption, checksum, private key, and OAuth2 tokens.
type AirpayClient struct {
	httpClient   *http.Client
	encryption   *security.AirpayEncryption
	tokenManager *security.OAuth2TokenManager
	privateKey   string
	merchantID   string
}

// APIResponse represents the standard decrypted Airpay API response.
type APIResponse struct {
	StatusCode   string          `json:"status_code"`
	ResponseCode string          `json:"response_code"`
	Status       string          `json:"status"`
	Message      string          `json:"message"`
	Data         json.RawMessage `json:"data"`
}

// NewAirpayClient creates a new Airpay API client.
func NewAirpayClient(
	encryption *security.AirpayEncryption,
	tokenManager *security.OAuth2TokenManager,
	privateKey string,
	merchantID string,
) *AirpayClient {
	return &AirpayClient{
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		encryption:   encryption,
		tokenManager: tokenManager,
		privateKey:   privateKey,
		merchantID:   merchantID,
	}
}

// PostEncrypted sends an encrypted POST request to the Airpay API.
// It handles: JSON payload encryption, checksum generation, private key attachment,
// OAuth2 token acquisition, and response decryption.
func (c *AirpayClient) PostEncrypted(endpoint string, params map[string]string) (*APIResponse, error) {
	return c.postEncryptedWithRetry(endpoint, params, true)
}

func (c *AirpayClient) postEncryptedWithRetry(endpoint string, params map[string]string, canRetry bool) (*APIResponse, error) {
	// Get OAuth2 token
	token, err := c.tokenManager.GetToken()
	if err != nil {
		return nil, fmt.Errorf("getting OAuth2 token: %w", err)
	}

	// Encrypt the JSON payload
	jsonData, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("marshaling payload: %w", err)
	}

	encdata, err := c.encryption.Encrypt(string(jsonData))
	if err != nil {
		return nil, fmt.Errorf("encrypting payload: %w", err)
	}

	// Generate checksum from the original params
	checksum := security.GenerateChecksum(params)

	// Append token to endpoint URL
	reqURL := appendToken(endpoint, token)

	// Build form data
	formData := url.Values{
		"merchant_id": {c.merchantID},
		"encdata":     {encdata},
		"checksum":    {checksum},
		"privatekey":  {c.privateKey},
	}

	log.Printf("[client] POST %s", endpoint)

	resp, err := c.httpClient.Post(reqURL, "application/x-www-form-urlencoded", strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("sending request to %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	// Parse the outer response which contains the encrypted "response" field
	var outerResp struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(body, &outerResp); err != nil {
		// Some endpoints return plain JSON without encrypted wrapper
		var apiResp APIResponse
		if err2 := json.Unmarshal(body, &apiResp); err2 == nil {
			return &apiResp, nil
		}
		return nil, fmt.Errorf("parsing response: %w (body: %s)", err, string(body))
	}

	if outerResp.Response == "" {
		// Try parsing as plain APIResponse
		var apiResp APIResponse
		if err := json.Unmarshal(body, &apiResp); err == nil && apiResp.Status != "" {
			return &apiResp, nil
		}
		return nil, fmt.Errorf("empty response field: %s", string(body))
	}

	// Decrypt the response
	decrypted, err := c.encryption.Decrypt(outerResp.Response)
	if err != nil {
		if canRetry {
			// Token may have expired, force refresh and retry once
			log.Println("[client] Decryption failed, forcing token refresh and retrying...")
			if _, refreshErr := c.tokenManager.ForceRefresh(); refreshErr == nil {
				return c.postEncryptedWithRetry(endpoint, params, false)
			}
		}
		return nil, fmt.Errorf("decrypting response: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(decrypted, &apiResp); err != nil {
		return nil, fmt.Errorf("parsing decrypted response: %w (raw: %s)", err, string(decrypted))
	}

	return &apiResp, nil
}

// PostFormDirect sends a direct (non-encrypted) form POST to the Airpay API.
// Used for endpoints like POS that don't use the encryption pipeline.
func (c *AirpayClient) PostFormDirect(endpoint string, formData url.Values) (*APIResponse, error) {
	log.Printf("[client] POST (direct) %s", endpoint)

	resp, err := c.httpClient.Post(endpoint, "application/x-www-form-urlencoded", strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("sending request to %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		// Some POS endpoints return non-standard JSON
		return &APIResponse{
			StatusCode: fmt.Sprintf("%d", resp.StatusCode),
			Status:     "raw",
			Data:       json.RawMessage(body),
		}, nil
	}

	return &apiResp, nil
}

// PostJSONDirect sends a direct JSON POST (used for split settlement with API-Key header).
func (c *AirpayClient) PostJSONDirect(endpoint string, jsonBody []byte, headers map[string]string) (*APIResponse, error) {
	log.Printf("[client] POST (json) %s", endpoint)

	req, err := http.NewRequest("POST", endpoint, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return &APIResponse{
			StatusCode: fmt.Sprintf("%d", resp.StatusCode),
			Status:     "raw",
			Data:       json.RawMessage(body),
		}, nil
	}

	return &apiResp, nil
}

// GetWithAuth sends an authenticated GET request using a Bearer token (used for payout APIs).
func (c *AirpayClient) GetWithAuth(endpoint string, authToken string, queryParams url.Values) (*APIResponse, error) {
	reqURL := endpoint
	if len(queryParams) > 0 {
		reqURL = endpoint + "?" + queryParams.Encode()
	}
	log.Printf("[client] GET %s", endpoint)

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return &APIResponse{
			StatusCode: fmt.Sprintf("%d", resp.StatusCode),
			Status:     "raw",
			Data:       json.RawMessage(body),
		}, nil
	}
	return &apiResp, nil
}

// PostJSONWithAuth sends an authenticated JSON POST using a Bearer token (used for payout APIs).
func (c *AirpayClient) PostJSONWithAuth(endpoint string, jsonBody []byte, authToken string) (*APIResponse, error) {
	log.Printf("[client] POST (bearer) %s", endpoint)

	req, err := http.NewRequest("POST", endpoint, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return &APIResponse{
			StatusCode: fmt.Sprintf("%d", resp.StatusCode),
			Status:     "raw",
			Data:       json.RawMessage(body),
		}, nil
	}
	return &apiResp, nil
}

// GetMerchantID returns the configured merchant ID.
func (c *AirpayClient) GetMerchantID() string {
	return c.merchantID
}

// appendToken appends the OAuth2 token to the URL as a query parameter.
func appendToken(endpoint, token string) string {
	if strings.Contains(endpoint, "?") {
		return endpoint + "&token=" + token
	}
	return endpoint + "?token=" + token
}
