package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nxhung/proxy-ipv6-generator-cli/internal/config"
	"github.com/nxhung/proxy-ipv6-generator-cli/internal/health"
	"github.com/nxhung/proxy-ipv6-generator-cli/internal/proxy"
	"github.com/nxhung/proxy-ipv6-generator-cli/pkg/types"
)

type HealthCmd struct {
	Check HealthCheckCmd `cmd:"check" help:"Run health check"`
}

type HealthCheckCmd struct {
	Pool string `short:"p" help:"Pool name"`
}

func (c *HealthCheckCmd) Run(ctx *CLIContext) error {
	runCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	healthCfg := health.CheckerConfig{
		Interval:    time.Duration(cfg.Health.IntervalSeconds) * time.Second,
		Timeout:     time.Duration(cfg.Health.TimeoutSeconds) * time.Second,
		MaxFailures: cfg.Health.MaxFailures,
	}

	var proxies []types.Proxy
	if c.Pool != "" {
		p, ok := ctx.PoolMgr.GetByName(c.Pool)
		if !ok {
			return fmt.Errorf("pool not found: %s", c.Pool)
		}
		proxies = p.GetProxies()
	}

	if len(proxies) == 0 {
		proxies = ctx.ProxyMgr.List()
	}

	if len(proxies) == 0 {
		fmt.Println("[]")
		return nil
	}

	proxyMgr := proxy.NewManager()
	proxyMgr.AddBatch(proxies)

	provider := health.NewProxyProviderAdapter(proxyMgr)
	checker := health.NewChecker(healthCfg, provider)

	results, err := checker.CheckAll(runCtx)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	data, _ := json.MarshalIndent(results, "", "  ")
	fmt.Println(string(data))
	return nil
}
