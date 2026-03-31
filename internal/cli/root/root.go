package root

import (
	"github.com/spf13/cobra"

	"github.com/ix64/hatch/internal/cli/buildcmd"
	"github.com/ix64/hatch/internal/cli/devcmd"
	"github.com/ix64/hatch/internal/cli/envcmd"
	"github.com/ix64/hatch/internal/cli/gencmd"
	"github.com/ix64/hatch/internal/cli/initcmd"
	"github.com/ix64/hatch/internal/cli/migratecmd"
	"github.com/ix64/hatch/internal/cli/startcmd"
	"github.com/ix64/hatch/internal/cli/toolscmd"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hatch",
		Short: "Hatch application tooling",
	}
	cmd.AddCommand(initcmd.New())
	cmd.AddCommand(buildcmd.New())
	cmd.AddCommand(startcmd.New())
	cmd.AddCommand(devcmd.New())
	cmd.AddCommand(envcmd.New())
	cmd.AddCommand(gencmd.New())
	cmd.AddCommand(migratecmd.New())
	cmd.AddCommand(toolscmd.New())
	return cmd
}

func Execute() error {
	return New().Execute()
}
