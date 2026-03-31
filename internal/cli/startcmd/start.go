package startcmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ix64/hatch/internal/cli/projectmeta"
)

type options struct {
	projectDir string
	binary     string
}

type execution struct {
	binaryPath string
	projectDir string
	args       []string
}

func New() *cobra.Command {
	opts := options{}
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Run the built application binary",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(opts)
		},
	}

	cmd.Flags().StringVar(&opts.projectDir, "project-dir", ".", "project directory")
	cmd.Flags().StringVar(&opts.binary, "binary", "", "built binary path")
	return cmd
}

func run(opts options) error {
	execSpec, err := resolveExecution(opts)
	if err != nil {
		return err
	}

	cmd := exec.Command(execSpec.binaryPath, execSpec.args...)
	cmd.Dir = execSpec.projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	fmt.Println(strings.Repeat("=", 40))
	fmt.Println(displayShellCommand(cmd))
	fmt.Println(strings.Repeat("=", 40))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run built binary: %w", err)
	}
	return nil
}

func resolveExecution(opts options) (execution, error) {
	projectDir, err := filepath.Abs(opts.projectDir)
	if err != nil {
		return execution{}, fmt.Errorf("resolve project directory: %w", err)
	}

	spec, err := projectmeta.Load(projectDir)
	if err != nil {
		return execution{}, err
	}

	binaryPath := spec.Paths.BuildOutput
	if opts.binary != "" {
		binaryPath = opts.binary
	}
	binaryPath = normalizeBinaryPath(binaryPath)
	if !filepath.IsAbs(binaryPath) {
		binaryPath = filepath.Join(projectDir, filepath.FromSlash(binaryPath))
	}
	if _, err := os.Stat(binaryPath); err != nil {
		if os.IsNotExist(err) {
			return execution{}, fmt.Errorf("built binary not found at %s; run `hatch build --project-dir %s` first", binaryPath, opts.projectDir)
		}
		return execution{}, fmt.Errorf("stat built binary %s: %w", binaryPath, err)
	}

	return execution{
		binaryPath: binaryPath,
		projectDir: projectDir,
		args:       append([]string(nil), spec.Run.Command...),
	}, nil
}

func normalizeBinaryPath(path string) string {
	if runtime.GOOS == "windows" && filepath.Ext(path) != ".exe" {
		return path + ".exe"
	}
	return path
}

func displayShellCommand(cmd *exec.Cmd) string {
	items := make([]string, 0, len(cmd.Args))
	for _, arg := range cmd.Args {
		if strings.ContainsAny(arg, " \t\n\"") {
			items = append(items, strconv.Quote(arg))
			continue
		}
		items = append(items, arg)
	}
	return strings.Join(items, " ")
}
