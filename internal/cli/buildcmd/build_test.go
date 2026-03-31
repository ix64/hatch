package buildcmd

import "testing"

func TestLDFlags(t *testing.T) {
	flags := ldflags("v1.2.3", "abc123", "2026-03-31T00:00:00Z")
	want := []string{
		"-s",
		"-w",
		"-X", "github.com/ix64/hatch/core.Version=v1.2.3",
		"-X", "github.com/ix64/hatch/core.CommitHash=abc123",
		"-X", "github.com/ix64/hatch/core.BuildTime=2026-03-31T00:00:00Z",
	}
	if len(flags) != len(want) {
		t.Fatalf("unexpected flag count: %d", len(flags))
	}
	for i := range want {
		if flags[i] != want[i] {
			t.Fatalf("flags[%d] = %q, want %q", i, flags[i], want[i])
		}
	}
}
