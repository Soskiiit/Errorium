package migrations

import (
	"io/fs"
	"strings"
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

	expectedFiles := []string{
		"000001_init_gateway_schema.up.sql",
		"000001_init_gateway_schema.down.sql",
		"000002_profile_and_message_content.up.sql",
		"000002_profile_and_message_content.down.sql",
	}

	for _, fileName := range expectedFiles {
		content, err := fs.ReadFile(FS, fileName)
		if err != nil {
			t.Fatalf("expected embedded migration %s: %v", fileName, err)
		}
		if len(content) == 0 {
			t.Fatalf("migration %s is empty", fileName)
		}
	}
}

func TestMigration000002_Statements(t *testing.T) {
	upContent, err := fs.ReadFile(FS, "000002_profile_and_message_content.up.sql")
	if err != nil {
		t.Fatalf("failed to read 000002 up: %v", err)
	}
	upSQL := string(upContent)

	expectedUpStatements := []string{
		"ALTER TABLE users ADD COLUMN profession TEXT NOT NULL DEFAULT 'Backend developer' CHECK (length(profession) <= 100);",
		"ALTER TABLE users ADD COLUMN gender TEXT NOT NULL DEFAULT 'Not specified' CHECK (gender IN ('Male', 'Female', 'Not specified'));",
		"ALTER TABLE users ADD COLUMN language TEXT NOT NULL DEFAULT 'English' CHECK (language IN ('English', 'Русский'));",
		"ALTER TABLE chat_messages ADD COLUMN content TEXT;",
		"ALTER TABLE chat_messages ADD COLUMN code TEXT;",
	}

	for _, stmt := range expectedUpStatements {
		if !strings.Contains(upSQL, stmt) {
			t.Errorf("000002 up migration missing expected statement:\n%s", stmt)
		}
	}

	downContent, err := fs.ReadFile(FS, "000002_profile_and_message_content.down.sql")
	if err != nil {
		t.Fatalf("failed to read 000002 down: %v", err)
	}
	downSQL := string(downContent)

	expectedDownStatements := []string{
		"ALTER TABLE chat_messages DROP COLUMN IF EXISTS code;",
		"ALTER TABLE chat_messages DROP COLUMN IF EXISTS content;",
		"ALTER TABLE users DROP COLUMN IF EXISTS language;",
		"ALTER TABLE users DROP COLUMN IF EXISTS gender;",
		"ALTER TABLE users DROP COLUMN IF EXISTS profession;",
	}

	for _, stmt := range expectedDownStatements {
		if !strings.Contains(downSQL, stmt) {
			t.Errorf("000002 down migration missing expected statement:\n%s", stmt)
		}
	}
}
