package migrations

import (
	"io/fs"
	"testing"
)

func TestEmbeddedMigrations(t *testing.T) {
	entries, err := fs.ReadDir(FS, ".")
	if err != nil {
		t.Fatalf("failed to read embedded migrations dir: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("expected embedded migrations, found 0")
	}

	foundUp := false
	for _, entry := range entries {
		if entry.Name() == "000001_init_gateway_schema.up.sql" {
			foundUp = true
			content, err := fs.ReadFile(FS, entry.Name())
			if err != nil {
				t.Fatalf("failed to read %s: %v", entry.Name(), err)
			}
			if len(content) == 0 {
				t.Fatalf("file %s is empty", entry.Name())
			}
		}
	}

	if !foundUp {
		t.Fatal("expected 000001_init_gateway_schema.up.sql to be embedded")
	}
}
