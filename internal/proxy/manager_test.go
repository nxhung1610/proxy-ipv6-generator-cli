package proxy

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nxhung/proxy-ipv6-generator-cli/pkg/types"
)

func TestManager_Add(t *testing.T) {
	m := NewManager()

	proxy := types.Proxy{
		ID:      "test-1",
		Address: "2001:db8::1",
		Port:    10000,
		Status:  string(types.ProxyStatusHealthy),
	}

	m.Add(proxy)

	got, ok := m.Get("test-1")
	if !ok {
		t.Fatal("expected proxy to be retrievable after Add")
	}
	if got.ID != proxy.ID {
		t.Errorf("expected ID %q, got %q", proxy.ID, got.ID)
	}
	if got.Address != proxy.Address {
		t.Errorf("expected Address %q, got %q", proxy.Address, got.Address)
	}
	if got.Port != proxy.Port {
		t.Errorf("expected Port %d, got %d", proxy.Port, got.Port)
	}
}

func TestManager_AddBatch(t *testing.T) {
	m := NewManager()

	proxies := []types.Proxy{
		{ID: "batch-1", Address: "2001:db8::1", Port: 10000},
		{ID: "batch-2", Address: "2001:db8::2", Port: 10001},
		{ID: "batch-3", Address: "2001:db8::3", Port: 10002},
	}

	m.AddBatch(proxies)

	if got := m.Count(); got != 3 {
		t.Errorf("expected Count=3, got %d", got)
	}

	// Verify each proxy is stored correctly — this is the regression test
	// for the dangling pointer bug where only the last item was reachable.
	for _, want := range proxies {
		got, ok := m.Get(want.ID)
		if !ok {
			t.Errorf("proxy %q not found after AddBatch", want.ID)
			continue
		}
		if got.Address != want.Address {
			t.Errorf("proxy %q: expected Address %q, got %q", want.ID, want.Address, got.Address)
		}
		if got.Port != want.Port {
			t.Errorf("proxy %q: expected Port %d, got %d", want.ID, want.Port, got.Port)
		}
	}
}

func TestManager_AddBatch_Large(t *testing.T) {
	m := NewManager()

	// Stress test: verify no dangling pointers with a larger batch
	proxies := make([]types.Proxy, 100)
	for i := range proxies {
		proxies[i] = types.Proxy{
			ID:      "large-batch-" + string(rune('a'+i%26)) + string(rune('0'+i/26%10)),
			Address: "2001:db8::" + string([]byte{byte((i + 1) >> 8), byte(i + 1)}),
			Port:    10000 + i,
		}
	}

	m.AddBatch(proxies)

	if got := m.Count(); got != 100 {
		t.Errorf("expected Count=100, got %d", got)
	}

	// List should return all proxies without corruption
	listed := m.List()
	if len(listed) != 100 {
		t.Errorf("expected List() to return 100 proxies, got %d", len(listed))
	}

	// Each proxy in List() should have the correct data
	found := make(map[string]bool)
	for _, p := range listed {
		found[p.ID] = true
	}
	for _, want := range proxies {
		if !found[want.ID] {
			t.Errorf("proxy %q missing from List()", want.ID)
		}
	}
}

func TestManager_Get(t *testing.T) {
	m := NewManager()
	m.Add(types.Proxy{ID: "exists", Address: "2001:db8::99", Port: 9999})

	tests := []struct {
		name    string
		id      string
		wantOk  bool
		wantID  string
	}{
		{"existing proxy", "exists", true, "exists"},
		{"non-existent proxy", "missing", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := m.Get(tt.id)
			if ok != tt.wantOk {
				t.Errorf("Get() ok = %v, want %v", ok, tt.wantOk)
			}
			if tt.wantOk && got.ID != tt.wantID {
				t.Errorf("Get() returned ID %q, want %q", got.ID, tt.wantID)
			}
		})
	}
}

func TestManager_List(t *testing.T) {
	m := NewManager()

	// Empty manager
	if got := m.List(); len(got) != 0 {
		t.Errorf("expected empty list, got %d items", len(got))
	}

	// Add proxies
	for i := 1; i <= 5; i++ {
		m.Add(types.Proxy{ID: "list-" + string(rune('0'+i)), Address: "2001:db8::1", Port: 10000 + i})
	}

	listed := m.List()
	if len(listed) != 5 {
		t.Errorf("expected 5 proxies, got %d", len(listed))
	}

	// All returned proxies should be complete copies (no pointers)
	for _, p := range listed {
		if p.ID == "" || p.Address == "" {
			t.Errorf("List() returned proxy with missing data: ID=%q Address=%q", p.ID, p.Address)
		}
	}
}

func TestManager_Remove(t *testing.T) {
	m := NewManager()
	m.Add(types.Proxy{ID: "to-remove"})

	if !m.Remove("to-remove") {
		t.Error("expected Remove to return true for existing proxy")
	}
	if m.Count() != 0 {
		t.Errorf("expected Count=0 after Remove, got %d", m.Count())
	}
	if _, ok := m.Get("to-remove"); ok {
		t.Error("proxy should not be retrievable after Remove")
	}

	// Remove non-existent should return false
	if m.Remove("missing") {
		t.Error("expected Remove to return false for non-existent proxy")
	}
}

func TestManager_UpdateStatus(t *testing.T) {
	m := NewManager()
	m.Add(types.Proxy{ID: "status-test", Status: string(types.ProxyStatusHealthy)})

	m.UpdateStatus("status-test", string(types.ProxyStatusUnhealthy))

	got, ok := m.Get("status-test")
	if !ok {
		t.Fatal("proxy not found")
	}
	if got.Status != string(types.ProxyStatusUnhealthy) {
		t.Errorf("expected status %q, got %q", string(types.ProxyStatusUnhealthy), got.Status)
	}
}

func TestManager_Count(t *testing.T) {
	m := NewManager()

	if got := m.Count(); got != 0 {
		t.Errorf("empty manager: expected Count=0, got %d", got)
	}

	m.Add(types.Proxy{ID: "c1"})
	if got := m.Count(); got != 1 {
		t.Errorf("expected Count=1, got %d", got)
	}

	m.Add(types.Proxy{ID: "c2"})
	if got := m.Count(); got != 2 {
		t.Errorf("expected Count=2, got %d", got)
	}

	m.Remove("c1")
	if got := m.Count(); got != 1 {
		t.Errorf("expected Count=1 after remove, got %d", got)
	}
}

func TestManager_CountByStatus(t *testing.T) {
	m := NewManager()

	m.Add(types.Proxy{ID: "s1", Status: string(types.ProxyStatusHealthy)})
	m.Add(types.Proxy{ID: "s2", Status: string(types.ProxyStatusHealthy)})
	m.Add(types.Proxy{ID: "s3", Status: string(types.ProxyStatusUnhealthy)})

	if got := m.CountByStatus(string(types.ProxyStatusHealthy)); got != 2 {
		t.Errorf("expected 2 healthy, got %d", got)
	}
	if got := m.CountByStatus(string(types.ProxyStatusUnhealthy)); got != 1 {
		t.Errorf("expected 1 unhealthy, got %d", got)
	}
	if got := m.CountByStatus("unknown"); got != 0 {
		t.Errorf("expected 0 for unknown status, got %d", got)
	}
}

func TestManager_Concurrent(t *testing.T) {
	m := NewManager()
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			m.Add(types.Proxy{ID: fmt.Sprintf("concurrent-%d", id), Port: 10000 + id})
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_ = m.Count()
			_ = m.List()
			_, _ = m.Get(fmt.Sprintf("concurrent-%d", id))
		}(i)
	}

	wg.Wait()

	// Should have 50 proxies — no duplicates or lost writes
	if got := m.Count(); got != 50 {
		t.Errorf("expected Count=50 after concurrent access, got %d", got)
	}
}

func TestNewPool(t *testing.T) {
	pm, err := NewPool("2001:db8::/64", 10000, 10, types.ProtocolSOCKS5)
	if err != nil {
		t.Fatalf("NewPool failed: %v", err)
	}

	if got := pm.Count(); got != 10 {
		t.Errorf("expected 10 proxies in pool, got %d", got)
	}

	// All proxies should have correct protocol
	for _, p := range pm.List() {
		if p.Protocol != types.ProtocolSOCKS5 {
			t.Errorf("expected Protocol=SOCKS5, got %v", p.Protocol)
		}
	}

	// Verify no dangling pointers — each proxy in List() should be individually retrievable
	listed := pm.List()
	for _, p := range listed {
		got, ok := pm.Get(p.ID)
		if !ok {
			t.Errorf("proxy %q from List() not retrievable via Get()", p.ID)
			continue
		}
		if got.Address != p.Address {
			t.Errorf("proxy %q: expected Address %q, got %q", p.ID, p.Address, got.Address)
		}
		if got.Port != p.Port {
			t.Errorf("proxy %q: expected Port %d, got %d", p.ID, p.Port, got.Port)
		}
	}
}

func TestManager_CheckHealth_Success(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skip("cannot bind test port")
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	m := NewManager()
	addr := listener.Addr().String()
	parts := strings.Split(addr, ":")
	port, _ := strconv.Atoi(parts[1])

	m.Add(types.Proxy{
		ID:      "health-test",
		Address: parts[0],
		Port:    port,
	})

	ctx := context.Background()
	timeout := 2 * time.Second
	results := m.CheckHealth(ctx, timeout)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Healthy {
		t.Error("expected healthy for listening port")
	}
}

func TestManager_CheckHealth_ConnectionRefused(t *testing.T) {
	m := NewManager()
	m.Add(types.Proxy{
		ID:      "refused",
		Address: "127.0.0.1",
		Port:    59999,
	})

	ctx := context.Background()
	results := m.CheckHealth(ctx, 500*time.Millisecond)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Healthy {
		t.Error("expected unhealthy for connection refused")
	}
}

func TestManager_CheckHealth_Cancellation(t *testing.T) {
	m := NewManager()
	m.Add(types.Proxy{
		ID:      "slow",
		Address: "10.255.255.1",
		Port:    80,
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	results := m.CheckHealth(ctx, 10*time.Second)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestManager_UpdateStatus_NonExistent(t *testing.T) {
	m := NewManager()
	m.UpdateStatus("non-existent", string(types.ProxyStatusUnhealthy))
	if m.Count() != 0 {
		t.Error("UpdateStatus should not create proxy")
	}
}

func TestManager_CheckHealth_Empty(t *testing.T) {
	m := NewManager()

	ctx := context.Background()
	results := m.CheckHealth(ctx, 1*time.Second)

	if len(results) != 0 {
		t.Errorf("expected 0 results for empty manager, got %d", len(results))
	}
}

func TestManager_CheckHealth_IPv6(t *testing.T) {
	if !supportsIPv6() {
		t.Skip("IPv6 not supported on this system")
	}

	listener, err := net.Listen("tcp", "[::1]:0")
	if err != nil {
		t.Skip("cannot bind IPv6 test port")
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	m := NewManager()
	addr := listener.Addr().String()
	m.Add(types.Proxy{
		ID:      "ipv6-health",
		Address: "::1",
		Port:    portFromAddr(addr),
	})

	ctx := context.Background()
	results := m.CheckHealth(ctx, 2*time.Second)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func portFromAddr(addr string) int {
	parts := strings.Split(addr, ":")
	last := parts[len(parts)-1]
	port, _ := strconv.Atoi(last)
	return port
}

func supportsIPv6() bool {
	conn, err := net.Listen("tcp", "[::1]:0")
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
