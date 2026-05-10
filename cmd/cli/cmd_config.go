package main

import (
	"fmt"
	"os"

	"github.com/nxhung/proxy-ipv6-generator-cli/internal/config"
	"github.com/nxhung/proxy-ipv6-generator-cli/internal/errs"
	"github.com/nxhung/proxy-ipv6-generator-cli/internal/pool"
	"github.com/nxhung/proxy-ipv6-generator-cli/pkg/types"
)

type ConfigCmd struct {
	Generate ConfigGenerateCmd `cmd:"generate" help:"Generate 3proxy config"`
}

type ConfigGenerateCmd struct {
	Pool   string `short:"p" help:"Pool name (finds running pool if not specified)"`
	Output string `short:"o" help:"Output file"`
}

func (c *ConfigGenerateCmd) Run(ctx *CLIContext) error {
	var p *pool.Pool
	var ok bool

	if c.Pool != "" {
		p, ok = ctx.PoolMgr.GetByName(c.Pool)
		if !ok {
			return fmt.Errorf("pool not found: %s", c.Pool)
		}
	} else {
		// Find the first running pool
		for _, pool := range ctx.PoolMgr.List() {
			if pool.Status == string(types.PoolStatusRunning) {
				p, ok = ctx.PoolMgr.Get(pool.ID)
				if ok {
					break
				}
			}
		}
		if p == nil {
			return fmt.Errorf("no running pool found; specify a pool with --pool")
		}
	}

	proxies := p.GetProxies()

	cfg, err := config.Load("")
	if err != nil {
		cfg = &types.Config{}
	}

	dnssrv := cfg.Proxy.DNSServer
	maxConn := cfg.Proxy.MaxConn
	if maxConn == 0 {
		maxConn = config.DefaultMaxConn
	}

	cfgOut, err := config.Generate3proxyConfig(p.Pool, proxies, maxConn, dnssrv)
	if err != nil {
		return fmt.Errorf("failed to generate config: %w", err)
	}

	if c.Output != "" {
		if err := os.WriteFile(c.Output, []byte(cfgOut), 0600); err != nil {
			return errs.WrapFileError("failed to write config file", c.Output, err)
		}
		return nil
	}

	fmt.Println(cfgOut)
	return nil
}
