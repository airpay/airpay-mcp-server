package qr_upi

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/airpay/airpay-mcp-server/internal/client"
	"github.com/airpay/airpay-mcp-server/internal/qrlink"
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
func HandleGenerateQR(apiClient *client.AirpayClient, baseURL, paymentDomain, qrMasterKeyRaw, mcpServerURL string, qrLinkTTL int) mcp.ToolHandlerFor[GenerateQRInput, any] {
	// Derive a stable 32-byte key: prefer QR_MASTER_KEY, fall back to nothing (links omitted).
	var masterKey []byte
	if qrMasterKeyRaw != "" {
		k := sha256.Sum256([]byte(qrMasterKeyRaw))
		masterKey = k[:]
	}

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

		// Parse response data — Airpay returns an object on success and an
		// array on error (e.g. [{"status":"400","message":"..."}]).
		var dataMap map[string]any
		if err := json.Unmarshal(resp.Data, &dataMap); err != nil {
			var arr []map[string]any
			if err2 := json.Unmarshal(resp.Data, &arr); err2 == nil && len(arr) > 0 {
				dataMap = arr[0]
			} else {
				return tools.JSONResult(resp), nil, nil
			}
		}

		qrcodeStr, _ := dataMap["qrcode_string"].(string)
		apTxID := fmt.Sprintf("%v", dataMap["ap_transactionid"])
		merchantID := fmt.Sprintf("%v", dataMap["merchant_id"])

		// Build clean response — omit qrcode_string (raw UPI deep link).
		clean := map[string]any{}
		for _, k := range []string{"ap_transactionid", "merchant_id", "status", "message", "response_code", "status_code"} {
			if v, ok := dataMap[k]; ok {
				clean[k] = v
			}
		}

		// Generate encrypted preview URL.
		if qrcodeStr != "" && len(masterKey) == 32 && mcpServerURL != "" {
			payload := &qrlink.QRPayload{
				QRCodeString:    qrcodeStr,
				APTransactionID: apTxID,
				Amount:          input.Amount,
				OrderID:         input.OrderID,
				MerchantID:      merchantID,
				ExpiresAt:       time.Now().Add(time.Duration(qrLinkTTL) * time.Second).Unix(),
			}
			if token, tokenErr := qrlink.EncryptToken(payload, masterKey); tokenErr == nil {
				clean["view_qr_page"] = mcpServerURL + "/qr/" + token + "/preview"
			}
		}

		if filtered, err := json.Marshal(clean); err == nil {
			resp.Data = filtered
		}
		return tools.JSONResult(resp), nil, nil
	}
}
