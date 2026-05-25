package payments

import (
	"context"
	"fmt"

	"github.com/airpay/airpay-mcp-server/internal/client"
	"github.com/airpay/airpay-mcp-server/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// VerifyOrderInput is the typed input for the verify-order tool.
type VerifyOrderInput struct {
	OrderID         string `json:"orderid" jsonschema:"description:Merchant generated order/transaction ID"`
	APTransactionID string `json:"ap_transactionid,omitempty" jsonschema:"description:Airpay transaction ID (alternative to orderid)"`
	RRN             string `json:"rrn,omitempty" jsonschema:"description:Retrieval Reference Number (alternative to orderid)"`
	TerminalID      string `json:"terminal_id,omitempty" jsonschema:"description:POS terminal ID (for POS transactions)"`
	TxnType         string `json:"txn_type,omitempty" jsonschema:"description:Transaction type e.g. pos"`
}

// HandleVerifyOrder verifies/confirms a payment order status.
func HandleVerifyOrder(apiClient *client.AirpayClient, baseURL string) mcp.ToolHandlerFor[VerifyOrderInput, any] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input VerifyOrderInput) (*mcp.CallToolResult, any, error) {
		if input.OrderID == "" && input.APTransactionID == "" && input.RRN == "" {
			return tools.ErrorResult(fmt.Errorf("at least one of orderid, ap_transactionid, or rrn is required")), nil, nil
		}

		params := make(map[string]string)
		if input.OrderID != "" {
			params["orderid"] = input.OrderID
		}
		if input.APTransactionID != "" {
			params["ap_transactionid"] = input.APTransactionID
		}
		if input.RRN != "" {
			params["rrn"] = input.RRN
		}
		if input.TerminalID != "" {
			params["terminal_id"] = input.TerminalID
		}
		if input.TxnType != "" {
			params["txn_type"] = input.TxnType
		}

		endpoint := baseURL + "/airpay/pay/v4/api/verify/"
		resp, err := apiClient.PostEncrypted(endpoint, params)
		if err != nil {
			return tools.ErrorResult(err), nil, nil
		}

		return tools.JSONResult(resp), nil, nil
	}
}
