package main

import (
	"fmt"
	"os"

	"github.com/nxhung/proxy-ipv6-generator-cli/internal/config"
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
	cfg, err := config.Generate3proxyConfig(p.Pool, proxies, 500)
	if err != nil {
		return fmt.Errorf("failed to generate config: %w", err)
	}

	if c.Output != "" {
		return os.WriteFile(c.Output, []byte(cfg), 0600)
	}

	fmt.Println(cfg)
	return nil
}
