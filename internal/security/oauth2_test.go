package security

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// buildTokenResponseBody builds `{"response":"<encrypted>"}` for a mock OAuth2 server.
func buildTokenResponseBody(enc *AirpayEncryption, accessToken string, expiresIn int) []byte {
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
	innerJSON, _ := json.Marshal(inner)
	encrypted, _ := enc.Encrypt(string(innerJSON))
	body, _ := json.Marshal(map[string]string{"response": encrypted})
	return body
}

func newTestTokenManager(t *testing.T, handler http.HandlerFunc) (*OAuth2TokenManager, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	enc := NewAirpayEncryption("testuser", "testpass")
	pk := GeneratePrivateKey("sec", "testuser", "testpass")
	tm := NewOAuth2TokenManager("cid", "csec", "12345", srv.URL+"/token", enc, pk)
	return tm, srv
}

// ─────────────────────────────────────────────────────────────────────────────
// NewOAuth2TokenManager
// ─────────────────────────────────────────────────────────────────────────────

func TestNewOAuth2TokenManager_ReturnsNonNil(t *testing.T) {
	enc := NewAirpayEncryption("u", "p")
	tm := NewOAuth2TokenManager("cid", "csec", "mid", "http://localhost/token", enc, "pk")
	if tm == nil {
		t.Fatal("expected non-nil token manager")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// GetToken / refresh
// ─────────────────────────────────────────────────────────────────────────────

func TestGetToken_FetchesOnFirstCall(t *testing.T) {
	enc := NewAirpayEncryption("testuser", "testpass")
	body := buildTokenResponseBody(enc, "tok-first", 3600)

	tm, srv := newTestTokenManager(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	})
	defer srv.Close()

	token, err := tm.GetToken()
	if err != nil {
		t.Fatalf("GetToken failed: %v", err)
	}
	if token != "tok-first" {
		t.Errorf("GetToken = %q, want tok-first", token)
	}
}

func TestGetToken_CachesToken(t *testing.T) {
	enc := NewAirpayEncryption("testuser", "testpass")
	body := buildTokenResponseBody(enc, "tok-cached", 3600)
	calls := 0

	tm, srv := newTestTokenManager(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Write(body)
	})
	defer srv.Close()

	tm.GetToken()
	tm.GetToken()
	tm.GetToken()

	if calls != 1 {
		t.Errorf("expected 1 HTTP call (cached), got %d", calls)
	}
}

func TestGetToken_RefreshesWhenExpired(t *testing.T) {
	enc := NewAirpayEncryption("testuser", "testpass")
	body := buildTokenResponseBody(enc, "tok-fresh", 3600)
	calls := 0

	tm, srv := newTestTokenManager(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Write(body)
	})
	defer srv.Close()

	// Seed an expired token directly
	tm.mu.Lock()
	tm.accessToken = "tok-old"
	tm.expiresAt = time.Now().Add(-1 * time.Second) // already expired
	tm.mu.Unlock()

	token, err := tm.GetToken()
	if err != nil {
		t.Fatalf("GetToken failed: %v", err)
	}
	if token != "tok-fresh" {
		t.Errorf("GetToken = %q, want tok-fresh", token)
	}
	if calls != 1 {
		t.Errorf("expected 1 HTTP call for refresh, got %d", calls)
	}
}

func TestGetToken_RefreshesWhenWithin60Seconds(t *testing.T) {
	enc := NewAirpayEncryption("testuser", "testpass")
	body := buildTokenResponseBody(enc, "tok-new", 3600)

	tm, srv := newTestTokenManager(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	})
	defer srv.Close()

	// Token valid but within 60-second refresh window
	tm.mu.Lock()
	tm.accessToken = "tok-expiring"
	tm.expiresAt = time.Now().Add(30 * time.Second)
	tm.mu.Unlock()

	token, err := tm.GetToken()
	if err != nil {
		t.Fatalf("GetToken failed: %v", err)
	}
	if token != "tok-new" {
		t.Errorf("GetToken = %q, want tok-new (proactive refresh)", token)
	}
}

func TestGetToken_ServerError_ReturnsError(t *testing.T) {
	tm, srv := newTestTokenManager(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	})
	defer srv.Close()

	_, err := tm.GetToken()
	if err == nil {
		t.Fatal("expected error from server failure, got nil")
	}
}

func TestGetToken_MalformedResponse_ReturnsError(t *testing.T) {
	tm, srv := newTestTokenManager(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not-json"))
	})
	defer srv.Close()

	_, err := tm.GetToken()
	if err == nil {
		t.Fatal("expected error for malformed response")
	}
}

func TestGetToken_EmptyResponseField_ReturnsError(t *testing.T) {
	tm, srv := newTestTokenManager(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"response":""}`))
	})
	defer srv.Close()

	_, err := tm.GetToken()
	if err == nil {
		t.Fatal("expected error for empty response field")
	}
	if !strings.Contains(err.Error(), "empty response field") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetToken_FailedStatus_ReturnsError(t *testing.T) {
	enc := NewAirpayEncryption("testuser", "testpass")
	inner := map[string]any{
		"status_code":   "400",
		"response_code": "99",
		"status":        "failure",
		"message":       "invalid credentials",
		"data":          map[string]any{"access_token": "", "expires_in": "0", "scope": nil},
	}
	innerJSON, _ := json.Marshal(inner)
	encrypted, _ := enc.Encrypt(string(innerJSON))
	body, _ := json.Marshal(map[string]string{"response": encrypted})

	tm, srv := newTestTokenManager(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	})
	defer srv.Close()

	_, err := tm.GetToken()
	if err == nil {
		t.Fatal("expected error for failed OAuth2 status")
	}
	if !strings.Contains(err.Error(), "invalid credentials") {
		t.Errorf("error should mention message, got: %v", err)
	}
}

func TestGetToken_EmptyAccessToken_ReturnsError(t *testing.T) {
	enc := NewAirpayEncryption("testuser", "testpass")
	inner := map[string]any{
		"status":  "success",
		"message": "ok",
		"data":    map[string]any{"access_token": "", "expires_in": "300", "scope": nil},
	}
	innerJSON, _ := json.Marshal(inner)
	encrypted, _ := enc.Encrypt(string(innerJSON))
	body, _ := json.Marshal(map[string]string{"response": encrypted})

	tm, srv := newTestTokenManager(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	})
	defer srv.Close()

	_, err := tm.GetToken()
	if err == nil {
		t.Fatal("expected error for empty access token")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// ForceRefresh
// ─────────────────────────────────────────────────────────────────────────────

func TestForceRefresh_ClearsAndRefetches(t *testing.T) {
	enc := NewAirpayEncryption("testuser", "testpass")
	body := buildTokenResponseBody(enc, "tok-forced", 3600)
	calls := 0

	tm, srv := newTestTokenManager(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Write(body)
	})
	defer srv.Close()

	// Set a valid cached token
	tm.mu.Lock()
	tm.accessToken = "tok-cached"
	tm.expiresAt = time.Now().Add(10 * time.Minute)
	tm.mu.Unlock()

	token, err := tm.ForceRefresh()
	if err != nil {
		t.Fatalf("ForceRefresh failed: %v", err)
	}
	if token != "tok-forced" {
		t.Errorf("ForceRefresh = %q, want tok-forced", token)
	}
	if calls != 1 {
		t.Errorf("expected 1 HTTP call, got %d", calls)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Masking (previously uncovered)
// ─────────────────────────────────────────────────────────────────────────────

func TestMaskSensitiveJSON_MasksAccessToken(t *testing.T) {
	input := `{"access_token":"abc123xyz","other":"value"}`
	result := MaskSensitiveJSON(input)
	if strings.Contains(result, "abc123xyz") {
		t.Errorf("access_token should be masked, got: %s", result)
	}
	if !strings.Contains(result, "***MASKED***") {
		t.Errorf("expected ***MASKED***, got: %s", result)
	}
}

func TestMaskSensitiveJSON_MasksMultipleFields(t *testing.T) {
	input := `{"token":"tok1","privatekey":"pk1","encdata":"enc1","checksum":"chk1"}`
	result := MaskSensitiveJSON(input)
	for _, sensitive := range []string{"tok1", "pk1", "enc1", "chk1"} {
		if strings.Contains(result, sensitive) {
			t.Errorf("sensitive value %q should be masked in: %s", sensitive, result)
		}
	}
}

func TestMaskSensitiveJSON_PreservesNonSensitiveFields(t *testing.T) {
	input := `{"status":"success","message":"ok"}`
	result := MaskSensitiveJSON(input)
	if result != input {
		t.Errorf("non-sensitive fields should be unchanged: got %s", result)
	}
}

func TestMaskResponseForLog_MasksBothTokenAndCard(t *testing.T) {
	input := `{"token":"secrettoken","card":"4622941234563713"}`
	result := MaskResponseForLog(input)
	if strings.Contains(result, "secrettoken") {
		t.Errorf("token should be masked")
	}
	if strings.Contains(result, "941234563") {
		t.Errorf("card middle digits should be masked")
	}
}

func TestMaskEmail_NoAtSign_ReturnsUnchanged(t *testing.T) {
	result := MaskEmail("notanemail")
	if result != "notanemail" {
		t.Errorf("invalid email should be returned unchanged, got: %s", result)
	}
}

func TestMaskPhone_ShortPhone_ReturnsUnchanged(t *testing.T) {
	result := MaskPhone("123")
	if result != "123" {
		t.Errorf("short phone should be unchanged, got: %s", result)
	}
}

func TestGenerateChecksum_UsesTodayDate(t *testing.T) {
	params := map[string]string{"amount": "100.00", "orderid": "ORD1"}
	// GenerateChecksum uses time.Now() internally — just verify it returns 64-char SHA-256
	checksum := GenerateChecksum(params)
	if len(checksum) != 64 {
		t.Errorf("GenerateChecksum length: got %d, want 64", len(checksum))
	}
}
