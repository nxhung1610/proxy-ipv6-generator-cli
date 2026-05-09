package main

import (
	"fmt"
)

type VersionCmd struct{}

func (c *VersionCmd) Run(ctx *CLIContext) error {
	fmt.Printf("rip version %s\n", version)
	return nil
}
