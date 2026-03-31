package migratecmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ix64/hatch/internal/cli/cmdutil"
	"github.com/ix64/hatch/internal/cli/projectmeta"
)

const defaultSchemaDumpName = "50_schema.dump.sql"

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Manage Atlas migrations",
	}
	cmd.AddCommand(newGenerateCmd())
	cmd.AddCommand(newHashCmd())
	cmd.AddCommand(newLintCmd())
	cmd.AddCommand(newApplyCmd())
	return cmd
}

func newGenerateCmd() *cobra.Command {
	var projectDir string
	var name string
	var envName string
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate a new Atlas migration from Ent schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return errors.New("missing required flag: --name")
			}
			return Generate(projectDir, envName, name)
		},
	}
	cmd.Flags().StringVar(&projectDir, "project-dir", ".", "project directory")
	cmd.Flags().StringVar(&envName, "env", "dev", "atlas environment name")
	cmd.Flags().StringVar(&name, "name", "", "migration name")
	return cmd
}

func newHashCmd() *cobra.Command {
	var projectDir string
	var envName string
	cmd := &cobra.Command{
		Use:   "hash",
		Short: "Recompute Atlas migration hash",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAtlas(projectDir, "migrate", "hash", "--env", envName)
		},
	}
	cmd.Flags().StringVar(&projectDir, "project-dir", ".", "project directory")
	cmd.Flags().StringVar(&envName, "env", "dev", "atlas environment name")
	return cmd
}

func newLintCmd() *cobra.Command {
	var projectDir string
	var envName string
	var latest int
	cmd := &cobra.Command{
		Use:   "lint",
		Short: "Lint recent Atlas migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAtlas(projectDir, "migrate", "lint", "--env", envName, "--latest", strconv.Itoa(latest))
		},
	}
	cmd.Flags().StringVar(&projectDir, "project-dir", ".", "project directory")
	cmd.Flags().StringVar(&envName, "env", "dev", "atlas environment name")
	cmd.Flags().IntVar(&latest, "latest", 1, "number of latest migrations to lint")
	return cmd
}

func newApplyCmd() *cobra.Command {
	var projectDir string
	var envName string
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply Atlas migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAtlas(projectDir, "migrate", "apply", "--env", envName)
		},
	}
	cmd.Flags().StringVar(&projectDir, "project-dir", ".", "project directory")
	cmd.Flags().StringVar(&envName, "env", "dev", "atlas environment name")
	return cmd
}

func Generate(projectDir, envName, name string) error {
	projectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	spec, err := projectmeta.Load(projectDir)
	if err != nil {
		return err
	}

	before, err := listMigrationFiles(projectDir, spec.Paths.MigrationsDir)
	if err != nil {
		return err
	}

	if err := generateSchemaDump(projectDir, spec); err != nil {
		return err
	}

	if err := runAtlas(projectDir, "migrate", "diff", name, "--env", envName, "--to", "file://"+spec.Paths.CompositeDir); err != nil {
		return err
	}

	after, err := listMigrationFiles(projectDir, spec.Paths.MigrationsDir)
	if err != nil {
		return err
	}

	newFiles := diffMigrationFiles(before, after)
	if len(newFiles) == 0 {
		fmt.Println("The migration directory is synced with the desired state, skip formatting and hashing.")
		return nil
	}
	if len(newFiles) > 1 {
		return fmt.Errorf("expected exactly one new migration file, got %d: %s", len(newFiles), strings.Join(newFiles, ", "))
	}

	migrationFile := newFiles[0]
	fmt.Printf("Generated migration: %s\n", migrationFile)
	if err := formatMigrationFile(projectDir, migrationFile); err != nil {
		return err
	}

	return runAtlas(projectDir, "migrate", "hash", "--env", envName)
}

func generateSchemaDump(projectDir string, spec projectmeta.ProjectSpec) error {
	dumpPath := filepath.Join(projectDir, filepath.FromSlash(spec.Paths.CompositeDir), defaultSchemaDumpName)
	file, err := os.Create(dumpPath)
	if err != nil {
		return fmt.Errorf("create schema dump: %w", err)
	}
	defer file.Close()

	cmd := exec.Command("ent", "schema", "./"+spec.Paths.SchemaDir, "--dialect", "postgres", "--version", "18")
	cmd.Dir = projectDir
	cmd.Stdout = file
	cmd.Stderr = os.Stderr

	fmt.Println(strings.Repeat("=", 40))
	fmt.Printf("%s > %s\n", cmdutil.DisplayShellCommand(cmd), filepath.ToSlash(filepath.Join(spec.Paths.CompositeDir, defaultSchemaDumpName)))
	fmt.Println(strings.Repeat("=", 40))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("generate schema dump: %w", err)
	}
	return nil
}

func listMigrationFiles(projectDir, migrationsDir string) ([]string, error) {
	dir := filepath.Join(projectDir, filepath.FromSlash(migrationsDir))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		files = append(files, filepath.ToSlash(filepath.Join(migrationsDir, entry.Name())))
	}
	slices.Sort(files)
	return files, nil
}

func diffMigrationFiles(before, after []string) []string {
	seen := make(map[string]struct{}, len(before))
	for _, file := range before {
		seen[file] = struct{}{}
	}
	newFiles := make([]string, 0, len(after))
	for _, file := range after {
		if _, ok := seen[file]; ok {
			continue
		}
		newFiles = append(newFiles, file)
	}
	return newFiles
}

func formatMigrationFile(projectDir, migrationFile string) error {
	args := []string{
		"run",
		"--rm",
		"--mount", fmt.Sprintf("type=bind,source=%s,target=/work", projectDir),
	}

	if userSpec, ok := dockerUserSpec(); ok {
		args = append(args, "--user", userSpec)
	}

	containerPath := filepath.ToSlash(filepath.Join("/work", migrationFile))
	args = append(args,
		"--entrypoint", "sh",
		"backplane/pgformatter:latest",
		"-c", fmt.Sprintf("pg_format -i %s", strconv.Quote(containerPath)),
	)
	return runCommand(projectDir, "docker", args...)
}

func dockerUserSpec() (string, bool) {
	if runtime.GOOS == "windows" {
		return "", false
	}

	current, err := user.Current()
	if err != nil || current.Uid == "" || current.Gid == "" {
		return "", false
	}

	return current.Uid + ":" + current.Gid, true
}

func runAtlas(projectDir string, args ...string) error {
	return runCommand(projectDir, "atlas", args...)
}

func runCommand(projectDir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println(strings.Repeat("=", 40))
	fmt.Println(cmdutil.DisplayShellCommand(cmd))
	fmt.Println(strings.Repeat("=", 40))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run %s: %w", name, err)
	}
	return nil
}

