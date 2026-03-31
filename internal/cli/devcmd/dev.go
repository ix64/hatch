package devcmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"text/template"

	"github.com/spf13/cobra"

	"github.com/ix64/hatch/internal/cli/projectmeta"
)

type options struct {
	projectDir string
	config     string
}

type execution struct {
	airPath     string
	projectDir  string
	configPath  string
	cleanupPath string
}

func New() *cobra.Command {
	opts := options{}
	cmd := &cobra.Command{
		Use:   "dev",
		Short: "Run the application with live reload via Air",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(opts)
		},
	}
	cmd.Flags().StringVar(&opts.projectDir, "project-dir", ".", "project directory")
	cmd.Flags().StringVar(&opts.config, "config", ".air.toml", "air config file")
	return cmd
}

func run(opts options) error {
	execSpec, err := resolveExecution(opts, exec.LookPath)
	if err != nil {
		return err
	}
	if execSpec.cleanupPath != "" {
		defer os.Remove(execSpec.cleanupPath)
	}

	cmd := exec.Command(execSpec.airPath, "-c", execSpec.configPath)
	cmd.Dir = execSpec.projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run air: %w", err)
	}
	return nil
}

func resolveExecution(opts options, lookPath func(string) (string, error)) (execution, error) {
	projectDir, err := filepath.Abs(opts.projectDir)
	if err != nil {
		return execution{}, fmt.Errorf("resolve project directory: %w", err)
	}

	spec, err := projectmeta.Load(projectDir)
	if err != nil {
		return execution{}, err
	}

	configPath := opts.config
	cleanupPath := ""
	if configPath == "" || configPath == ".air.toml" {
		configPath, err = writeManagedConfig(projectDir, spec)
		if err != nil {
			return execution{}, err
		}
		cleanupPath = configPath
	} else {
		if !filepath.IsAbs(configPath) {
			configPath = filepath.Join(projectDir, configPath)
		}
		if _, err := os.Stat(configPath); err != nil {
			if os.IsNotExist(err) {
				return execution{}, fmt.Errorf("air config not found at %s", configPath)
			}
			return execution{}, fmt.Errorf("stat air config %s: %w", configPath, err)
		}
	}

	airPath, err := lookPath("air")
	if err != nil {
		return execution{}, fmt.Errorf("air is not installed; run `hatch tools install` first")
	}

	return execution{
		airPath:     airPath,
		projectDir:  projectDir,
		configPath:  configPath,
		cleanupPath: cleanupPath,
	}, nil
}

const managedAirTemplate = `#:schema https://json.schemastore.org/any.json

[build]
  args_bin = [{{ range $i, $arg := .Run.Command }}{{ if $i }}, {{ end }}{{ quote $arg }}{{ end }}]
  bin = {{ quote .BinPath }}
  cmd = {{ quote .BuildCommand }}
  entrypoint = [{{ quote .BinPath }}]
  exclude_dir = ["tmp", "build", ".git"]

[misc]
  clean_on_exit = true

[proxy]
  enabled = false
`

func writeManagedConfig(projectDir string, spec projectmeta.ProjectSpec) (string, error) {
	type managedAirConfig struct {
		Run          projectmeta.Run
		BinPath      string
		BuildCommand string
	}

	data := managedAirConfig{
		Run:          spec.Run,
		BinPath:      filepath.ToSlash(filepath.Join(".", "tmp", spec.BinaryName)),
		BuildCommand: fmt.Sprintf("go build -o %s %s", filepath.ToSlash(filepath.Join(".", "tmp", spec.BinaryName)), spec.Paths.MainPackage),
	}

	tpl, err := template.New("managed-air").Funcs(template.FuncMap{
		"quote": strconv.Quote,
	}).Parse(managedAirTemplate)
	if err != nil {
		return "", fmt.Errorf("parse managed air config: %w", err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render managed air config: %w", err)
	}

	file, err := os.CreateTemp("", "hatch-air-*.toml")
	if err != nil {
		return "", fmt.Errorf("create managed air config: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(buf.Bytes()); err != nil {
		return "", fmt.Errorf("write managed air config: %w", err)
	}
	return file.Name(), nil
}
