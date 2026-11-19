package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/soskiiit/errorium/services/gateway/pkg/config"
)

// DB wraps sql.DB to provide a typed pool.
type DB struct {
	*sql.DB
}

// New opens a PostgreSQL connection pool using the provided DBConfig.
// Settings: MaxOpenConns=25, MaxIdleConns=5, ConnMaxLifetime=5min.
func New(cfg config.DBConfig) (*DB, error) {
	sqlDB, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("db: open: %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	return &DB{sqlDB}, nil
}
