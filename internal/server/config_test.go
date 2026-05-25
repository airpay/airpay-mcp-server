package server

import (
	"os"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// LoadConfig — missing required env vars
// ---------------------------------------------------------------------------

func TestLoadConfig_MissingMerchantID_ReturnsError(t *testing.T) {
	t.Setenv("AIRPAY_MERCHANT_ID", "")
	t.Setenv("AIRPAY_USERNAME", "user")
	t.Setenv("AIRPAY_PASSWORD", "pass")
	t.Setenv("AIRPAY_SECRET", "sec")
	t.Setenv("AIRPAY_CLIENT_ID", "cid")
	t.Setenv("AIRPAY_CLIENT_SECRET", "csec")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for missing AIRPAY_MERCHANT_ID, got nil")
	}
	if !strings.Contains(err.Error(), "AIRPAY_MERCHANT_ID") {
		t.Errorf("error message should mention AIRPAY_MERCHANT_ID, got: %v", err)
	}
}

func TestLoadConfig_MissingUsername_ReturnsError(t *testing.T) {
	t.Setenv("AIRPAY_MERCHANT_ID", "12345")
	t.Setenv("AIRPAY_USERNAME", "")
	t.Setenv("AIRPAY_PASSWORD", "pass")
	t.Setenv("AIRPAY_SECRET", "sec")
	t.Setenv("AIRPAY_CLIENT_ID", "cid")
	t.Setenv("AIRPAY_CLIENT_SECRET", "csec")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for missing AIRPAY_USERNAME, got nil")
	}
	if !strings.Contains(err.Error(), "AIRPAY_USERNAME") {
		t.Errorf("error message should mention AIRPAY_USERNAME, got: %v", err)
	}
}

func TestLoadConfig_MissingPassword_ReturnsError(t *testing.T) {
	t.Setenv("AIRPAY_MERCHANT_ID", "12345")
	t.Setenv("AIRPAY_USERNAME", "user")
	t.Setenv("AIRPAY_PASSWORD", "")
	t.Setenv("AIRPAY_SECRET", "sec")
	t.Setenv("AIRPAY_CLIENT_ID", "cid")
	t.Setenv("AIRPAY_CLIENT_SECRET", "csec")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for missing AIRPAY_PASSWORD, got nil")
	}
	if !strings.Contains(err.Error(), "AIRPAY_PASSWORD") {
		t.Errorf("error message should mention AIRPAY_PASSWORD, got: %v", err)
	}
}

func TestLoadConfig_MissingSecret_ReturnsError(t *testing.T) {
	t.Setenv("AIRPAY_MERCHANT_ID", "12345")
	t.Setenv("AIRPAY_USERNAME", "user")
	t.Setenv("AIRPAY_PASSWORD", "pass")
	t.Setenv("AIRPAY_SECRET", "")
	t.Setenv("AIRPAY_CLIENT_ID", "cid")
	t.Setenv("AIRPAY_CLIENT_SECRET", "csec")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for missing AIRPAY_SECRET, got nil")
	}
	if !strings.Contains(err.Error(), "AIRPAY_SECRET") {
		t.Errorf("error message should mention AIRPAY_SECRET, got: %v", err)
	}
}

func TestLoadConfig_MissingClientID_ReturnsError(t *testing.T) {
	t.Setenv("AIRPAY_MERCHANT_ID", "12345")
	t.Setenv("AIRPAY_USERNAME", "user")
	t.Setenv("AIRPAY_PASSWORD", "pass")
	t.Setenv("AIRPAY_SECRET", "sec")
	t.Setenv("AIRPAY_CLIENT_ID", "")
	t.Setenv("AIRPAY_CLIENT_SECRET", "csec")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for missing AIRPAY_CLIENT_ID, got nil")
	}
	if !strings.Contains(err.Error(), "AIRPAY_CLIENT_ID") {
		t.Errorf("error message should mention AIRPAY_CLIENT_ID, got: %v", err)
	}
}

func TestLoadConfig_MissingClientSecret_ReturnsError(t *testing.T) {
	t.Setenv("AIRPAY_MERCHANT_ID", "12345")
	t.Setenv("AIRPAY_USERNAME", "user")
	t.Setenv("AIRPAY_PASSWORD", "pass")
	t.Setenv("AIRPAY_SECRET", "sec")
	t.Setenv("AIRPAY_CLIENT_ID", "cid")
	t.Setenv("AIRPAY_CLIENT_SECRET", "")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for missing AIRPAY_CLIENT_SECRET, got nil")
	}
	if !strings.Contains(err.Error(), "AIRPAY_CLIENT_SECRET") {
		t.Errorf("error message should mention AIRPAY_CLIENT_SECRET, got: %v", err)
	}
}

func TestLoadConfig_MissingPaymentDomain_ReturnsError(t *testing.T) {
	t.Setenv("AIRPAY_MERCHANT_ID", "12345")
	t.Setenv("AIRPAY_USERNAME", "user")
	t.Setenv("AIRPAY_PASSWORD", "pass")
	t.Setenv("AIRPAY_SECRET", "sec")
	t.Setenv("AIRPAY_CLIENT_ID", "cid")
	t.Setenv("AIRPAY_CLIENT_SECRET", "csec")
	t.Setenv("PAYMENT_DOMAIN", "")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for missing PAYMENT_DOMAIN, got nil")
	}
	if !strings.Contains(err.Error(), "PAYMENT_DOMAIN") {
		t.Errorf("error message should mention PAYMENT_DOMAIN, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// LoadConfig — success with all required env vars
// ---------------------------------------------------------------------------

// setRequiredEnv is a helper that sets all required env vars for LoadConfig.
func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("AIRPAY_MERCHANT_ID", "mid-001")
	t.Setenv("AIRPAY_USERNAME", "user")
	t.Setenv("AIRPAY_PASSWORD", "pass")
	t.Setenv("AIRPAY_SECRET", "sec")
	t.Setenv("AIRPAY_CLIENT_ID", "cid")
	t.Setenv("AIRPAY_CLIENT_SECRET", "csec")
	t.Setenv("PAYMENT_DOMAIN", "https://merchant.example.com")
}

func TestLoadConfig_SuccessWithAllRequired(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil Config")
	}
	if cfg.MerchantID != "mid-001" {
		t.Errorf("MerchantID = %q, want mid-001", cfg.MerchantID)
	}
}

// ---------------------------------------------------------------------------
// LoadConfig — default values
// ---------------------------------------------------------------------------

func TestLoadConfig_DefaultPort(t *testing.T) {
	setRequiredEnv(t)
	os.Unsetenv("PORT")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "8888" {
		t.Errorf("default Port = %q, want 8888", cfg.Port)
	}
}

func TestLoadConfig_DefaultTransport(t *testing.T) {
	setRequiredEnv(t)
	os.Unsetenv("TRANSPORT")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Transport != "stdio" {
		t.Errorf("default Transport = %q, want stdio", cfg.Transport)
	}
}

func TestLoadConfig_DefaultEnvironment(t *testing.T) {
	setRequiredEnv(t)
	os.Unsetenv("ENVIRONMENT")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Environment != "sandbox" {
		t.Errorf("default Environment = %q, want sandbox", cfg.Environment)
	}
}

func TestLoadConfig_DefaultToolsets_EnablesAll(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("TOOLSETS", "")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty TOOLSETS means all toolsets enabled (Toolsets slice is nil/empty).
	if len(cfg.Toolsets) != 0 {
		t.Errorf("default Toolsets should be empty (all enabled), got %v", cfg.Toolsets)
	}
}

func TestLoadConfig_DefaultReadOnly_IsFalse(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("READ_ONLY", "")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ReadOnly {
		t.Error("default ReadOnly should be false")
	}
}

// ---------------------------------------------------------------------------
// LoadConfig — custom values
// ---------------------------------------------------------------------------

func TestLoadConfig_CustomPort(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("PORT", "9090")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want 9090", cfg.Port)
	}
}

func TestLoadConfig_CustomTransportHTTP(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("TRANSPORT", "http")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Transport != "http" {
		t.Errorf("Transport = %q, want http", cfg.Transport)
	}
}

func TestLoadConfig_CustomTransportSSE(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("TRANSPORT", "sse")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Transport != "sse" {
		t.Errorf("Transport = %q, want sse", cfg.Transport)
	}
}

func TestLoadConfig_ReadOnlyTrue(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("READ_ONLY", "true")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.ReadOnly {
		t.Error("ReadOnly should be true when READ_ONLY=true")
	}
}

// ---------------------------------------------------------------------------
// LoadConfig — URL derivation
// Notes: The implementation uses fixed base URLs regardless of environment.
// The environment setting uses sandbox MID rather than separate base URLs.
// ---------------------------------------------------------------------------

func TestLoadConfig_BaseURLsAlwaysSet(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("ENVIRONMENT", "sandbox")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIBaseURL == "" {
		t.Error("APIBaseURL should not be empty")
	}
	if cfg.PaymentBaseURL == "" {
		t.Error("PaymentBaseURL should not be empty")
	}
	if cfg.PayoutBaseURL == "" {
		t.Error("PayoutBaseURL should not be empty")
	}
	if cfg.OffersBaseURL == "" {
		t.Error("OffersBaseURL should not be empty")
	}
}

func TestLoadConfig_SandboxEnvironment_APIBaseURL(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("ENVIRONMENT", "sandbox")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(cfg.APIBaseURL, "airpay.co.in") {
		t.Errorf("APIBaseURL %q does not contain expected domain", cfg.APIBaseURL)
	}
}

func TestLoadConfig_ProductionEnvironment_APIBaseURL(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("ENVIRONMENT", "production")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(cfg.APIBaseURL, "airpay.co.in") {
		t.Errorf("APIBaseURL %q does not contain expected domain", cfg.APIBaseURL)
	}
}

// ---------------------------------------------------------------------------
// IsToolsetEnabled
// ---------------------------------------------------------------------------

func TestIsToolsetEnabled_EmptyToolsets_ReturnsTrueForAll(t *testing.T) {
	cfg := &Config{Toolsets: []string{}}

	cases := []string{"payments", "refunds", "payouts", "offers", "anything"}
	for _, tc := range cases {
		if !cfg.IsToolsetEnabled(tc) {
			t.Errorf("IsToolsetEnabled(%q) = false, want true when Toolsets is empty", tc)
		}
	}
}

func TestIsToolsetEnabled_AllKeyword_TreatedAsEmpty(t *testing.T) {
	// When TOOLSETS=all, the Toolsets slice is kept empty (meaning all enabled).
	setRequiredEnv(t)
	t.Setenv("TOOLSETS", "all")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Toolsets) != 0 {
		t.Errorf("Toolsets should be empty for TOOLSETS=all, got %v", cfg.Toolsets)
	}
	if !cfg.IsToolsetEnabled("payments") {
		t.Error("IsToolsetEnabled should return true for all toolsets when TOOLSETS=all")
	}
}

func TestIsToolsetEnabled_SingleToolset_MatchReturnsTrue(t *testing.T) {
	cfg := &Config{Toolsets: []string{"payments"}}

	if !cfg.IsToolsetEnabled("payments") {
		t.Error("IsToolsetEnabled(payments) should return true")
	}
}

func TestIsToolsetEnabled_SingleToolset_NoMatchReturnsFalse(t *testing.T) {
	cfg := &Config{Toolsets: []string{"payments"}}

	if cfg.IsToolsetEnabled("refunds") {
		t.Error("IsToolsetEnabled(refunds) should return false when only payments is enabled")
	}
}

func TestIsToolsetEnabled_CommaSeparatedList_MatchReturnsTrue(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("TOOLSETS", "payments,refunds,payouts")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, ts := range []string{"payments", "refunds", "payouts"} {
		if !cfg.IsToolsetEnabled(ts) {
			t.Errorf("IsToolsetEnabled(%q) = false, want true", ts)
		}
	}
}

func TestIsToolsetEnabled_CommaSeparatedList_NoMatchReturnsFalse(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("TOOLSETS", "payments,refunds")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.IsToolsetEnabled("payouts") {
		t.Error("IsToolsetEnabled(payouts) should return false when not in list")
	}
	if cfg.IsToolsetEnabled("offers") {
		t.Error("IsToolsetEnabled(offers) should return false when not in list")
	}
}

func TestIsToolsetEnabled_ToolsetsWithSpaces_TrimsCorrectly(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("TOOLSETS", " payments , refunds ")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cfg.IsToolsetEnabled("payments") {
		t.Error("IsToolsetEnabled(payments) should return true after trimming spaces")
	}
	if !cfg.IsToolsetEnabled("refunds") {
		t.Error("IsToolsetEnabled(refunds) should return true after trimming spaces")
	}
}
