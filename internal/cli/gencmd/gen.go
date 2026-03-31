package gencmd

import (
	"github.com/spf13/cobra"

	"github.com/ix64/hatch/internal/cli/entcmd"
	"github.com/ix64/hatch/internal/cli/protocmd"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gen",
		Short: "Generate application code",
	}
	cmd.AddCommand(entcmd.New())
	cmd.AddCommand(protocmd.New())
	return cmd
}
