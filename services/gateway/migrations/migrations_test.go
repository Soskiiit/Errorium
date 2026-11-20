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
		"000003_iteration2.up.sql",
		"000003_iteration2.down.sql",
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

func TestMigration000003_Statements(t *testing.T) {
	upContent, err := fs.ReadFile(FS, "000003_iteration2.up.sql")
	if err != nil {
		t.Fatalf("failed to read 000003 up: %v", err)
	}
	upSQL := string(upContent)

	expectedUpStatements := []string{
		"CREATE TABLE IF NOT EXISTS password_reset_tokens",
		"id UUID PRIMARY KEY DEFAULT uuidv7()",
		"user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE",
		"token_hash TEXT NOT NULL UNIQUE CHECK (length(token_hash) = 64)",
		"expires_at TIMESTAMPTZ NOT NULL",
		"CREATE INDEX IF NOT EXISTS idx_password_reset_user_id ON password_reset_tokens(user_id);",
		"CREATE TABLE IF NOT EXISTS sso_auth_states",
		"state TEXT PRIMARY KEY",
		"provider TEXT NOT NULL CHECK (provider IN ('github', 'google', 'yandex'))",
		"CREATE TABLE IF NOT EXISTS rag_config",
		"id INTEGER PRIMARY KEY CHECK (id = 1)",
		"INSERT INTO rag_config",
		"ON CONFLICT (id) DO NOTHING;",
	}

	for _, stmt := range expectedUpStatements {
		// Normalize whitespace for flexible matching
		normalizedStmt := strings.Join(strings.Fields(stmt), " ")
		normalizedSQL := strings.Join(strings.Fields(upSQL), " ")
		if !strings.Contains(normalizedSQL, normalizedStmt) {
			t.Errorf("000003 up migration missing expected snippet:\n%s", stmt)
		}
	}

	// Ensure mcp_tokens is NOT created (ADR-012)
	if strings.Contains(strings.ToLower(upSQL), "mcp_tokens") {
		t.Errorf("000003 up migration must NOT reference mcp_tokens (ADR-012)")
	}

	downContent, err := fs.ReadFile(FS, "000003_iteration2.down.sql")
	if err != nil {
		t.Fatalf("failed to read 000003 down: %v", err)
	}
	downSQL := string(downContent)

	expectedDownStatements := []string{
		"DROP TABLE IF EXISTS rag_config;",
		"DROP TABLE IF EXISTS sso_auth_states;",
		"DROP TABLE IF EXISTS password_reset_tokens;",
	}

	for _, stmt := range expectedDownStatements {
		if !strings.Contains(downSQL, stmt) {
			t.Errorf("000003 down migration missing expected statement:\n%s", stmt)
		}
	}
}
