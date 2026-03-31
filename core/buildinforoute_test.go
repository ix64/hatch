package core

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuildInfoRoute(t *testing.T) {
	t.Parallel()

	prevVersion := Version
	prevCommitHash := CommitHash
	prevBuildTime := BuildTime
	Version = "v1.2.3"
	CommitHash = "abc123"
	BuildTime = "2026-03-31T08:00:00Z"
	t.Cleanup(func() {
		Version = prevVersion
		CommitHash = prevCommitHash
		BuildTime = prevBuildTime
	})

	mux := http.NewServeMux()
	NewBuildInfoRoute().Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	var payload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if payload["version"] != "v1.2.3" {
		t.Fatalf("version = %q", payload["version"])
	}
	if payload["commit_hash"] != "abc123" {
		t.Fatalf("commit_hash = %q", payload["commit_hash"])
	}
	if payload["build_time"] != "2026-03-31T08:00:00Z" {
		t.Fatalf("build_time = %q", payload["build_time"])
	}
	if payload["module"] == "" {
		t.Fatal("module is empty")
	}
}
