package envcmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnvCommandIncludesAddSubcommand(t *testing.T) {
	cmd := New()

	var found bool
	for _, child := range cmd.Commands() {
		if child.Name() == "add" {
			found = true
			if len(child.ValidArgs) != 5 {
				t.Fatalf("unexpected add valid args: %v", child.ValidArgs)
			}
		}
	}
	if !found {
		t.Fatal("expected add subcommand")
	}
}

func TestAddServiceMinioAddsComposeAndConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeEnvAddProject(t, dir, baseComposeYAML, baseConfigExample)

	if err := addService(dir, "minio"); err != nil {
		t.Fatalf("addService() error = %v", err)
	}

	compose := readFile(t, filepath.Join(dir, "dev", "compose.yaml"))
	for _, snippet := range []string{"minio:", "minio-setup:", "minio-data:", "29000:9000", "demo-dev"} {
		if !strings.Contains(compose, snippet) {
			t.Fatalf("compose missing %q:\n%s", snippet, compose)
		}
	}

	config := readFile(t, filepath.Join(dir, "config.toml.example"))
	for _, snippet := range []string{"[object_storage]", `endpoint = "http://localhost:29000"`, `bucket = "demo-dev"`, `access_key = "demo"`} {
		if !strings.Contains(config, snippet) {
			t.Fatalf("config missing %q:\n%s", snippet, config)
		}
	}
}

func TestAddServicePostgresAddsComposeAndConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeEnvAddProject(t, dir, emptyComposeYAML, baseConfigExample)

	if err := addService(dir, "postgres"); err != nil {
		t.Fatalf("addService() error = %v", err)
	}

	compose := readFile(t, filepath.Join(dir, "dev", "compose.yaml"))
	for _, snippet := range []string{"postgres:", "25432:5432", `POSTGRES_DB: "app"`, `POSTGRES_USER: "postgres"`, `POSTGRES_PASSWORD: "postgres"`} {
		if !strings.Contains(compose, snippet) {
			t.Fatalf("compose missing %q:\n%s", snippet, compose)
		}
	}

	config := readFile(t, filepath.Join(dir, "config.toml.example"))
	for _, snippet := range []string{"[db]", `dsn = "postgres://postgres:postgres@localhost:25432/app?search_path=public&sslmode=disable"`, "migrate = true"} {
		if !strings.Contains(config, snippet) {
			t.Fatalf("config missing %q:\n%s", snippet, config)
		}
	}
}

func TestAddServiceMailpitAddsComposeAndCommentOnly(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeEnvAddProject(t, dir, baseComposeYAML, baseConfigExample)

	if err := addService(dir, "mailpit"); err != nil {
		t.Fatalf("addService() error = %v", err)
	}

	compose := readFile(t, filepath.Join(dir, "dev", "compose.yaml"))
	for _, snippet := range []string{"mailpit:", "1025:1025", "8025:8025"} {
		if !strings.Contains(compose, snippet) {
			t.Fatalf("compose missing %q:\n%s", snippet, compose)
		}
	}

	config := readFile(t, filepath.Join(dir, "config.toml.example"))
	if !strings.Contains(config, "Mailpit for local dev") {
		t.Fatalf("config missing mailpit note:\n%s", config)
	}
	if strings.Contains(config, "[mailpit]") {
		t.Fatalf("config unexpectedly added [mailpit] section:\n%s", config)
	}
}

func TestAddServiceValkeyAddsComposeAndConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeEnvAddProject(t, dir, baseComposeYAML, baseConfigExample)

	if err := addService(dir, "valkey"); err != nil {
		t.Fatalf("addService() error = %v", err)
	}

	compose := readFile(t, filepath.Join(dir, "dev", "compose.yaml"))
	for _, snippet := range []string{"valkey:", "valkey-data:", "26379:6379", "demo-dev"} {
		if !strings.Contains(compose, snippet) {
			t.Fatalf("compose missing %q:\n%s", snippet, compose)
		}
	}

	config := readFile(t, filepath.Join(dir, "config.toml.example"))
	if !strings.Contains(config, `[valkey]`) || !strings.Contains(config, `redis://:demo-dev@localhost:26379/0`) {
		t.Fatalf("config missing valkey example:\n%s", config)
	}
}

func TestAddServiceOpenBaoAddsComposeAndConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeEnvAddProject(t, dir, baseComposeYAML, baseConfigExample)

	if err := addService(dir, "openbao"); err != nil {
		t.Fatalf("addService() error = %v", err)
	}

	compose := readFile(t, filepath.Join(dir, "dev", "compose.yaml"))
	for _, snippet := range []string{"openbao:", "28200:8200", "IPC_LOCK", "demo-dev-root"} {
		if !strings.Contains(compose, snippet) {
			t.Fatalf("compose missing %q:\n%s", snippet, compose)
		}
	}

	config := readFile(t, filepath.Join(dir, "config.toml.example"))
	for _, snippet := range []string{"[openbao]", `address = "http://localhost:28200"`, `token = "demo-dev-root"`} {
		if !strings.Contains(config, snippet) {
			t.Fatalf("config missing %q:\n%s", snippet, config)
		}
	}
}

func TestAddServiceIsIdempotent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeEnvAddProject(t, dir, baseComposeYAML, baseConfigExample)

	if err := addService(dir, "minio"); err != nil {
		t.Fatalf("addService() error = %v", err)
	}
	if err := addService(dir, "minio"); err != nil {
		t.Fatalf("second addService() error = %v", err)
	}

	compose := readFile(t, filepath.Join(dir, "dev", "compose.yaml"))
	for _, name := range []string{"\n  minio:\n", "\n  minio-setup:\n", "\n  minio-data:"} {
		if got := strings.Count(compose, name); got != 1 {
			t.Fatalf("expected %q once, got %d:\n%s", name, got, compose)
		}
	}

	config := readFile(t, filepath.Join(dir, "config.toml.example"))
	if got := strings.Count(config, "[object_storage]"); got != 1 {
		t.Fatalf("expected [object_storage] once, got %d:\n%s", got, config)
	}
}

func TestAddServicePostgresIsIdempotent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeEnvAddProject(t, dir, emptyComposeYAML, baseConfigExample)

	if err := addService(dir, "postgres"); err != nil {
		t.Fatalf("addService() error = %v", err)
	}
	if err := addService(dir, "postgres"); err != nil {
		t.Fatalf("second addService() error = %v", err)
	}

	compose := readFile(t, filepath.Join(dir, "dev", "compose.yaml"))
	root, err := parseComposeRoot([]byte(compose))
	if err != nil {
		t.Fatalf("parseComposeRoot() error = %v", err)
	}
	services := mapValue(root, "services")
	if services == nil {
		t.Fatalf("expected services section:\n%s", compose)
	}
	if got := mapValue(services, "postgres"); got == nil {
		t.Fatalf("expected postgres service:\n%s", compose)
	}

	config := readFile(t, filepath.Join(dir, "config.toml.example"))
	if got := strings.Count(config, "[db]"); got != 1 {
		t.Fatalf("expected [db] once, got %d:\n%s", got, config)
	}
}

func TestAddServiceMinioCompletesPartialGroup(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeEnvAddProject(t, dir, `name: demo_dev

services:
  postgres:
    image: "postgres:18"
  minio:
    image: "pgsty/minio:latest"
`, baseConfigExample)

	if err := addService(dir, "minio"); err != nil {
		t.Fatalf("addService() error = %v", err)
	}

	compose := readFile(t, filepath.Join(dir, "dev", "compose.yaml"))
	if got := strings.Count(compose, "\n  minio:\n"); got != 1 {
		t.Fatalf("expected minio once, got %d:\n%s", got, compose)
	}
	for _, snippet := range []string{"minio-setup:", "minio-data:"} {
		if !strings.Contains(compose, snippet) {
			t.Fatalf("compose missing %q:\n%s", snippet, compose)
		}
	}
}

func TestAddServiceRejectsUnknownService(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeEnvAddProject(t, dir, baseComposeYAML, baseConfigExample)

	err := addService(dir, "unknown")
	if err == nil || !strings.Contains(err.Error(), "unsupported env service") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddServiceRejectsInvalidComposeYAML(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeEnvAddProject(t, dir, "services: [\n", baseConfigExample)

	err := addService(dir, "mailpit")
	if err == nil || !strings.Contains(err.Error(), "parse dev compose file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddServiceRejectsMissingCompose(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeEnvAddProjectFiles(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "config.toml.example"), []byte(baseConfigExample), 0o644); err != nil {
		t.Fatal(err)
	}

	err := addService(dir, "mailpit")
	if err == nil || !strings.Contains(err.Error(), "dev compose file not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddServiceRejectsMissingConfigExample(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeEnvAddProjectFiles(t, dir)
	if err := os.MkdirAll(filepath.Join(dir, "dev"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "dev", "compose.yaml"), []byte(baseComposeYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	err := addService(dir, "mailpit")
	if err == nil || !strings.Contains(err.Error(), "config example file not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

const baseComposeYAML = `name: demo_dev

services:
  postgres:
    image: "postgres:18"
`

const emptyComposeYAML = `name: demo_dev

services: {}
`

const baseConfigExample = `[server]
addr = ":9580"
debug = true

[logger]
level = "DEBUG"
debug = true
`

func writeEnvAddProject(t *testing.T, dir, compose, config string) {
	t.Helper()
	writeEnvAddProjectFiles(t, dir)
	if err := os.MkdirAll(filepath.Join(dir, "dev"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "dev", "compose.yaml"), []byte(compose), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml.example"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeEnvAddProjectFiles(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/demo\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "hatch.toml"), []byte("binary_name = \"demo\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
