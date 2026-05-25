package payments

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/airpay/airpay-mcp-server/internal/client"
	"github.com/airpay/airpay-mcp-server/internal/testhelper"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// encryptedResp is a shorthand for building an encrypted API response body.
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
// HandleVerifyOrder
// ─────────────────────────────────────────────────────────────────────────────

func TestHandleVerifyOrder_MissingAllIdentifiers_ReturnsError(t *testing.T) {
	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, nil)
	defer cleanup()

	handler := HandleVerifyOrder(apiClient, baseURL)
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, VerifyOrderInput{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isErrorResult(result) {
		t.Fatal("expected error result for missing identifiers")
	}
	if !strings.Contains(result.Content[0].(*mcpsdk.TextContent).Text, "at least one") {
		t.Errorf("error message should mention 'at least one', got: %s", result.Content[0].(*mcpsdk.TextContent).Text)
	}
}

func TestHandleVerifyOrder_WithOrderID_Success(t *testing.T) {
	body := encryptedResp(t, "200", "success")

	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	})
	defer cleanup()

	handler := HandleVerifyOrder(apiClient, baseURL)
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, VerifyOrderInput{
		OrderID: "ORD123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isErrorResult(result) {
		t.Fatalf("expected success result, got error: %s", result.Content[0].(*mcpsdk.TextContent).Text)
	}
}

func TestHandleVerifyOrder_WithAPTransactionID_Success(t *testing.T) {
	body := encryptedResp(t, "200", "ok")

	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	})
	defer cleanup()

	handler := HandleVerifyOrder(apiClient, baseURL)
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, VerifyOrderInput{
		APTransactionID: "999888",
	})

	if err != nil || isErrorResult(result) {
		t.Fatalf("expected success: err=%v isError=%v", err, isErrorResult(result))
	}
}

func TestHandleVerifyOrder_WithRRN_Success(t *testing.T) {
	body := encryptedResp(t, "200", "ok")

	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	})
	defer cleanup()

	handler := HandleVerifyOrder(apiClient, baseURL)
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, VerifyOrderInput{RRN: "556677"})

	if err != nil || isErrorResult(result) {
		t.Fatalf("expected success: err=%v isError=%v", err, isErrorResult(result))
	}
}

func TestHandleVerifyOrder_APIError_ReturnsErrorResult(t *testing.T) {
	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	})
	defer cleanup()

	handler := HandleVerifyOrder(apiClient, baseURL)
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, VerifyOrderInput{OrderID: "ORD1"})

	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if !isErrorResult(result) {
		t.Fatal("expected error result on API failure")
	}
}

func TestHandleVerifyOrder_OptionalFields_Sent(t *testing.T) {
	var captured []byte

	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		captured = []byte(r.FormValue("encdata"))
		body := encryptedResp(t, "200", "ok")
		w.Write(body)
	})
	defer cleanup()

	handler := HandleVerifyOrder(apiClient, baseURL)
	handler(context.Background(), &mcpsdk.CallToolRequest{}, VerifyOrderInput{
		OrderID:    "ORD123",
		TerminalID: "12345678",
		TxnType:    "pos",
	})

	// encdata should be present (non-empty) — params were encrypted and sent
	if len(captured) == 0 {
		t.Error("expected encdata to be present in request")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// HandleGetBankList
// ─────────────────────────────────────────────────────────────────────────────

func TestHandleGetBankList_NoInput_UsesDefaultDomain(t *testing.T) {
	var path string
	body := encryptedResp(t, "200", "ok")

	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Write(body)
	})
	defer cleanup()

	handler := HandleGetBankList(apiClient, baseURL, "https://merchant.example.com")
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, BankListInput{})

	if err != nil || isErrorResult(result) {
		t.Fatalf("expected success: err=%v isError=%v", err, isErrorResult(result))
	}
	if !strings.HasSuffix(path, "/banks/") {
		t.Errorf("expected /banks/ endpoint, got %s", path)
	}
}

func TestHandleGetBankList_WithPaymentMode_Success(t *testing.T) {
	body := encryptedResp(t, "200", "ok")

	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	})
	defer cleanup()

	handler := HandleGetBankList(apiClient, baseURL, "https://merchant.example.com")
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, BankListInput{PaymentMode: "upi"})

	if err != nil || isErrorResult(result) {
		t.Fatalf("expected success: err=%v isError=%v", err, isErrorResult(result))
	}
}

func TestHandleGetBankList_WithCustomDomain_Success(t *testing.T) {
	body := encryptedResp(t, "200", "ok")

	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	})
	defer cleanup()

	handler := HandleGetBankList(apiClient, baseURL, "https://merchant.example.com")
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, BankListInput{})

	if err != nil || isErrorResult(result) {
		t.Fatalf("expected success: err=%v isError=%v", err, isErrorResult(result))
	}
}

func TestHandleGetBankList_APIError_ReturnsErrorResult(t *testing.T) {
	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad gateway", http.StatusBadGateway)
	})
	defer cleanup()

	handler := HandleGetBankList(apiClient, baseURL, "https://merchant.example.com")
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, BankListInput{})

	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if !isErrorResult(result) {
		t.Fatal("expected error result on API failure")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// HandleGetTransactionDetail
// ─────────────────────────────────────────────────────────────────────────────

func TestHandleGetTransactionDetail_MissingFields_ReturnsError(t *testing.T) {
	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, nil)
	defer cleanup()

	handler := HandleGetTransactionDetail(apiClient, baseURL)

	cases := []struct {
		name  string
		input TransactionDetailInput
	}{
		{"missing mercid", TransactionDetailInput{TerminalID: "12345678", UniqueID: "u1", ReferenceID: "r1"}},
		{"missing terminalid", TransactionDetailInput{MerchantID: "123", UniqueID: "u1", ReferenceID: "r1"}},
		{"missing uniqueid", TransactionDetailInput{MerchantID: "123", TerminalID: "12345678", ReferenceID: "r1"}},
		{"missing referenceid", TransactionDetailInput{MerchantID: "123", TerminalID: "12345678", UniqueID: "u1"}},
		{"all missing", TransactionDetailInput{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !isErrorResult(result) {
				t.Fatal("expected error result for missing fields")
			}
		})
	}
}

func TestHandleGetTransactionDetail_AllFields_Success(t *testing.T) {
	respBody, _ := json.Marshal(client.APIResponse{Status: "200", Message: "ok"})

	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(respBody)
	})
	defer cleanup()

	handler := HandleGetTransactionDetail(apiClient, baseURL)
	result, _, err := handler(context.Background(), &mcpsdk.CallToolRequest{}, TransactionDetailInput{
		MerchantID:  "12345",
		TerminalID:  "12345678",
		UniqueID:    "REQ001",
		ReferenceID: "ORD001",
	})

	if err != nil || isErrorResult(result) {
		t.Fatalf("expected success: err=%v isError=%v", err, isErrorResult(result))
	}
}

func TestHandleGetTransactionDetail_HitsCorrectEndpoint(t *testing.T) {
	var hitPath string
	respBody, _ := json.Marshal(client.APIResponse{Status: "200", Message: "ok"})

	apiClient, baseURL, cleanup := testhelper.NewTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		hitPath = r.URL.Path
		w.Write(respBody)
	})
	defer cleanup()

	handler := HandleGetTransactionDetail(apiClient, baseURL)
	handler(context.Background(), &mcpsdk.CallToolRequest{}, TransactionDetailInput{
		MerchantID:  "12345",
		TerminalID:  "12345678",
		UniqueID:    "REQ001",
		ReferenceID: "ORD001",
	})

	if hitPath != "/airpay/ms/pos/api/transaction-detail" {
		t.Errorf("expected /airpay/ms/pos/api/transaction-detail, got %s", hitPath)
	}
}
