package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nxhung/proxy-ipv6-generator-cli/internal/config"
	"github.com/nxhung/proxy-ipv6-generator-cli/internal/platform"
	"github.com/nxhung/proxy-ipv6-generator-cli/internal/pool"
	"github.com/nxhung/proxy-ipv6-generator-cli/pkg/types"
)

type PoolCmd struct {
	Start  PoolStartCmd  `cmd:"start" help:"Start a pool"`
	Stop   PoolStopCmd   `cmd:"stop" help:"Stop a pool"`
	Status PoolStatusCmd `cmd:"status" help:"Show pool status"`
	Create PoolCreateCmd `cmd:"create" help:"Create a pool"`
	Remove PoolRemoveCmd `cmd:"remove" help:"Remove a pool"`
	List   PoolListCmd   `cmd:"list" help:"List all pools"`
}

type PoolCreateCmd struct {
	Name     string `arg:"" help:"Pool name"`
	Prefix   string `arg:"" help:"IPv6 prefix"`
	Ports    string `arg:"" help:"Port range (e.g., 10000-10100)"`
	Protocol string `short:"p" default:"socks5" help:"Protocol (socks5 or http)"`
}

func (c *PoolCreateCmd) Run(ctx *CLIContext) error {
	cfg := types.PoolConfig{
		Name:   c.Name,
		Prefix: c.Prefix,
		Ports:  c.Ports,
	}

	p, err := ctx.PoolMgr.Create(cfg, platform.NewThreeProxyServer(ctx.ProcMgr))
	if err != nil {
		return fmt.Errorf("failed to create pool: %w", err)
	}

	fmt.Printf("Pool '%s' created with ID: %s\n", p.Name, p.ID)
	return nil
}

type PoolStartCmd struct {
	Name string `arg:"" optional:"" help:"Pool name"`
}

func (c *PoolStartCmd) Run(ctx *CLIContext) error {
	name := c.Name
	if name == "" {
		name = "default"
	}

	p, ok := ctx.PoolMgr.GetByName(name)
	if !ok {
		return fmt.Errorf("pool not found: %s", name)
	}

	if p.Status == string(types.PoolStatusRunning) {
		if pid, _ := ctx.ProcMgr.GetPIDFromFile(ctx.ProcMgr.PIDPath(p.ID)); pid > 0 && ctx.ProcMgr.IsRunning(pid) {
			return fmt.Errorf("pool '%s' is already running (PID %d)", name, pid)
		}
	}

	if err := ctx.PoolMgr.Start(nil, p.ID); err != nil {
		return fmt.Errorf("failed to start pool: %w", err)
	}

	proxyList := p.GetProxies()
	cfgStr, err := config.Generate3proxyConfig(p.Pool, proxyList, 500)
	if err != nil {
		return fmt.Errorf("failed to generate 3proxy config: %w", err)
	}

	poolDir := ctx.ProcMgr.ConfigPath(p.ID)
	if err := platform.EnsureDir(poolDir); err != nil {
		return fmt.Errorf("failed to create pool dir: %w", err)
	}

	cfgPath := filepath.Join(poolDir, "3proxy.cfg")
	if err := os.WriteFile(cfgPath, []byte(cfgStr), 0600); err != nil {
		return fmt.Errorf("failed to write 3proxy config: %w", err)
	}

	binPath, err := ctx.ProcMgr.Detect3proxy()
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	proc, err := ctx.ProcMgr.Start(binPath, cfgPath)
	if err != nil {
		return fmt.Errorf("failed to start 3proxy: %w", err)
	}

	pidPath := ctx.ProcMgr.PIDPath(p.ID)
	if err := platform.EnsureDir(filepath.Dir(pidPath)); err != nil {
		ctx.ProcMgr.Stop(proc.PID)
		return fmt.Errorf("failed to create pid dir: %w", err)
	}
	if err := ctx.ProcMgr.WritePIDFile(pidPath, proc.PID); err != nil {
		ctx.ProcMgr.Stop(proc.PID)
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	fmt.Printf("Pool '%s' started (PID %d, %d proxies)\n", name, proc.PID, len(proxyList))
	return nil
}

type PoolStopCmd struct {
	Name string `arg:"" optional:"" help:"Pool name"`
}

func (c *PoolStopCmd) Run(ctx *CLIContext) error {
	name := c.Name
	if name == "" {
		name = "default"
	}

	p, ok := ctx.PoolMgr.GetByName(name)
	if !ok {
		return fmt.Errorf("pool not found: %s", name)
	}

	pidPath := ctx.ProcMgr.PIDPath(p.ID)
	pid, _ := ctx.ProcMgr.GetPIDFromFile(pidPath)
	if pid > 0 && ctx.ProcMgr.IsRunning(pid) {
		if err := ctx.ProcMgr.Stop(pid); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to stop 3proxy (PID %d): %v\n", pid, err)
		}
	}
	_ = os.Remove(pidPath)

	if err := ctx.PoolMgr.Stop(nil, p.ID); err != nil {
		return fmt.Errorf("failed to stop pool: %w", err)
	}

	fmt.Printf("Pool '%s' stopped\n", name)
	return nil
}

type PoolStatusCmd struct {
	Name string `arg:"" optional:"" help:"Pool name"`
}

func (c *PoolStatusCmd) Run(ctx *CLIContext) error {
	var p *pool.Pool
	var ok bool

	if c.Name != "" {
		p, ok = ctx.PoolMgr.GetByName(c.Name)
		if !ok {
			return fmt.Errorf("pool not found: %s", c.Name)
		}
	} else {
		pools := ctx.PoolMgr.List()
		if len(pools) == 0 {
			fmt.Println("No pools found")
			return nil
		}
		for _, pl := range pools {
			p, ok = ctx.PoolMgr.Get(pl.ID)
			if ok {
				break
			}
		}
	}

	if p == nil {
		return fmt.Errorf("pool not found")
	}

	status := map[string]interface{}{
		"pool":     p.Name,
		"status":   p.Status,
		"prefix":   p.Prefix,
		"proxies":  p.Count,
		"protocol": p.Protocol,
	}

	data, _ := json.MarshalIndent(status, "", "  ")
	fmt.Println(string(data))
	return nil
}

type PoolRemoveCmd struct {
	Name string `arg:"" optional:"" help:"Pool name"`
	Yes  bool   `short:"y" help:"Skip confirmation"`
}

func (c *PoolRemoveCmd) Run(ctx *CLIContext) error {
	if !c.Yes {
		fmt.Printf("Are you sure you want to remove pool '%s'? (y/N): ", c.Name)
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "y" && confirm != "Y" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	p, ok := ctx.PoolMgr.GetByName(c.Name)
	if !ok {
		return fmt.Errorf("pool not found: %s", c.Name)
	}

	ctx.PoolMgr.Remove(p.ID)
	fmt.Printf("Pool '%s' removed\n", c.Name)
	return nil
}

type PoolListCmd struct{}

func (c *PoolListCmd) Run(ctx *CLIContext) error {
	pools := ctx.PoolMgr.List()

	if len(pools) == 0 {
		fmt.Println("No pools found")
		return nil
	}

	data, _ := json.MarshalIndent(pools, "", "  ")
	fmt.Println(string(data))
	return nil
}
