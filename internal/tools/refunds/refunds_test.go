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

const testSecret = "testsecret"

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

	handler := HandleInitiateRefund(apiClient, baseURL, testSecret)
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

	handler := HandleInitiateRefund(apiClient, baseURL, testSecret)
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, InitiateRefundInput{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isErrorResult(result) {
		t.Fatal("expected error result for nil transactions")
	}
}

func TestHandleInitiateRefund_Phase1_ReturnsPreviewWithToken(t *testing.T) {
	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, nil)
	defer cleanup()

	handler := HandleInitiateRefund(apiClient, baseURL, testSecret)
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, InitiateRefundInput{
		Transactions: []RefundTransaction{
			{APTransactionID: 123456, Amount: "100.00"},
		},
		Confirmed: false,
	})

	if err != nil || isErrorResult(result) {
		t.Fatalf("expected Phase 1 success: err=%v isError=%v", err, isErrorResult(result))
	}
	text := result.Content[0].(*mcpsdk.TextContent).Text
	if !strings.Contains(text, "confirmation_token") {
		t.Errorf("Phase 1 response missing confirmation_token: %s", text)
	}
	if !strings.Contains(text, "phase") {
		t.Errorf("Phase 1 response missing phase field: %s", text)
	}
}

func TestHandleInitiateRefund_Phase2_ValidToken_HitsAPI(t *testing.T) {
	body := encryptedResp(t, "200", "refund initiated")
	var hitPath string

	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		hitPath = r.URL.Path
		w.Write(body)
	})
	defer cleanup()

	txns := []RefundTransaction{{APTransactionID: 123456, Amount: "100.00"}}
	token, err := signTransactions(txns, testSecret)
	if err != nil {
		t.Fatalf("signing transactions: %v", err)
	}

	handler := HandleInitiateRefund(apiClient, baseURL, testSecret)
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, InitiateRefundInput{
		Transactions:      txns,
		Confirmed:         true,
		ConfirmationToken: token,
	})

	if err != nil || isErrorResult(result) {
		t.Fatalf("expected Phase 2 success: err=%v isError=%v", err, isErrorResult(result))
	}
	if hitPath != "/airpay/pay/v4/api/refund/" {
		t.Errorf("expected /airpay/pay/v4/api/refund/, got %s", hitPath)
	}
}

func TestHandleInitiateRefund_Phase2_InvalidToken_ReturnsError(t *testing.T) {
	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, nil)
	defer cleanup()

	handler := HandleInitiateRefund(apiClient, baseURL, testSecret)
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, InitiateRefundInput{
		Transactions:      []RefundTransaction{{APTransactionID: 1, Amount: "10.00"}},
		Confirmed:         true,
		ConfirmationToken: "invalidsignature==",
	})

	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if !isErrorResult(result) {
		t.Fatal("expected error result for invalid token")
	}
	if !strings.Contains(result.Content[0].(*mcpsdk.TextContent).Text, "mismatch") {
		t.Errorf("unexpected error message: %s", result.Content[0].(*mcpsdk.TextContent).Text)
	}
}

func TestHandleInitiateRefund_Phase2_MissingToken_ReturnsError(t *testing.T) {
	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, nil)
	defer cleanup()

	handler := HandleInitiateRefund(apiClient, baseURL, testSecret)
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, InitiateRefundInput{
		Transactions: []RefundTransaction{{APTransactionID: 1, Amount: "10.00"}},
		Confirmed:    true,
	})

	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if !isErrorResult(result) {
		t.Fatal("expected error result for missing token")
	}
}

func TestHandleInitiateRefund_Phase2_APIError_ReturnsErrorResult(t *testing.T) {
	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	})
	defer cleanup()

	txns := []RefundTransaction{{APTransactionID: 1, Amount: "10.00"}}
	token, err := signTransactions(txns, testSecret)
	if err != nil {
		t.Fatalf("signing transactions: %v", err)
	}

	handler := HandleInitiateRefund(apiClient, baseURL, testSecret)
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, InitiateRefundInput{
		Transactions:      txns,
		Confirmed:         true,
		ConfirmationToken: token,
	})

	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if !isErrorResult(result) {
		t.Fatal("expected error result on API failure")
	}
}
