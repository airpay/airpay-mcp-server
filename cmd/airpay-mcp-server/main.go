package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/airpay/airpay-mcp-server/internal/client"
	"github.com/airpay/airpay-mcp-server/internal/security"
	"github.com/airpay/airpay-mcp-server/internal/server"
	"github.com/airpay/airpay-mcp-server/internal/tools/payments"
	"github.com/airpay/airpay-mcp-server/internal/tools/qr_upi"
	"github.com/airpay/airpay-mcp-server/internal/tools/refunds"
	"github.com/airpay/airpay-mcp-server/internal/tools/subscriptions"
	"github.com/joho/godotenv"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	serverName    = "airpay-mcp-server"
	serverVersion = "1.0.0"
	commit        = "none"
	buildDate     = "unknown"
)

func main() {
	_ = godotenv.Load()
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	cfg, err := server.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if cfg.LogFile != "" {
		f, err := os.OpenFile(cfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("WARNING: cannot open log file %s: %v — using stderr", cfg.LogFile, err)
		} else {
			defer f.Close()
			log.SetOutput(f)
		}
	}

	log.Printf("Starting %s v%s (commit=%s built=%s transport=%s)", serverName, serverVersion, commit, buildDate, cfg.Transport)

	// Security layer
	enc := security.NewAirpayEncryption(cfg.Username, cfg.Password)
	privateKey := security.GeneratePrivateKey(cfg.Secret, cfg.Username, cfg.Password)
	tokenURL := cfg.APIBaseURL + "/airpay/pay/v4/api/oauth2"
	tokenMgr := security.NewOAuth2TokenManager(
		cfg.ClientID, cfg.ClientSecret, cfg.MerchantID, tokenURL, enc, privateKey,
	)

	apiClient := client.NewAirpayClient(enc, tokenMgr, privateKey, cfg.MerchantID)

	mcpServer := mcpsdk.NewServer(
		&mcpsdk.Implementation{
			Name:    serverName,
			Version: serverVersion,
		},
		&mcpsdk.ServerOptions{
			Instructions: `You are connected to the Airpay Payment Gateway MCP Server.

## Authentication
All requests are pre-authenticated. Trust tool results — do not infer session state from prior conversation. Report errors exactly as returned.

## Available Tool Groups
- payments: airpay_verify_order, airpay_get_bank_list, airpay_pos_transaction_detail
- refunds: airpay_initiate_refund
- qr_upi: airpay_validate_vpa, airpay_generate_qr
- subscriptions: airpay_check_subscription_status

## Typical Workflows
1. Check payment status: airpay_verify_order (by orderid OR ap_transactionid OR rrn)
2. Process refund: airpay_verify_order → airpay_initiate_refund (two-phase: confirm=false then confirm=true)
3. UPI QR payment: airpay_validate_vpa (optional) → airpay_generate_qr → airpay_verify_order (poll until status=200)
4. Subscription check: airpay_check_subscription_status (by subscription_id or orderId)

## Key Identifiers
- ap_transactionid: Airpay internal numeric ID. Prefer this when available.
- orderid: Merchant-generated order ID (alphanumeric, max 30 chars).
- rrn: Retrieval Reference Number from the bank.

## Transaction Status Codes
- 200 = Success
- 211 = In Processing (poll again)
- 400 = Failed
- 401 = Not registered properly
- 402 = Not yet processed
- 403 = No callback from bank
- 405 = Bounced
- 503 = No records found

## Subscription Status Values
- SUBSCRIBED = active, future debits will occur on NEXT_TRAN_DATE
- UNSUBSCRIBED = cancelled
- COMPLETED = all cycles finished
- PAUSED = temporarily halted

## Important Constraints
- verify-order works on LIVE merchant IDs only, not sandbox.
- Amounts must always be strings with 2 decimal places e.g. "100.00".
- initiate-refund is irreversible — always confirm with user before Phase 2.
- generate-qr QR code expires after successful payment.`,
		},
	)

	registerTools(mcpServer, cfg, apiClient)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("[main] Shutdown signal received")
		cancel()
	}()

	authToken := os.Getenv("AIRPAY_MCP_AUTH_TOKEN")
	if authToken != "" {
		log.Println("[mcp] Bearer token authentication enabled")
	} else {
		log.Println("[mcp] WARNING: AIRPAY_MCP_AUTH_TOKEN not set — /mcp endpoint is unauthenticated")
	}

	if err := runMCPServer(ctx, mcpServer, cfg, authToken); err != nil {
		log.Fatalf("[main] Server error: %v", err)
	}

	log.Println("[main] Shutdown complete")
}

func runMCPServer(ctx context.Context, mcpServer *mcpsdk.Server, cfg *server.Config, authToken string) error {
	switch cfg.Transport {
	case "stdio":
		log.Println("[mcp] Running over stdio")
		return mcpServer.Run(ctx, &mcpsdk.StdioTransport{})

	case "sse":
		addr := ":" + cfg.Port
		log.Printf("[mcp] Running over SSE on %s", addr)

		sseHandler := mcpsdk.NewSSEHandler(func(r *http.Request) *mcpsdk.Server {
			return mcpServer
		}, nil)

		var sseRoot http.Handler = http.HandlerFunc(sseHandler.ServeHTTP)
		if authToken != "" {
			sseRoot = bearerAuthMiddleware(authToken, sseRoot)
		}

		mux := http.NewServeMux()
		mux.HandleFunc("/health", healthHandler(cfg.Transport))
		mux.Handle("/", sseRoot)

		srv := &http.Server{
			Addr:        addr,
			Handler:     mux,
			ReadTimeout: 15 * time.Second,
			// WriteTimeout intentionally 0: SSE connections are long-lived streams;
			// a non-zero value would forcibly close open event streams.
			WriteTimeout: 0,
			IdleTimeout:  60 * time.Second,
		}
		go func() {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("[mcp] SSE server error: %v", err)
			}
		}()
		<-ctx.Done()
		log.Println("[mcp] Shutting down SSE server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)

	case "http":
		addr := ":" + cfg.Port
		log.Printf("[mcp] Running over Streamable HTTP on %s", addr)

		mcpHandler := mcpsdk.NewStreamableHTTPHandler(func(r *http.Request) *mcpsdk.Server {
			return mcpServer
		}, &mcpsdk.StreamableHTTPOptions{
			Stateless: true,
		})

		var mcpRoute http.Handler = http.HandlerFunc(mcpHandler.ServeHTTP)
		if authToken != "" {
			mcpRoute = bearerAuthMiddleware(authToken, mcpRoute)
		}

		mux := http.NewServeMux()
		mux.HandleFunc("/health", healthHandler(cfg.Transport))
		mux.Handle("/mcp", mcpRoute)
		mux.Handle("/mcp/", mcpRoute)

		srv := &http.Server{
			Addr:         addr,
			Handler:      mux,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		}
		go func() {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("[mcp] HTTP server error: %v", err)
			}
		}()
		<-ctx.Done()
		log.Println("[mcp] Shutting down HTTP server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)

	default:
		return fmt.Errorf("unknown transport: %s (valid: stdio, sse, http)", cfg.Transport)
	}
}

func bearerAuthMiddleware(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, `{"error":"unauthorized"}`)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func healthHandler(transport string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","service":"airpay-mcp-server","transport":%q}`, transport)
	}
}

func registerTools(srv *mcpsdk.Server, cfg *server.Config, apiClient *client.AirpayClient) {
	baseURL := cfg.APIBaseURL
	registered := 0

	boolp := func(b bool) *bool { return &b }
	ro := &mcpsdk.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true, OpenWorldHint: boolp(true)}
	wr := &mcpsdk.ToolAnnotations{ReadOnlyHint: false, DestructiveHint: boolp(false), IdempotentHint: false, OpenWorldHint: boolp(true)}
	dest := &mcpsdk.ToolAnnotations{ReadOnlyHint: false, DestructiveHint: boolp(true), IdempotentHint: false, OpenWorldHint: boolp(true)}

	// --- Payments ---
	if cfg.IsToolsetEnabled("payments") {
		mcpsdk.AddTool(srv, &mcpsdk.Tool{
			Name: "airpay_verify_order",
			Description: `Pull the current status of a payment transaction from Airpay.

At least ONE of the following is required:
- orderid: Alphanumeric, 1-30 chars (e.g. "ORD123456")
- ap_transactionid: Numeric (e.g. "123456")
- rrn: Numeric, 1-12 digits (e.g. "556677")

Optional:
- terminal_id: Exactly 8 numeric digits (POS transactions only)
- txn_type: Alphanumeric (e.g. "pos")

Common errors:
❌ orderid: "ORD-123" (hyphen not allowed) → ✅ "ORD123"
❌ terminal_id: "1234" (too short) → ✅ "12345678"
❌ rrn: "12345678901234" (too long) → ✅ max 12 digits`,
			Annotations: ro,
		}, payments.HandleVerifyOrder(apiClient, baseURL))
		registered++

		mcpsdk.AddTool(srv, &mcpsdk.Tool{
			Name: "airpay_get_bank_list",
			Description: `Retrieve the list of banks and payment options available for the merchant.

Optional:
- chmod: Payment mode filter. One of: pg, ppc, nb, cash, emi, upi, btqr, payltr, va, enach. Leave empty for all.

Mode reference:
- pg: Payment Gateway (credit/debit cards)
- nb: Net Banking
- upi: UPI payments
- emi: EMI/installments
- enach: E-NACH mandates

Common errors:
❌ chmod: "card" (invalid) → ✅ "pg"
❌ chmod: "UPI" (uppercase) → ✅ "upi"`,
			Annotations: ro,
		}, payments.HandleGetBankList(apiClient, baseURL, cfg.PaymentDomain))
		registered++

		mcpsdk.AddTool(srv, &mcpsdk.Tool{
			Name: "airpay_pos_transaction_detail",
			Description: `Get details of a specific POS terminal transaction.

All fields required:
- mercid: Merchant ID (numeric, e.g. "123356")
- terminalid: Exactly 8 numeric digits (e.g. "12345678")
- uniqueid: Unique request identifier (alphanumeric, e.g. "REQ123456")
- referenceid: Order reference to look up (alphanumeric, 1-30 chars)

Common errors:
❌ terminalid: "1234" (too short) → ✅ "12345678"
❌ terminalid: "12345678ABC" (letters) → ✅ "12345678" (numeric only)`,
			Annotations: ro,
		}, payments.HandleGetTransactionDetail(apiClient, baseURL))
		registered++
	}

	// --- Refunds ---
	if cfg.IsToolsetEnabled("refunds") && !cfg.ReadOnly {
		mcpsdk.AddTool(srv, &mcpsdk.Tool{
			Name: "airpay_initiate_refund",
			Description: `Initiate a refund for one or more Airpay transactions.

TWO-PHASE FLOW — call this tool twice:

PHASE 1 (confirmed=false, the default):
  Pass the transactions array. Tool echoes back ap_transactionid + amount per transaction
  and returns a confirmation_token. Show details to user and get explicit approval.

PHASE 2 (confirmed=true):
  Re-call with the same transactions array, confirmed=true, and the confirmation_token
  from Phase 1. Any mismatch aborts the refund.

Each transaction requires:
- ap_transactionid: Integer from verify-order response (NOT an order ID). e.g. 123456
- amount: String with exactly 2 decimal places. e.g. "100.00"

Common errors:
❌ ap_transactionid: "ORD123" → ✅ 123456 (integer)
❌ amount: 100 or "100" or "100.0" → ✅ "100.00"
❌ transactions: [] (empty) → ✅ at least 1 transaction

CRITICAL: This action is IRREVERSIBLE. Never proceed to Phase 2 without explicit user confirmation.`,
			Annotations: dest,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"transactions"},
				"properties": map[string]any{
					"transactions": map[string]any{
						"type":        "array",
						"description": "List of transactions to refund.",
						"items": map[string]any{
							"type":     "object",
							"required": []string{"ap_transactionid", "amount"},
							"properties": map[string]any{
								"ap_transactionid": map[string]any{
									"type":        "integer",
									"description": "The ap_transactionid integer from verify-order. Not an order ID.",
								},
								"amount": map[string]any{
									"type":        "string",
									"pattern":     `^\d+\.\d{2}$`,
									"description": "Refund amount as string with exactly 2 decimal places. e.g. '100.00'",
								},
							},
						},
					},
					"confirmed": map[string]any{
						"type":        "boolean",
						"default":     false,
						"description": "Set true for Phase 2 only — after user has approved the echoed details.",
					},
					"confirmation_token": map[string]any{
						"type":        "string",
						"description": "Token returned from Phase 1. Required when confirmed=true.",
					},
				},
			},
		}, refunds.HandleInitiateRefund(apiClient, baseURL, cfg.Secret))
		registered++
	}

	// --- QR & UPI ---
	if cfg.IsToolsetEnabled("qr_upi") {
		mcpsdk.AddTool(srv, &mcpsdk.Tool{
			Name: "airpay_validate_vpa",
			Description: `Validate a UPI Virtual Payment Address (VPA/UPI ID) before initiating payment.

Required:
- customer_vpa: UPI address in format username@bankname (e.g. "user@paytm")

Common UPI handles: @paytm, @ybl, @oksbi, @okicici, @okaxis, @okhdfcbank, @upi

Common errors:
❌ "user" (missing @handle) → ✅ "user@paytm"
❌ "9876543210" (no handle) → ✅ "9876543210@ybl"
❌ "user@" (incomplete) → ✅ "user@paytm"

Returns vpa_name if valid — show to user to confirm correct recipient.`,
			Annotations: ro,
		}, qr_upi.HandleValidateVPA(apiClient, baseURL))
		registered++

		if !cfg.ReadOnly {
			mcpsdk.AddTool(srv, &mcpsdk.Tool{
				Name: "airpay_generate_qr",
				Description: `Generate a UPI QR code for payment collection.

All fields required:
- orderid: Alphanumeric only, 1-30 chars (e.g. "ORD123456")
- amount: Numeric with exactly 2 decimals (e.g. "100.00")
- buyer_email: Valid email (e.g. "customer@example.com")
- buyer_phone: Digits only, 8-15 digits (e.g. "9876543210")

Optional:
- tid: Terminal ID, max 15 chars
- customvar: Alphanumeric + spaces + equals, 1-4096 chars
- customer_consent: "Y" or "N" (default "Y")

Common errors:
❌ amount: "100" or "100.0" → ✅ "100.00"
❌ orderid: "ORD-123" (hyphen) → ✅ "ORD123"
❌ buyer_phone: "+919876543210" or "98765 43210" → ✅ "9876543210"

ALWAYS display the returned QR code image and payment link to the user.
After payment, poll airpay_verify_order with ap_transactionid to confirm status.`,
				Annotations: wr,
			}, qr_upi.HandleGenerateQR(apiClient, baseURL, cfg.PaymentDomain))
			registered++
		}
	}

	// --- Subscriptions ---
	if cfg.IsToolsetEnabled("subscriptions") {
		mcpsdk.AddTool(srv, &mcpsdk.Tool{
			Name: "airpay_check_subscription_status",
			Description: `Check the current status of an Airpay eNACH/SI subscription.

At least ONE of the following is required:
- subscription_id: Numeric Airpay subscription ID (e.g. "10234982")
- orderId: Numeric merchant order ID linked to the subscription (e.g. "1012")

Optional:
- pgno: Page number for transaction history pagination (default 0)

Common errors:
❌ Both subscription_id and orderId empty → ✅ provide at least one
❌ subscription_id: "SUB123" (non-numeric) → ✅ "10234982"

Response fields:
- SUBSCRIPTION_STATUS: SUBSCRIBED, UNSUBSCRIBED, COMPLETED, or PAUSED
- NEXT_TRAN_DATE: Next scheduled debit date
- LAST_TRAN_DATE: Date of last successful debit
- SUBSCRIPTION_AMOUNT: Authorized mandate amount
- TRANSACTION_HISTORY: Past debit attempts`,
			Annotations: ro,
		}, subscriptions.HandleCheckSubscriptionStatus(apiClient, baseURL))
		registered++
	}

	log.Printf("[mcp] Registered %d tools (read_only=%v, toolsets=%v)", registered, cfg.ReadOnly, cfg.Toolsets)
}
