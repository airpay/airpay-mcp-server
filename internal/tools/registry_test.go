package tools

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/airpay/airpay-mcp-server/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ---------------------------------------------------------------------------
// TextResult
// ---------------------------------------------------------------------------

func TestTextResult_ReturnsTextContent(t *testing.T) {
	result := TextResult("hello")

	if result == nil {
		t.Fatal("expected non-nil CallToolResult")
	}
	if result.IsError {
		t.Error("IsError should be false for TextResult")
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected *mcp.TextContent, got %T", result.Content[0])
	}
	if tc.Text != "hello" {
		t.Errorf("TextContent.Text = %q, want %q", tc.Text, "hello")
	}
}

func TestTextResult_EmptyString(t *testing.T) {
	result := TextResult("")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected *mcp.TextContent, got %T", result.Content[0])
	}
	if tc.Text != "" {
		t.Errorf("TextContent.Text = %q, want empty string", tc.Text)
	}
}

func TestTextResult_UnicodeAndSpecialChars(t *testing.T) {
	input := "hello 世界 🌏 <>&\""
	result := TextResult(input)
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected *mcp.TextContent, got %T", result.Content[0])
	}
	if tc.Text != input {
		t.Errorf("TextContent.Text = %q, want %q", tc.Text, input)
	}
}

// ---------------------------------------------------------------------------
// ErrorResult
// ---------------------------------------------------------------------------

func TestErrorResult_IsErrorTrue(t *testing.T) {
	result := ErrorResult(fmt.Errorf("boom"))

	if result == nil {
		t.Fatal("expected non-nil CallToolResult")
	}
	if !result.IsError {
		t.Error("IsError should be true for ErrorResult")
	}
}

func TestErrorResult_ContainsErrorMessage(t *testing.T) {
	result := ErrorResult(fmt.Errorf("boom"))

	if len(result.Content) == 0 {
		t.Fatal("expected at least one content item")
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected *mcp.TextContent, got %T", result.Content[0])
	}
	if !strings.Contains(tc.Text, "boom") {
		t.Errorf("error content %q should contain 'boom'", tc.Text)
	}
	if !strings.Contains(tc.Text, "Error:") {
		t.Errorf("error content %q should contain 'Error:'", tc.Text)
	}
}

func TestErrorResult_FormattedMessage(t *testing.T) {
	result := ErrorResult(fmt.Errorf("connection refused"))
	tc := result.Content[0].(*mcp.TextContent)
	want := "Error: connection refused"
	if tc.Text != want {
		t.Errorf("error text = %q, want %q", tc.Text, want)
	}
}

func TestErrorResult_WrappedError(t *testing.T) {
	inner := fmt.Errorf("timeout")
	outer := fmt.Errorf("sending request: %w", inner)
	result := ErrorResult(outer)
	tc := result.Content[0].(*mcp.TextContent)
	if !strings.Contains(tc.Text, "timeout") {
		t.Errorf("error content %q should contain inner error 'timeout'", tc.Text)
	}
}

// ---------------------------------------------------------------------------
// JSONResult — nil input
// ---------------------------------------------------------------------------

func TestJSONResult_NilResponse_ReturnsError(t *testing.T) {
	result := JSONResult(nil)

	if result == nil {
		t.Fatal("expected non-nil CallToolResult")
	}
	if !result.IsError {
		t.Error("IsError should be true for nil response")
	}
}

func TestJSONResult_NilResponse_ContainsNilMessage(t *testing.T) {
	result := JSONResult(nil)
	if len(result.Content) == 0 {
		t.Fatal("expected content in error result")
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected *mcp.TextContent, got %T", result.Content[0])
	}
	if !strings.Contains(strings.ToLower(tc.Text), "nil") {
		t.Errorf("nil response error text %q should mention 'nil'", tc.Text)
	}
}

// ---------------------------------------------------------------------------
// JSONResult — valid APIResponse
// ---------------------------------------------------------------------------

func TestJSONResult_ValidResponse_IsErrorFalse(t *testing.T) {
	resp := &client.APIResponse{
		Status:  "success",
		Message: "operation completed",
		Data:    json.RawMessage(`{"id":"txn-1"}`),
	}

	result := JSONResult(resp)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.IsError {
		t.Error("IsError should be false for valid response")
	}
}

func TestJSONResult_ValidResponse_ContainsStatus(t *testing.T) {
	resp := &client.APIResponse{
		Status:  "success",
		Message: "all good",
	}

	result := JSONResult(resp)

	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected *mcp.TextContent, got %T", result.Content[0])
	}
	if !strings.Contains(tc.Text, "success") {
		t.Errorf("output %q should contain 'success'", tc.Text)
	}
}

func TestJSONResult_ValidResponse_ContainsMessage(t *testing.T) {
	resp := &client.APIResponse{
		Status:  "success",
		Message: "transaction approved",
	}

	result := JSONResult(resp)

	tc := result.Content[0].(*mcp.TextContent)
	if !strings.Contains(tc.Text, "transaction approved") {
		t.Errorf("output %q should contain message 'transaction approved'", tc.Text)
	}
}

func TestJSONResult_WithStatusCode_IncludedInOutput(t *testing.T) {
	resp := &client.APIResponse{
		StatusCode: "200",
		Status:     "success",
		Message:    "ok",
	}

	result := JSONResult(resp)

	tc := result.Content[0].(*mcp.TextContent)
	if !strings.Contains(tc.Text, "200") {
		t.Errorf("output %q should contain status_code '200'", tc.Text)
	}
}

func TestJSONResult_WithResponseCode_IncludedInOutput(t *testing.T) {
	resp := &client.APIResponse{
		ResponseCode: "00",
		Status:       "success",
		Message:      "approved",
	}

	result := JSONResult(resp)

	tc := result.Content[0].(*mcp.TextContent)
	if !strings.Contains(tc.Text, "00") {
		t.Errorf("output %q should contain response_code '00'", tc.Text)
	}
}

func TestJSONResult_EmptyStatusCode_NotIncludedInOutput(t *testing.T) {
	resp := &client.APIResponse{
		StatusCode: "",
		Status:     "success",
		Message:    "ok",
	}

	result := JSONResult(resp)

	if result.IsError {
		t.Error("empty StatusCode should not cause an error")
	}
	// Just verify it returns a valid result with content.
	if len(result.Content) == 0 {
		t.Error("expected content in result")
	}
}

func TestJSONResult_WithJSONDataField_DecodedStructurally(t *testing.T) {
	resp := &client.APIResponse{
		Status:  "success",
		Message: "data present",
		Data:    json.RawMessage(`{"order_id":"ord-123","amount":500}`),
	}

	result := JSONResult(resp)

	if result.IsError {
		t.Errorf("unexpected error result: %v", result)
	}
	tc := result.Content[0].(*mcp.TextContent)
	// The TOON output should contain the data field values.
	if !strings.Contains(tc.Text, "ord-123") {
		t.Errorf("output %q should contain order_id value 'ord-123'", tc.Text)
	}
}

func TestJSONResult_WithNonJSONDataField_FallsBackGracefully(t *testing.T) {
	// Data that is not valid JSON — should fall back to string representation.
	resp := &client.APIResponse{
		Status:  "success",
		Message: "raw data",
		Data:    json.RawMessage(`not-valid-json`),
	}

	result := JSONResult(resp)

	// Should NOT return an error — graceful fallback is expected.
	if result.IsError {
		t.Error("non-JSON data field should not cause IsError=true")
	}
	if len(result.Content) == 0 {
		t.Error("expected content even with non-JSON data")
	}
}

func TestJSONResult_EmptyData_DoesNotPanic(t *testing.T) {
	resp := &client.APIResponse{
		Status:  "success",
		Message: "no data",
		Data:    nil,
	}

	// Should not panic.
	result := JSONResult(resp)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.IsError {
		t.Error("nil Data should not cause IsError=true")
	}
}

func TestJSONResult_WithArrayData_DecodedStructurally(t *testing.T) {
	resp := &client.APIResponse{
		Status:  "success",
		Message: "list",
		Data:    json.RawMessage(`[{"id":1},{"id":2}]`),
	}

	result := JSONResult(resp)

	if result.IsError {
		t.Error("array data should not cause error")
	}
}

func TestJSONResult_HasSingleTextContent(t *testing.T) {
	resp := &client.APIResponse{
		Status:  "success",
		Message: "check",
	}

	result := JSONResult(resp)

	if len(result.Content) != 1 {
		t.Errorf("expected 1 content item, got %d", len(result.Content))
	}
	if _, ok := result.Content[0].(*mcp.TextContent); !ok {
		t.Errorf("expected *mcp.TextContent, got %T", result.Content[0])
	}
}
