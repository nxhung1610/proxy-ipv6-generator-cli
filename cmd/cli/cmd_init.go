package main

import (
	"github.com/nxhung/proxy-ipv6-generator-cli/internal/config"
)

type InitCmd struct {
	Path string `arg:"" optional:"" help:"Config path"`
}

func (c *InitCmd) Run(ctx *CLIContext) error {
	path := c.Path
	if path == "" {
		path = ""
	}
	return config.InitConfig(path)
}
