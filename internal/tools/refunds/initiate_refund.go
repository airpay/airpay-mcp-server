package refunds

import (
	"context"
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
	Transactions []RefundTransaction `json:"transactions" jsonschema:"description:List of transactions to refund. Each must have ap_transactionid (int) and amount (string with 2 decimals),required"`
}

// HandleInitiateRefund initiates full or partial refund for one or more transactions.
func HandleInitiateRefund(apiClient *client.AirpayClient, baseURL string) mcp.ToolHandlerFor[InitiateRefundInput, any] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input InitiateRefundInput) (*mcp.CallToolResult, any, error) {
		if len(input.Transactions) == 0 {
			return tools.ErrorResult(fmt.Errorf("at least one transaction is required for refund")), nil, nil
		}

		// Encode transactions as base64 JSON (Airpay API requirement)
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
