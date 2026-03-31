package scaffold

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ix64/hatch/internal/cli/projectmeta"
)

func TestWriteProject(t *testing.T) {
	dir := t.TempDir()
	spec := projectmeta.New("example.com/acme/demo", "Demo Service", "demo-service")
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	spec.HatchReplacePath = repoRoot

	if err := WriteProject(dir, spec); err != nil {
		t.Fatal(err)
	}

	paths := []string{
		"go.mod",
		"hatch.toml",
		".gitignore",
		"dev/compose.yaml",
		".air.toml",
		"cmd/server/internal/root.go",
		"internal/register/module.go",
		"ddl/schema/task.go",
		"proto/buf.yaml",
		"proto/app/v1/service.proto",
	}
	for _, rel := range paths {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("expected %s: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "Taskfile.yaml")); !os.IsNotExist(err) {
		t.Fatalf("expected Taskfile.yaml to be absent, got err=%v", err)
	}

	goMod, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(goMod), "module example.com/acme/demo") {
		t.Fatalf("go.mod missing module declaration:\n%s", goMod)
	}
	if !strings.Contains(string(goMod), "replace github.com/ix64/hatch => "+repoRoot) {
		t.Fatalf("go.mod missing replace directive:\n%s", goMod)
	}

	readme, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readme), "# Demo Service") {
		t.Fatalf("README.md missing rendered app name:\n%s", readme)
	}

	compose, err := os.ReadFile(filepath.Join(dir, "dev", "compose.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, snippet := range []string{"postgres:", "name: demo-service_dev"} {
		if !strings.Contains(string(compose), snippet) {
			t.Fatalf("dev/compose.yaml missing %q:\n%s", snippet, compose)
		}
	}
	for _, service := range []string{"minio:", "mailpit:", "valkey:", "openbao:"} {
		if strings.Contains(string(compose), service) {
			t.Fatalf("dev/compose.yaml unexpectedly includes %q:\n%s", service, compose)
		}
	}

	gitignore, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	for _, pattern := range []string{"build/", "tmp/", "config.toml", "hatch.local.toml"} {
		if !strings.Contains(string(gitignore), pattern) {
			t.Fatalf(".gitignore missing %q:\n%s", pattern, gitignore)
		}
	}

	metadata, err := os.ReadFile(filepath.Join(dir, "hatch.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(metadata), "[ent]") {
		t.Fatalf("hatch.toml missing ent section:\n%s", metadata)
	}
	if !strings.Contains(string(metadata), "\"sql/upsert\"") {
		t.Fatalf("hatch.toml missing default ent features:\n%s", metadata)
	}
	if !strings.Contains(string(metadata), "[run]") || !strings.Contains(string(metadata), "\"serve\"") {
		t.Fatalf("hatch.toml missing default run command:\n%s", metadata)
	}

	runCmd(t, dir, "go", "mod", "tidy")
	runCmd(t, dir, "go", "build", "./cmd/server")
	if _, err := exec.LookPath("buf"); err == nil {
		runCmd(t, dir, "go", "run", "-mod=mod", "github.com/ix64/hatch/cmd/hatch", "gen", "rpc", "--project-dir", ".")
		runCmd(t, dir, "go", "test", "./...")
	}
}

func runCmd(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s failed: %v\n%s", name, strings.Join(args, " "), err, output)
	}
}
