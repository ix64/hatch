package projectmeta

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJSONSchemaMatchesSnapshot(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	got, err := JSONSchema()
	if err != nil {
		t.Fatal(err)
	}

	want, err := os.ReadFile(filepath.Join(repoRoot, "hatch.schema.json"))
	if err != nil {
		t.Fatal(err)
	}

	if string(got)+"\n" != string(want) {
		t.Fatalf("hatch.schema.json is out of date\nwant:\n%s\ngot:\n%s", want, got)
	}
}
