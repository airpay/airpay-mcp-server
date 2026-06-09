package qr_upi

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/airpay/airpay-mcp-server/internal/client"
	"github.com/airpay/airpay-mcp-server/internal/testhelper"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func encryptedResp(t *testing.T, status, message string) []byte {
	t.Helper()
	enc := testhelper.NewTestEncryption()
	body, err := testhelper.BuildEncryptedAPIResponse(enc, client.APIResponse{
		Status:  status,
		Message: message,
	})
	if err != nil {
		t.Fatalf("building encrypted response: %v", err)
	}
	return body
}

func isErrorResult(r *mcpsdk.CallToolResult) bool {
	return r != nil && r.IsError
}

// ─────────────────────────────────────────────────────────────────────────────
// HandleValidateVPA
// ─────────────────────────────────────────────────────────────────────────────

func TestHandleValidateVPA_EmptyVPA_ReturnsError(t *testing.T) {
	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, nil)
	defer cleanup()

	handler := HandleValidateVPA(apiClient, baseURL)
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, ValidateVPAInput{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isErrorResult(result) {
		t.Fatal("expected error result for empty customer_vpa")
	}
	if !strings.Contains(result.Content[0].(*mcpsdk.TextContent).Text, "customer_vpa") {
		t.Errorf("error should mention customer_vpa, got: %s", result.Content[0].(*mcpsdk.TextContent).Text)
	}
}

func TestHandleValidateVPA_ValidVPA_Success(t *testing.T) {
	body := encryptedResp(t, "200", "VPA valid")

	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	})
	defer cleanup()

	handler := HandleValidateVPA(apiClient, baseURL)
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, ValidateVPAInput{
		CustomerVPA: "user@paytm",
	})

	if err != nil || isErrorResult(result) {
		t.Fatalf("expected success: err=%v isError=%v", err, isErrorResult(result))
	}
}

func TestHandleValidateVPA_HitsCorrectEndpoint(t *testing.T) {
	var hitPath string
	body := encryptedResp(t, "200", "ok")

	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		hitPath = r.URL.Path
		w.Write(body)
	})
	defer cleanup()

	handler := HandleValidateVPA(apiClient, baseURL)
	handler(context.Background(), &mcpsdk.CallToolRequest{}, ValidateVPAInput{CustomerVPA: "user@upi"})

	if hitPath != "/airpay/pay/v4/api/vpavalidate/" {
		t.Errorf("expected /airpay/pay/v4/api/vpavalidate/, got %s", hitPath)
	}
}

func TestHandleValidateVPA_APIError_ReturnsErrorResult(t *testing.T) {
	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
	})
	defer cleanup()

	handler := HandleValidateVPA(apiClient, baseURL)
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, ValidateVPAInput{CustomerVPA: "user@upi"})

	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if !isErrorResult(result) {
		t.Fatal("expected error result on API failure")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// HandleGenerateQR
// ─────────────────────────────────────────────────────────────────────────────

func TestHandleGenerateQR_MissingRequiredFields_ReturnsError(t *testing.T) {
	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, nil)
	defer cleanup()

	handler := HandleGenerateQR(apiClient, baseURL, "https://merchant.example.com", "", "", 86400)

	cases := []struct {
		name  string
		input GenerateQRInput
	}{
		{"missing orderid", GenerateQRInput{Amount: "100.00", BuyerEmail: "a@b.com", BuyerPhone: "9876543210"}},
		{"missing amount", GenerateQRInput{OrderID: "ORD1", BuyerEmail: "a@b.com", BuyerPhone: "9876543210"}},
		{"missing buyer_email", GenerateQRInput{OrderID: "ORD1", Amount: "100.00", BuyerPhone: "9876543210"}},
		{"missing buyer_phone", GenerateQRInput{OrderID: "ORD1", Amount: "100.00", BuyerEmail: "a@b.com"}},
		{"all missing", GenerateQRInput{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !isErrorResult(result) {
				t.Fatal("expected error result for missing required fields")
			}
		})
	}
}

func TestHandleGenerateQR_AllRequiredFields_Success(t *testing.T) {
	body := encryptedResp(t, "200", "QR generated")

	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	})
	defer cleanup()

	handler := HandleGenerateQR(apiClient, baseURL, "https://merchant.example.com", "", "", 86400)
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, GenerateQRInput{
		OrderID:    "ORD123",
		Amount:     "250.00",
		BuyerEmail: "buyer@example.com",
		BuyerPhone: "9876543210",
	})

	if err != nil || isErrorResult(result) {
		t.Fatalf("expected success: err=%v isError=%v", err, isErrorResult(result))
	}
}

func TestHandleGenerateQR_DefaultConsentIsY(t *testing.T) {
	body := encryptedResp(t, "200", "ok")

	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	})
	defer cleanup()

	handler := HandleGenerateQR(apiClient, baseURL, "https://merchant.example.com", "", "", 86400)
	// No customer_consent provided — should default to "Y"
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, GenerateQRInput{
		OrderID:    "ORD1",
		Amount:     "10.00",
		BuyerEmail: "a@b.com",
		BuyerPhone: "9876543210",
	})

	if err != nil || isErrorResult(result) {
		t.Fatalf("expected success: err=%v isError=%v", err, isErrorResult(result))
	}
}

func TestHandleGenerateQR_ExplicitConsentN(t *testing.T) {
	body := encryptedResp(t, "200", "ok")

	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	})
	defer cleanup()

	handler := HandleGenerateQR(apiClient, baseURL, "https://merchant.example.com", "", "", 86400)
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, GenerateQRInput{
		OrderID:         "ORD2",
		Amount:          "10.00",
		BuyerEmail:      "a@b.com",
		BuyerPhone:      "9876543210",
		CustomerConsent: "N",
	})

	if err != nil || isErrorResult(result) {
		t.Fatalf("expected success: err=%v isError=%v", err, isErrorResult(result))
	}
}

func TestHandleGenerateQR_HitsCorrectEndpoint(t *testing.T) {
	var hitPath string
	body := encryptedResp(t, "200", "ok")

	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		hitPath = r.URL.Path
		w.Write(body)
	})
	defer cleanup()

	handler := HandleGenerateQR(apiClient, baseURL, "https://merchant.example.com", "", "", 86400)
	handler(context.Background(), &mcpsdk.CallToolRequest{}, GenerateQRInput{
		OrderID:    "ORD1",
		Amount:     "10.00",
		BuyerEmail: "a@b.com",
		BuyerPhone: "9876543210",
	})

	if hitPath != "/airpay/pay/v4/api/generateorder/" {
		t.Errorf("expected /airpay/pay/v4/api/generateorder/, got %s", hitPath)
	}
}

func TestHandleGenerateQR_OptionalFields_DoNotBreak(t *testing.T) {
	body := encryptedResp(t, "200", "ok")

	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	})
	defer cleanup()

	handler := HandleGenerateQR(apiClient, baseURL, "https://merchant.example.com", "", "", 86400)
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, GenerateQRInput{
		OrderID:    "ORD3",
		Amount:     "10.00",
		BuyerEmail: "a@b.com",
		BuyerPhone: "9876543210",
		TerminalID: "T001",
		CustomVar:  "ref=abc123",
	})

	if err != nil || isErrorResult(result) {
		t.Fatalf("expected success with optional fields: err=%v isError=%v", err, isErrorResult(result))
	}
}

func TestHandleGenerateQR_APIError_ReturnsErrorResult(t *testing.T) {
	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	})
	defer cleanup()

	handler := HandleGenerateQR(apiClient, baseURL, "https://merchant.example.com", "", "", 86400)
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, GenerateQRInput{
		OrderID:    "ORD1",
		Amount:     "10.00",
		BuyerEmail: "a@b.com",
		BuyerPhone: "9876543210",
	})

	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if !isErrorResult(result) {
		t.Fatal("expected error result on API failure")
	}
}
