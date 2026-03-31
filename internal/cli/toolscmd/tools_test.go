package toolscmd

import "testing"

func TestInstallPlanLinux(t *testing.T) {
	t.Parallel()

	plan := installPlan("linux")
	if len(plan) == 0 {
		t.Fatal("installPlan() returned no steps")
	}
	assertHasTool(t, plan, "ent")
	assertHasTool(t, plan, "atlas")
	assertHasTool(t, plan, "buf")
	assertHasTool(t, plan, "protoc-gen-go")
	assertHasTool(t, plan, "protoc-gen-connect-go")
	assertHasTool(t, plan, "golangci-lint")
	assertHasTool(t, plan, "air")
}

func TestInstallPlanWindows(t *testing.T) {
	t.Parallel()

	plan := installPlan("windows")
	assertHasTool(t, plan, "atlas")
	assertHasTool(t, plan, "golangci-lint")
	for _, step := range plan {
		if step.name != "atlas" {
			continue
		}
		if step.run == nil {
			t.Fatalf("windows atlas installer should use internal downloader, got %v", step.cmd)
		}
		if step.display == "" {
			t.Fatal("windows atlas installer should describe the internal downloader")
		}
	}
}

func assertHasTool(t *testing.T, plan []installStep, name string) {
	t.Helper()
	for _, step := range plan {
		if step.name == name {
			return
		}
	}
	t.Fatalf("missing tool %q in install plan", name)
}
