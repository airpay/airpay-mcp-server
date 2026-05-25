package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/airpay/airpay-mcp-server/internal/security"
)

// testCredentials holds shared test credential constants.
const (
	testUsername     = "testuser"
	testPassword     = "testpass"
	testMerchantID   = "12345"
	testClientID     = "cid"
	testClientSecret = "csec"
	testPrivateKey   = "testprivkey"
)

// newTestEncryption builds an AirpayEncryption using test credentials.
func newTestEncryption() *security.AirpayEncryption {
	return security.NewAirpayEncryption(testUsername, testPassword)
}

// buildEncryptedTokenResponse builds the `{"response":"<encrypted>"}` envelope that
// the OAuth2 endpoint returns. It encrypts the oauth2Response payload so that the
// token manager's Decrypt call succeeds.
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
	outer := map[string]string{"response": encrypted}
	return json.Marshal(outer)
}

// buildEncryptedAPIResponse builds the `{"response":"<encrypted>"}` envelope that
// standard Airpay API endpoints return.
func buildEncryptedAPIResponse(enc *security.AirpayEncryption, resp APIResponse) ([]byte, error) {
	innerJSON, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}
	encrypted, err := enc.Encrypt(string(innerJSON))
	if err != nil {
		return nil, err
	}
	outer := map[string]string{"response": encrypted}
	return json.Marshal(outer)
}

// newTestClient creates an AirpayClient wired to a mock OAuth2 server that
// returns a valid token, and returns both the client and the oauth2 test server
// so callers can close it.
func newTestClient(t *testing.T, enc *security.AirpayEncryption) (*AirpayClient, *httptest.Server) {
	t.Helper()

	tokenBody, err := buildEncryptedTokenResponse(enc, "tok-abc123", 3600)
	if err != nil {
		t.Fatalf("building token response: %v", err)
	}

	oauthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(tokenBody)
	}))

	tm := security.NewOAuth2TokenManager(
		testClientID, testClientSecret, testMerchantID,
		oauthServer.URL,
		enc,
		testPrivateKey,
	)

	c := NewAirpayClient(enc, tm, testPrivateKey, testMerchantID)
	return c, oauthServer
}

// ---------------------------------------------------------------------------
// NewAirpayClient
// ---------------------------------------------------------------------------

func TestNewAirpayClient_ReturnsNonNil(t *testing.T) {
	enc := newTestEncryption()
	tm := security.NewOAuth2TokenManager(
		testClientID, testClientSecret, testMerchantID,
		"http://localhost",
		enc,
		testPrivateKey,
	)

	c := NewAirpayClient(enc, tm, testPrivateKey, testMerchantID)
	if c == nil {
		t.Fatal("expected non-nil AirpayClient")
	}
}

func TestNewAirpayClient_StoresMerchantIDAndPrivateKey(t *testing.T) {
	enc := newTestEncryption()
	tm := security.NewOAuth2TokenManager(
		testClientID, testClientSecret, testMerchantID,
		"http://localhost",
		enc,
		testPrivateKey,
	)

	c := NewAirpayClient(enc, tm, testPrivateKey, testMerchantID)

	if got := c.GetMerchantID(); got != testMerchantID {
		t.Errorf("GetMerchantID() = %q, want %q", got, testMerchantID)
	}
}

// ---------------------------------------------------------------------------
// PostEncrypted
// ---------------------------------------------------------------------------

func TestPostEncrypted_SuccessPath(t *testing.T) {
	enc := newTestEncryption()
	c, oauthSrv := newTestClient(t, enc)
	defer oauthSrv.Close()

	expected := APIResponse{
		StatusCode:   "200",
		ResponseCode: "00",
		Status:       "success",
		Message:      "OK",
		Data:         json.RawMessage(`{"order_id":"ord-1"}`),
	}
	respBody, err := buildEncryptedAPIResponse(enc, expected)
	if err != nil {
		t.Fatalf("building API response: %v", err)
	}

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(respBody)
	}))
	defer apiServer.Close()

	got, err := c.PostEncrypted(apiServer.URL, map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("PostEncrypted returned error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil APIResponse")
	}
	if got.Status != expected.Status {
		t.Errorf("Status = %q, want %q", got.Status, expected.Status)
	}
	if got.Message != expected.Message {
		t.Errorf("Message = %q, want %q", got.Message, expected.Message)
	}
}

func TestPostEncrypted_ServerError_Non200(t *testing.T) {
	enc := newTestEncryption()
	c, oauthSrv := newTestClient(t, enc)
	defer oauthSrv.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer apiServer.Close()

	// The client does not check HTTP status codes; it tries to parse the body.
	// A non-JSON body will cause a parsing error.
	_, err := c.PostEncrypted(apiServer.URL, map[string]string{"key": "value"})
	if err == nil {
		t.Error("expected error for non-JSON error response, got nil")
	}
}

func TestPostEncrypted_PlainJSONResponse(t *testing.T) {
	// Some endpoints return plain JSON without an encrypted wrapper.
	enc := newTestEncryption()
	c, oauthSrv := newTestClient(t, enc)
	defer oauthSrv.Close()

	plain := APIResponse{
		Status:  "success",
		Message: "plain response",
	}
	body, _ := json.Marshal(plain)

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}))
	defer apiServer.Close()

	got, err := c.PostEncrypted(apiServer.URL, map[string]string{"a": "b"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != plain.Status {
		t.Errorf("Status = %q, want %q", got.Status, plain.Status)
	}
}

func TestPostEncrypted_RetryOnDecryptionFailure(t *testing.T) {
	// First call returns garbage that can't be decrypted.
	// Second call (after ForceRefresh) returns a valid encrypted response.
	enc := newTestEncryption()

	callCount := 0

	// Token refresh counter — the test oauth server tracks how many times it's hit.
	tokenRefreshCount := 0
	tokenBody, err := buildEncryptedTokenResponse(enc, "tok-refreshed", 3600)
	if err != nil {
		t.Fatalf("building token body: %v", err)
	}
	oauthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenRefreshCount++
		w.Header().Set("Content-Type", "application/json")
		w.Write(tokenBody)
	}))
	defer oauthServer.Close()

	tm := security.NewOAuth2TokenManager(
		testClientID, testClientSecret, testMerchantID,
		oauthServer.URL,
		enc,
		testPrivateKey,
	)
	c := NewAirpayClient(enc, tm, testPrivateKey, testMerchantID)

	goodResp := APIResponse{Status: "success", Message: "retry worked"}
	goodBody, err := buildEncryptedAPIResponse(enc, goodResp)
	if err != nil {
		t.Fatalf("building good response: %v", err)
	}

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			// Return a malformed "response" field that will fail decryption.
			w.Write([]byte(`{"response":"AAAAAAAAAAAAAAAA" }`))
			return
		}
		w.Write(goodBody)
	}))
	defer apiServer.Close()

	got, err := c.PostEncrypted(apiServer.URL, map[string]string{"x": "y"})
	if err != nil {
		t.Fatalf("PostEncrypted returned error on retry path: %v", err)
	}
	if got.Status != "success" {
		t.Errorf("Status after retry = %q, want %q", got.Status, "success")
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls (initial + retry), got %d", callCount)
	}
}

func TestPostEncrypted_OAuthError_ReturnsError(t *testing.T) {
	enc := newTestEncryption()

	// OAuth2 server returns bad JSON so token fetch fails.
	oauthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not-json`))
	}))
	defer oauthServer.Close()

	tm := security.NewOAuth2TokenManager(
		testClientID, testClientSecret, testMerchantID,
		oauthServer.URL,
		enc,
		testPrivateKey,
	)
	c := NewAirpayClient(enc, tm, testPrivateKey, testMerchantID)

	_, err := c.PostEncrypted("http://unused", map[string]string{})
	if err == nil {
		t.Error("expected error when OAuth2 token fetch fails, got nil")
	}
}

// ---------------------------------------------------------------------------
// PostFormDirect
// ---------------------------------------------------------------------------

func TestPostFormDirect_Success(t *testing.T) {
	enc := newTestEncryption()
	c, oauthSrv := newTestClient(t, enc)
	defer oauthSrv.Close()

	expected := APIResponse{Status: "success", Message: "form ok"}
	body, _ := json.Marshal(expected)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}))
	defer srv.Close()

	got, err := c.PostFormDirect(srv.URL, url.Values{"field": {"val"}})
	if err != nil {
		t.Fatalf("PostFormDirect error: %v", err)
	}
	if got.Status != expected.Status {
		t.Errorf("Status = %q, want %q", got.Status, expected.Status)
	}
}

func TestPostFormDirect_NonJSONResponseFallback(t *testing.T) {
	enc := newTestEncryption()
	c, oauthSrv := newTestClient(t, enc)
	defer oauthSrv.Close()

	rawBody := []byte("raw-non-json-data")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(rawBody)
	}))
	defer srv.Close()

	got, err := c.PostFormDirect(srv.URL, url.Values{"k": {"v"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Falls back to raw APIResponse wrapper.
	if got.Status != "raw" {
		t.Errorf("expected Status=raw, got %q", got.Status)
	}
}

func TestPostFormDirect_NetworkError(t *testing.T) {
	enc := newTestEncryption()
	c, oauthSrv := newTestClient(t, enc)
	defer oauthSrv.Close()

	// Use an address with no server.
	_, err := c.PostFormDirect("http://127.0.0.1:1", url.Values{})
	if err == nil {
		t.Error("expected network error, got nil")
	}
}

// ---------------------------------------------------------------------------
// PostJSONDirect
// ---------------------------------------------------------------------------

func TestPostJSONDirect_Success(t *testing.T) {
	enc := newTestEncryption()
	c, oauthSrv := newTestClient(t, enc)
	defer oauthSrv.Close()

	expected := APIResponse{Status: "success", Message: "json ok"}
	body, _ := json.Marshal(expected)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		customHeader := r.Header.Get("X-Custom")
		if customHeader != "hval" {
			t.Errorf("X-Custom header = %q, want hval", customHeader)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}))
	defer srv.Close()

	payload := []byte(`{"transaction_id":"txn-1"}`)
	got, err := c.PostJSONDirect(srv.URL, payload, map[string]string{"X-Custom": "hval"})
	if err != nil {
		t.Fatalf("PostJSONDirect error: %v", err)
	}
	if got.Status != expected.Status {
		t.Errorf("Status = %q, want %q", got.Status, expected.Status)
	}
}

func TestPostJSONDirect_NonJSONFallback(t *testing.T) {
	enc := newTestEncryption()
	c, oauthSrv := newTestClient(t, enc)
	defer oauthSrv.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("plain text"))
	}))
	defer srv.Close()

	got, err := c.PostJSONDirect(srv.URL, []byte(`{}`), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != "raw" {
		t.Errorf("expected Status=raw, got %q", got.Status)
	}
}

func TestPostJSONDirect_NetworkError(t *testing.T) {
	enc := newTestEncryption()
	c, oauthSrv := newTestClient(t, enc)
	defer oauthSrv.Close()

	_, err := c.PostJSONDirect("http://127.0.0.1:1", []byte(`{}`), nil)
	if err == nil {
		t.Error("expected network error, got nil")
	}
}

// ---------------------------------------------------------------------------
// GetWithAuth
// ---------------------------------------------------------------------------

func TestGetWithAuth_Success(t *testing.T) {
	enc := newTestEncryption()
	c, oauthSrv := newTestClient(t, enc)
	defer oauthSrv.Close()

	expected := APIResponse{Status: "success", Message: "get ok"}
	body, _ := json.Marshal(expected)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer mytoken" {
			t.Errorf("Authorization = %q, want Bearer mytoken", authHeader)
		}
		// Verify query param forwarding.
		if r.URL.Query().Get("ref") != "abc" {
			t.Errorf("query param ref = %q, want abc", r.URL.Query().Get("ref"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}))
	defer srv.Close()

	got, err := c.GetWithAuth(srv.URL, "mytoken", url.Values{"ref": {"abc"}})
	if err != nil {
		t.Fatalf("GetWithAuth error: %v", err)
	}
	if got.Status != expected.Status {
		t.Errorf("Status = %q, want %q", got.Status, expected.Status)
	}
}

func TestGetWithAuth_NoQueryParams(t *testing.T) {
	enc := newTestEncryption()
	c, oauthSrv := newTestClient(t, enc)
	defer oauthSrv.Close()

	expected := APIResponse{Status: "success"}
	body, _ := json.Marshal(expected)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query params, got %q", r.URL.RawQuery)
		}
		w.Write(body)
	}))
	defer srv.Close()

	got, err := c.GetWithAuth(srv.URL, "tok", nil)
	if err != nil {
		t.Fatalf("GetWithAuth error: %v", err)
	}
	if got.Status != "success" {
		t.Errorf("Status = %q, want success", got.Status)
	}
}

func TestGetWithAuth_NonJSONFallback(t *testing.T) {
	enc := newTestEncryption()
	c, oauthSrv := newTestClient(t, enc)
	defer oauthSrv.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not-json"))
	}))
	defer srv.Close()

	got, err := c.GetWithAuth(srv.URL, "tok", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != "raw" {
		t.Errorf("expected Status=raw, got %q", got.Status)
	}
}

func TestGetWithAuth_NetworkError(t *testing.T) {
	enc := newTestEncryption()
	c, oauthSrv := newTestClient(t, enc)
	defer oauthSrv.Close()

	_, err := c.GetWithAuth("http://127.0.0.1:1", "tok", nil)
	if err == nil {
		t.Error("expected network error, got nil")
	}
}

// ---------------------------------------------------------------------------
// PostJSONWithAuth
// ---------------------------------------------------------------------------

func TestPostJSONWithAuth_Success(t *testing.T) {
	enc := newTestEncryption()
	c, oauthSrv := newTestClient(t, enc)
	defer oauthSrv.Close()

	expected := APIResponse{Status: "success", Message: "bearer ok"}
	body, _ := json.Marshal(expected)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer bearer-token" {
			t.Errorf("Authorization = %q, want Bearer bearer-token", authHeader)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}))
	defer srv.Close()

	got, err := c.PostJSONWithAuth(srv.URL, []byte(`{"amount":100}`), "bearer-token")
	if err != nil {
		t.Fatalf("PostJSONWithAuth error: %v", err)
	}
	if got.Status != expected.Status {
		t.Errorf("Status = %q, want %q", got.Status, expected.Status)
	}
}

func TestPostJSONWithAuth_NonJSONFallback(t *testing.T) {
	enc := newTestEncryption()
	c, oauthSrv := newTestClient(t, enc)
	defer oauthSrv.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("created"))
	}))
	defer srv.Close()

	got, err := c.PostJSONWithAuth(srv.URL, []byte(`{}`), "tok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != "raw" {
		t.Errorf("expected Status=raw, got %q", got.Status)
	}
}

func TestPostJSONWithAuth_NetworkError(t *testing.T) {
	enc := newTestEncryption()
	c, oauthSrv := newTestClient(t, enc)
	defer oauthSrv.Close()

	_, err := c.PostJSONWithAuth("http://127.0.0.1:1", []byte(`{}`), "tok")
	if err == nil {
		t.Error("expected network error, got nil")
	}
}

// ---------------------------------------------------------------------------
// appendToken (package-internal helper)
// ---------------------------------------------------------------------------

func TestAppendToken_NoExistingQuery(t *testing.T) {
	got := appendToken("https://example.com/api", "abc123")
	want := "https://example.com/api?token=abc123"
	if got != want {
		t.Errorf("appendToken = %q, want %q", got, want)
	}
}

func TestAppendToken_WithExistingQuery(t *testing.T) {
	got := appendToken("https://example.com/api?x=1", "abc123")
	want := "https://example.com/api?x=1&token=abc123"
	if got != want {
		t.Errorf("appendToken = %q, want %q", got, want)
	}
}
