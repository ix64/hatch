package projectmeta

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/pelletier/go-toml/v2"
	"golang.org/x/mod/modfile"
)

const (
	MetadataFile      = "hatch.toml"
	LocalMetadataFile = "hatch.local.toml"
	DefaultGo         = "1.26.0"
	DefaultHatchMod   = "github.com/ix64/hatch"
	DefaultHatchVer   = "v0.0.0"
)

type Paths struct {
	MainPackage   string `toml:"main_package"`
	BuildOutput   string `toml:"build_output"`
	ConfigFile    string `toml:"config_file"`
	ConfigExample string `toml:"config_example"`
	AtlasFile     string `toml:"atlas_file"`
	DevCompose    string `toml:"dev_compose"`
	SchemaDir     string `toml:"schema_dir"`
	CompositeDir  string `toml:"composite_dir"`
	MigrationsDir string `toml:"migrations_dir"`
	EntDir        string `toml:"ent_dir"`
}

type Proto struct {
	Enabled          bool   `toml:"enabled"`
	Source           string `toml:"source"`
	Dir              string `toml:"dir"`
	OutDir           string `toml:"out_dir"`
	GitRepo          string `toml:"git_repo"`
	GitBranch        string `toml:"git_branch"`
	LocalOverrideDir string `toml:"local_override_dir"`
}

type Ent struct {
	Features []string `toml:"features"`
}

type Run struct {
	Command []string `toml:"command"`
}

type ProjectSpec struct {
	ModulePath       string `toml:"module_path"`
	AppName          string `toml:"app_name"`
	BinaryName       string `toml:"binary_name"`
	GoVersion        string `toml:"go_version"`
	HatchModulePath  string `toml:"hatch_module_path"`
	HatchVersion     string `toml:"hatch_version"`
	HatchReplacePath string `toml:"hatch_replace_path,omitempty"`
	Paths            Paths  `toml:"paths"`
	Ent              Ent    `toml:"ent"`
	Run              Run    `toml:"run"`
	Proto            Proto  `toml:"proto"`
}

type rawPaths struct {
	MainPackage   string `toml:"main_package"`
	BuildOutput   string `toml:"build_output"`
	ConfigFile    string `toml:"config_file"`
	ConfigExample string `toml:"config_example"`
	AtlasFile     string `toml:"atlas_file"`
	DevCompose    string `toml:"dev_compose"`
	SchemaDir     string `toml:"schema_dir"`
	CompositeDir  string `toml:"composite_dir"`
	MigrationsDir string `toml:"migrations_dir"`
	EntDir        string `toml:"ent_dir"`
	BufGenFile    string `toml:"buf_gen_file"`
	RPCDir        string `toml:"rpc_dir"`
}

type rawProjectSpec struct {
	ModulePath       string   `toml:"module_path"`
	AppName          string   `toml:"app_name"`
	BinaryName       string   `toml:"binary_name"`
	GoVersion        string   `toml:"go_version"`
	ProtoEnabled     *bool    `toml:"proto_enabled"`
	HatchModulePath  string   `toml:"hatch_module_path"`
	HatchVersion     string   `toml:"hatch_version"`
	HatchReplacePath string   `toml:"hatch_replace_path,omitempty"`
	Paths            rawPaths `toml:"paths"`
	Ent              Ent      `toml:"ent"`
	Run              Run      `toml:"run"`
	Proto            Proto    `toml:"proto"`
}

var DefaultEntFeatures = []string{
	"intercept",
	"sql/versioned-migration",
	"sql/modifier",
	"sql/execquery",
	"sql/upsert",
}

func New(modulePath, appName, binaryName string) ProjectSpec {
	spec := ProjectSpec{
		ModulePath:      strings.TrimSpace(modulePath),
		AppName:         strings.TrimSpace(appName),
		BinaryName:      strings.TrimSpace(binaryName),
		GoVersion:       DefaultGo,
		HatchModulePath: DefaultHatchMod,
		HatchVersion:    DefaultHatchVer,
		Ent: Ent{
			Features: append([]string(nil), DefaultEntFeatures...),
		},
		Run: Run{
			Command: []string{"serve"},
		},
		Proto: Proto{
			Enabled:   true,
			Source:    "local",
			Dir:       "proto",
			OutDir:    "rpc",
			GitBranch: "main",
		},
	}
	spec.Normalize()
	return spec
}

func Load(projectDir string) (ProjectSpec, error) {
	projectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return ProjectSpec{}, err
	}

	spec := New("", "", "")
	if raw, found, err := loadRawProjectSpec(projectDir); err != nil {
		return ProjectSpec{}, err
	} else if found {
		spec = raw.toProjectSpec()
	}

	if spec.ModulePath == "" {
		modPath, err := ReadModulePath(projectDir)
		if err != nil {
			return ProjectSpec{}, err
		}
		spec.ModulePath = modPath
	}

	spec.Normalize()
	return spec, nil
}

func loadRawProjectSpec(projectDir string) (rawProjectSpec, bool, error) {
	merged := map[string]any{}
	loaded := false

	for _, filename := range []string{MetadataFile, LocalMetadataFile} {
		path := filepath.Join(projectDir, filename)
		data, err := os.ReadFile(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return rawProjectSpec{}, false, fmt.Errorf("read %s: %w", filename, err)
		}

		var current map[string]any
		if err := toml.Unmarshal(data, &current); err != nil {
			return rawProjectSpec{}, false, fmt.Errorf("parse %s: %w", filename, err)
		}
		mergeMaps(merged, current)
		loaded = true
	}

	if !loaded {
		return rawProjectSpec{}, false, nil
	}

	data, err := toml.Marshal(merged)
	if err != nil {
		return rawProjectSpec{}, false, fmt.Errorf("marshal merged metadata: %w", err)
	}

	var raw rawProjectSpec
	if err := toml.Unmarshal(data, &raw); err != nil {
		return rawProjectSpec{}, false, fmt.Errorf("parse merged metadata: %w", err)
	}
	return raw, true, nil
}

func mergeMaps(dst, src map[string]any) {
	for key, value := range src {
		srcMap, srcIsMap := value.(map[string]any)
		if !srcIsMap {
			dst[key] = value
			continue
		}

		dstMap, dstIsMap := dst[key].(map[string]any)
		if !dstIsMap {
			dst[key] = cloneMap(srcMap)
			continue
		}

		mergeMaps(dstMap, srcMap)
		dst[key] = dstMap
	}
}

func cloneMap(src map[string]any) map[string]any {
	dst := make(map[string]any, len(src))
	for key, value := range src {
		if nested, ok := value.(map[string]any); ok {
			dst[key] = cloneMap(nested)
			continue
		}
		dst[key] = value
	}
	return dst
}

func Save(projectDir string, spec ProjectSpec) error {
	spec.Normalize()
	data, err := toml.Marshal(spec)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", MetadataFile, err)
	}
	metaPath := filepath.Join(projectDir, MetadataFile)
	return os.WriteFile(metaPath, data, 0o644)
}

func ReadModulePath(projectDir string) (string, error) {
	goModPath := filepath.Join(projectDir, "go.mod")
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return "", fmt.Errorf("read go.mod: %w", err)
	}
	file, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return "", fmt.Errorf("parse go.mod: %w", err)
	}
	if file.Module == nil || file.Module.Mod.Path == "" {
		return "", errors.New("go.mod missing module path")
	}
	return file.Module.Mod.Path, nil
}

func (s *ProjectSpec) Normalize() {
	if s.GoVersion == "" {
		s.GoVersion = DefaultGo
	}
	if s.HatchModulePath == "" {
		s.HatchModulePath = DefaultHatchMod
	}
	if s.HatchVersion == "" {
		s.HatchVersion = DefaultHatchVer
	}
	if s.ModulePath != "" && s.BinaryName == "" {
		s.BinaryName = sanitizeBinaryName(filepath.Base(s.ModulePath))
	}
	if s.AppName == "" && s.BinaryName != "" {
		s.AppName = humanizeName(s.BinaryName)
	}
	if s.Paths.MainPackage == "" {
		s.Paths.MainPackage = "./cmd/server"
	}
	if s.Paths.BuildOutput == "" {
		outputName := s.BinaryName
		if outputName == "" {
			outputName = "app"
		}
		s.Paths.BuildOutput = filepath.ToSlash(filepath.Join("build", outputName))
	}
	if s.Paths.ConfigFile == "" {
		s.Paths.ConfigFile = "config.toml"
	}
	if s.Paths.ConfigExample == "" {
		s.Paths.ConfigExample = "config.toml.example"
	}
	if s.Paths.AtlasFile == "" {
		s.Paths.AtlasFile = "atlas.hcl"
	}
	if s.Paths.DevCompose == "" {
		s.Paths.DevCompose = filepath.ToSlash(filepath.Join("dev", "compose.yaml"))
	}
	if s.Paths.SchemaDir == "" {
		s.Paths.SchemaDir = filepath.ToSlash(filepath.Join("ddl", "schema"))
	}
	if s.Paths.CompositeDir == "" {
		s.Paths.CompositeDir = filepath.ToSlash(filepath.Join("ddl", "composite"))
	}
	if s.Paths.MigrationsDir == "" {
		s.Paths.MigrationsDir = filepath.ToSlash(filepath.Join("ddl", "migrations"))
	}
	if s.Paths.EntDir == "" {
		s.Paths.EntDir = filepath.ToSlash(filepath.Join("ddl", "ent"))
	}
	if s.Ent.Features == nil {
		s.Ent.Features = append([]string(nil), DefaultEntFeatures...)
	}
	if s.Run.Command == nil {
		s.Run.Command = []string{"serve"}
	}
	if !s.Proto.Enabled && s.Proto.Source == "" && s.Proto.Dir == "" && s.Proto.OutDir == "" && s.Proto.GitRepo == "" && s.Proto.LocalOverrideDir == "" {
		s.Proto.Enabled = true
	}
	if s.Proto.Source == "" {
		s.Proto.Source = "local"
	}
	if s.Proto.Dir == "" {
		s.Proto.Dir = "proto"
	}
	if s.Proto.OutDir == "" {
		s.Proto.OutDir = "rpc"
	}
	s.Proto.Dir = filepath.ToSlash(s.Proto.Dir)
	s.Proto.OutDir = filepath.ToSlash(s.Proto.OutDir)
	s.Proto.LocalOverrideDir = filepath.ToSlash(s.Proto.LocalOverrideDir)
	if s.Proto.Source == "git" && s.Proto.GitBranch == "" {
		s.Proto.GitBranch = "main"
	}
}

func (s ProjectSpec) ValidateForInit() error {
	if strings.TrimSpace(s.ModulePath) == "" {
		return errors.New("missing required flag: --module")
	}
	if strings.TrimSpace(s.AppName) == "" {
		return errors.New("missing required flag: --name")
	}
	if strings.TrimSpace(s.BinaryName) == "" {
		return errors.New("missing required flag: --binary")
	}
	return nil
}

func (s ProjectSpec) RootPackageImport() string {
	return s.ModulePath
}

func (r rawProjectSpec) toProjectSpec() ProjectSpec {
	spec := ProjectSpec{
		ModulePath:       r.ModulePath,
		AppName:          r.AppName,
		BinaryName:       r.BinaryName,
		GoVersion:        r.GoVersion,
		HatchModulePath:  r.HatchModulePath,
		HatchVersion:     r.HatchVersion,
		HatchReplacePath: r.HatchReplacePath,
		Paths: Paths{
			MainPackage:   r.Paths.MainPackage,
			BuildOutput:   r.Paths.BuildOutput,
			ConfigFile:    r.Paths.ConfigFile,
			ConfigExample: r.Paths.ConfigExample,
			AtlasFile:     r.Paths.AtlasFile,
			DevCompose:    r.Paths.DevCompose,
			SchemaDir:     r.Paths.SchemaDir,
			CompositeDir:  r.Paths.CompositeDir,
			MigrationsDir: r.Paths.MigrationsDir,
			EntDir:        r.Paths.EntDir,
		},
		Ent:   r.Ent,
		Run:   r.Run,
		Proto: r.Proto,
	}

	if r.ProtoEnabled != nil {
		spec.Proto.Enabled = *r.ProtoEnabled
	}
	if spec.Proto.OutDir == "" && r.Paths.RPCDir != "" {
		spec.Proto.OutDir = r.Paths.RPCDir
	}
	return spec
}

func sanitizeBinaryName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "app"
	}
	var b strings.Builder
	for _, r := range value {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
		case r == '-', r == '_':
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "app"
	}
	return b.String()
}

func humanizeName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Hatch App"
	}
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '-' || r == '_' || r == '.'
	})
	for i, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(strings.ToLower(part))
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.Join(parts, " ")
}
