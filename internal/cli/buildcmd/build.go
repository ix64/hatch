package buildcmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ix64/hatch/internal/cli/projectmeta"
)

type options struct {
	projectDir string
	mainPkg    string
	output     string
	version    string
	commit     string
	buildTime  string
}

func New() *cobra.Command {
	opts := options{}
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build the application binary",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(opts)
		},
	}

	cmd.Flags().StringVar(&opts.projectDir, "project-dir", ".", "project directory")
	cmd.Flags().StringVar(&opts.mainPkg, "main", "", "main package to build")
	cmd.Flags().StringVar(&opts.output, "output", "", "output path")
	cmd.Flags().StringVar(&opts.version, "version", "", "override version")
	cmd.Flags().StringVar(&opts.commit, "commit", "", "override commit hash")
	cmd.Flags().StringVar(&opts.buildTime, "build-time", "", "override build time")
	return cmd
}

func run(opts options) error {
	projectDir, err := filepath.Abs(opts.projectDir)
	if err != nil {
		return err
	}

	spec, err := projectmeta.Load(projectDir)
	if err != nil {
		return err
	}

	mainPkg := spec.Paths.MainPackage
	if opts.mainPkg != "" {
		mainPkg = opts.mainPkg
	}
	output := spec.Paths.BuildOutput
	if opts.output != "" {
		output = opts.output
	}

	version := opts.version
	if version == "" {
		version, err = gitOutput(projectDir, "describe", "--tags", "--always", "--dirty")
		if err != nil {
			version = "v0.0.0-devel"
		}
	}

	commit := opts.commit
	if commit == "" {
		commit, err = gitOutput(projectDir, "rev-parse", "HEAD")
		if err != nil {
			commit = "unknown"
		}
	}

	buildTime := opts.buildTime
	if buildTime == "" {
		buildTime = time.Now().UTC().Format(time.RFC3339)
	}

	output = normalizeOutputPath(output)
	if err := os.MkdirAll(filepath.Join(projectDir, filepath.Dir(output)), 0o755); err != nil {
		return fmt.Errorf("create build output dir: %w", err)
	}

	ldflags := ldflags(version, commit, buildTime)
	args := []string{
		"build",
		"-trimpath",
		"-ldflags", strings.Join(ldflags, " "),
		"-o", output,
		mainPkg,
	}

	return runGoCommand(projectDir, args...)
}

func ldflags(version, commit, buildTime string) []string {
	return []string{
		"-s",
		"-w",
		"-X", fmt.Sprintf("%s.Version=%s", "github.com/ix64/hatch/core", version),
		"-X", fmt.Sprintf("%s.CommitHash=%s", "github.com/ix64/hatch/core", commit),
		"-X", fmt.Sprintf("%s.BuildTime=%s", "github.com/ix64/hatch/core", buildTime),
	}
}

func normalizeOutputPath(output string) string {
	if runtime.GOOS == "windows" && filepath.Ext(output) != ".exe" {
		return output + ".exe"
	}
	return output
}

func gitOutput(projectDir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = projectDir
	buf, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(buf)), nil
}

func runGoCommand(projectDir string, args ...string) error {
	cmd := exec.Command("go", args...)
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println(strings.Repeat("=", 40))
	fmt.Println(displayShellCommand(cmd))
	fmt.Println(strings.Repeat("=", 40))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go %s: %w", strings.Join(args, " "), err)
	}
	return nil
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
