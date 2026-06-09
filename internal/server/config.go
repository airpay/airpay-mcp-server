package server

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all configuration for the Airpay MCP Server.
type Config struct {
	// Airpay Credentials (Required)
	MerchantID    string
	Username      string
	Password      string
	Secret        string
	ClientID      string
	ClientSecret  string
	PaymentDomain string

	// Server Configuration (Optional)
	Toolsets    []string // Which toolsets to enable (empty = all)
	ReadOnly    bool     // Restrict to read-only tools only
	LogFile     string
	LogLevel    string
	Port        string
	Environment string // sandbox | production
	Transport   string // stdio | sse | http

	// Derived URLs
	APIBaseURL     string
	PaymentBaseURL string
	PayoutBaseURL  string
	OffersBaseURL  string

	// QR Preview (optional — auto-derived from Secret if not set)
	QRMasterKeyRaw string
	MCPServerURL   string
	QRLinkTTL      int
}

// LoadConfig loads configuration from environment variables.
func LoadConfig() (*Config, error) {
	cfg := &Config{
		MerchantID:    getEnv("AIRPAY_MERCHANT_ID", ""),
		Username:      getEnv("AIRPAY_USERNAME", ""),
		Password:      getEnv("AIRPAY_PASSWORD", ""),
		Secret:        getEnv("AIRPAY_SECRET", ""),
		ClientID:      getEnv("AIRPAY_CLIENT_ID", ""),
		ClientSecret:  getEnv("AIRPAY_CLIENT_SECRET", ""),
		PaymentDomain: getEnv("PAYMENT_DOMAIN", ""),
		ReadOnly:     getEnv("READ_ONLY", "false") == "true",
		LogFile:      getEnv("LOG_FILE", ""),
		LogLevel:     getEnv("LOG_LEVEL", "info"),
		Port:         getEnv("PORT", "8888"),
		Environment:    getEnv("ENVIRONMENT", "sandbox"),
		Transport:      getEnv("TRANSPORT", "stdio"),
		QRMasterKeyRaw: getEnv("QR_MASTER_KEY", ""),
		MCPServerURL:   getEnv("MCP_SERVER_URL", ""),
		QRLinkTTL:      getEnvInt("QR_LINK_TTL", 86400),
	}

	// Parse toolsets
	toolsetsStr := getEnv("TOOLSETS", "all")
	if toolsetsStr != "all" && toolsetsStr != "" {
		cfg.Toolsets = strings.Split(toolsetsStr, ",")
		for i := range cfg.Toolsets {
			cfg.Toolsets[i] = strings.TrimSpace(cfg.Toolsets[i])
		}
	}

	// Base URLs (same for sandbox and production — sandbox uses sandbox MID)
	cfg.APIBaseURL = "https://kraken.airpay.co.in"
	cfg.PaymentBaseURL = "https://payments.airpay.co.in"
	cfg.PayoutBaseURL = "https://kraken.airpay.co.in:8000"
	cfg.OffersBaseURL = "https://offers.airpay.co.in"

	// Default MCP server URL for QR preview links
	if cfg.MCPServerURL == "" {
		cfg.MCPServerURL = "http://localhost:" + cfg.Port
	}

	return cfg, cfg.validate()
}

func (c *Config) validate() error {
	if c.MerchantID == "" {
		return fmt.Errorf("AIRPAY_MERCHANT_ID is required")
	}
	if c.Username == "" {
		return fmt.Errorf("AIRPAY_USERNAME is required")
	}
	if c.Password == "" {
		return fmt.Errorf("AIRPAY_PASSWORD is required")
	}
	if c.Secret == "" {
		return fmt.Errorf("AIRPAY_SECRET is required")
	}
	if c.ClientID == "" {
		return fmt.Errorf("AIRPAY_CLIENT_ID is required")
	}
	if c.ClientSecret == "" {
		return fmt.Errorf("AIRPAY_CLIENT_SECRET is required")
	}
	if c.PaymentDomain == "" {
		return fmt.Errorf("PAYMENT_DOMAIN is required")
	}
	return nil
}

// IsToolsetEnabled checks if a given toolset is enabled.
func (c *Config) IsToolsetEnabled(toolset string) bool {
	if len(c.Toolsets) == 0 {
		return true
	}
	for _, t := range c.Toolsets {
		if t == toolset {
			return true
		}
	}
	return false
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	v := getEnv(key, "")
	if v == "" {
		return defaultValue
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultValue
	}
	return n
}
