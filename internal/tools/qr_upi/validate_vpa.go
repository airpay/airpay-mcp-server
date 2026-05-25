package qr_upi

import (
	"context"
	"fmt"

	"github.com/airpay/airpay-mcp-server/internal/client"
	"github.com/airpay/airpay-mcp-server/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ValidateVPAInput is the typed input for the validate-vpa tool.
type ValidateVPAInput struct {
	CustomerVPA string `json:"customer_vpa" jsonschema:"description:UPI Virtual Payment Address to validate e.g. user@upi,required"`
}

// HandleValidateVPA validates a UPI Virtual Payment Address.
func HandleValidateVPA(apiClient *client.AirpayClient, baseURL string) mcp.ToolHandlerFor[ValidateVPAInput, any] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input ValidateVPAInput) (*mcp.CallToolResult, any, error) {
		if input.CustomerVPA == "" {
			return tools.ErrorResult(fmt.Errorf("customer_vpa is required")), nil, nil
		}

		params := map[string]string{
			"customer_vpa": input.CustomerVPA,
		}

		endpoint := baseURL + "/airpay/pay/v4/api/vpavalidate/"
		resp, err := apiClient.PostEncrypted(endpoint, params)
		if err != nil {
			return tools.ErrorResult(err), nil, nil
		}

		return tools.JSONResult(resp), nil, nil
	}
}
