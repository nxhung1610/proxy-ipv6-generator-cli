package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/nxhung/proxy-ipv6-generator-cli/internal/errs"
)

type ExportCmd struct {
	Format string `short:"f" default:"json" help:"Output format (json, txt, csv)"`
	Output string `short:"o" help:"Output file"`
}

func (c *ExportCmd) Run(ctx *CLIContext) error {
	proxies := ctx.ProxyMgr.List()

	var output string
	switch c.Format {
	case "json":
		data, _ := json.MarshalIndent(proxies, "", "  ")
		output = string(data)
	case "txt":
		var lines []string
		for _, p := range proxies {
			lines = append(lines, fmt.Sprintf("%s:%d", p.Address, p.Port))
		}
		output = strings.Join(lines, "\n") + "\n"
	case "csv":
		var lines []string
		lines = append(lines, "id,address,port,protocol,status")
		for _, p := range proxies {
			lines = append(lines, fmt.Sprintf("%s,%s,%d,%s,%s", p.ID, p.Address, p.Port, p.Protocol, p.Status))
		}
		output = strings.Join(lines, "\n") + "\n"
	default:
		return fmt.Errorf("unsupported format: %s", c.Format)
	}

	if c.Output != "" {
		if err := os.WriteFile(c.Output, []byte(output), 0600); err != nil {
			return errs.WrapFileError("failed to write", c.Output, err)
		}
		return nil
	}

	fmt.Print(output)
	return nil
}
