package refunds

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/airpay/airpay-mcp-server/internal/client"
	"github.com/airpay/airpay-mcp-server/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RefundTransaction represents a single transaction to refund.
type RefundTransaction struct {
	APTransactionID int    `json:"ap_transactionid"`
	Amount          string `json:"amount"`
}

// InitiateRefundInput is the typed input for the initiate-refund tool.
type InitiateRefundInput struct {
	Transactions      []RefundTransaction `json:"transactions"`
	Confirmed         bool               `json:"confirmed"`
	ConfirmationToken string             `json:"confirmation_token"`
}

func signTransactions(transactions []RefundTransaction, secret string) (string, error) {
	txnJSON, err := json.Marshal(transactions)
	if err != nil {
		return "", fmt.Errorf("marshalling transactions: %w", err)
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(txnJSON)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil)), nil
}

// HandleInitiateRefund initiates full or partial refund using a two-phase flow.
// Phase 1 (confirmed=false): returns transaction summary + confirmation_token.
// Phase 2 (confirmed=true): validates token then executes the refund.
func HandleInitiateRefund(apiClient *client.AirpayClient, baseURL, secret string) mcp.ToolHandlerFor[InitiateRefundInput, any] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input InitiateRefundInput) (*mcp.CallToolResult, any, error) {
		if len(input.Transactions) == 0 {
			return tools.ErrorResult(fmt.Errorf("at least one transaction is required for refund")), nil, nil
		}

		if !input.Confirmed {
			// Phase 1: preview — compute token, return summary, do NOT call API
			token, err := signTransactions(input.Transactions, secret)
			if err != nil {
				return tools.ErrorResult(fmt.Errorf("generating confirmation token: %w", err)), nil, nil
			}
			preview := map[string]any{
				"phase":              1,
				"confirmation_token": token,
				"transactions":       input.Transactions,
				"message":            "Review the transactions above. Re-call with confirmed=true and this confirmation_token to execute the refund. This action is IRREVERSIBLE.",
			}
			previewJSON, err := json.Marshal(preview)
			if err != nil {
				return tools.ErrorResult(fmt.Errorf("encoding preview: %w", err)), nil, nil
			}
			return tools.TextResult(string(previewJSON)), nil, nil
		}

		// Phase 2: validate token matches transactions before executing
		if input.ConfirmationToken == "" {
			return tools.ErrorResult(fmt.Errorf("confirmation_token is required when confirmed=true")), nil, nil
		}
		expectedToken, err := signTransactions(input.Transactions, secret)
		if err != nil {
			return tools.ErrorResult(fmt.Errorf("validating confirmation token: %w", err)), nil, nil
		}
		if !hmac.Equal([]byte(input.ConfirmationToken), []byte(expectedToken)) {
			return tools.ErrorResult(fmt.Errorf("confirmation_token mismatch — transactions may have been modified; restart with confirmed=false")), nil, nil
		}

		// Execute refund
		txnJSON, err := json.Marshal(input.Transactions)
		if err != nil {
			return tools.ErrorResult(fmt.Errorf("encoding transactions: %w", err)), nil, nil
		}
		params := map[string]string{
			"mode":         "refund",
			"transactions": base64.StdEncoding.EncodeToString(txnJSON),
		}

		endpoint := baseURL + "/airpay/pay/v4/api/refund/"
		resp, err := apiClient.PostEncrypted(endpoint, params)
		if err != nil {
			return tools.ErrorResult(err), nil, nil
		}

		return tools.JSONResult(resp), nil, nil
	}
}
