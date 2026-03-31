package scaffold

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/ix64/hatch/internal/cli/projectmeta"
)

type fileTemplate struct {
	Path         string
	TemplatePath string
}

//go:embed templates
var templateFS embed.FS

var templates = []fileTemplate{
	{Path: "go.mod", TemplatePath: "templates/go.mod.tmpl"},
	{Path: projectmeta.MetadataFile, TemplatePath: "templates/hatch.toml.tmpl"},
	{Path: "README.md", TemplatePath: "templates/README.md.tmpl"},
	{Path: ".gitignore", TemplatePath: "templates/gitignore.tmpl"},
	{Path: ".air.toml", TemplatePath: "templates/air.toml.tmpl"},
	{Path: "cmd/server/main.go", TemplatePath: "templates/cmd/server/main.go.tmpl"},
	{Path: "cmd/server/internal/root.go", TemplatePath: "templates/cmd/server/internal/root.go.tmpl"},
	{Path: "cmd/server/internal/serve.go", TemplatePath: "templates/cmd/server/internal/serve.go.tmpl"},
	{Path: "internal/config/config.go", TemplatePath: "templates/internal/config/config.go.tmpl"},
	{Path: "internal/register/module.go", TemplatePath: "templates/internal/register/module.go.tmpl"},
	{Path: "internal/register/root.go", TemplatePath: "templates/internal/register/root.go.tmpl"},
	{Path: "config.toml.example", TemplatePath: "templates/config.toml.example.tmpl"},
	{Path: "atlas.hcl", TemplatePath: "templates/atlas.hcl.tmpl"},
	{Path: "dev/compose.yaml", TemplatePath: "templates/dev/compose.yaml.tmpl"},
	{Path: "ddl/embed.go", TemplatePath: "templates/ddl/embed.go.tmpl"},
	{Path: "ddl/schema/task.go", TemplatePath: "templates/ddl/schema/task.go.tmpl"},
	{Path: "proto/buf.yaml", TemplatePath: "templates/proto/buf.yaml.tmpl"},
	{Path: "proto/app/v1/service.proto", TemplatePath: "templates/proto/app/v1/service.proto.tmpl"},
}

func WriteProject(projectDir string, spec projectmeta.ProjectSpec) error {
	spec.Normalize()
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return err
	}

	for _, file := range templates {
		contents, err := render(file.TemplatePath, spec)
		if err != nil {
			return fmt.Errorf("render %s: %w", file.Path, err)
		}
		if err := writeFile(projectDir, file.Path, contents); err != nil {
			return err
		}
	}

	for _, dir := range []string{
		spec.Paths.CompositeDir,
		spec.Proto.OutDir,
	} {
		if err := writeFile(projectDir, filepath.ToSlash(filepath.Join(dir, ".gitkeep")), ""); err != nil {
			return err
		}
	}
	if err := writeFile(projectDir, filepath.ToSlash(filepath.Join(spec.Paths.MigrationsDir, "atlas.sum")), "h1:47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU=\n"); err != nil {
		return err
	}

	return nil
}

func render(templatePath string, spec projectmeta.ProjectSpec) (string, error) {
	var buf bytes.Buffer
	parsed, err := template.ParseFS(templateFS, templatePath)
	if err != nil {
		return "", err
	}
	if err := parsed.Execute(&buf, spec); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func writeFile(projectDir, relPath, contents string) error {
	fullPath := filepath.Join(projectDir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return fmt.Errorf("create dir for %s: %w", relPath, err)
	}
	if err := os.WriteFile(fullPath, []byte(contents), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", relPath, err)
	}
	return nil
}
