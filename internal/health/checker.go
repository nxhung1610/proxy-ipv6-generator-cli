package health

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nxhung/proxy-ipv6-generator-cli/pkg/types"
)

type Checker struct {
	mu             sync.RWMutex
	failures       map[string]int
	maxFailures    int
	timeout        time.Duration
	interval       time.Duration
	proxyProvider  ProxyProvider
	statusCallback func(string, string)
	stopCh         chan struct{}
	running        bool
}

type CheckerConfig struct {
	Interval    time.Duration
	Timeout     time.Duration
	MaxFailures int
}

func NewChecker(cfg CheckerConfig, provider ProxyProvider) *Checker {
	if cfg.Timeout == 0 {
		cfg.Timeout = 3 * time.Second
	}
	if cfg.Interval == 0 {
		cfg.Interval = 5 * time.Second
	}
	if cfg.MaxFailures == 0 {
		cfg.MaxFailures = 3
	}

	return &Checker{
		failures:       make(map[string]int),
		maxFailures:    cfg.MaxFailures,
		timeout:        cfg.Timeout,
		interval:       cfg.Interval,
		proxyProvider:  provider,
		stopCh:         make(chan struct{}),
	}
}

func (c *Checker) CheckAll(ctx context.Context) ([]types.HealthResult, error) {
	proxies := c.proxyProvider.List()
	results := make([]types.HealthResult, 0, len(proxies))

	for _, p := range proxies {
		result := c.checkProxy(ctx, &p)
		results = append(results, *result)

		c.mu.Lock()
		if result.Healthy {
			delete(c.failures, p.ID)
		} else {
			c.failures[p.ID]++
			if c.failures[p.ID] >= c.maxFailures {
				c.markUnhealthy(p.ID)
				if c.statusCallback != nil {
					c.statusCallback(p.ID, string(types.ProxyStatusUnhealthy))
				}
			}
		}
		c.mu.Unlock()
	}

	return results, nil
}

func (c *Checker) CheckOne(ctx context.Context, proxy *types.Proxy) *types.HealthResult {
	result := c.checkProxy(ctx, proxy)

	c.mu.Lock()
	if result.Healthy {
		delete(c.failures, proxy.ID)
	} else {
		c.failures[proxy.ID]++
		if c.failures[proxy.ID] >= c.maxFailures {
			c.markUnhealthy(proxy.ID)
		}
	}
	c.mu.Unlock()

	return result
}

func (c *Checker) checkProxy(ctx context.Context, proxy *types.Proxy) *types.HealthResult {
	result := &types.HealthResult{
		ProxyID: proxy.ID,
		Address: proxy.Address,
		Port:    proxy.Port,
		Healthy: false,
	}

	start := time.Now()

	addr := fmt.Sprintf("[%s]:%d", proxy.Address, proxy.Port)

	conn, err := DialProxy(ctx, addr, c.timeout)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer conn.Close()

	result.LatencyMs = time.Since(start).Milliseconds()
	result.Healthy = true
	return result
}

func (c *Checker) markUnhealthy(proxyID string) {
	c.proxyProvider.UpdateStatus(proxyID, string(types.ProxyStatusUnhealthy))
	if c.statusCallback != nil {
		c.statusCallback(proxyID, string(types.ProxyStatusUnhealthy))
	}
}

func (c *Checker) Start(ctx context.Context) {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return
	}
	c.running = true
	c.stopCh = make(chan struct{}) // Fresh channel per Start()
	c.mu.Unlock()

	go func() {
		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.CheckAll(ctx)
			case <-c.stopCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (c *Checker) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return // Already stopped — no-op
	}

	close(c.stopCh)
	c.running = false
	// NOTE: do NOT recreate stopCh here
	// Let the background goroutine handle the closed channel
	// If Start() is called again, create a new channel there
}

func (c *Checker) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

func (c *Checker) GetFailureCount(proxyID string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.failures[proxyID]
}

func (c *Checker) ResetFailures(proxyID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.failures, proxyID)
}

func (c *Checker) OnStatusChange(cb func(string, string)) {
	c.statusCallback = cb
}
