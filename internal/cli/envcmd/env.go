package envcmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ix64/hatch/internal/cli/projectmeta"
)

type options struct {
	projectDir string
}

type execution struct {
	dockerPath  string
	projectDir  string
	composePath string
	args        []string
}

type actionSpec struct {
	use   string
	short string
	args  []string
}

var actions = map[string]actionSpec{
	"start": {
		use:   "start",
		short: "Start local development dependencies",
		args:  []string{"up", "--pull", "always", "--detach"},
	},
	"stop": {
		use:   "stop",
		short: "Stop local development dependencies",
		args:  []string{"down"},
	},
	"clean": {
		use:   "clean",
		short: "Stop and remove local development dependency volumes",
		args:  []string{"down", "--volumes"},
	},
}

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Manage local development dependencies",
	}
	cmd.AddCommand(newActionCmd("start"))
	cmd.AddCommand(newActionCmd("stop"))
	cmd.AddCommand(newActionCmd("clean"))
	cmd.AddCommand(newAddCmd())
	return cmd
}

func newActionCmd(name string) *cobra.Command {
	action := actions[name]
	opts := options{}

	cmd := &cobra.Command{
		Use:   action.use,
		Short: action.short,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(name, opts)
		},
	}
	cmd.Flags().StringVar(&opts.projectDir, "project-dir", ".", "project directory")
	return cmd
}

func run(action string, opts options) error {
	execSpec, err := resolveExecution(action, opts, exec.LookPath)
	if err != nil {
		return err
	}

	cmd := exec.Command(execSpec.dockerPath, execSpec.args...)
	cmd.Dir = execSpec.projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	fmt.Println(strings.Repeat("=", 40))
	fmt.Println(displayShellCommand(cmd))
	fmt.Println(strings.Repeat("=", 40))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s local development dependencies: %w", action, err)
	}
	return nil
}

func resolveExecution(action string, opts options, lookPath func(string) (string, error)) (execution, error) {
	actionSpec, ok := actions[action]
	if !ok {
		return execution{}, fmt.Errorf("unsupported env action: %s", action)
	}

	projectDir, err := filepath.Abs(opts.projectDir)
	if err != nil {
		return execution{}, fmt.Errorf("resolve project directory: %w", err)
	}

	spec, err := projectmeta.Load(projectDir)
	if err != nil {
		return execution{}, err
	}

	composePath := spec.Paths.DevCompose
	if composePath == "" {
		return execution{}, fmt.Errorf("dev compose path is empty")
	}
	if !filepath.IsAbs(composePath) {
		composePath = filepath.Join(projectDir, filepath.FromSlash(composePath))
	}
	if _, err := os.Stat(composePath); err != nil {
		if os.IsNotExist(err) {
			return execution{}, fmt.Errorf("dev compose file not found at %s", composePath)
		}
		return execution{}, fmt.Errorf("stat dev compose file %s: %w", composePath, err)
	}

	dockerPath, err := lookPath("docker")
	if err != nil {
		return execution{}, fmt.Errorf("docker is not installed; install Docker first")
	}

	args := append([]string{"compose", "-f", composePath}, actionSpec.args...)
	return execution{
		dockerPath:  dockerPath,
		projectDir:  projectDir,
		composePath: composePath,
		args:        args,
	}, nil
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
