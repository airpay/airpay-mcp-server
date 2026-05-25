package tools

import (
	"encoding/json"
	"fmt"

	"github.com/airpay/airpay-mcp-server/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/toon-format/toon-go"
)

// TextResult creates a simple text CallToolResult.
func TextResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}
}

// ErrorResult creates an error CallToolResult.
func ErrorResult(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Error: %v", err)},
		},
		IsError: true,
	}
}

// JSONResult encodes the Airpay API response as a TOON document for the AI agent.
// TOON is compact and token-efficient compared to JSON, improving LLM comprehension.
func JSONResult(resp *client.APIResponse) *mcp.CallToolResult {
	if resp == nil {
		return ErrorResult(fmt.Errorf("nil response from API"))
	}

	// Decode Data JSON into a generic value so TOON can render it structurally.
	var data any
	if len(resp.Data) > 0 {
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			// Fall back to raw string if data is not valid JSON.
			data = string(resp.Data)
		}
	}

	payload := map[string]any{
		"status":  resp.Status,
		"message": resp.Message,
		"data":    data,
	}
	if resp.StatusCode != "" {
		payload["status_code"] = resp.StatusCode
	}
	if resp.ResponseCode != "" {
		payload["response_code"] = resp.ResponseCode
	}

	text, err := toon.MarshalString(payload)
	if err != nil {
		// Fallback to plain text if TOON encoding fails.
		text = fmt.Sprintf("status: %s\nmessage: %s\ndata: %s", resp.Status, resp.Message, string(resp.Data))
	}

	return TextResult(text)
}
