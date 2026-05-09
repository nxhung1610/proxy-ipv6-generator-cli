package generator

import (
	"net"
	"strings"
	"sync"
	"testing"
)

func TestNew_ValidPrefix(t *testing.T) {
	g, err := New("2001:db8::/64", 10000, 10)
	if err != nil {
		t.Fatalf("expected valid prefix, got: %v", err)
	}
	if g.prefix.String() != "2001:db8::/64" {
		t.Errorf("prefix mismatch: got %s", g.prefix.String())
	}
}

func TestNew_InvalidPrefix(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"IPv4", "192.168.1.0/24", true},
		{"Too wide prefix", "2001:db8::/48", true},
		{"Malformed", "not-an-address", true},
		{"Empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.input, 10000, 10)
			if (err != nil) != tt.wantErr {
				t.Errorf("New(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestGenerate_ValidAddresses(t *testing.T) {
	g, err := New("2001:db8::/64", 10000, 10)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	proxies, err := g.Generate(5)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if len(proxies) != 5 {
		t.Errorf("expected 5 proxies, got %d", len(proxies))
	}

	for i, p := range proxies {
		ip := net.ParseIP(p.Address)
		if ip == nil {
			t.Errorf("proxy[%d].Address is not a valid IP: %s", i, p.Address)
		}
		if !strings.Contains(p.Address, "2001:db8::") {
			t.Errorf("proxy[%d] address %s not in prefix 2001:db8::/64", i, p.Address)
		}
	}
}

func TestGenerate_UniqueAddresses(t *testing.T) {
	g, err := New("2001:db8::/64", 10000, 10)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	proxies, err := g.Generate(100)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	seen := make(map[string]bool)
	for _, p := range proxies {
		if seen[p.Address] {
			t.Errorf("duplicate address: %s", p.Address)
		}
		seen[p.Address] = true
	}
}

func TestGenerate_SkipsNetworkAndLoopback(t *testing.T) {
	g, err := New("2001:db8::/64", 10000, 10000)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	proxies, err := g.Generate(10000)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Generate indices 0-9999, skip index 0 (::) and index 1 (::1)
	for i, p := range proxies {
		idx := i + 2 // Start from index 2 to skip :: and ::1
		if idx == 0 && strings.HasSuffix(p.Address, "::") {
			t.Errorf("Generate produced reserved address (network): %s", p.Address)
		}
		if idx == 1 && strings.HasSuffix(p.Address, "::1") {
			t.Errorf("Generate produced reserved address (loopback): %s", p.Address)
		}
	}
}

func TestGenerate_ZeroCount(t *testing.T) {
	g, err := New("2001:db8::/64", 10000, 10)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	proxies, err := g.Generate(0)
	if err != nil {
		t.Fatalf("Generate with count=0 should use internal count: %v", err)
	}
	if len(proxies) != 10 {
		t.Errorf("expected 10 proxies with count=0 (using default), got %d", len(proxies))
	}
}

func TestGenerate_NegativeCount(t *testing.T) {
	g, err := New("2001:db8::/64", 10000, 5)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	proxies, err := g.Generate(-1)
	if err != nil {
		t.Fatalf("Generate with negative count should use internal count: %v", err)
	}
	if len(proxies) != 5 {
		t.Errorf("expected 5 proxies with negative count (using default), got %d", len(proxies))
	}
}

func TestGenerate_Concurrent(t *testing.T) {
	g, err := New("2001:db8::/64", 10000, 10)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = g.Generate(10)
		}()
	}
	wg.Wait()
}

func TestGenerate_PrefixTooWide(t *testing.T) {
	_, err := New("2001:db8::/48", 10000, 10)
	if err == nil {
		t.Error("expected error for prefix wider than /64")
	}
}

func TestGenerate_LargeBatch(t *testing.T) {
	g, err := New("2001:db8::/64", 10000, 10)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	proxies, err := g.Generate(1000)
	if err != nil {
		t.Fatalf("Generate 1000 proxies failed: %v", err)
	}
	if len(proxies) != 1000 {
		t.Errorf("expected 1000 proxies, got %d", len(proxies))
	}

	seen := make(map[string]bool)
	for _, p := range proxies {
		if seen[p.Address] {
			t.Errorf("duplicate in large batch: %s", p.Address)
		}
		seen[p.Address] = true
	}
}

func TestParsePrefix_Valid(t *testing.T) {
	result, err := ParsePrefix("2001:db8::/64")
	if err != nil {
		t.Fatalf("ParsePrefix failed: %v", err)
	}
	if result != "2001:db8::/64" {
		t.Errorf("expected 2001:db8::/64, got %s", result)
	}
}

func TestParsePrefix_Invalid(t *testing.T) {
	_, err := ParsePrefix("invalid")
	if err == nil {
		t.Error("expected error for invalid prefix")
	}
}

func TestValidatePrefix_ValidIPv6(t *testing.T) {
	err := ValidatePrefix("2001:db8::/64")
	if err != nil {
		t.Errorf("ValidatePrefix should accept valid IPv6: %v", err)
	}
}

func TestValidatePrefix_InvalidIPv4(t *testing.T) {
	err := ValidatePrefix("192.168.1.0/24")
	if err == nil {
		t.Error("expected error for IPv4 prefix")
	}
}

func TestValidatePrefix_InvalidAddress(t *testing.T) {
	err := ValidatePrefix("not-an-address")
	if err == nil {
		t.Error("expected error for malformed address")
	}
}

func BenchmarkGenerate_10K(b *testing.B) {
	g, _ := New("2001:db8::/64", 10000, 10)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = g.Generate(10000) //nolint:errcheck
	}
}
