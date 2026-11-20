package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

func TestMigration000004_Statements(t *testing.T) {
	upContent, err := fs.ReadFile(FS, "000004_chat_memory.up.sql")
	if err != nil {
		t.Fatalf("failed to read 000004 up: %v", err)
	}
	upSQL := string(upContent)

	if !strings.Contains(upSQL, "ALTER TABLE chats ADD COLUMN IF NOT EXISTS memory TEXT NOT NULL DEFAULT ''") {
		t.Errorf("000004 up migration missing memory column statement:\n%s", upSQL)
	}

	downContent, err := fs.ReadFile(FS, "000004_chat_memory.down.sql")
	if err != nil {
		t.Fatalf("failed to read 000004 down: %v", err)
	}
	if !strings.Contains(string(downContent), "ALTER TABLE chats DROP COLUMN IF EXISTS memory;") {
		t.Errorf("000004 down migration missing drop statement:\n%s", downContent)
	}
}