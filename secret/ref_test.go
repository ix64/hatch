package secret

import "testing"

func TestParseRef(t *testing.T) {
	t.Run("with explicit key", func(t *testing.T) {
		ref, err := ParseRef("sparkvm/dev/db#dsn")
		if err != nil {
			t.Fatalf("ParseRef returned error: %v", err)
		}
		if ref.Path != "sparkvm/dev/db" || ref.Key != "dsn" {
			t.Fatalf("unexpected ref: %+v", ref)
		}
	})

	t.Run("default key", func(t *testing.T) {
		ref, err := ParseRef("sparkvm/dev/object_storage")
		if err != nil {
			t.Fatalf("ParseRef returned error: %v", err)
		}
		if ref.Path != "sparkvm/dev/object_storage" || ref.Key != "value" {
			t.Fatalf("unexpected ref: %+v", ref)
		}
	})
}
