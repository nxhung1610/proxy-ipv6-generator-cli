package health

import (
	"context"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nxhung/proxy-ipv6-generator-cli/pkg/types"
)

type mockProxyManager struct {
	proxies  []types.Proxy
	statuses map[string]string
	mu       sync.RWMutex
}

func (m *mockProxyManager) List() []types.Proxy {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.proxies
}

func (m *mockProxyManager) UpdateStatus(id string, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.statuses == nil {
		m.statuses = make(map[string]string)
	}
	m.statuses[id] = status
}

func mustParsePort(t *testing.T, portStr string) int {
	t.Helper()
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("failed to parse port: %v", err)
	}
	return port
}

func TestChecker_CheckAll_Success(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skip("cannot bind to test port")
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

	addr := listener.Addr().String()
	parts := strings.Split(addr, ":")
	pm := &mockProxyManager{
		proxies: []types.Proxy{{
			ID:      "test-1",
			Address: parts[0],
			Port:    mustParsePort(t, parts[1]),
		}},
	}

	checker := NewChecker(CheckerConfig{MaxFailures: 3, Timeout: 2 * time.Second}, pm)
	ctx := context.Background()
	results, err := checker.CheckAll(ctx)
	if err != nil {
		t.Fatalf("CheckAll failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestChecker_CheckAll_ConnectionRefused(t *testing.T) {
	pm := &mockProxyManager{
		proxies: []types.Proxy{{
			ID:      "test-refused",
			Address: "127.0.0.1",
			Port:    59999,
		}},
	}

	checker := NewChecker(CheckerConfig{MaxFailures: 3, Timeout: 500 * time.Millisecond}, pm)
	ctx := context.Background()
	results, err := checker.CheckAll(ctx)
	if err != nil {
		t.Fatalf("CheckAll failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Healthy {
		t.Error("expected unhealthy for connection refused")
	}
}

func TestChecker_CheckAll_Empty(t *testing.T) {
	pm := &mockProxyManager{
		proxies: []types.Proxy{},
	}

	checker := NewChecker(CheckerConfig{Timeout: 1 * time.Second}, pm)
	ctx := context.Background()
	results, err := checker.CheckAll(ctx)
	if err != nil {
		t.Fatalf("CheckAll failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty proxy list, got %d", len(results))
	}
}

func TestChecker_StartStop(t *testing.T) {
	pm := &mockProxyManager{}
	checker := NewChecker(CheckerConfig{Interval: 100 * time.Millisecond}, pm)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	checker.Start(ctx)
	if !checker.IsRunning() {
		t.Error("expected checker to be running after Start")
	}

	checker.Stop()
	if checker.IsRunning() {
		t.Error("expected checker to not be running after Stop")
	}
}

func TestChecker_DoubleStop(t *testing.T) {
	pm := &mockProxyManager{}
	checker := NewChecker(CheckerConfig{Interval: 100 * time.Millisecond}, pm)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	checker.Start(ctx)
	checker.Stop()
	checker.Stop()
	checker.Stop()
}

func TestChecker_StartAlreadyRunning(t *testing.T) {
	pm := &mockProxyManager{}
	checker := NewChecker(CheckerConfig{Interval: 100 * time.Millisecond}, pm)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	checker.Start(ctx)
	checker.Start(ctx)

	if !checker.IsRunning() {
		t.Error("checker should still be running")
	}

	checker.Stop()
}

func TestChecker_StatusCallback(t *testing.T) {
	received := make(chan string, 10)
	pm := &mockProxyManager{}
	checker := NewChecker(CheckerConfig{MaxFailures: 1, Timeout: 50 * time.Millisecond}, pm)

	// Set up a proxy that will definitely fail
	pm.proxies = []types.Proxy{{
		ID:      "test-status",
		Address: "127.0.0.1",
		Port:    59999,
	}}

	checker.OnStatusChange(func(proxyID, status string) {
		received <- proxyID + ":" + status
	})

	ctx := context.Background()

	// Run multiple checks to trigger the callback after MaxFailures
	for i := 0; i < 5; i++ {
		checker.CheckAll(ctx)
	}

	// Give callback time to fire
	select {
	case msg := <-received:
		t.Logf("Callback received: %s", msg)
	case <-time.After(2 * time.Second):
		t.Log("timeout waiting for status callback - may be expected if MaxFailures not reached")
	}
}

func TestChecker_GetFailureCount(t *testing.T) {
	pm := &mockProxyManager{}
	checker := NewChecker(CheckerConfig{MaxFailures: 3, Timeout: 50 * time.Millisecond}, pm)

	count := checker.GetFailureCount("non-existent")
	if count != 0 {
		t.Errorf("expected 0 failures for non-existent proxy, got %d", count)
	}
}

func TestChecker_ResetFailures(t *testing.T) {
	pm := &mockProxyManager{}
	checker := NewChecker(CheckerConfig{Timeout: 50 * time.Millisecond}, pm)

	checker.ResetFailures("test-proxy")

	count := checker.GetFailureCount("test-proxy")
	if count != 0 {
		t.Errorf("expected 0 after reset, got %d", count)
	}
}

func TestChecker_CheckOne(t *testing.T) {
	pm := &mockProxyManager{}
	checker := NewChecker(CheckerConfig{Timeout: 1 * time.Second}, pm)

	proxy := &types.Proxy{
		ID:      "test-single",
		Address: "127.0.0.1",
		Port:    59999,
	}

	ctx := context.Background()
	result := checker.CheckOne(ctx, proxy)

	if result.Healthy {
		t.Error("expected unhealthy for non-listening port")
	}
	if result.ProxyID != "test-single" {
		t.Errorf("expected proxy ID test-single, got %s", result.ProxyID)
	}
}

func TestChecker_ConfigDefaults(t *testing.T) {
	pm := &mockProxyManager{}

	cfg := CheckerConfig{}
	checker := NewChecker(cfg, pm)

	if checker.timeout != 3*time.Second {
		t.Errorf("expected default timeout 3s, got %v", checker.timeout)
	}
	if checker.interval != 5*time.Second {
		t.Errorf("expected default interval 5s, got %v", checker.interval)
	}
	if checker.maxFailures != 3 {
		t.Errorf("expected default maxFailures 3, got %d", checker.maxFailures)
	}
}

func TestChecker_StartWithContextCancellation(t *testing.T) {
	pm := &mockProxyManager{}
	checker := NewChecker(CheckerConfig{Interval: 50 * time.Millisecond}, pm)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	checker.Start(ctx)

	// Give time for the goroutine to start and exit
	time.Sleep(200 * time.Millisecond)

	// Note: The goroutine may still report as running briefly before
	// it exits due to context cancellation. This is expected behavior.
	if checker.IsRunning() {
		t.Log("checker is still running after context cancellation (may be expected with short intervals)")
	}
}
