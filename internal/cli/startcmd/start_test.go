package startcmd

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestResolveExecutionDefaults(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProjectFiles(t, dir, filepath.ToSlash(filepath.Join("build", "demo")))

	got, err := resolveExecution(options{projectDir: dir})
	if err != nil {
		t.Fatalf("resolveExecution() error = %v", err)
	}

	wantBinary := filepath.Join(dir, normalizeBinaryPath(filepath.Join("build", "demo")))
	if got.projectDir != dir {
		t.Fatalf("projectDir = %q", got.projectDir)
	}
	if got.binaryPath != wantBinary {
		t.Fatalf("binaryPath = %q, want %q", got.binaryPath, wantBinary)
	}
	if len(got.args) != 1 || got.args[0] != "serve" {
		t.Fatalf("args = %q", got.args)
	}
}

func TestResolveExecutionCustomBinary(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProjectFiles(t, dir, filepath.ToSlash(filepath.Join("build", "demo")))

	customPath := filepath.Join(dir, "out", normalizeBinaryPath("app"))
	if err := os.MkdirAll(filepath.Dir(customPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(customPath, []byte(""), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := resolveExecution(options{projectDir: dir, binary: filepath.ToSlash(filepath.Join("out", "app"))})
	if err != nil {
		t.Fatalf("resolveExecution() error = %v", err)
	}
	if got.binaryPath != customPath {
		t.Fatalf("binaryPath = %q, want %q", got.binaryPath, customPath)
	}
}

func TestResolveExecutionCustomBuildOutputFromMetadata(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProjectFiles(t, dir, "dist/service")

	got, err := resolveExecution(options{projectDir: dir})
	if err != nil {
		t.Fatalf("resolveExecution() error = %v", err)
	}

	wantBinary := filepath.Join(dir, normalizeBinaryPath(filepath.Join("dist", "service")))
	if got.binaryPath != wantBinary {
		t.Fatalf("binaryPath = %q, want %q", got.binaryPath, wantBinary)
	}
}

func TestResolveExecutionUsesConfiguredRunCommand(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProjectFiles(t, dir, "dist/service", "worker", "--queue", "critical")

	got, err := resolveExecution(options{projectDir: dir})
	if err != nil {
		t.Fatalf("resolveExecution() error = %v", err)
	}
	if len(got.args) != 3 || got.args[0] != "worker" || got.args[1] != "--queue" || got.args[2] != "critical" {
		t.Fatalf("args = %q", got.args)
	}
}

func TestResolveExecutionMissingBinary(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/acme/demo\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "hatch.toml"), []byte("[paths]\nbuild_output = \"build/demo\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := resolveExecution(options{projectDir: dir})
	if err == nil {
		t.Fatal("resolveExecution() unexpectedly succeeded")
	}
}

func writeProjectFiles(t *testing.T, projectDir string, buildOutput string, runCommand ...string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module example.com/acme/demo\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if buildOutput != "" {
		content := "[paths]\n" + `build_output = "` + buildOutput + "\"\n"
		if len(runCommand) > 0 {
			content += "\n[run]\ncommand = ["
			for i, arg := range runCommand {
				if i > 0 {
					content += ", "
				}
				content += strconv.Quote(arg)
			}
			content += "]\n"
		}
		if err := os.WriteFile(filepath.Join(projectDir, "hatch.toml"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	} else {
		buildOutput = filepath.ToSlash(filepath.Join("build", "demo"))
	}

	binaryPath := filepath.Join(projectDir, filepath.FromSlash(normalizeBinaryPath(buildOutput)))
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binaryPath, []byte(""), 0o755); err != nil {
		t.Fatal(err)
	}
}
