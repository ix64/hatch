package devcmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveExecutionDefaults(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := writeProjectFiles(dir, "[run]\ncommand = [\"serve\"]\n"); err != nil {
		t.Fatal(err)
	}

	got, err := resolveExecution(options{projectDir: dir}, func(name string) (string, error) {
		if name != "air" {
			t.Fatalf("unexpected lookup for %q", name)
		}
		return "/usr/bin/air", nil
	})
	if err != nil {
		t.Fatalf("resolveExecution() error = %v", err)
	}
	if got.projectDir != dir {
		t.Fatalf("projectDir = %q", got.projectDir)
	}
	if got.configPath == filepath.Join(dir, ".air.toml") {
		t.Fatalf("configPath unexpectedly used on-disk default config: %q", got.configPath)
	}
	data, err := os.ReadFile(got.configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `args_bin = ["serve"]`) {
		t.Fatalf("managed config missing default run command:\n%s", data)
	}
	if got.airPath != "/usr/bin/air" {
		t.Fatalf("airPath = %q", got.airPath)
	}
	if got.cleanupPath != got.configPath {
		t.Fatalf("cleanupPath = %q, want %q", got.cleanupPath, got.configPath)
	}
}

func TestResolveExecutionCustomConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := writeProjectFiles(dir, ""); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "configs", "air.toml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := resolveExecution(options{
		projectDir: dir,
		config:     filepath.ToSlash(filepath.Join("configs", "air.toml")),
	}, func(string) (string, error) { return "/usr/bin/air", nil })
	if err != nil {
		t.Fatalf("resolveExecution() error = %v", err)
	}
	if got.configPath != configPath {
		t.Fatalf("configPath = %q", got.configPath)
	}
	if got.cleanupPath != "" {
		t.Fatalf("cleanupPath = %q", got.cleanupPath)
	}
}

func TestResolveExecutionUsesConfiguredRunCommand(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := writeProjectFiles(dir, "[run]\ncommand = [\"worker\", \"--queue\", \"critical\"]\n"); err != nil {
		t.Fatal(err)
	}

	got, err := resolveExecution(options{projectDir: dir}, func(string) (string, error) {
		return "/usr/bin/air", nil
	})
	if err != nil {
		t.Fatalf("resolveExecution() error = %v", err)
	}

	data, err := os.ReadFile(got.configPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, `args_bin = ["worker", "--queue", "critical"]`) {
		t.Fatalf("managed config missing configured run command:\n%s", text)
	}
}

func TestResolveExecutionMissingAir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := writeProjectFiles(dir, ""); err != nil {
		t.Fatal(err)
	}

	_, err := resolveExecution(options{projectDir: dir}, func(string) (string, error) {
		return "", errors.New("not found")
	})
	if err == nil || err.Error() != "air is not installed; run `hatch tools install` first" {
		t.Fatalf("resolveExecution() error = %v", err)
	}
}

func TestResolveExecutionMissingConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := writeProjectFiles(dir, ""); err != nil {
		t.Fatal(err)
	}

	_, err := resolveExecution(options{
		projectDir: dir,
		config:     filepath.ToSlash(filepath.Join("configs", "air.toml")),
	}, func(string) (string, error) {
		return "/usr/bin/air", nil
	})
	if err == nil {
		t.Fatal("resolveExecution() unexpectedly succeeded")
	}
	if want := "air config not found"; !strings.HasPrefix(err.Error(), want) {
		t.Fatalf("resolveExecution() error = %v", err)
	}
}

func writeProjectFiles(dir string, extra string) error {
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/acme/demo\n\ngo 1.26.0\n"), 0o644); err != nil {
		return err
	}
	if extra == "" {
		return os.WriteFile(filepath.Join(dir, "hatch.toml"), []byte(""), 0o644)
	}
	return os.WriteFile(filepath.Join(dir, "hatch.toml"), []byte(extra), 0o644)
}
