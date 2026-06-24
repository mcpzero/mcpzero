package main

import (
	"fmt"
	"os"

	"github.com/mcpzero/mcpzero/cli/internal/cmd"
)

func main() {
	if err := cmd.Execute(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
