package core

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the application version",
		Run: func(cmd *cobra.Command, args []string) {
			info := CurrentBuildInfo()
			fmt.Println("Version:", info.Version)
			fmt.Println("Build Time:", info.BuildTime)
			fmt.Println("Commit Hash:", info.CommitHash)
			fmt.Println("Module:", info.Module)
			fmt.Println("Package:", info.Package)
			fmt.Println("Compiler:", info.Compiler)
		},
	}
}
