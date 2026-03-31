package envcmd

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
	writeProjectFiles(t, dir, "")

	got, err := resolveExecution("start", options{projectDir: dir}, func(name string) (string, error) {
		if name != "docker" {
			t.Fatalf("unexpected lookup for %q", name)
		}
		return "/usr/bin/docker", nil
	})
	if err != nil {
		t.Fatalf("resolveExecution() error = %v", err)
	}

	wantCompose := filepath.Join(dir, "dev", "compose.yaml")
	if got.projectDir != dir {
		t.Fatalf("projectDir = %q", got.projectDir)
	}
	if got.composePath != wantCompose {
		t.Fatalf("composePath = %q, want %q", got.composePath, wantCompose)
	}
	wantArgs := []string{"compose", "-f", wantCompose, "up", "--pull", "always", "--detach"}
	assertArgs(t, got.args, wantArgs)
}

func TestResolveExecutionLegacyComposePath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProjectFiles(t, dir, "deploy/dev/compose.yaml")

	got, err := resolveExecution("clean", options{projectDir: dir}, func(string) (string, error) {
		return "/usr/bin/docker", nil
	})
	if err != nil {
		t.Fatalf("resolveExecution() error = %v", err)
	}

	wantCompose := filepath.Join(dir, "deploy", "dev", "compose.yaml")
	if got.composePath != wantCompose {
		t.Fatalf("composePath = %q, want %q", got.composePath, wantCompose)
	}
	wantArgs := []string{"compose", "-f", wantCompose, "down", "--volumes"}
	assertArgs(t, got.args, wantArgs)
}

func TestResolveExecutionStopAction(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProjectFiles(t, dir, "")

	got, err := resolveExecution("stop", options{projectDir: dir}, func(string) (string, error) {
		return "/usr/bin/docker", nil
	})
	if err != nil {
		t.Fatalf("resolveExecution() error = %v", err)
	}

	wantCompose := filepath.Join(dir, "dev", "compose.yaml")
	wantArgs := []string{"compose", "-f", wantCompose, "down"}
	assertArgs(t, got.args, wantArgs)
}

func TestResolveExecutionMissingCompose(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/demo\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := resolveExecution("start", options{projectDir: dir}, func(string) (string, error) {
		return "/usr/bin/docker", nil
	})
	if err == nil {
		t.Fatal("resolveExecution() unexpectedly succeeded")
	}
	if want := "dev compose file not found"; !strings.HasPrefix(err.Error(), want) {
		t.Fatalf("resolveExecution() error = %v", err)
	}
}

func TestResolveExecutionMissingDocker(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProjectFiles(t, dir, "")

	_, err := resolveExecution("start", options{projectDir: dir}, func(string) (string, error) {
		return "", errors.New("not found")
	})
	if err == nil || err.Error() != "docker is not installed; install Docker first" {
		t.Fatalf("resolveExecution() error = %v", err)
	}
}

func TestResolveExecutionInvalidProjectDir(t *testing.T) {
	t.Parallel()

	_, err := resolveExecution("start", options{projectDir: string([]byte{0})}, func(string) (string, error) {
		return "/usr/bin/docker", nil
	})
	if err == nil {
		t.Fatal("resolveExecution() unexpectedly succeeded")
	}
}

func writeProjectFiles(t *testing.T, projectDir string, devCompose string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module example.com/demo\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if devCompose != "" {
		content := "[paths]\n" + `dev_compose = "` + devCompose + "\"\n"
		if err := os.WriteFile(filepath.Join(projectDir, "hatch.toml"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	} else {
		devCompose = "dev/compose.yaml"
	}

	composePath := filepath.Join(projectDir, filepath.FromSlash(devCompose))
	if err := os.MkdirAll(filepath.Dir(composePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(composePath, []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertArgs(t *testing.T, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("args length = %d, want %d; args=%q", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("args[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
