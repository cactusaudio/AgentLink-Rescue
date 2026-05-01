package main

import (
	"os"

	"cactus-agentlink-rescue/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:], os.Stdout, os.Stderr))
}
