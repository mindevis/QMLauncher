//go:build cli

package main

import (
	"QMLauncher/internal/cli"
	_ "QMLauncher/internal/cli/cmd" // Import CLI commands
)

func main() {
	// Run CLI-only version without GUI
	exiter, code := cli.Run()
	exiter(code)
}
