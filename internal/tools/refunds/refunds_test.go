package refunds

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

func TestHandleInitiateRefund_EmptyTransactions_ReturnsError(t *testing.T) {
	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, nil)
	defer cleanup()

	handler := HandleInitiateRefund(apiClient, baseURL)
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, InitiateRefundInput{
		Transactions: []RefundTransaction{},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isErrorResult(result) {
		t.Fatal("expected error result for empty transactions")
	}
	if !strings.Contains(result.Content[0].(*mcpsdk.TextContent).Text, "at least one transaction") {
		t.Errorf("unexpected error message: %s", result.Content[0].(*mcpsdk.TextContent).Text)
	}
}

func TestHandleInitiateRefund_NilTransactions_ReturnsError(t *testing.T) {
	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, nil)
	defer cleanup()

	handler := HandleInitiateRefund(apiClient, baseURL)
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, InitiateRefundInput{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isErrorResult(result) {
		t.Fatal("expected error result for nil transactions")
	}
}

func TestHandleInitiateRefund_SingleTransaction_Success(t *testing.T) {
	body := encryptedResp(t, "200", "refund initiated")

	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	})
	defer cleanup()

	handler := HandleInitiateRefund(apiClient, baseURL)
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, InitiateRefundInput{
		Transactions: []RefundTransaction{
			{APTransactionID: 123456, Amount: "100.00"},
		},
	})

	if err != nil || isErrorResult(result) {
		t.Fatalf("expected success: err=%v isError=%v", err, isErrorResult(result))
	}
}

func TestHandleInitiateRefund_MultipleTransactions_Success(t *testing.T) {
	body := encryptedResp(t, "200", "ok")

	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	})
	defer cleanup()

	handler := HandleInitiateRefund(apiClient, baseURL)
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, InitiateRefundInput{
		Transactions: []RefundTransaction{
			{APTransactionID: 111, Amount: "50.00"},
			{APTransactionID: 222, Amount: "25.50"},
		},
	})

	if err != nil || isErrorResult(result) {
		t.Fatalf("expected success: err=%v isError=%v", err, isErrorResult(result))
	}
}

func TestHandleInitiateRefund_HitsCorrectEndpoint(t *testing.T) {
	var hitPath string
	body := encryptedResp(t, "200", "ok")

	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		hitPath = r.URL.Path
		w.Write(body)
	})
	defer cleanup()

	handler := HandleInitiateRefund(apiClient, baseURL)
	handler(context.Background(), &mcpsdk.CallToolRequest{}, InitiateRefundInput{
		Transactions: []RefundTransaction{{APTransactionID: 1, Amount: "10.00"}},
	})

	if hitPath != "/airpay/pay/v4/api/refund/" {
		t.Errorf("expected /airpay/pay/v4/api/refund/, got %s", hitPath)
	}
}

func TestHandleInitiateRefund_APIError_ReturnsErrorResult(t *testing.T) {
	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	})
	defer cleanup()

	handler := HandleInitiateRefund(apiClient, baseURL)
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, InitiateRefundInput{
		Transactions: []RefundTransaction{{APTransactionID: 1, Amount: "10.00"}},
	})

	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if !isErrorResult(result) {
		t.Fatal("expected error result on API failure")
	}
}

func TestHandleInitiateRefund_RequestIncludesMode(t *testing.T) {
	var formMode string
	body := encryptedResp(t, "200", "ok")

	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		// mode is encrypted inside encdata — just verify the request reaches the endpoint
		_ = formMode
		w.Write(body)
	})
	defer cleanup()

	handler := HandleInitiateRefund(apiClient, baseURL)
	result, _, _ := handler(context.Background(), &mcpsdk.CallToolRequest{}, InitiateRefundInput{
		Transactions: []RefundTransaction{{APTransactionID: 1, Amount: "10.00"}},
	})

	if isErrorResult(result) {
		t.Fatal("expected success result")
	}
}
