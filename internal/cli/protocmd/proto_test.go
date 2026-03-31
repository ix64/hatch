package protocmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ix64/hatch/internal/cli/projectmeta"
)

func TestResolveInputPrefersLocalOverride(t *testing.T) {
	dir := t.TempDir()
	overrideDir := filepath.Join(dir, "..", "proto")
	if err := os.MkdirAll(overrideDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(overrideDir, "buf.yaml"), []byte("version: v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	spec := projectmeta.New("example.com/app", "App", "app")
	spec.Proto.Source = "git"
	spec.Proto.GitRepo = "ssh://git@example.com/proto.git"
	spec.Proto.LocalOverrideDir = "../proto"

	in, err := resolveInput(dir, spec)
	if err != nil {
		t.Fatal(err)
	}
	if in.Directory != "../proto" {
		t.Fatalf("unexpected directory: %s", in.Directory)
	}
}

func TestResolveInputFallsBackToGit(t *testing.T) {
	dir := t.TempDir()
	spec := projectmeta.New("example.com/app", "App", "app")
	spec.Proto.Source = "git"
	spec.Proto.GitRepo = "ssh://git@example.com/proto.git"

	in, err := resolveInput(dir, spec)
	if err != nil {
		t.Fatal(err)
	}
	if in.GitRepo != "ssh://git@example.com/proto.git" {
		t.Fatalf("unexpected git repo: %s", in.GitRepo)
	}
	if in.Branch != "main" {
		t.Fatalf("unexpected branch: %s", in.Branch)
	}
}

func TestGenerateLocalProto(t *testing.T) {
	if _, err := exec.LookPath("buf"); err != nil {
		t.Skip("buf not installed")
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/demo\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "hatch.toml"), []byte(`
module_path = "example.com/demo"
app_name = "Demo"
binary_name = "demo"

[proto]
enabled = true
source = "local"
dir = "proto"
out_dir = "rpc"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	protoDir := filepath.Join(dir, "proto")
	if err := os.MkdirAll(filepath.Join(protoDir, "example", "v1"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(protoDir, "buf.yaml"), []byte("version: v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	protoContent := `syntax = "proto3";
package example.v1;
service DemoService {
  rpc Ping(PingRequest) returns (PingResponse);
}
message PingRequest {}
message PingResponse {
  string message = 1;
}
`
	if err := os.WriteFile(filepath.Join(protoDir, "example", "v1", "demo.proto"), []byte(protoContent), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Generate(dir); err != nil {
		t.Fatal(err)
	}

	var pbFound bool
	var connectFound bool
	walkErr := filepath.Walk(filepath.Join(dir, "rpc"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		switch {
		case strings.HasSuffix(path, ".pb.go"):
			pbFound = true
		case strings.HasSuffix(path, ".connect.go"):
			connectFound = true
		}
		return nil
	})
	if walkErr != nil {
		t.Fatal(walkErr)
	}
	if !pbFound {
		t.Fatal("expected at least one generated .pb.go file")
	}
	if !connectFound {
		t.Fatal("expected at least one generated .connect.go file")
	}
}

func TestWriteTemplateFileIncludesGitInput(t *testing.T) {
	spec := projectmeta.New("example.com/demo", "Demo", "demo")
	spec.Proto.Source = "git"
	spec.Proto.OutDir = "rpc"

	path, err := writeTemplateFile(spec, input{
		GitRepo: "ssh://git@example.com/demo-proto.git",
		Branch:  "master",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "git_repo: ssh://git@example.com/demo-proto.git") {
		t.Fatalf("missing git repo in template:\n%s", text)
	}
	if !strings.Contains(text, "value: example.com/demo/rpc") {
		t.Fatalf("missing go package prefix in template:\n%s", text)
	}
}
