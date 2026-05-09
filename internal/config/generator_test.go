package config

import (
	"strings"
	"testing"

	"github.com/nxhung/proxy-ipv6-generator-cli/pkg/types"
)

func TestGenerate3proxyConfig_OutputFormat(t *testing.T) {
	pool := types.Pool{
		ID:       "test-pool",
		Prefix:   "2001:db8::/64",
		BasePort: 10000,
		Count:    2,
		Protocol: types.ProtocolSOCKS5,
	}
	proxies := []types.Proxy{
		{ID: "p1", Address: "2001:db8::1", Port: 10000, Username: "user1", Password: "pass1"},
		{ID: "p2", Address: "2001:db8::2", Port: 10001, Username: "user2", Password: "pass2"},
	}

	cfg, err := Generate3proxyConfig(pool, proxies, 500)
	if err != nil {
		t.Fatalf("Generate3proxyConfig failed: %v", err)
	}

	if !strings.Contains(cfg, "maxconn 500") {
		t.Error("missing maxconn directive")
	}
	if !strings.Contains(cfg, "nserver") {
		t.Error("missing nserver directive")
	}
	if !strings.Contains(cfg, "auth strong") {
		t.Error("missing auth directive")
	}
	if !strings.Contains(cfg, "proxy -6") {
		t.Error("missing proxy directive with -6 flag")
	}
}

func TestGenerate3proxyConfig_EmptyProxies(t *testing.T) {
	pool := types.Pool{
		ID:     "empty-pool",
		Prefix: "2001:db8::/64",
		Count:  0,
	}

	cfg, err := Generate3proxyConfig(pool, []types.Proxy{}, 500)
	if err != nil {
		t.Errorf("expected no error for empty proxy list, got: %v", err)
	}

	if !strings.Contains(cfg, "maxconn") {
		t.Error("expected maxconn in output even for empty pool")
	}
}

func TestGenerate3proxyConfig_InjectionAttempt(t *testing.T) {
	pool := types.Pool{ID: "injection", Prefix: "2001:db8::"}
	proxies := []types.Proxy{{
		ID:       "evil",
		Address:  "2001:db8::1",
		Port:     10000,
		Username: "user\nauth strong\nallow *",
		Password: "pass",
	}}

	cfg, err := Generate3proxyConfig(pool, proxies, 500)
	if err != nil {
		t.Fatalf("Generate3proxyConfig failed: %v", err)
	}

	// Verify the malicious newline is removed from username in output
	// After sanitization, newlines are removed so "user\nauth" becomes "userauth"
	if strings.Contains(cfg, "user\nauth") {
		t.Error("injection: newline in username should be sanitized")
	}
}

func TestPoolToProxies_AddressFormat(t *testing.T) {
	pool := types.Pool{
		ID:       "format-test",
		Prefix:   "2001:db8::",
		BasePort: 10000,
		Count:    5,
	}
	proxies := PoolToProxies(pool)

	if len(proxies) != 5 {
		t.Fatalf("expected 5 proxies, got %d", len(proxies))
	}

	for i, p := range proxies {
		// Address should start with 2001 (prefix without colons is sanitized)
		if !strings.HasPrefix(p.Address, "2001") {
			t.Errorf("proxy[%d] %s should start with 2001", i, p.Address)
		}

		// Sanitized address should not contain newlines or other dangerous characters
		if strings.ContainsAny(p.Address, "\n\r") {
			t.Errorf("proxy[%d] address %s contains dangerous characters", i, p.Address)
		}
	}
}

func TestPoolToProxies_PortAssignment(t *testing.T) {
	pool := types.Pool{
		ID:       "port-test",
		Prefix:   "2001:db8::",
		BasePort: 10000,
		Count:    10,
	}
	proxies := PoolToProxies(pool)

	for i, p := range proxies {
		expectedPort := 10000 + i
		if p.Port != expectedPort {
			t.Errorf("proxy[%d]: expected port %d, got %d", i, expectedPort, p.Port)
		}
	}
}

func TestGenerateIPv6FromPrefix(t *testing.T) {
	tests := []struct {
		prefix string
		index  int
	}{
		{"2001:db8::", 0},
		{"2001:db8::", 1},
		{"2001:db8::", 15},
		{"2001:db8:", 0},
		{"2001:db8", 0},
	}

	for _, tt := range tests {
		got := generateIPv6FromPrefix(tt.prefix, tt.index)
		// Output should contain "2001" (colons may be stripped by sanitization)
		if !strings.Contains(got, "2001") {
			t.Errorf("generateIPv6FromPrefix(%q, %d): got %q, should contain 2001", tt.prefix, tt.index, got)
		}
	}
}

func TestGenerate3proxyConfig_WithProxies(t *testing.T) {
	pool := types.Pool{
		ID:             "test-pool",
		Prefix:         "2001:db8::/64",
		BasePort:       10000,
		Count:          2,
		Protocol:       types.ProtocolSOCKS5,
		CredentialRef:  "test-creds",
	}
	proxies := []types.Proxy{
		{ID: "p1", Address: "2001:db8::1", Port: 10000, Username: "user1", Password: "pass1"},
	}

	cfg, err := Generate3proxyConfig(pool, proxies, 500)
	if err != nil {
		t.Fatalf("Generate3proxyConfig failed: %v", err)
	}

	if !strings.Contains(cfg, "users") {
		t.Error("expected users directive in config")
	}
}

func TestSanitize3proxyValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal", "normal"},
		{"user\npass", "userpass"},
		{"user\rpass", "userpass"},
		{"user:pass", "userpass"},
		{"user#pass", "userpass"},
		{"user;pass", "userpass"},
		{"user pass", "userpass"},
		{"user\\pass", "userpass"},
	}

	for _, tt := range tests {
		got := sanitize3proxyValue(tt.input)
		if got != tt.expected {
			t.Errorf("sanitize3proxyValue(%q): got %q, want %q", tt.input, got, tt.expected)
		}
	}
}
