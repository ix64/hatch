package entcmd

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"github.com/spf13/cobra"

	"github.com/ix64/hatch/internal/cli/projectmeta"
)

func New() *cobra.Command {
	var projectDir string
	var scratch bool
	cmd := &cobra.Command{
		Use:   "ent",
		Short: "Generate Ent code",
		RunE: func(cmd *cobra.Command, args []string) error {
			return Generate(projectDir, scratch)
		},
	}
	cmd.Flags().StringVar(&projectDir, "project-dir", ".", "project directory")
	cmd.Flags().BoolVar(&scratch, "scratch", false, "rebuild Ent code from scratch before generating incrementally")
	return cmd
}

func Generate(projectDir string, scratch bool) error {
	projectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	spec, err := projectmeta.Load(projectDir)
	if err != nil {
		return err
	}

	targetPath := filepath.Join(projectDir, filepath.FromSlash(spec.Paths.EntDir))
	schemaPath := filepath.Join(projectDir, filepath.FromSlash(spec.Paths.SchemaDir))
	genPackage := path.Join(spec.ModulePath, filepath.ToSlash(spec.Paths.EntDir))

	options := entOptions(spec)
	if len(spec.Ent.Features) > 0 {
		fmt.Printf("using Ent features: %s\n", strings.Join(spec.Ent.Features, ", "))
	} else {
		fmt.Println("using Ent features: none")
	}

	fromScratch := scratch
	if dirInfo, err := os.Stat(targetPath); err != nil {
		if os.IsNotExist(err) {
			fromScratch = true
		} else {
			return fmt.Errorf("stat %s: %w", targetPath, err)
		}
	} else if !dirInfo.IsDir() {
		fromScratch = true
	}

	if fromScratch {
		if err := os.RemoveAll(targetPath); err != nil {
			return fmt.Errorf("remove %s: %w", targetPath, err)
		}
		if err := entc.Generate(schemaPath, &gen.Config{
			Target:  targetPath,
			Package: genPackage,
		}, append(options, entc.BuildTags("ent_scratch"))...); err != nil {
			return fmt.Errorf("build ent from scratch: %w", err)
		}
	}

	if err := entc.Generate(schemaPath, &gen.Config{
		Target:  targetPath,
		Package: genPackage,
	}, options...); err != nil {
		return fmt.Errorf("build ent incrementally: %w", err)
	}

	return nil
}

func entOptions(spec projectmeta.ProjectSpec) []entc.Option {
	options := make([]entc.Option, 0, 1)
	if len(spec.Ent.Features) > 0 {
		options = append(options, entc.FeatureNames(spec.Ent.Features...))
	}
	return options
}
