package proxy

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nxhung/proxy-ipv6-generator-cli/internal/generator"
	"github.com/nxhung/proxy-ipv6-generator-cli/internal/health"
	"github.com/nxhung/proxy-ipv6-generator-cli/pkg/types"
)

// Ensure proxy.Manager implements types.ProxyStore at compile time.
var _ types.ProxyStore = (*Manager)(nil)

type Manager struct {
	mu      sync.RWMutex
	proxies map[string]*types.Proxy
}

func NewManager() *Manager {
	return &Manager{
		proxies: make(map[string]*types.Proxy),
	}
}

func (m *Manager) Add(proxy types.Proxy) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p := new(types.Proxy)
	*p = proxy
	m.proxies[p.ID] = p
}

func (m *Manager) AddBatch(proxies []types.Proxy) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range proxies {
		p := new(types.Proxy)
		*p = proxies[i]
		m.proxies[p.ID] = p
	}
}

func (m *Manager) Get(id string) (types.Proxy, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.proxies[id]
	if !ok {
		return types.Proxy{}, false
	}
	return *p, true
}

func (m *Manager) List() []types.Proxy {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]types.Proxy, 0, len(m.proxies))
	for _, p := range m.proxies {
		result = append(result, *p)
	}
	return result
}

func (m *Manager) Remove(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.proxies[id]; ok {
		delete(m.proxies, id)
		return true
	}
	return false
}

func (m *Manager) UpdateStatus(id string, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.proxies[id]; ok {
		p.Status = status
	}
}

func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.proxies)
}

func (m *Manager) CountByStatus(status string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for _, p := range m.proxies {
		if p.Status == status {
			count++
		}
	}
	return count
}

func NewPool(prefix string, basePort, count int, proto types.Protocol) (*Manager, error) {
	gen, err := generator.New(prefix, basePort, count)
	if err != nil {
		return nil, fmt.Errorf("failed to create generator: %w", err)
	}

	proxies, err := gen.Generate(count)
	if err != nil {
		return nil, fmt.Errorf("failed to generate proxies: %w", err)
	}

	m := NewManager()
	for i := range proxies {
		proxies[i].Protocol = proto
		p := new(types.Proxy)
		*p = proxies[i]
		m.proxies[p.ID] = p
	}

	return m, nil
}

func (m *Manager) CheckHealth(ctx context.Context, timeout time.Duration) []types.HealthResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make([]types.HealthResult, 0, len(m.proxies))
	for _, p := range m.proxies {
		result := CheckHealth(ctx, p, timeout)
		results = append(results, *result)
	}
	return results
}

func CheckHealth(ctx context.Context, proxy *types.Proxy, timeout time.Duration) *types.HealthResult {
	result := &types.HealthResult{
		ProxyID: proxy.ID,
		Address: proxy.Address,
		Port:    proxy.Port,
		Healthy: false,
	}

	start := time.Now()

	addr := fmt.Sprintf("[%s]:%d", proxy.Address, proxy.Port)

	conn, err := health.DialProxy(ctx, addr, timeout)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer conn.Close()

	result.LatencyMs = time.Since(start).Milliseconds()
	result.Healthy = true
	return result
}
