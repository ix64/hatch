package main

import (
	"fmt"
	"os"

	"github.com/ix64/hatch/internal/cli/root"
)

func main() {
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
