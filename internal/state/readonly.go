package state

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// NewReadOnlyStateStore opens the state database in read-only mode for concurrent
// dashboard queries. It uses ?mode=ro, PRAGMA query_only=ON, higher MaxOpenConns
// for concurrent HTTP handlers, and WAL mode for concurrent reads alongside a writer.
// Migrations are skipped since this is a read-only connection.
func NewReadOnlyStateStore(dbPath string) (StateStore, error) {
	// Open in read-only mode via URI parameter
	dsn := dbPath + "?mode=ro"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open read-only database: %w", err)
	}

	// Higher connection limit for concurrent HTTP handler queries
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping read-only database: %w", err)
	}

	// Defense-in-depth: ensure this connection cannot write
	if _, err := db.Exec("PRAGMA query_only=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set query_only pragma: %w", err)
	}

	// WAL mode required for concurrent reads alongside a writer
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}

	// Set busy timeout for read contention
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	// No schema initialization or migrations - read-only connection
	return &stateStore{db: db}, nil
}
