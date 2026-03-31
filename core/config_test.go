package core

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

type testConfig struct {
	Server struct {
		Addr string
	} `mapstructure:"server"`
	Logger struct {
		Level string
	} `mapstructure:"logger"`
}

func TestNewViperWithoutConfigFile(t *testing.T) {
	t.Parallel()

	vp, err := NewViper("", ConfigOptions{
		DefaultName: "config",
		DefaultType: "toml",
		SearchPaths: []string{t.TempDir()},
	})
	if err != nil {
		t.Fatalf("NewViper() error = %v", err)
	}
	if got := vp.ConfigFileUsed(); got != "" {
		t.Fatalf("ConfigFileUsed() = %q", got)
	}
}

func TestNewViperWithExplicitConfigFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "custom.toml")
	if err := os.WriteFile(configPath, []byte("[server]\naddr = \":9999\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	vp, err := NewViper(configPath, ConfigOptions{})
	if err != nil {
		t.Fatalf("NewViper() error = %v", err)
	}
	if got := vp.ConfigFileUsed(); got != configPath {
		t.Fatalf("ConfigFileUsed() = %q, want %q", got, configPath)
	}
}

func TestLoadConfigAppliesDefaultsAndOverrides(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(configPath, []byte("[server]\naddr = \":7777\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	vp, err := NewViper(configPath, ConfigOptions{})
	if err != nil {
		t.Fatalf("NewViper() error = %v", err)
	}

	cfg, err := LoadConfig[testConfig](vp, LoadConfigOptions[testConfig]{
		Defaults: map[string]any{
			"server.addr":  ":9580",
			"logger.level": "INFO",
		},
	})
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.Server.Addr != ":7777" {
		t.Fatalf("server.addr = %q", cfg.Server.Addr)
	}
	if cfg.Logger.Level != "INFO" {
		t.Fatalf("logger.level = %q", cfg.Logger.Level)
	}
}

func TestLoadConfigEnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(configPath, []byte("[server]\naddr = \":7777\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SERVER_ADDR", ":8888")

	vp, err := NewViper(configPath, ConfigOptions{})
	if err != nil {
		t.Fatalf("NewViper() error = %v", err)
	}

	cfg, err := LoadConfig[testConfig](vp, LoadConfigOptions[testConfig]{})
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.Server.Addr != ":8888" {
		t.Fatalf("server.addr = %q", cfg.Server.Addr)
	}
}

func TestLoadConfigValidate(t *testing.T) {
	t.Parallel()

	vp := viper.New()
	_, err := LoadConfig[testConfig](vp, LoadConfigOptions[testConfig]{
		Validate: func(cfg *testConfig) error {
			if cfg.Server.Addr == "" {
				return errors.New("server.addr is required")
			}
			return nil
		},
	})
	if err == nil {
		t.Fatal("LoadConfig() unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), "server.addr is required") {
		t.Fatalf("LoadConfig() error = %v", err)
	}
}
