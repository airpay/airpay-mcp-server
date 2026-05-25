package subscriptions

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

func TestHandleCheckSubscriptionStatus_BothEmpty_ReturnsError(t *testing.T) {
	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, nil)
	defer cleanup()

	handler := HandleCheckSubscriptionStatus(apiClient, baseURL)
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, CheckSubscriptionStatusInput{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isErrorResult(result) {
		t.Fatal("expected error result when both identifiers are empty")
	}
	if !strings.Contains(result.Content[0].(*mcpsdk.TextContent).Text, "at least one") {
		t.Errorf("error should mention 'at least one', got: %s", result.Content[0].(*mcpsdk.TextContent).Text)
	}
}

func TestHandleCheckSubscriptionStatus_WithSubscriptionID_Success(t *testing.T) {
	body := encryptedResp(t, "200", "SUBSCRIBED")

	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	})
	defer cleanup()

	handler := HandleCheckSubscriptionStatus(apiClient, baseURL)
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, CheckSubscriptionStatusInput{
		SubscriptionID: "10234982",
	})

	if err != nil || isErrorResult(result) {
		t.Fatalf("expected success: err=%v isError=%v", err, isErrorResult(result))
	}
}

func TestHandleCheckSubscriptionStatus_WithOrderID_Success(t *testing.T) {
	body := encryptedResp(t, "200", "SUBSCRIBED")

	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	})
	defer cleanup()

	handler := HandleCheckSubscriptionStatus(apiClient, baseURL)
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, CheckSubscriptionStatusInput{
		OrderID: "1012",
	})

	if err != nil || isErrorResult(result) {
		t.Fatalf("expected success: err=%v isError=%v", err, isErrorResult(result))
	}
}

func TestHandleCheckSubscriptionStatus_WithBothIDs_Success(t *testing.T) {
	body := encryptedResp(t, "200", "ok")

	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	})
	defer cleanup()

	handler := HandleCheckSubscriptionStatus(apiClient, baseURL)
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, CheckSubscriptionStatusInput{
		SubscriptionID: "10234982",
		OrderID:        "1012",
	})

	if err != nil || isErrorResult(result) {
		t.Fatalf("expected success: err=%v isError=%v", err, isErrorResult(result))
	}
}

func TestHandleCheckSubscriptionStatus_WithPageNo_Success(t *testing.T) {
	body := encryptedResp(t, "200", "ok")

	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	})
	defer cleanup()

	handler := HandleCheckSubscriptionStatus(apiClient, baseURL)
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, CheckSubscriptionStatusInput{
		SubscriptionID: "10234982",
		PageNo:         "2",
	})

	if err != nil || isErrorResult(result) {
		t.Fatalf("expected success: err=%v isError=%v", err, isErrorResult(result))
	}
}

func TestHandleCheckSubscriptionStatus_HitsCorrectEndpoint(t *testing.T) {
	var hitPath string
	body := encryptedResp(t, "200", "ok")

	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		hitPath = r.URL.Path
		w.Write(body)
	})
	defer cleanup()

	handler := HandleCheckSubscriptionStatus(apiClient, baseURL)
	handler(context.Background(), &mcpsdk.CallToolRequest{}, CheckSubscriptionStatusInput{
		SubscriptionID: "123",
	})

	if hitPath != "/airpay/pay/v4/api/subscription/status" {
		t.Errorf("expected /airpay/pay/v4/api/subscription/status, got %s", hitPath)
	}
}

func TestHandleCheckSubscriptionStatus_APIError_ReturnsErrorResult(t *testing.T) {
	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	})
	defer cleanup()

	handler := HandleCheckSubscriptionStatus(apiClient, baseURL)
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, CheckSubscriptionStatusInput{
		SubscriptionID: "123",
	})

	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if !isErrorResult(result) {
		t.Fatal("expected error result on API failure")
	}
}
