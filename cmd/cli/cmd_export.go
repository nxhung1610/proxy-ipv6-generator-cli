package main

import (
	"encoding/json"
	"fmt"
	"os"
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
		for _, p := range proxies {
			output += fmt.Sprintf("%s:%d\n", p.Address, p.Port)
		}
	case "csv":
		output = "id,address,port,protocol,status\n"
		for _, p := range proxies {
			output += fmt.Sprintf("%s,%s,%d,%s,%s\n", p.ID, p.Address, p.Port, p.Protocol, p.Status)
		}
	default:
		return fmt.Errorf("unsupported format: %s", c.Format)
	}

	if c.Output != "" {
		return os.WriteFile(c.Output, []byte(output), 0600)
	}

	fmt.Print(output)
	return nil
}
