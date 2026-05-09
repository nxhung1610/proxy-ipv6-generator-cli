package pool

import (
	"context"
	"testing"

	"github.com/nxhung/proxy-ipv6-generator-cli/internal/platform"
	"github.com/nxhung/proxy-ipv6-generator-cli/pkg/types"
)

func TestManager_Create(t *testing.T) {
	m := NewManager(nil)

	cfg := types.PoolConfig{
		Name:   "test-pool",
		Prefix: "2001:db8::/64",
		Ports:  "10000-10099",
	}

	p, err := m.Create(cfg, platform.NewThreeProxyServer(platform.NewProcessManager()))
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if p.Name != "test-pool" {
		t.Errorf("expected name %q, got %q", "test-pool", p.Name)
	}
	if p.Prefix != "2001:db8::/64" {
		t.Errorf("expected prefix %q, got %q", "2001:db8::/64", p.Prefix)
	}
	if p.BasePort != 10000 {
		t.Errorf("expected BasePort=10000, got %d", p.BasePort)
	}
	if p.Count != 100 {
		t.Errorf("expected Count=100, got %d", p.Count)
	}
	if p.Status != string(types.PoolStatusStopped) {
		t.Errorf("expected status %q, got %q", string(types.PoolStatusStopped), p.Status)
	}
	if p.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestManager_CreateThenGetByName(t *testing.T) {
	m := NewManager(nil)
	server := platform.NewThreeProxyServer(platform.NewProcessManager())

	m.Create(types.PoolConfig{Name: "find-me", Prefix: "2001:db8::/64", Ports: "10000-10010"}, server)
	m.Create(types.PoolConfig{Name: "other", Prefix: "2001:db8:1::/64", Ports: "20000-20005"}, server)

	got, ok := m.GetByName("find-me")
	if !ok {
		t.Fatal("GetByName returned false for existing pool")
	}
	if got.Prefix != "2001:db8::/64" {
		t.Errorf("expected prefix %q, got %q", "2001:db8::/64", got.Prefix)
	}
}

func TestManager_List(t *testing.T) {
	m := NewManager(nil)
	server := platform.NewThreeProxyServer(platform.NewProcessManager())

	if pools := m.List(); len(pools) != 0 {
		t.Errorf("empty manager should return empty list, got %d", len(pools))
	}

	m.Create(types.PoolConfig{Name: "pool-1", Prefix: "2001:db8::/64", Ports: "10000-10004"}, server)
	m.Create(types.PoolConfig{Name: "pool-2", Prefix: "2001:db8:1::/64", Ports: "20000-20009"}, server)

	pools := m.List()
	if len(pools) != 2 {
		t.Errorf("expected 2 pools, got %d", len(pools))
	}

	names := make(map[string]bool)
	for _, p := range pools {
		names[p.Name] = true
	}
	if !names["pool-1"] || !names["pool-2"] {
		t.Errorf("expected both pools in List(), got %v", names)
	}
}

func TestManager_Remove(t *testing.T) {
	m := NewManager(nil)
	server := platform.NewThreeProxyServer(platform.NewProcessManager())

	p, _ := m.Create(types.PoolConfig{Name: "to-remove", Prefix: "2001:db8::/64", Ports: "10000-10000"}, server)
	if !m.Remove(p.ID) {
		t.Error("expected Remove to return true for existing pool")
	}
	if _, ok := m.Get(p.ID); ok {
		t.Error("pool should not be retrievable after Remove")
	}
	if m.Count() != 0 {
		t.Errorf("expected Count=0, got %d", m.Count())
	}
}

func TestManager_Start(t *testing.T) {
	m := NewManager(nil)
	server := platform.NewThreeProxyServer(platform.NewProcessManager())

	p, _ := m.Create(types.PoolConfig{Name: "running-pool", Prefix: "2001:db8::/64", Ports: "10000-10009"}, server)

	err := m.Start(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	got, _ := m.Get(p.ID)
	if got.Status != string(types.PoolStatusRunning) {
		t.Errorf("expected status %q, got %q", string(types.PoolStatusRunning), got.Status)
	}
	if got.startedAt.IsZero() {
		t.Error("expected startedAt to be set")
	}

	// Start should create proxies in the proxyStore
	if got.proxyStore == nil {
		t.Fatal("proxyStore should not be nil after Start")
	}
	if got.proxyStore.Count() != 10 {
		t.Errorf("expected 10 proxies, got %d", got.proxyStore.Count())
	}
}

func TestManager_StartThenStop(t *testing.T) {
	m := NewManager(nil)
	server := platform.NewThreeProxyServer(platform.NewProcessManager())

	p, _ := m.Create(types.PoolConfig{Name: "stop-test", Prefix: "2001:db8::/64", Ports: "10000-10002"}, server)

	if err := m.Start(context.Background(), p.ID); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if err := m.Stop(context.Background(), p.ID); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	got, _ := m.Get(p.ID)
	if got.Status != string(types.PoolStatusStopped) {
		t.Errorf("expected status %q, got %q", string(types.PoolStatusStopped), got.Status)
	}
}

func TestManager_StartNonExistent(t *testing.T) {
	m := NewManager(nil)
	err := m.Start(context.Background(), "does-not-exist")
	if err == nil {
		t.Error("expected error when starting non-existent pool")
	}
}

func TestManager_StopNonExistent(t *testing.T) {
	m := NewManager(nil)
	err := m.Stop(context.Background(), "does-not-exist")
	if err == nil {
		t.Error("expected error when stopping non-existent pool")
	}
}

func TestManager_ScaleUp(t *testing.T) {
	m := NewManager(nil)
	server := platform.NewThreeProxyServer(platform.NewProcessManager())

	p, _ := m.Create(types.PoolConfig{Name: "scale-up", Prefix: "2001:db8::/64", Ports: "10000-10004"}, server)

	// Scale up requires a running pool (Scale calls proxy.NewPool internally)
	if err := m.Start(context.Background(), p.ID); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	got, _ := m.Get(p.ID)
	if got.proxyStore.Count() != 5 {
		t.Fatalf("expected 5 proxies after Start, got %d", got.proxyStore.Count())
	}

	err := m.Scale(context.Background(), p.ID, 10)
	if err != nil {
		t.Fatalf("Scale up failed: %v", err)
	}

	got, _ = m.Get(p.ID)
	if got.Count != 10 {
		t.Errorf("expected Count=10, got %d", got.Count)
	}
	if got.proxyStore.Count() != 10 {
		t.Errorf("expected 10 proxies in proxyStore, got %d", got.proxyStore.Count())
	}
}

func TestManager_ScaleDown(t *testing.T) {
	m := NewManager(nil)
	server := platform.NewThreeProxyServer(platform.NewProcessManager())

	p, _ := m.Create(types.PoolConfig{Name: "scale-down", Prefix: "2001:db8::/64", Ports: "10000-10009"}, server)
	if err := m.Start(context.Background(), p.ID); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	err := m.Scale(context.Background(), p.ID, 3)
	if err != nil {
		t.Fatalf("Scale down failed: %v", err)
	}

	got, _ := m.Get(p.ID)
	if got.Count != 3 {
		t.Errorf("expected Count=3, got %d", got.Count)
	}
	if got.proxyStore.Count() != 3 {
		t.Errorf("expected 3 proxies in proxyStore, got %d", got.proxyStore.Count())
	}
}

func TestManager_ScaleNonExistent(t *testing.T) {
	m := NewManager(nil)
	err := m.Scale(context.Background(), "missing", 10)
	if err == nil {
		t.Error("expected error when scaling non-existent pool")
	}
}

func TestManager_Concurrent(t *testing.T) {
	m := NewManager(nil)
	server := platform.NewThreeProxyServer(platform.NewProcessManager())
	done := make(chan struct{}, 50)

	for i := 0; i < 50; i++ {
		go func(idx int) {
			cfg := types.PoolConfig{
				Name:   "concurrent-pool-" + string(rune('a'+idx%26)),
				Prefix: "2001:db8::/64",
				Ports:  "10000-10004",
			}
			_, _ = m.Create(cfg, server)
			_ = m.Count()
			_ = m.List()
			done <- struct{}{}
		}(i)
	}

	for i := 0; i < 50; i++ {
		<-done
	}

	if m.Count() != 50 {
		t.Errorf("expected 50 pools after concurrent creates, got %d", m.Count())
	}
}
