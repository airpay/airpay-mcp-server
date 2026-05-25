# Airpay MCP Server

<div align="center">

**A Model Context Protocol (MCP) server for the Airpay Payment Gateway**

[![Go Version](https://img.shields.io/badge/Go-1.25%2B-00ADD8?style=flat&logo=go)](https://go.dev/)
[![MCP](https://img.shields.io/badge/MCP-1.4.1-blue)](https://github.com/modelcontextprotocol/go-sdk)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

Built with the [official MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk)

</div>

---

## Table of Contents

- [Overview](#overview)
- [Available Tools](#available-tools)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Configuration](#configuration)
- [Usage](#usage)
- [MCP Client Setup](#mcp-client-setup)
- [Security Architecture](#security-architecture)
- [Project Structure](#project-structure)
- [Development](#development)
- [Docker Deployment](#docker-deployment)
- [Troubleshooting](#troubleshooting)

---

## Overview

The **Airpay MCP Server** enables AI agents (Claude Desktop, Cursor, VS Code Copilot, and custom agents) to interact with Airpay's payment gateway APIs through the Model Context Protocol.

**Transport:** Streamable HTTP (MCP spec 2025-03-26) — stateless, load-balancer-friendly, production-ready.

**MCP endpoint:** `http://localhost:8888/mcp`

---

## Available Tools

### Payments (3 tools)
| Tool | MCP Name | Read-Only | Description |
|------|----------|-----------|-------------|
| verify-order | `airpay_verify_order` | ✅ | Verify payment status by orderid, ap_transactionid, or rrn |
| get-bank-list | `airpay_get_bank_list` | ✅ | List supported banks and payment options |
| get-transaction-detail | `airpay_pos_transaction_detail` | ✅ | Get specific POS transaction details |

### Refunds (1 tool)
| Tool | MCP Name | Read-Only | Description |
|------|----------|-----------|-------------|
| initiate-refund | `airpay_initiate_refund` | ❌ | Full or partial refund — two-phase with user confirmation |

### QR & UPI (2 tools)
| Tool | MCP Name | Read-Only | Description |
|------|----------|-----------|-------------|
| validate-vpa | `airpay_validate_vpa` | ✅ | Validate UPI Virtual Payment Address |
| generate-qr | `airpay_generate_qr` | ❌ | Generate dynamic UPI QR code for payment |

### Subscriptions (1 tool)
| Tool | MCP Name | Read-Only | Description |
|------|----------|-----------|-------------|
| check-subscription-status | `airpay_check_subscription_status` | ✅ | Get eNACH/SI subscription status |

**Total: 7 tools — Phase 1**

---

## Prerequisites

- **Go 1.25+** — [Download Go](https://go.dev/dl/)
- **Airpay Merchant Account** with API credentials:
  - Merchant ID, API Username, API Password
  - Secret Key, OAuth2 Client ID, OAuth2 Client Secret

---

## Installation

### Step 1: Clone

```bash
git clone https://github.com/airpay/airpay-mcp-server.git
cd airpay-mcp-server
```

### Step 2: Install Dependencies

```bash
go mod download
```

### Step 3: Configure

```bash
cp .env.example .env
```

Edit `.env` with your Airpay credentials.

### Step 4: Build

```bash
go build -o airpay-mcp-server ./cmd/airpay-mcp-server
```

**Windows:**
```bash
go build -o airpay-mcp-server.exe ./cmd/airpay-mcp-server
```

### Step 5: Run

```bash
./airpay-mcp-server
```

Server starts on `http://localhost:8888/mcp`.

---

## Configuration

All configuration via environment variables. See `.env.example` for the full template.

### Required

| Variable | Description | Example |
|----------|-------------|---------|
| `AIRPAY_MERCHANT_ID` | Airpay merchant ID | `12345` |
| `AIRPAY_USERNAME` | API username | `merchant@example.com` |
| `AIRPAY_PASSWORD` | API password | `your_password` |
| `AIRPAY_SECRET` | Secret key for encryption | `your_secret_key` |
| `AIRPAY_CLIENT_ID` | OAuth2 client ID | `your_client_id` |
| `AIRPAY_CLIENT_SECRET` | OAuth2 client secret | `your_client_secret` |
| `PAYMENT_DOMAIN` | Merchant domain for QR/bank-list requests | `https://yourstore.com` |

### Optional

| Variable | Default | Description |
|----------|---------|-------------|
| `TRANSPORT` | `http` | Transport protocol — `http` (Streamable HTTP) |
| `PORT` | `8888` | Listening port |
| `ENVIRONMENT` | `sandbox` | `sandbox` or `production` |
| `TOOLSETS` | `all` | Comma-separated toolsets or `all` |
| `READ_ONLY` | `false` | Restrict to read-only tools |
| `LOG_FILE` | - | Log file path (empty = stderr) |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |

### Toolset Names

`payments`, `refunds`, `qr_upi`, `subscriptions`

```bash
# Payments and refunds only
TOOLSETS=payments,refunds

# Read-only production mode
READ_ONLY=true TOOLSETS=payments,subscriptions
```

---

## Usage

### Start the Server

```bash
./airpay-mcp-server
# [mcp] Running over Streamable HTTP on :8888
```

MCP endpoint: `http://localhost:8888/mcp`  
Health check: `http://localhost:8888/health`

### Using Environment File

```bash
export $(cat .env | xargs) && ./airpay-mcp-server
```

---

## MCP Client Setup

### Claude Desktop

1. Locate config file:
   - **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
   - **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

2. Add server configuration:

```json
{
  "mcpServers": {
    "airpay": {
      "type": "http",
      "url": "http://localhost:8888/mcp"
    }
  }
}
```

> If running the binary directly with credentials baked in via env, use `type: "stdio"` with the binary path. For HTTP transport (Docker/remote), use `type: "http"` with the URL.

### Cursor IDE

Create `.cursor/mcp.json` in your project root:

```json
{
  "mcpServers": {
    "airpay": {
      "type": "http",
      "url": "http://localhost:8888/mcp"
    }
  }
}
```

### VS Code

Create `.vscode/mcp.json` in your workspace:

```json
{
  "servers": {
    "airpay": {
      "type": "http",
      "url": "http://localhost:8888/mcp"
    }
  }
}
```

### MCP Inspector (Testing)

```bash
npx @modelcontextprotocol/inspector http://localhost:8888/mcp
```

---

## Security Architecture

Multi-layered security — unique among payment gateway MCP implementations:

### Encryption Key
```
MD5(username + "~:~" + password) → 32-byte hex key
```

### Private Key
```
SHA-256(secret + "@" + username + ":|:" + password)
```

### Checksum
```
SHA-256(alphabetically_sorted_param_values + YYYY-MM-DD)
```

### OAuth2 Token Management
- Auto-refresh 60 seconds before 5-minute TTL expires
- Token exchange itself is encrypted

### Request Pipeline
```
Params → AES-256-CBC Encrypt → Checksum + Private Key + OAuth2 Token → POST
```

### Response Pipeline
```
Encrypted Response → AES-256-CBC Decrypt → JSON → AI Agent
```

---

## Project Structure

```
airpay-mcp-server/
├── cmd/
│   └── airpay-mcp-server/
│       └── main.go                  # Entry point, transport, tool registration
├── internal/
│   ├── client/
│   │   └── http.go                  # Airpay HTTP client with encryption middleware
│   ├── security/
│   │   ├── encryption.go            # AES-256-CBC encrypt/decrypt
│   │   ├── checksum.go              # SHA-256 checksum
│   │   ├── privatekey.go            # Private key derivation
│   │   ├── oauth2.go                # OAuth2 token manager
│   │   ├── masking.go               # Sensitive data masking
│   │   └── encryption_test.go       # Security unit tests
│   ├── server/
│   │   └── config.go                # Configuration from env vars
│   └── tools/
│       ├── registry.go              # TextResult / ErrorResult / JSONResult
│       ├── payments/                # verify_order, bank_list, transaction_detail
│       ├── refunds/                 # initiate_refund
│       ├── qr_upi/                  # validate_vpa, generate_qr
│       └── subscriptions/           # check_status
├── .env.example
├── Dockerfile
├── go.mod
└── README.md
```

---

## Development

### Run Tests

```bash
go test ./...
go test -cover ./...
go test -race ./internal/security/...
```

### Code Quality

```bash
go vet ./...
go fmt ./...
```

### Cross-Platform Builds

```bash
GOOS=linux   GOARCH=amd64 go build -o airpay-mcp-server-linux   ./cmd/airpay-mcp-server
GOOS=darwin  GOARCH=amd64 go build -o airpay-mcp-server-macos   ./cmd/airpay-mcp-server
GOOS=darwin  GOARCH=arm64 go build -o airpay-mcp-server-arm64   ./cmd/airpay-mcp-server
GOOS=windows GOARCH=amd64 go build -o airpay-mcp-server.exe     ./cmd/airpay-mcp-server
```

### Adding New Tools

1. Create handler in the appropriate `internal/tools/<toolset>/` directory
2. Register in `cmd/airpay-mcp-server/main.go` → `registerTools()`
3. Update this README tool table

---

## Docker Deployment

### Build

```bash
docker build -t airpay-mcp-server .
```

### Run

```bash
docker run \
  -e AIRPAY_MERCHANT_ID=your_id \
  -e AIRPAY_USERNAME=your_username \
  -e AIRPAY_PASSWORD=your_password \
  -e AIRPAY_SECRET=your_secret \
  -e AIRPAY_CLIENT_ID=your_client_id \
  -e AIRPAY_CLIENT_SECRET=your_client_secret \
  -e PAYMENT_DOMAIN=https://yourstore.com \
  -p 8888:8888 \
  airpay-mcp-server
```

Or use an env file:

```bash
docker run --env-file .env -p 8888:8888 airpay-mcp-server
```

MCP endpoint: `http://localhost:8888/mcp`

### Docker Compose

```yaml
services:
  airpay-mcp:
    build: .
    ports:
      - "8888:8888"
    env_file:
      - .env
    restart: unless-stopped
```

```bash
docker compose up -d
```

---

## Troubleshooting

### "AIRPAY_MERCHANT_ID is required"

All required env vars must be set. Check `.env` file and re-export.

### OAuth2 token errors

- Verify `AIRPAY_CLIENT_ID` and `AIRPAY_CLIENT_SECRET`
- Confirm credentials have OAuth2 access enabled
- Check `ENVIRONMENT` matches your account type (sandbox vs production)

### Encryption/Decryption errors

`AIRPAY_USERNAME`, `AIRPAY_PASSWORD`, and `AIRPAY_SECRET` derive the encryption key — must be exact.

### Connection refused at /mcp

- Confirm server is running: `curl http://localhost:8888/health`
- Check `PORT` matches what your client is connecting to

### Debug logging

```bash
LOG_LEVEL=debug ./airpay-mcp-server
```

---

## License

MIT License — see [LICENSE](LICENSE) for details.
