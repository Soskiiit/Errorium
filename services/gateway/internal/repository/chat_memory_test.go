package repository_test

import (
	"context"
	"database/sql/driver"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/soskiiit/errorium/services/gateway/internal/repository"
)

func TestChatRepository_MemoryColumn(t *testing.T) {
	database, mock := setupMockDB(t)
	repo := repository.NewChatRepository(database)

	ctx := context.Background()
	now := time.Now().UTC()
	id := uuid.New()
	userID := uuid.New()

	mock.nextRowsCols = []string{"id", "user_id", "title", "is_pinned", "memory", "created_at", "updated_at"}
	mock.nextRowsVals = [][]driver.Value{
		{id.String(), userID.String(), "t", false, "стек: python", now, now},
	}

	c, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if c.Memory != "стек: python" {
		t.Errorf("expected memory scanned, got %q", c.Memory)
	}

	c.Memory = "новый факт"
	if err := repo.Update(ctx, c); err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	q := mock.queries[len(mock.queries)-1]
	if !strings.Contains(q.query, "memory") {
		t.Errorf("update query must write memory column: %s", q.query)
	}
	found := false
	for _, a := range q.args {
		if a.Value == "новый факт" {
			found = true
		}
	}
	if !found {
		t.Errorf("update args missing memory value: %v", q.args)
	}
}