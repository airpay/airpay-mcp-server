package subscriptions

import (
	"context"
	"fmt"

	"github.com/airpay/airpay-mcp-server/internal/client"
	"github.com/airpay/airpay-mcp-server/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CheckSubscriptionStatusInput is the typed input for the check-subscription-status tool.
type CheckSubscriptionStatusInput struct {
	SubscriptionID string `json:"subscription_id,omitempty" jsonschema:"description:Airpay subscription ID (numeric). Required if orderId not provided."`
	OrderID        string `json:"orderId,omitempty" jsonschema:"description:Merchant order ID linked to the subscription (numeric). Required if subscription_id not provided."`
	PageNo         string `json:"pgno,omitempty" jsonschema:"description:Page number for transaction history pagination (default 0)."`
}

// HandleCheckSubscriptionStatus retrieves the current status of an eNACH/SI subscription.
func HandleCheckSubscriptionStatus(apiClient *client.AirpayClient, baseURL string) mcp.ToolHandlerFor[CheckSubscriptionStatusInput, any] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input CheckSubscriptionStatusInput) (*mcp.CallToolResult, any, error) {
		if input.SubscriptionID == "" && input.OrderID == "" {
			return tools.ErrorResult(fmt.Errorf("at least one of subscription_id or orderId is required")), nil, nil
		}

		params := make(map[string]string)
		if input.SubscriptionID != "" {
			params["subscription_id"] = input.SubscriptionID
		}
		if input.OrderID != "" {
			params["orderId"] = input.OrderID
		}
		if input.PageNo != "" {
			params["pgno"] = input.PageNo
		}

		endpoint := baseURL + "/airpay/pay/v4/api/subscription/status"
		resp, err := apiClient.PostEncrypted(endpoint, params)
		if err != nil {
			return tools.ErrorResult(err), nil, nil
		}

		return tools.JSONResult(resp), nil, nil
	}
}
