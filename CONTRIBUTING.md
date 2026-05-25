# Contributing to Airpay MCP Server

## Prerequisites

- Go 1.25+
- An Airpay sandbox merchant account (for integration testing)

## Setup

```bash
git clone https://github.com/airpay/airpay-mcp-server.git
cd airpay-mcp-server
go mod download
cp .env.example .env
# Edit .env with your sandbox credentials
```

## Development Workflow

### Run the server

```bash
go run ./cmd/airpay-mcp-server
```

### Run tests

```bash
go test ./...
go test -cover ./...
go test -race ./...
```

### Lint and format

```bash
go vet ./...
go fmt ./...
```

## Project Structure

```
cmd/airpay-mcp-server/   # Entry point
internal/
  client/                # Airpay HTTP client (encryption, OAuth2, retry)
  security/              # AES-256-CBC, checksum, private key, token manager
  server/                # Config loader
  tools/
    payments/            # verify_order, bank_list, transaction_detail
    refunds/             # initiate_refund
    qr_upi/              # validate_vpa, generate_qr
    subscriptions/       # check_status
```

## Adding a New Tool

1. Create a handler file in the appropriate `internal/tools/<toolset>/` directory:

```go
package payments

import (
    "context"
    "github.com/airpay/airpay-mcp-server/internal/client"
    "github.com/airpay/airpay-mcp-server/internal/tools"
    "github.com/modelcontextprotocol/go-sdk/mcp"
)

type MyToolInput struct {
    Field string `json:"field" jsonschema:"description:...,required"`
}

func HandleMyTool(apiClient *client.AirpayClient, baseURL string) mcp.ToolHandlerFor[MyToolInput, any] {
    return func(ctx context.Context, req *mcp.CallToolRequest, input MyToolInput) (*mcp.CallToolResult, any, error) {
        if input.Field == "" {
            return tools.ErrorResult(fmt.Errorf("field is required")), nil, nil
        }
        resp, err := apiClient.PostEncrypted(baseURL+"/your/endpoint", map[string]string{"field": input.Field})
        if err != nil {
            return tools.ErrorResult(err), nil, nil
        }
        return tools.JSONResult(resp), nil, nil
    }
}
```

2. Register in `cmd/airpay-mcp-server/main.go` → `registerTools()` with appropriate annotations:
   - Read-only tools: use `ro` annotations
   - Write tools (non-destructive): use `wr` annotations
   - Irreversible actions (refunds, deletions): use `dest` annotations

3. Write tests — minimum 80% coverage required.

4. Update the tool table in `README.md`.

## Pull Request Guidelines

- One logical change per PR
- All tests must pass: `go test ./...`
- No linter errors: `go vet ./...`
- Coverage must not drop below 80% on modified packages
- No secrets, credentials, or internal URLs in code
- Commit message format: `type: description` (types: `feat`, `fix`, `refactor`, `docs`, `test`, `chore`)

## Reporting Bugs

Open a GitHub issue with:
- Go version (`go version`)
- OS and architecture
- Steps to reproduce
- Expected vs actual behavior
- Relevant log output (redact any credentials)

## Security Issues

**Do not open public issues for security vulnerabilities.** See [SECURITY.md](SECURITY.md).
