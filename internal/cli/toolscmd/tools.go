package toolscmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

type installStep struct {
	name    string
	display string
	cmd     []string
	run     func() error
}

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tools",
		Short: "Manage Hatch development tools",
	}
	cmd.AddCommand(newInstallCmd())
	return cmd
}

func newInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install codegen, migration, and lint tools",
		RunE: func(cmd *cobra.Command, args []string) error {
			return install(runtime.GOOS)
		},
	}
}

func install(goos string) error {
	for _, step := range installPlan(goos) {
		if err := runStep(step); err != nil {
			return err
		}
	}
	return nil
}

func installPlan(goos string) []installStep {
	steps := []installStep{
		{name: "ent", cmd: []string{"go", "install", "entgo.io/ent/cmd/ent@latest"}},
		{name: "buf", cmd: []string{"go", "install", "github.com/bufbuild/buf/cmd/buf@latest"}},
		{name: "protoc-gen-go", cmd: []string{"go", "install", "google.golang.org/protobuf/cmd/protoc-gen-go@latest"}},
		{name: "protoc-gen-connect-go", cmd: []string{"go", "install", "connectrpc.com/connect/cmd/protoc-gen-connect-go@latest"}},
		{name: "air", cmd: []string{"go", "install", "github.com/air-verse/air@latest"}},
	}

	switch goos {
	case "windows":
		steps = append(steps,
			installStep{name: "atlas", display: "download atlas.exe to GOPATH/bin", run: installAtlasWindows},
			installStep{name: "golangci-lint", cmd: []string{"go", "install", "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest"}},
		)
	default:
		steps = append(steps,
			installStep{name: "atlas", cmd: []string{"bash", "-lc", "curl -sSf https://atlasgo.sh | sh -s -- --community -y"}},
			installStep{name: "golangci-lint", cmd: []string{"bash", "-lc", "curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(go env GOPATH)/bin"}},
		)
	}

	return steps
}

func runStep(step installStep) error {
	if step.run != nil {
		fmt.Println(strings.Repeat("=", 40))
		fmt.Println(step.display)
		fmt.Println(strings.Repeat("=", 40))
		if err := step.run(); err != nil {
			return fmt.Errorf("install %s: %w", step.name, err)
		}
		return nil
	}

	if len(step.cmd) == 0 {
		return fmt.Errorf("tool %s has no command", step.name)
	}

	cmd := exec.Command(step.cmd[0], step.cmd[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println(strings.Repeat("=", 40))
	fmt.Println(strings.Join(step.cmd, " "))
	fmt.Println(strings.Repeat("=", 40))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("install %s: %w", step.name, err)
	}
	return nil
}

func installAtlasWindows() error {
	const atlasURL = "https://atlasbinaries.com/atlas/atlas-community-windows-amd64-latest.exe"

	gopath, err := goEnv("GOPATH")
	if err != nil {
		return err
	}

	destDir := filepath.Join(gopath, "bin")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", destDir, err)
	}

	destPath := filepath.Join(destDir, "atlas.exe")
	resp, err := http.Get(atlasURL)
	if err != nil {
		return fmt.Errorf("download atlas.exe: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download atlas.exe: unexpected status %s", resp.Status)
	}

	file, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create %s: %w", destPath, err)
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("write %s: %w", destPath, err)
	}
	return nil
}

func goEnv(key string) (string, error) {
	cmd := exec.Command("go", "env", key)
	buf, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("go env %s: %w", key, err)
	}
	return strings.TrimSpace(string(buf)), nil
}
