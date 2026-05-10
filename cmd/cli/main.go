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
	"github.com/nxhung/proxy-ipv6-generator-cli/internal/errs"
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
	Setup    SetupCmd      `cmd:"setup" help:"Initialize config and install 3proxy (combines init+install)"`
	Init     InitCmd       `cmd:"init" help:"Initialize configuration (deprecated: use setup)"`
	Install  InstallCmd    `cmd:"install" help:"Install 3proxy binary (deprecated: use setup)"`
	Generate GenerateCmd   `cmd:"generate" help:"Generate IPv6 addresses"`
	Pool     PoolCmd       `cmd:"pool" help:"Manage proxy pools"`
	Config   ConfigCmd     `cmd:"config" help:"Configuration management"`
	Health   HealthCmd     `cmd:"health" help:"Health check management"`
	Export   ExportCmd    `cmd:"export" help:"Export proxies"`
	Version  VersionCmd    `cmd:"version" help:"Show version"`
}

type GenerateCmd struct {
	Count  int    `short:"n" help:"Number of addresses to generate"`
	Prefix string `short:"p" help:"IPv6 prefix (e.g., 2001:db8::/64)"`
	Output string `short:"o" help:"Output file (default: stdout)"`
	Pool   string `short:"P" help:"Pool name to add generated proxies to"`
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

	// Add to pool if specified
	if c.Pool != "" {
		_, ok := ctx.PoolMgr.GetByName(c.Pool)
		if !ok {
			return fmt.Errorf("pool not found: %s", c.Pool)
		}
		ctx.ProxyMgr.AddBatch(proxies)
		fmt.Printf("Added %d proxies to pool '%s'\n", len(proxies), c.Pool)
		return nil
	}

	data, _ := json.MarshalIndent(proxies, "", "  ")
	if c.Output != "" {
		if err := os.WriteFile(c.Output, data, 0600); err != nil {
			return errs.WrapFileError("failed to write", c.Output, err)
		}
		return nil
	}

	fmt.Println(string(data))
	return nil
}

type InstallCmd struct{}

func (c *InstallCmd) Run(ctx *CLIContext) error {
	fmt.Fprintf(os.Stderr, "\nWARNING: 'rip install' is deprecated. Use 'rip setup' instead.\n\n")
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

// copyFile copies a file from src to dst atomically using rename to avoid TOCTOU races.
func copyFile(src, dst string) error {
	// Create temp file in same directory as dst for atomic rename
	dir := filepath.Dir(dst)
	tmp, err := os.CreateTemp(dir, "3proxy-*.bin")
	if err != nil {
		return errs.WrapFileError("cannot create temp file in "+dir, "", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // Best-effort cleanup if we fail

	s, err := os.Open(src) // #nosec G304 // src: from Detect3proxy() system paths only
	if err != nil {
		tmp.Close()
		return errs.WrapFileError("cannot read", src, err)
	}
	defer s.Close()

	if _, err := io.Copy(tmp, s); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to copy %s -> %s: %w", src, tmpPath, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Set executable bit before atomic rename
	if err := os.Chmod(tmpPath, 0755); err != nil {
		return fmt.Errorf("failed to chmod temp file: %w", err)
	}

	// Atomic rename replaces dst safely
	if err := os.Rename(tmpPath, dst); err != nil {
		return fmt.Errorf("failed to install binary to %s: %w", dst, err)
	}

	return nil
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
