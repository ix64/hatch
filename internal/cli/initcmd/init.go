package initcmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/ix64/hatch/internal/cli/projectmeta"
	"github.com/ix64/hatch/internal/cli/scaffold"
)

type options struct {
	modulePath       string
	appName          string
	binaryName       string
	hatchReplacePath string
}

func New() *cobra.Command {
	opts := options{}
	cmd := &cobra.Command{
		Use:   "init <dir>",
		Short: "Initialize a new Hatch application",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("expected target directory")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(args[0], opts)
		},
	}

	cmd.Flags().StringVar(&opts.modulePath, "module", "", "go module path")
	cmd.Flags().StringVar(&opts.appName, "name", "", "application display name")
	cmd.Flags().StringVar(&opts.binaryName, "binary", "", "binary name")
	cmd.Flags().StringVar(&opts.hatchReplacePath, "hatch-replace-path", "", "optional local hatch replace path for development")
	return cmd
}

func run(targetDir string, opts options) error {
	spec := projectmeta.New(opts.modulePath, opts.appName, opts.binaryName)
	spec.HatchReplacePath = opts.hatchReplacePath
	if err := spec.ValidateForInit(); err != nil {
		return err
	}

	targetDir, err := filepath.Abs(targetDir)
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(targetDir)
	switch {
	case err == nil && len(entries) > 0:
		return fmt.Errorf("target directory is not empty: %s", targetDir)
	case err == nil:
	case errors.Is(err, os.ErrNotExist):
	default:
		return err
	}

	if err := scaffold.WriteProject(targetDir, spec); err != nil {
		return err
	}
	if err := runGoModTidy(targetDir); err != nil {
		return err
	}

	fmt.Printf("initialized %s in %s\n", spec.AppName, targetDir)
	return nil
}

func runGoModTidy(projectDir string) error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
