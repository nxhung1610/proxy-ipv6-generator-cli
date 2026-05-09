// Package main is the entry point for proxy-ipv6-generator-cli.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/alecthomas/kong"
	"github.com/nxhung/proxy-ipv6-generator-cli/internal/config"
	"github.com/nxhung/proxy-ipv6-generator-cli/internal/generator"
	"github.com/nxhung/proxy-ipv6-generator-cli/internal/platform"
	"github.com/nxhung/proxy-ipv6-generator-cli/internal/pool"
	"github.com/nxhung/proxy-ipv6-generator-cli/internal/proxy"
	"github.com/nxhung/proxy-ipv6-generator-cli/internal/state"
)

var version = "dev"

// CLIContext holds shared manager instances injected into subcommands via kong.Bind.
type CLIContext struct {
	PoolMgr  *pool.Manager
	ProxyMgr *proxy.Manager
	StateMgr *state.StateManager
	ProcMgr  *platform.ProcessManager
}

type CLI struct {
	Init     InitCmd      `cmd:"" help:"Initialize configuration"`
	Install  InstallCmd   `cmd:"install" help:"Install 3proxy (bundled)"`
	Generate GenerateCmd  `cmd:"generate" help:"Generate IPv6 addresses"`
	Pool     PoolCmd      `cmd:"pool" help:"Manage proxy pools"`
	Config   ConfigCmd    `cmd:"config" help:"Configuration management"`
	Health   HealthCmd    `cmd:"health" help:"Health check management"`
	Export   ExportCmd    `cmd:"export" help:"Export proxies"`
	Version  VersionCmd   `cmd:"version" help:"Show version"`
}

type GenerateCmd struct {
	Count  int    `short:"n" help:"Number of addresses to generate"`
	Prefix string `short:"p" help:"IPv6 prefix (e.g., 2001:db8::/64)"`
	Output string `short:"o" help:"Output file (default: stdout)"`
}

func (c *GenerateCmd) Run(ctx *CLIContext) error {
	prefix := c.Prefix
	if prefix == "" {
		cfg, err := config.Load("")
		if err == nil {
			prefix = cfg.Server.Prefix
		}
	}
	if prefix == "" {
		prefix = "2001:db8::"
	}

	count := c.Count
	if count == 0 {
		cfg, err := config.Load("")
		if err == nil {
			count = cfg.Server.Count
		}
		if count == 0 {
			count = 100
		}
	}

	gen, err := generator.New(prefix, 10000, count)
	if err != nil {
		return fmt.Errorf("failed to create generator: %w", err)
	}

	proxies, err := gen.Generate(count)
	if err != nil {
		return fmt.Errorf("failed to generate proxies: %w", err)
	}

	data, _ := json.MarshalIndent(proxies, "", "  ")
	if c.Output != "" {
		return os.WriteFile(c.Output, data, 0600)
	}

	fmt.Println(string(data))
	return nil
}

type InstallCmd struct{}

func (c *InstallCmd) Run(ctx *CLIContext) error {
	return install3proxy(ctx.ProcMgr)
}

// install3proxy installs 3proxy for the current host platform.
func install3proxy(pm *platform.ProcessManager) error {
	installDir := pm.InstallPath()
	if err := platform.EnsureDir(installDir); err != nil {
		return fmt.Errorf("failed to create install directory: %w", err)
	}
	installDst := filepath.Join(installDir, platform.BinaryName())

	vendorSrc := ""
	if runtime.GOOS != "darwin" {
		exePath, err := os.Executable()
		if err == nil {
			vendorSrc = filepath.Join(filepath.Dir(exePath), "3proxy-bin", platform.BinaryName())
		}
	}

	if vendorSrc != "" {
		if _, err := os.Stat(vendorSrc); err == nil {
			if err := copyFile(vendorSrc, installDst); err != nil {
				return fmt.Errorf("failed to copy bundled 3proxy: %w", err)
			}
			fmt.Printf("3proxy installed to %s\n", installDst)
			return nil
		}
	}

	systemPath, err := pm.Detect3proxy()
	if err == nil && systemPath != "" {
		if systemPath == installDst {
			fmt.Printf("3proxy already installed at %s\n", installDst)
			return nil
		}
		if err := copyFile(systemPath, installDst); err != nil {
			return fmt.Errorf("failed to copy system 3proxy: %w", err)
		}
		fmt.Printf("3proxy installed to %s (copied from %s)\n", installDst, systemPath)
		return nil
	}

	return fmt.Errorf("no 3proxy binary available; compile from source:\n  git clone https://github.com/3proxy/3proxy.git && cd 3proxy && make -f Makefile.Linux && make install\nthen run 'rip install' again")
}

func copyFile(src, dst string) error {
	s, err := os.Open(src) // #nosec G304 // src: from Detect3proxy() system paths only
	if err != nil {
		return err
	}
	defer s.Close()
	d, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600) // #nosec G304 // dst: internal InstallPath() construction only
	if err != nil {
		return err
	}
	defer d.Close()
	if _, err := io.Copy(d, s); err != nil {
		return err
	}
	if err := d.Close(); err != nil {
		return err
	}
	return os.Chmod(dst, 0755)
}

func main() {
	stateMgr, _ := state.New("")

	cliCtx := &CLIContext{
		PoolMgr:  pool.NewManager(stateMgr),
		ProxyMgr: proxy.NewManager(),
		ProcMgr:  platform.NewProcessManager(),
		StateMgr: stateMgr,
	}

	var cli CLI

	kongCtx := kong.Parse(&cli,
		kong.Name("rip"),
		kong.Description("Generate and manage IPv6 proxy pools with 3proxy"),
		kong.UsageOnError(),
		kong.Bind(cliCtx),
	)

	if err := kongCtx.Run(cliCtx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
