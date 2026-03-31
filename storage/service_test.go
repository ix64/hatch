package storage

import (
	"testing"
	"time"
)

func TestNormalizeRemotePath(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		got, err := NormalizeRemotePath("/demo/test.txt")
		if err != nil {
			t.Fatalf("NormalizeRemotePath() error = %v", err)
		}
		if got != "demo/test.txt" {
			t.Fatalf("NormalizeRemotePath() = %q, want %q", got, "demo/test.txt")
		}
	})

	t.Run("invalid segment", func(t *testing.T) {
		if _, err := NormalizeRemotePath("demo/../test.txt"); err == nil {
			t.Fatal("NormalizeRemotePath() expected error, got nil")
		}
	})
}

func TestResolveExpire(t *testing.T) {
	svc := NewDisabledService(Config{PresignExpireSeconds: 900})

	if got := svc.ResolveExpire(0); got != 15*time.Minute {
		t.Fatalf("ResolveExpire(0) = %v, want %v", got, 15*time.Minute)
	}
	if got := svc.ResolveExpire(60); got != time.Minute {
		t.Fatalf("ResolveExpire(60) = %v, want %v", got, time.Minute)
	}
}
