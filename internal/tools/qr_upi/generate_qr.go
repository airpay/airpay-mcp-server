package qr_upi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/airpay/airpay-mcp-server/internal/client"
	"github.com/airpay/airpay-mcp-server/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GenerateQRInput is the typed input for the generate-qr tool.
type GenerateQRInput struct {
	OrderID         string `json:"orderid" jsonschema:"description:Merchant generated order/transaction ID,required"`
	Amount          string `json:"amount" jsonschema:"description:Amount with two decimals e.g. 100.00,required"`
	BuyerEmail      string `json:"buyer_email" jsonschema:"description:Buyer email address,required"`
	BuyerPhone      string `json:"buyer_phone" jsonschema:"description:Buyer phone number,required"`
	TerminalID      string `json:"tid,omitempty" jsonschema:"description:Terminal ID of POS device (optional)"`
	CustomVar       string `json:"customvar,omitempty" jsonschema:"description:Custom tracking information (optional)"`
	CustomerConsent string `json:"customer_consent,omitempty" jsonschema:"description:Customer consent flag Y/N (default Y)"`
}

// HandleGenerateQR generates a UPI QR code for payment.
func HandleGenerateQR(apiClient *client.AirpayClient, baseURL, paymentDomain string) mcp.ToolHandlerFor[GenerateQRInput, any] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input GenerateQRInput) (*mcp.CallToolResult, any, error) {
		if input.OrderID == "" || input.Amount == "" || input.BuyerEmail == "" || input.BuyerPhone == "" {
			return tools.ErrorResult(fmt.Errorf("orderid, amount, buyer_email, and buyer_phone are required")), nil, nil
		}

		params := map[string]string{
			"orderid":     input.OrderID,
			"amount":      input.Amount,
			"buyer_email": input.BuyerEmail,
			"buyer_phone": input.BuyerPhone,
			"call_type":   "upiqr",
		}
		params["mer_dom"] = base64.StdEncoding.EncodeToString([]byte(paymentDomain))
		if input.TerminalID != "" {
			params["tid"] = input.TerminalID
		}
		if input.CustomVar != "" {
			params["customvar"] = input.CustomVar
		}
		consent := input.CustomerConsent
		if consent == "" {
			consent = "Y"
		}
		params["customer_consent"] = consent

		endpoint := baseURL + "/airpay/pay/v4/api/generateorder/"
		resp, err := apiClient.PostEncrypted(endpoint, params)
		if err != nil {
			return tools.ErrorResult(err), nil, nil
		}

		filterQRData(resp)
		return tools.JSONResult(resp), nil, nil
	}
}

// filterQRData removes the QR PNG download URL and UPI deep-link from the
// response data, keeping only the View QR Page link and other fields.
func filterQRData(resp *client.APIResponse) {
	if len(resp.Data) == 0 {
		return
	}
	var data map[string]interface{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return
	}
	for k, v := range data {
		if s, ok := v.(string); ok {
			if strings.HasPrefix(s, "upi://") || strings.Contains(s, "/qr/") {
				delete(data, k)
			}
		}
	}
	if filtered, err := json.Marshal(data); err == nil {
		resp.Data = filtered
	}
}
