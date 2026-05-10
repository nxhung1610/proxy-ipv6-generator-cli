package main

import (
	"fmt"
	"os"

	"github.com/nxhung/proxy-ipv6-generator-cli/internal/config"
	"github.com/nxhung/proxy-ipv6-generator-cli/internal/errs"
	"github.com/nxhung/proxy-ipv6-generator-cli/pkg/types"
)

type ConfigCmd struct {
	Generate ConfigGenerateCmd `cmd:"generate" help:"Generate 3proxy config"`
}

type ConfigGenerateCmd struct {
	Pool   string `short:"p" default:"default" help:"Pool name"`
	Output string `short:"o" help:"Output file"`
}

func (c *ConfigGenerateCmd) Run(ctx *CLIContext) error {
	p, ok := ctx.PoolMgr.GetByName(c.Pool)
	if !ok {
		return fmt.Errorf("pool not found: %s", c.Pool)
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
