package state

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// MigrationStatus contains the current state of migrations
type MigrationStatus struct {
	CurrentVersion    int
	AllMigrations     []Migration
	PendingMigrations []Migration
}

// MigrationRunner provides CLI-friendly migration operations
type MigrationRunner struct {
	db      *sql.DB
	manager *MigrationManager
}

// NewMigrationRunner creates a new migration runner with database connection
func NewMigrationRunner(dbPath string) (*MigrationRunner, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool for SQLite
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Configure SQLite for concurrent access
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}

	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	manager := NewMigrationManager(db)

	// Initialize migration table if it doesn't exist
	if err := manager.InitializeMigrationTable(); err != nil {
		return nil, fmt.Errorf("failed to initialize migration table: %w", err)
	}

	return &MigrationRunner{
		db:      db,
		manager: manager,
	}, nil
}

// Close closes the database connection
func (r *MigrationRunner) Close() error {
	return r.db.Close()
}

// MigrateUp applies all pending migrations up to the target version
func (r *MigrationRunner) MigrateUp(targetVersion int) error {
	allMigrations := GetAllMigrations()
	return r.manager.MigrateUp(allMigrations, targetVersion)
}

// MigrateDown rolls back migrations down to the target version
func (r *MigrationRunner) MigrateDown(targetVersion int) error {
	allMigrations := GetAllMigrations()
	return r.manager.MigrateDown(allMigrations, targetVersion)
}

// GetStatus returns the current migration status
func (r *MigrationRunner) GetStatus() (*MigrationStatus, error) {
	currentVersion, err := r.manager.GetCurrentVersion()
	if err != nil {
		return nil, err
	}

	allMigrations := GetAllMigrations()

	// Get applied migrations to merge with all migrations
	applied, err := r.manager.GetAppliedMigrations()
	if err != nil {
		return nil, err
	}

	appliedMap := make(map[int]*time.Time)
	for _, migration := range applied {
		appliedMap[migration.Version] = migration.AppliedAt
	}

	// Set applied time for all migrations
	for i := range allMigrations {
		if appliedAt, exists := appliedMap[allMigrations[i].Version]; exists {
			allMigrations[i].AppliedAt = appliedAt
		}
	}

	pending, err := r.manager.PendingMigrations(allMigrations)
	if err != nil {
		return nil, err
	}

	return &MigrationStatus{
		CurrentVersion:    currentVersion,
		AllMigrations:     allMigrations,
		PendingMigrations: pending,
	}, nil
}

// ValidateIntegrity validates migration integrity
func (r *MigrationRunner) ValidateIntegrity() error {
	allMigrations := GetAllMigrations()
	return r.manager.ValidateMigrationIntegrity(allMigrations)
}

// ForceMarkApplied marks a migration as applied without running it
// This is useful for fixing migration state in exceptional circumstances
func (r *MigrationRunner) ForceMarkApplied(version int, description string) error {
	allMigrations := GetAllMigrations()

	var targetMigration *Migration
	for _, migration := range allMigrations {
		if migration.Version == version {
			targetMigration = &migration
			break
		}
	}

	if targetMigration == nil {
		return fmt.Errorf("migration version %d not found", version)
	}

	checksum := calculateChecksum(targetMigration.Up)
	now := time.Now().Unix()

	_, err := r.db.Exec(
		"INSERT INTO schema_migrations (version, description, applied_at, checksum) VALUES (?, ?, ?, ?)",
		version, description, now, checksum,
	)
	if err != nil {
		return fmt.Errorf("failed to mark migration %d as applied: %w", version, err)
	}

	return nil
}