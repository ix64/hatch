package envcmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/ix64/hatch/internal/cli/projectmeta"
)

type addOptions struct {
	projectDir string
}

type serviceDefinition struct {
	names          []string
	composeEntries map[string]func(projectmeta.ProjectSpec) string
	volumes        []string
	configMarkers  []string
	configSnippet  func(projectmeta.ProjectSpec) string
}

var serviceCatalog = map[string]serviceDefinition{
	"postgres": {
		names: []string{"postgres"},
		composeEntries: map[string]func(projectmeta.ProjectSpec) string{
			"postgres": postgresServiceYAML,
		},
		configMarkers: []string{"[db]"},
		configSnippet: postgresConfigSnippet,
	},
	"minio": {
		names: []string{"minio", "minio-setup"},
		composeEntries: map[string]func(projectmeta.ProjectSpec) string{
			"minio":       minioServiceYAML,
			"minio-setup": minioSetupServiceYAML,
		},
		volumes:       []string{"minio-data"},
		configMarkers: []string{"[object_storage]"},
		configSnippet: minioConfigSnippet,
	},
	"mailpit": {
		names: []string{"mailpit"},
		composeEntries: map[string]func(projectmeta.ProjectSpec) string{
			"mailpit": mailpitServiceYAML,
		},
		configMarkers: []string{"# Mailpit for local dev via `hatch env add mailpit`"},
		configSnippet: mailpitConfigSnippet,
	},
	"valkey": {
		names: []string{"valkey"},
		composeEntries: map[string]func(projectmeta.ProjectSpec) string{
			"valkey": valkeyServiceYAML,
		},
		volumes:       []string{"valkey-data"},
		configMarkers: []string{"[valkey]"},
		configSnippet: valkeyConfigSnippet,
	},
	"openbao": {
		names: []string{"openbao"},
		composeEntries: map[string]func(projectmeta.ProjectSpec) string{
			"openbao": openbaoServiceYAML,
		},
		configMarkers: []string{"[openbao]"},
		configSnippet: openbaoConfigSnippet,
	},
}

func newAddCmd() *cobra.Command {
	opts := addOptions{}
	cmd := &cobra.Command{
		Use:       "add <service>",
		Short:     "Add a local development dependency to the compose file",
		Long:      "Add a local development dependency to dev/compose.yaml and append matching config.toml.example guidance. Supported services: postgres, minio, mailpit, valkey, openbao.",
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"postgres", "minio", "mailpit", "valkey", "openbao"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return addService(opts.projectDir, args[0])
		},
	}
	cmd.Flags().StringVar(&opts.projectDir, "project-dir", ".", "project directory")
	return cmd
}

func addService(projectDir, serviceName string) error {
	def, ok := serviceCatalog[serviceName]
	if !ok {
		return fmt.Errorf("unsupported env service: %s", serviceName)
	}

	projectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return fmt.Errorf("resolve project directory: %w", err)
	}

	spec, err := projectmeta.Load(projectDir)
	if err != nil {
		return err
	}

	composePath := spec.Paths.DevCompose
	if !filepath.IsAbs(composePath) {
		composePath = filepath.Join(projectDir, filepath.FromSlash(composePath))
	}
	configPath := spec.Paths.ConfigExample
	if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(projectDir, filepath.FromSlash(configPath))
	}

	composeData, err := os.ReadFile(composePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("dev compose file not found at %s", composePath)
		}
		return fmt.Errorf("read dev compose file %s: %w", composePath, err)
	}
	configData, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("config example file not found at %s", configPath)
		}
		return fmt.Errorf("read config example file %s: %w", configPath, err)
	}

	updatedCompose, err := applyComposeService(composeData, spec, def)
	if err != nil {
		return err
	}
	updatedConfig := applyConfigSnippet(configData, spec, def)

	if err := os.WriteFile(composePath, updatedCompose, 0o644); err != nil {
		return fmt.Errorf("write dev compose file %s: %w", composePath, err)
	}
	if err := os.WriteFile(configPath, updatedConfig, 0o644); err != nil {
		return fmt.Errorf("write config example file %s: %w", configPath, err)
	}

	fmt.Printf("added %s to %s and %s\n", serviceName, filepath.ToSlash(spec.Paths.DevCompose), filepath.ToSlash(spec.Paths.ConfigExample))
	return nil
}

func applyComposeService(data []byte, spec projectmeta.ProjectSpec, def serviceDefinition) ([]byte, error) {
	root, err := parseComposeRoot(data)
	if err != nil {
		return nil, err
	}

	services := ensureMapValue(root, "services")
	for _, name := range def.names {
		if mapValue(services, name) != nil {
			continue
		}
		entry, err := parseMapValue(def.composeEntries[name](spec))
		if err != nil {
			return nil, fmt.Errorf("parse compose service %s: %w", name, err)
		}
		appendMapEntry(services, name, entry)
	}

	if len(def.volumes) > 0 {
		volumes := ensureMapValue(root, "volumes")
		for _, name := range def.volumes {
			if mapValue(volumes, name) != nil {
				continue
			}
			appendMapEntry(volumes, name, &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"})
		}
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		return nil, fmt.Errorf("encode dev compose file: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("close dev compose encoder: %w", err)
	}
	return buf.Bytes(), nil
}

func parseComposeRoot(data []byte) (*yaml.Node, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return &yaml.Node{
			Kind: yaml.MappingNode,
			Tag:  "!!map",
		}, nil
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse dev compose file: %w", err)
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil, fmt.Errorf("parse dev compose file: missing document root")
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("parse dev compose file: root must be a mapping")
	}
	return root, nil
}

func ensureMapValue(root *yaml.Node, key string) *yaml.Node {
	if value := mapValue(root, key); value != nil {
		if value.Kind == 0 {
			value.Kind = yaml.MappingNode
			value.Tag = "!!map"
		}
		return value
	}

	value := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	appendMapEntry(root, key, value)
	return value
}

func mapValue(root *yaml.Node, key string) *yaml.Node {
	if root == nil || root.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(root.Content)-1; i += 2 {
		if root.Content[i].Value == key {
			return root.Content[i+1]
		}
	}
	return nil
}

func appendMapEntry(root *yaml.Node, key string, value *yaml.Node) {
	root.Content = append(root.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		value,
	)
}

func parseMapValue(snippet string) (*yaml.Node, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(snippet), &doc); err != nil {
		return nil, err
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil, fmt.Errorf("missing document root")
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode || len(root.Content) != 2 {
		return nil, fmt.Errorf("expected a single mapping entry")
	}
	return root.Content[1], nil
}

func applyConfigSnippet(data []byte, spec projectmeta.ProjectSpec, def serviceDefinition) []byte {
	text := string(data)
	for _, marker := range def.configMarkers {
		if strings.Contains(text, marker) {
			return data
		}
	}

	snippet := strings.TrimSpace(def.configSnippet(spec))
	if snippet == "" {
		return data
	}
	if strings.TrimSpace(text) == "" {
		return []byte(snippet + "\n")
	}

	text = strings.TrimRight(text, "\n")
	return []byte(text + "\n\n" + snippet + "\n")
}

func minioServiceYAML(spec projectmeta.ProjectSpec) string {
	user := spec.BinaryName
	password := minioSecret(spec)
	return fmt.Sprintf(`minio:
  image: "pgsty/minio:latest"
  restart: unless-stopped
  ports:
    - "0.0.0.0:29000:9000"
    - "0.0.0.0:29001:9001"
  command: server /data --console-address ":9001"
  environment:
    MINIO_ROOT_USER: %q
    MINIO_ROOT_PASSWORD: %q
  volumes:
    - minio-data:/data
`, user, password)
}

func minioSetupServiceYAML(spec projectmeta.ProjectSpec) string {
	user := spec.BinaryName
	password := minioSecret(spec)
	bucket := minioBucket(spec)
	return fmt.Sprintf(`minio-setup:
  image: "minio/mc:latest"
  restart: on-failure
  depends_on:
    - minio
  entrypoint: >
    sh -c "
      sleep 5 &&
      mc alias set local http://minio:9000 %s %s &&
      mc mb --ignore-existing local/%s
    "
`, user, password, bucket)
}

func valkeyServiceYAML(spec projectmeta.ProjectSpec) string {
	return fmt.Sprintf(`valkey:
  image: "valkey/valkey:9"
  restart: unless-stopped
  ports:
    - "0.0.0.0:26379:6379"
  volumes:
    - valkey-data:/data
  command: >
    valkey-server
      --port 6379
      --requirepass %q
      --appendonly yes
      --dir "/data"
`, valkeyPassword(spec))
}

func openbaoServiceYAML(spec projectmeta.ProjectSpec) string {
	return fmt.Sprintf(`openbao:
  image: "openbao/openbao:latest"
  restart: unless-stopped
  ports:
    - "0.0.0.0:28200:8200"
  command: server -dev -dev-root-token-id=%s -dev-listen-address=0.0.0.0:8200
  cap_add:
    - IPC_LOCK
`, openbaoToken(spec))
}

func mailpitServiceYAML(spec projectmeta.ProjectSpec) string {
	return `mailpit:
  image: "axllent/mailpit:latest"
  restart: unless-stopped
  ports:
    - "0.0.0.0:1025:1025"
    - "0.0.0.0:8025:8025"
`
}

func postgresServiceYAML(spec projectmeta.ProjectSpec) string {
	return `postgres:
  image: "postgres:18"
  restart: unless-stopped
  ports:
    - "0.0.0.0:25432:5432"
  environment:
    POSTGRES_DB: "app"
    POSTGRES_USER: "postgres"
    POSTGRES_PASSWORD: "postgres"
`
}

func postgresConfigSnippet(spec projectmeta.ProjectSpec) string {
	return `# PostgreSQL for local dev via ` + "`hatch env add postgres`" + `
[db]
dsn = "postgres://postgres:postgres@localhost:25432/app?search_path=public&sslmode=disable"
migrate = true`
}

func minioConfigSnippet(spec projectmeta.ProjectSpec) string {
	return fmt.Sprintf(`# Object storage for local dev via `+"`hatch env add minio`"+`
[object_storage]
enabled = true
endpoint = "http://localhost:29000"
bucket = %q
bucket_lookup = "path"
access_key = %q
secret_key = %q
presign_expire_seconds = 900`, minioBucket(spec), spec.BinaryName, minioSecret(spec))
}

func valkeyConfigSnippet(spec projectmeta.ProjectSpec) string {
	return fmt.Sprintf(`# Valkey for local dev via `+"`hatch env add valkey`"+`
[valkey]
url = "redis://:%s@localhost:26379/0"`, valkeyPassword(spec))
}

func openbaoConfigSnippet(spec projectmeta.ProjectSpec) string {
	return fmt.Sprintf(`# OpenBao for local dev via `+"`hatch env add openbao`"+`
[openbao]
address = "http://localhost:28200"
token = %q
kv_mount = "secret"
insecure = false
timeout_seconds = 5`, openbaoToken(spec))
}

func mailpitConfigSnippet(spec projectmeta.ProjectSpec) string {
	return `# Mailpit for local dev via ` + "`hatch env add mailpit`" + `
# SMTP: localhost:1025
# Web UI: http://localhost:8025`
}

func minioSecret(spec projectmeta.ProjectSpec) string {
	return spec.BinaryName + "-dev-secret"
}

func minioBucket(spec projectmeta.ProjectSpec) string {
	return spec.BinaryName + "-dev"
}

func valkeyPassword(spec projectmeta.ProjectSpec) string {
	return spec.BinaryName + "-dev"
}

func openbaoToken(spec projectmeta.ProjectSpec) string {
	return spec.BinaryName + "-dev-root"
}
