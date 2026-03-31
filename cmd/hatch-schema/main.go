package main

import (
	"fmt"
	"os"

	"github.com/ix64/hatch/internal/cli/projectmeta"
)

func main() {
	schema, err := projectmeta.JSONSchema()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if _, err := os.Stdout.Write(append(schema, '\n')); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

