package payments

import (
	"context"
	"encoding/base64"

	"github.com/airpay/airpay-mcp-server/internal/client"
	"github.com/airpay/airpay-mcp-server/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// BankListInput is the typed input for the get-bank-list tool.
type BankListInput struct {
	PaymentMode string `json:"chmod,omitempty" jsonschema:"description:Payment mode filter: pg/ppc/nb/cash/emi/upi/btqr/payltr/va/enach. Leave empty for all modes."`
}

// HandleGetBankList retrieves the list of supported banks and payment options.
func HandleGetBankList(apiClient *client.AirpayClient, baseURL, paymentDomain string) mcp.ToolHandlerFor[BankListInput, any] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input BankListInput) (*mcp.CallToolResult, any, error) {
		params := make(map[string]string)

		params["mer_dom"] = base64.StdEncoding.EncodeToString([]byte(paymentDomain))

		if input.PaymentMode != "" {
			params["chmod"] = input.PaymentMode
		}

		endpoint := baseURL + "/airpay/pay/v4/api/banks/"
		resp, err := apiClient.PostEncrypted(endpoint, params)
		if err != nil {
			return tools.ErrorResult(err), nil, nil
		}

		return tools.JSONResult(resp), nil, nil
	}
}
