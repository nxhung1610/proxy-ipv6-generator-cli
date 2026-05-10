package main

import (
	"fmt"
	"os"

	"github.com/nxhung/proxy-ipv6-generator-cli/internal/config"
)

type InitCmd struct {
	Path string `arg:"" optional:"" help:"Config path"`
}

func (c *InitCmd) Run(ctx *CLIContext) error {
	fmt.Fprintf(os.Stderr, "\nWARNING: 'rip init' is deprecated. Use 'rip setup' instead.\n\n")
	return config.InitConfig(c.Path)
}
