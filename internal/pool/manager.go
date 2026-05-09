package pool

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nxhung/proxy-ipv6-generator-cli/internal/platform"
	"github.com/nxhung/proxy-ipv6-generator-cli/internal/proxy"
	"github.com/nxhung/proxy-ipv6-generator-cli/internal/state"
	"github.com/nxhung/proxy-ipv6-generator-cli/pkg/types"
)

type Manager struct {
	mu       sync.RWMutex
	pools    map[string]*Pool
	stateMgr *state.StateManager
}

type Pool struct {
	types.Pool
	proxyStore types.ProxyStore
	server     platform.ProxyServer
	startedAt  time.Time
}

func (p *Pool) GetProxies() []types.Proxy {
	if p.proxyStore == nil {
		return nil
	}
	return p.proxyStore.List()
}

func NewManager(stateMgr *state.StateManager) *Manager {
	m := &Manager{
		pools:    make(map[string]*Pool),
		stateMgr: stateMgr,
	}
	m.loadState()
	return m
}

func (m *Manager) Create(cfg types.PoolConfig, server platform.ProxyServer) (*Pool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	pool := &Pool{
		Pool: types.Pool{
			ID:            uuid.New().String(),
			Name:          cfg.Name,
			Prefix:        cfg.Prefix,
			BasePort:      parsePortRange(cfg.Ports),
			Count:         countPorts(cfg.Ports),
			Protocol:      types.ProtocolSOCKS5,
			Status:        string(types.PoolStatusStopped),
			CreatedAt:     time.Now().Format(time.RFC3339),
			CredentialRef: "",
		},
		proxyStore: proxy.NewManager(),
		server:     server,
	}

	m.pools[pool.ID] = pool
	m.saveState()
	return pool, nil
}

func (m *Manager) Get(id string) (*Pool, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.pools[id]
	return p, ok
}

func (m *Manager) GetByName(name string) (*Pool, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, p := range m.pools {
		if p.Name == name {
			return p, true
		}
	}
	return nil, false
}

func (m *Manager) List() []types.Pool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]types.Pool, 0, len(m.pools))
	for _, p := range m.pools {
		result = append(result, p.Pool)
	}
	return result
}

func (m *Manager) Remove(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.pools[id]; ok {
		delete(m.pools, id)
		m.saveState()
		return true
	}
	return false
}

func (m *Manager) Start(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pool, ok := m.pools[id]
	if !ok {
		return fmt.Errorf("pool not found: %s", id)
	}

	pm, err := proxy.NewPool(pool.Prefix, pool.BasePort, pool.Count, pool.Protocol)
	if err != nil {
		return fmt.Errorf("failed to create proxy pool: %w", err)
	}

	pool.proxyStore = pm
	pool.Status = string(types.PoolStatusRunning)
	pool.startedAt = time.Now()
	m.saveState()
	return nil
}

func (m *Manager) Stop(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pool, ok := m.pools[id]
	if !ok {
		return fmt.Errorf("pool not found: %s", id)
	}

	pool.Status = string(types.PoolStatusStopped)
	m.saveState()
	return nil
}

func (m *Manager) Scale(ctx context.Context, id string, count int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pool, ok := m.pools[id]
	if !ok {
		return fmt.Errorf("pool not found: %s", id)
	}

	if count < pool.Count {
		proxies := pool.proxyStore.List()
		for i := pool.Count - 1; i >= count; i-- {
			if i < len(proxies) {
				pool.proxyStore.Remove(proxies[i].ID)
			}
		}
	} else if count > pool.Count {
		extra, err := proxy.NewPool(pool.Prefix, pool.BasePort+pool.Count, count-pool.Count, pool.Protocol)
		if err != nil {
			return err
		}
		for _, p := range extra.List() {
			pool.proxyStore.Add(p)
		}
	}

	pool.Count = count
	m.saveState()
	return nil
}

func (m *Manager) saveState() {
	if m.stateMgr == nil {
		return
	}
	current := make(map[string]bool)
	for _, p := range m.pools {
		m.stateMgr.UpdatePool(p.Pool)
		current[p.ID] = true
	}
	for _, p := range m.stateMgr.GetState().Pools {
		if !current[p.ID] {
			m.stateMgr.RemovePool(p.ID)
		}
	}
}

func (m *Manager) loadState() {
	if m.stateMgr == nil {
		return
	}
	state := m.stateMgr.GetState()
	for _, p := range state.Pools {
		pool := &Pool{
			Pool:       p,
			proxyStore: proxy.NewManager(),
		}
		if p.Status == string(types.PoolStatusRunning) {
			pm, err := proxy.NewPool(p.Prefix, p.BasePort, p.Count, p.Protocol)
			if err == nil {
				pool.proxyStore = pm
			}
		}
		m.pools[p.ID] = pool
	}
}

func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.pools)
}

func parsePortRange(ports string) int {
	var start int
	fmt.Sscanf(ports, "%d-", &start)
	return start
}

func countPorts(ports string) int {
	var start, end int
	fmt.Sscanf(ports, "%d-%d", &start, &end)
	return end - start + 1
}

func ParsePortRange(ports string) (int, int, error) {
	var start, end int
	n, err := fmt.Sscanf(ports, "%d-%d", &start, &end)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid port range format %q: %w", ports, err)
	}
	if n != 2 {
		return 0, 0, fmt.Errorf("port range must be in format START-END")
	}
	return start, end, nil
}
