package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nxhung/proxy-ipv6-generator-cli/internal/config"
	"github.com/nxhung/proxy-ipv6-generator-cli/internal/platform"
)

type SetupCmd struct {
	ConfigPath string `short:"c" help:"Config file path"`
	Force      bool   `short:"f" help:"Force overwrite config"`
}

func (c *SetupCmd) Run(ctx *CLIContext) error {
	// Step 1: Initialize config
	fmt.Println("==> Initializing configuration...")
	if err := config.InitConfig(c.ConfigPath); err != nil {
		return fmt.Errorf("config init failed: %w", err)
	}
	fmt.Println("    Config initialized")

	// Step 2: Install 3proxy
	fmt.Println("==> Installing 3proxy...")
	installDir := ctx.ProcMgr.InstallPath()
	if err := platform.EnsureDir(installDir); err != nil {
		return fmt.Errorf("failed to create install directory: %w", err)
	}
	installDst := filepath.Join(installDir, platform.BinaryName())

	// Check if already installed
	if !c.Force {
		if _, err := os.Stat(installDst); err == nil {
			fmt.Printf("    3proxy already installed at %s\n", installDst)
			fmt.Println("    Run with --force to reinstall")
		}
	}

	binPath, err := ctx.ProcMgr.Detect3proxy()
	if err == nil && binPath != "" {
		if binPath == installDst {
			fmt.Printf("    3proxy already installed at %s\n", installDst)
		} else {
			if err := copyFile(binPath, installDst); err != nil {
				return fmt.Errorf("failed to copy 3proxy: %w", err)
			}
			fmt.Printf("    3proxy installed to %s (copied from %s)\n", installDst, binPath)
		}
	} else {
		return fmt.Errorf("no 3proxy binary found; compile from source:\n  git clone https://github.com/3proxy/3proxy.git && cd 3proxy && make -f Makefile.Linux && make install")
	}

	fmt.Println("==> Setup complete!")
	return nil
}
