package entcmd

import (
	"testing"

	"github.com/ix64/hatch/internal/cli/projectmeta"
)

func TestEntOptions(t *testing.T) {
	spec := projectmeta.New("example.com/acme/demo", "Demo Service", "demo")
	if got := len(entOptions(spec)); got != 1 {
		t.Fatalf("expected one ent option for default features, got %d", got)
	}

	spec.Ent.Features = []string{}
	if got := len(entOptions(spec)); got != 0 {
		t.Fatalf("expected no ent options for explicit empty features, got %d", got)
	}
}

func TestNewEntCmdSupportsScratchFlag(t *testing.T) {
	cmd := New()

	flag := cmd.Flags().Lookup("scratch")
	if flag == nil {
		t.Fatal("expected scratch flag to exist")
	}
	if flag.DefValue != "false" {
		t.Fatalf("unexpected scratch default: %s", flag.DefValue)
	}
}

func TestEntCommandOnlyRegistersGenerate(t *testing.T) {
	cmd := New()

	if cmd.Name() != "ent" {
		t.Fatalf("unexpected command name: %s", cmd.Name())
	}
	if len(cmd.Commands()) != 0 {
		t.Fatalf("expected no nested subcommands, got %d", len(cmd.Commands()))
	}
}
