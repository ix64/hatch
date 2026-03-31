package projectmeta

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFallsBackToGoMod(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/acme/sample\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	spec, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	if spec.ModulePath != "example.com/acme/sample" {
		t.Fatalf("unexpected module path: %s", spec.ModulePath)
	}
	if spec.BinaryName != "sample" {
		t.Fatalf("unexpected binary name: %s", spec.BinaryName)
	}
	if spec.Paths.MainPackage != "./cmd/server" {
		t.Fatalf("unexpected main package: %s", spec.Paths.MainPackage)
	}
	if spec.Paths.DevCompose != "dev/compose.yaml" {
		t.Fatalf("unexpected dev compose path: %s", spec.Paths.DevCompose)
	}
	if len(spec.Run.Command) != 1 || spec.Run.Command[0] != "serve" {
		t.Fatalf("unexpected run command: %v", spec.Run.Command)
	}
}

func TestSaveAndLoadMetadata(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/acme/app\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	want := New("example.com/acme/app", "Acme App", "acme-app")
	want.HatchReplacePath = "../hatch"
	if err := Save(dir, want); err != nil {
		t.Fatal(err)
	}

	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	if got.HatchReplacePath != "../hatch" {
		t.Fatalf("unexpected hatch replace path: %s", got.HatchReplacePath)
	}
	if got.AppName != "Acme App" {
		t.Fatalf("unexpected app name: %s", got.AppName)
	}
	if got.Proto.Source != "local" {
		t.Fatalf("unexpected proto source: %s", got.Proto.Source)
	}
	if got.Proto.OutDir != "rpc" {
		t.Fatalf("unexpected proto out dir: %s", got.Proto.OutDir)
	}
	if len(got.Run.Command) != 1 || got.Run.Command[0] != "serve" {
		t.Fatalf("unexpected run command: %v", got.Run.Command)
	}
	if len(got.Ent.Features) != len(DefaultEntFeatures) {
		t.Fatalf("unexpected ent feature count: %v", got.Ent.Features)
	}
	for i, feature := range DefaultEntFeatures {
		if got.Ent.Features[i] != feature {
			t.Fatalf("unexpected ent feature at %d: %s", i, got.Ent.Features[i])
		}
	}
}

func TestLoadLegacyProtoFields(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/acme/app\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	content := `
module_path = "example.com/acme/app"
app_name = "Acme App"
binary_name = "acme-app"
proto_enabled = true

[paths]
rpc_dir = "rpc"
buf_gen_file = "buf.gen.yaml"
`
	if err := os.WriteFile(filepath.Join(dir, MetadataFile), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Proto.Enabled {
		t.Fatal("expected proto to be enabled")
	}
	if got.Proto.OutDir != "rpc" {
		t.Fatalf("unexpected proto out dir: %s", got.Proto.OutDir)
	}
	if got.Proto.Source != "local" {
		t.Fatalf("unexpected proto source: %s", got.Proto.Source)
	}
	if len(got.Ent.Features) != len(DefaultEntFeatures) {
		t.Fatalf("unexpected ent feature count: %v", got.Ent.Features)
	}
}

func TestLoadSupportsEmptyEntFeatures(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/acme/app\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	content := `
module_path = "example.com/acme/app"
app_name = "Acme App"
binary_name = "acme-app"

[ent]
features = []
`
	if err := os.WriteFile(filepath.Join(dir, MetadataFile), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Ent.Features == nil {
		t.Fatal("expected ent features to preserve explicit empty list")
	}
	if len(got.Ent.Features) != 0 {
		t.Fatalf("expected no ent features, got: %v", got.Ent.Features)
	}
}

func TestLoadSupportsConfiguredRunCommand(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/acme/app\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	content := `
module_path = "example.com/acme/app"
app_name = "Acme App"
binary_name = "acme-app"

[run]
command = ["worker", "--queue", "critical"]
`
	if err := os.WriteFile(filepath.Join(dir, MetadataFile), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Run.Command) != 3 {
		t.Fatalf("unexpected run command: %v", got.Run.Command)
	}
}

func TestLoadAppliesLocalMetadataOverrides(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/acme/app\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	base := `
module_path = "example.com/acme/app"
app_name = "Acme App"
binary_name = "acme-app"

[paths]
build_output = "build/acme-app"

[run]
command = ["serve"]
`
	if err := os.WriteFile(filepath.Join(dir, MetadataFile), []byte(base), 0o644); err != nil {
		t.Fatal(err)
	}

	local := `
[paths]
build_output = "build/acme-local"

[run]
command = ["worker", "--queue", "critical"]
`
	if err := os.WriteFile(filepath.Join(dir, LocalMetadataFile), []byte(local), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.ModulePath != "example.com/acme/app" {
		t.Fatalf("unexpected module path: %s", got.ModulePath)
	}
	if got.Paths.BuildOutput != "build/acme-local" {
		t.Fatalf("unexpected build output: %s", got.Paths.BuildOutput)
	}
	if len(got.Run.Command) != 3 || got.Run.Command[0] != "worker" {
		t.Fatalf("unexpected run command: %v", got.Run.Command)
	}
}
