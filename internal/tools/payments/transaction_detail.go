package payments

import (
	"context"
	"fmt"
	"net/url"

	"github.com/airpay/airpay-mcp-server/internal/client"
	"github.com/airpay/airpay-mcp-server/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TransactionDetailInput is the typed input for the get-transaction-detail tool.
type TransactionDetailInput struct {
	MerchantID  string `json:"mercid" jsonschema:"description:Merchant ID,required"`
	TerminalID  string `json:"terminalid" jsonschema:"description:POS terminal ID (8 digits),required"`
	UniqueID    string `json:"uniqueid" jsonschema:"description:Unique identifier for the request,required"`
	ReferenceID string `json:"referenceid" jsonschema:"description:Reference ID (order ID) to look up,required"`
}

// HandleGetTransactionDetail gets details of a specific POS transaction.
func HandleGetTransactionDetail(apiClient *client.AirpayClient, baseURL string) mcp.ToolHandlerFor[TransactionDetailInput, any] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input TransactionDetailInput) (*mcp.CallToolResult, any, error) {
		if input.MerchantID == "" || input.TerminalID == "" || input.UniqueID == "" || input.ReferenceID == "" {
			return tools.ErrorResult(fmt.Errorf("mercid, terminalid, uniqueid, and referenceid are all required")), nil, nil
		}

		endpoint := baseURL + "/airpay/ms/pos/api/transaction-detail"
		formData := url.Values{
			"mercid":      {input.MerchantID},
			"terminalid":  {input.TerminalID},
			"uniqueid":    {input.UniqueID},
			"referenceid": {input.ReferenceID},
		}

		resp, err := apiClient.PostFormDirect(endpoint, formData)
		if err != nil {
			return tools.ErrorResult(err), nil, nil
		}

		return tools.JSONResult(resp), nil, nil
	}
}
