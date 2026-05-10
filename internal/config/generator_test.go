package config

import (
	"strings"
	"testing"

	"github.com/nxhung/proxy-ipv6-generator-cli/pkg/types"
)

// =============================================================================
// Sanitization Function Tests (CRITICAL - security functions)
// =============================================================================

func TestSanitizeCredential(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"preserves valid chars", "normaluser", "normaluser"},
		{"removes newline", "user\nname", "username"},
		{"removes carriage return", "user\rname", "username"},
		{"removes both CR+LF", "user\r\nname", "username"},
		{"removes colon", "user:name", "username"},
		{"removes multiple colons", "user:name:pass", "usernamepass"},
		{"removes hash", "user#pass", "userpass"},
		{"removes semicolon", "user;pass", "userpass"},
		{"removes space", "user pass", "userpass"},
		{"removes multiple spaces", "user  pass", "userpass"},
		{"removes backslash", "user\\pass", "userpass"},
		{"removes control chars", "user\x00pass", "userpass"},
		{"removes tab", "user\tpass", "userpass"},
		{"removes all dangerous", "user\nauth\rstrong\nallow *", "userauthstrongallow*"},
		{"empty string", "", ""},
		{"only dangerous chars", ":\n\r\\#; ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeCredential(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeCredential(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSanitizeCredential_OutputNeverLongerThanInput(t *testing.T) {
	inputs := []string{
		"normaluser",
		"user\nwith\nnewlines",
		"user\r\n\r\n",
		"u:n:a:m:e:p:a:s:s",
		"user space pass",
		"user\\backslash\\pass",
	}
	for _, input := range inputs {
		got := sanitizeCredential(input)
		if len(got) > len(input) {
			t.Errorf("sanitizeCredential(%q): output longer than input: %d > %d", input, len(got), len(input))
		}
	}
}

func TestSanitizeAddress(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"preserves valid IPv6", "2001:db8::1", "2001:db8::1"},
		{"preserves full IPv6", "2001:db8:85a3::8a2e:370:7334", "2001:db8:85a3::8a2e:370:7334"},
		{"preserves loopback", "::1", "::1"},
		{"preserves unspecific", "::", "::"},
		{"removes newline", "2001:db8::1\n", "2001:db8::1"},
		{"removes CR", "2001:db8::1\r", "2001:db8::1"},
		{"removes backslash-n", "2001:db8::1\\n", "2001:db8::1n"},  // backslash removed, n kept
		{"removes backslash-r", "2001:db8::1\\r", "2001:db8::1r"},
		{"preserves colon (IPv6 separator)", "2001:db8::1:2:3:4", "2001:db8::1:2:3:4"},
		{"empty string", "", ""},
		{"only dangerous chars", "\n\r\\", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeAddress(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeAddress(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSanitizeAddress_OutputNeverLongerThanInput(t *testing.T) {
	inputs := []string{
		"2001:db8::1",
		"2001:db8::1\nwith\nnewlines",
		"::1\r\r\r",
	}
	for _, input := range inputs {
		got := sanitizeAddress(input)
		if len(got) > len(input) {
			t.Errorf("sanitizeAddress(%q): output longer than input: %d > %d", input, len(got), len(input))
		}
	}
}

func TestSanitizeDNSServer(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"preserves valid DNS", "8.8.8.8", "8.8.8.8"},
		{"preserves IPv6 DNS", "2001:4860:4860::8888", "2001:4860:4860::8888"},
		{"removes newline", "8.8.8.8\n", "8.8.8.8"},
		{"removes CR", "8.8.8.8\r", "8.8.8.8"},
		{"removes injection attempt", "8.8.8.8\nauth none", "8.8.8.8auth none"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeDNSServer(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeDNSServer(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// =============================================================================
// Generate3proxyConfig Tests
// =============================================================================

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

	cfg, err := Generate3proxyConfig(pool, proxies, 500, "8.8.8.8")
	if err != nil {
		t.Fatalf("Generate3proxyConfig failed: %v", err)
	}

	if !strings.Contains(cfg, "maxconn 500") {
		t.Error("missing maxconn directive")
	}
	if !strings.Contains(cfg, "nserver 8.8.8.8") {
		t.Error("missing nserver directive with correct value")
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

	cfg, err := Generate3proxyConfig(pool, []types.Proxy{}, 500, "8.8.8.8")
	if err != nil {
		t.Errorf("expected no error for empty proxy list, got: %v", err)
	}

	if !strings.Contains(cfg, "maxconn 500") {
		t.Error("expected maxconn 500 in output even for empty pool")
	}
	if strings.Contains(cfg, "users") {
		t.Error("should not contain 'users' directive for empty proxy list")
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

	cfg, err := Generate3proxyConfig(pool, proxies, 500, "8.8.8.8")
	if err != nil {
		t.Fatalf("Generate3proxyConfig failed: %v", err)
	}

	// Verify the malicious newline is removed from username in output
	if strings.Contains(cfg, "user\nauth") {
		t.Error("injection: newline in username should be sanitized")
	}
	if strings.Contains(cfg, "allow *") {
		t.Error("injection: 'allow *' should not appear in output")
	}
}

func TestGenerate3proxyConfig_DNSServerDefault(t *testing.T) {
	pool := types.Pool{ID: "default-dns", Prefix: "2001:db8::/64", Count: 1}
	proxies := []types.Proxy{
		{ID: "p1", Address: "2001:db8::1", Port: 10000, Username: "user1", Password: "pass1"},
	}

	cfg, err := Generate3proxyConfig(pool, proxies, 500, "")  // empty = should default
	if err != nil {
		t.Fatalf("Generate3proxyConfig failed: %v", err)
	}

	if !strings.Contains(cfg, "nserver 8.8.8.8") {
		t.Error("expected default DNS server 8.8.8.8 when dnssrv is empty")
	}
}

func TestGenerate3proxyConfig_DNSServerCustom(t *testing.T) {
	pool := types.Pool{ID: "custom-dns", Prefix: "2001:db8::/64", Count: 1}
	proxies := []types.Proxy{
		{ID: "p1", Address: "2001:db8::1", Port: 10000, Username: "user1", Password: "pass1"},
	}

	cfg, err := Generate3proxyConfig(pool, proxies, 500, "1.1.1.1")
	if err != nil {
		t.Fatalf("Generate3proxyConfig failed: %v", err)
	}

	if !strings.Contains(cfg, "nserver 1.1.1.1") {
		t.Error("expected custom DNS server 1.1.1.1 in config")
	}
}

func TestGenerate3proxyConfig_DNSServerInjectionSanitized(t *testing.T) {
	pool := types.Pool{ID: "dns-injection", Prefix: "2001:db8::/64", Count: 1}
	proxies := []types.Proxy{
		{ID: "p1", Address: "2001:db8::1", Port: 10000, Username: "user1", Password: "pass1"},
	}

	// Try injection via newline - should be sanitized
	cfg, err := Generate3proxyConfig(pool, proxies, 500, "8.8.8.8\nauth none")
	if err != nil {
		t.Fatalf("Generate3proxyConfig failed: %v", err)
	}

	// The newline should be removed, so "auth none" gets concatenated
	if strings.Contains(cfg, "\nauth none") {
		t.Error("DNS injection: newline in DNS server should be sanitized")
	}
}

func TestGenerate3proxyConfig_MaxConnDefault(t *testing.T) {
	pool := types.Pool{ID: "maxconn-test", Prefix: "2001:db8::/64", Count: 1}
	proxies := []types.Proxy{
		{ID: "p1", Address: "2001:db8::1", Port: 10000, Username: "user1", Password: "pass1"},
	}

	// Pass 0 maxConn - should default to 500
	cfg, err := Generate3proxyConfig(pool, proxies, 0, "8.8.8.8")
	if err != nil {
		t.Fatalf("Generate3proxyConfig failed: %v", err)
	}

	if !strings.Contains(cfg, "maxconn 500") {
		t.Error("expected default maxconn 500 when maxConn is 0")
	}
}

func TestGenerate3proxyConfig_MaxConnCustom(t *testing.T) {
	pool := types.Pool{ID: "conn-test", Prefix: "2001:db8::/64", Count: 1}
	proxies := []types.Proxy{
		{ID: "p1", Address: "2001:db8::1", Port: 10000, Username: "user1", Password: "pass1"},
	}

	cfg, err := Generate3proxyConfig(pool, proxies, 1000, "8.8.8.8")
	if err != nil {
		t.Fatalf("Generate3proxyConfig failed: %v", err)
	}

	if !strings.Contains(cfg, "maxconn 1000") {
		t.Error("expected custom maxconn 1000 in config")
	}
}

// =============================================================================
// PoolToProxies Tests
// =============================================================================

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
		// Address should start with 2001
		if !strings.HasPrefix(p.Address, "2001") {
			t.Errorf("proxy[%d] %s should start with 2001", i, p.Address)
		}

		// Sanitized address should not contain newlines or other dangerous characters
		if strings.ContainsAny(p.Address, "\n\r\\") {
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

// =============================================================================
// generateIPv6FromPrefix Tests
// =============================================================================

func TestGenerateIPv6FromPrefix(t *testing.T) {
	tests := []struct {
		prefix string
		index  int
		want   string
	}{
		// Function strips ALL trailing colons then appends :XXXX::1
		{"2001:db8::", 0, "2001:db8:0000::1"},  // strips both colons
		{"2001:db8::", 1, "2001:db8:0001::1"},
		{"2001:db8::", 15, "2001:db8:000f::1"},
		{"2001:db8::", 255, "2001:db8:00ff::1"},
		{"2001:db8::", 65535, "2001:db8:ffff::1"},
		// Single trailing colon stripped: "2001:db8:" → "2001:db8" + ":0000" + "::1"
		{"2001:db8:", 0, "2001:db8:0000::1"},
		// No trailing colon: "2001:db8" + ":0000" + "::1"
		{"2001:db8", 0, "2001:db8:0000::1"},
		// Full IPv6 prefix: appends extra segment
		{"2001:db8:a:b:c:d:e:f", 0, "2001:db8:a:b:c:d:e:f:0000::1"},
	}

	for _, tt := range tests {
		got := generateIPv6FromPrefix(tt.prefix, tt.index)
		if got != tt.want {
			t.Errorf("generateIPv6FromPrefix(%q, %d): got %q, want %q", tt.prefix, tt.index, got, tt.want)
		}
	}
}

// =============================================================================
// Constants Tests
// =============================================================================

func TestDefaultConstants(t *testing.T) {
	if DefaultMaxConn != 500 {
		t.Errorf("DefaultMaxConn = %d, want 500", DefaultMaxConn)
	}
	if DefaultDNSServer != "8.8.8.8" {
		t.Errorf("DefaultDNSServer = %q, want 8.8.8.8", DefaultDNSServer)
	}
	if DefaultNSCache != 65536 {
		t.Errorf("DefaultNSCache = %d, want 65536", DefaultNSCache)
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkSanitizeCredential(b *testing.B) {
	input := "user\nwith\r\nmany\ndangerous\r\nchars"
	for i := 0; i < b.N; i++ {
		sanitizeCredential(input)
	}
}

func BenchmarkGenerate3proxyConfig(b *testing.B) {
	pool := types.Pool{
		ID:       "bench-pool",
		Prefix:   "2001:db8::/64",
		BasePort: 10000,
		Count:    100,
	}
	proxies := make([]types.Proxy, 100)
	for i := 0; i < 100; i++ {
		proxies[i] = types.Proxy{
			ID:       "p",
			Address:  "2001:db8::1",
			Port:     10000 + i,
			Username: "user",
			Password: "pass",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Generate3proxyConfig(pool, proxies, 500, "8.8.8.8")
	}
}
