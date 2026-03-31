package protocmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"

	"github.com/ix64/hatch/internal/cli/projectmeta"
)

type input struct {
	Directory string
	GitRepo   string
	Branch    string
}

type templateData struct {
	GoPackagePrefix string
	OutDir          string
	Input           input
}

const bufGenTemplate = `version: v2
managed:
  enabled: true
  override:
    - file_option: go_package_prefix
      value: {{ .GoPackagePrefix }}
plugins:
  - local: protoc-gen-go
    out: {{ .OutDir }}
    opt: paths=source_relative
  - local: protoc-gen-connect-go
    out: {{ .OutDir }}
    opt: paths=source_relative
inputs:
{{- if .Input.Directory }}
  - directory: {{ .Input.Directory }}
{{- else }}
  - git_repo: {{ .Input.GitRepo }}
    branch: {{ .Input.Branch }}
{{- end }}
`

func New() *cobra.Command {
	var projectDir string
	cmd := &cobra.Command{
		Use:   "rpc",
		Short: "Generate protobuf and Connect RPC code",
		RunE: func(cmd *cobra.Command, args []string) error {
			return Generate(projectDir)
		},
	}
	cmd.Flags().StringVar(&projectDir, "project-dir", ".", "project directory")
	return cmd
}

func Generate(projectDir string) error {
	projectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	spec, err := projectmeta.Load(projectDir)
	if err != nil {
		return err
	}
	if !spec.Proto.Enabled {
		return errors.New("proto generation is disabled in hatch.toml")
	}

	in, err := resolveInput(projectDir, spec)
	if err != nil {
		return err
	}
	outDir := filepath.Join(projectDir, filepath.FromSlash(spec.Proto.OutDir))
	if err := os.RemoveAll(outDir); err != nil {
		return fmt.Errorf("clear proto output dir: %w", err)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create proto output dir: %w", err)
	}
	templatePath, err := writeTemplateFile(spec, in)
	if err != nil {
		return err
	}
	defer os.Remove(templatePath)

	cmd := exec.Command("buf", "generate", "--template", templatePath)
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("buf generate: %w", err)
	}

	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = projectDir
	tidy.Stdout = os.Stdout
	tidy.Stderr = os.Stderr
	if err := tidy.Run(); err != nil {
		return fmt.Errorf("go mod tidy: %w", err)
	}
	return nil
}

func resolveInput(projectDir string, spec projectmeta.ProjectSpec) (input, error) {
	switch spec.Proto.Source {
	case "local":
		dir := spec.Proto.Dir
		if err := requireBufYAML(projectDir, dir); err != nil {
			return input{}, err
		}
		return input{Directory: dir}, nil
	case "git":
		if spec.Proto.LocalOverrideDir != "" {
			if info, err := os.Stat(filepath.Join(projectDir, filepath.FromSlash(spec.Proto.LocalOverrideDir))); err == nil && info.IsDir() {
				if err := requireBufYAML(projectDir, spec.Proto.LocalOverrideDir); err != nil {
					return input{}, err
				}
				return input{Directory: spec.Proto.LocalOverrideDir}, nil
			}
		}
		if spec.Proto.GitRepo == "" {
			return input{}, errors.New("proto.git_repo is required when proto.source is git")
		}
		branch := spec.Proto.GitBranch
		if branch == "" {
			branch = "main"
		}
		return input{GitRepo: spec.Proto.GitRepo, Branch: branch}, nil
	default:
		return input{}, fmt.Errorf("unsupported proto source: %s", spec.Proto.Source)
	}
}

func requireBufYAML(projectDir, protoDir string) error {
	bufYAMLPath := filepath.Join(projectDir, filepath.FromSlash(protoDir), "buf.yaml")
	if _, err := os.Stat(bufYAMLPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("missing buf.yaml in %s", filepath.ToSlash(filepath.Join(protoDir, "buf.yaml")))
		}
		return err
	}
	return nil
}

func writeTemplateFile(spec projectmeta.ProjectSpec, in input) (string, error) {
	file, err := os.CreateTemp("", "hatch-buf-gen-*.yaml")
	if err != nil {
		return "", err
	}
	defer file.Close()

	data := templateData{
		GoPackagePrefix: strings.TrimSuffix(spec.ModulePath, "/") + "/" + strings.TrimPrefix(spec.Proto.OutDir, "./"),
		OutDir:          filepath.ToSlash(spec.Proto.OutDir),
		Input:           in,
	}

	tpl, err := template.New("buf").Parse(bufGenTemplate)
	if err != nil {
		return "", err
	}
	if err := tpl.Execute(file, data); err != nil {
		return "", err
	}
	return file.Name(), nil
}
