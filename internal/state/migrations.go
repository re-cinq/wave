package state

import (
	"database/sql"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// Migration represents a database migration with forward and rollback SQL
type Migration struct {
	Version     int
	Description string
	Up          string
	Down        string
	AppliedAt   *time.Time
}

// MigrationManager handles database schema migrations
type MigrationManager struct {
	db *sql.DB
}

// NewMigrationManager creates a new migration manager
func NewMigrationManager(db *sql.DB) *MigrationManager {
	return &MigrationManager{db: db}
}

// InitializeMigrationTable creates the migration tracking table if it doesn't exist
func (m *MigrationManager) InitializeMigrationTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		description TEXT NOT NULL,
		applied_at INTEGER NOT NULL,
		checksum TEXT NOT NULL
	)`

	_, err := m.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	return nil
}

// GetAppliedMigrations returns a list of all applied migrations
func (m *MigrationManager) GetAppliedMigrations() ([]Migration, error) {
	query := `SELECT version, description, applied_at FROM schema_migrations ORDER BY version`

	rows, err := m.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	var migrations []Migration
	for rows.Next() {
		var migration Migration
		var appliedAt int64

		err := rows.Scan(&migration.Version, &migration.Description, &appliedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan migration: %w", err)
		}

		appliedAtTime := time.Unix(appliedAt, 0)
		migration.AppliedAt = &appliedAtTime
		migrations = append(migrations, migration)
	}

	return migrations, nil
}

// GetCurrentVersion returns the highest applied migration version
func (m *MigrationManager) GetCurrentVersion() (int, error) {
	var version sql.NullInt32
	err := m.db.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("failed to get current version: %w", err)
	}

	if !version.Valid {
		return 0, nil // No migrations applied yet
	}

	return int(version.Int32), nil
}

// ApplyMigration applies a single migration and records it
func (m *MigrationManager) ApplyMigration(migration Migration) error {
	// Start transaction for atomic migration
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start migration transaction: %w", err)
	}
	defer tx.Rollback() // Will be ignored if commit succeeds

	// Execute the migration SQL
	_, err = tx.Exec(migration.Up)
	if err != nil {
		return fmt.Errorf("failed to execute migration %d: %w", migration.Version, err)
	}

	// Record the migration as applied
	checksum := calculateChecksum(migration.Up)
	now := time.Now().Unix()

	_, err = tx.Exec(
		"INSERT INTO schema_migrations (version, description, applied_at, checksum) VALUES (?, ?, ?, ?)",
		migration.Version, migration.Description, now, checksum,
	)
	if err != nil {
		return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit migration %d: %w", migration.Version, err)
	}

	return nil
}

// RollbackMigration rolls back a single migration
func (m *MigrationManager) RollbackMigration(migration Migration) error {
	if migration.Down == "" {
		return fmt.Errorf("migration %d has no rollback script", migration.Version)
	}

	// Start transaction for atomic rollback
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start rollback transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute the rollback SQL
	_, err = tx.Exec(migration.Down)
	if err != nil {
		return fmt.Errorf("failed to execute rollback %d: %w", migration.Version, err)
	}

	// Remove the migration record
	_, err = tx.Exec("DELETE FROM schema_migrations WHERE version = ?", migration.Version)
	if err != nil {
		return fmt.Errorf("failed to remove migration record %d: %w", migration.Version, err)
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit rollback %d: %w", migration.Version, err)
	}

	return nil
}

// ValidateMigrationIntegrity checks if applied migrations match expected checksums
func (m *MigrationManager) ValidateMigrationIntegrity(migrations []Migration) error {
	applied, err := m.GetAppliedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	appliedMap := make(map[int]Migration)
	for _, migration := range applied {
		appliedMap[migration.Version] = migration
	}

	for _, expected := range migrations {
		if _, exists := appliedMap[expected.Version]; exists {
			// Check if checksums match (in real implementation)
			expectedChecksum := calculateChecksum(expected.Up)

			var storedChecksum string
			err := m.db.QueryRow("SELECT checksum FROM schema_migrations WHERE version = ?", expected.Version).Scan(&storedChecksum)
			if err != nil {
				return fmt.Errorf("failed to get checksum for migration %d: %w", expected.Version, err)
			}

			if expectedChecksum != storedChecksum {
				return fmt.Errorf("migration %d checksum mismatch: expected %s, got %s",
					expected.Version, expectedChecksum, storedChecksum)
			}
		}
	}

	return nil
}

// PendingMigrations returns migrations that haven't been applied yet
func (m *MigrationManager) PendingMigrations(allMigrations []Migration) ([]Migration, error) {
	applied, err := m.GetAppliedMigrations()
	if err != nil {
		return nil, err
	}

	appliedVersions := make(map[int]bool)
	for _, migration := range applied {
		appliedVersions[migration.Version] = true
	}

	var pending []Migration
	for _, migration := range allMigrations {
		if !appliedVersions[migration.Version] {
			pending = append(pending, migration)
		}
	}

	// Sort by version to apply in order
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].Version < pending[j].Version
	})

	return pending, nil
}

// MigrateUp applies all pending migrations up to the target version
func (m *MigrationManager) MigrateUp(allMigrations []Migration, targetVersion int) error {
	pending, err := m.PendingMigrations(allMigrations)
	if err != nil {
		return err
	}

	for _, migration := range pending {
		if targetVersion > 0 && migration.Version > targetVersion {
			break
		}

		fmt.Fprintf(os.Stderr, "Applying migration %d: %s\n", migration.Version, migration.Description)
		err := m.ApplyMigration(migration)
		if err != nil {
			return err
		}
	}

	return nil
}

// MigrateDown rolls back migrations down to the target version
func (m *MigrationManager) MigrateDown(allMigrations []Migration, targetVersion int) error {
	applied, err := m.GetAppliedMigrations()
	if err != nil {
		return err
	}

	// Create a map of all migrations for easy lookup
	migrationMap := make(map[int]Migration)
	for _, migration := range allMigrations {
		migrationMap[migration.Version] = migration
	}

	// Sort applied migrations in descending order for rollback
	sort.Slice(applied, func(i, j int) bool {
		return applied[i].Version > applied[j].Version
	})

	for _, appliedMigration := range applied {
		if appliedMigration.Version <= targetVersion {
			break
		}

		// Find the full migration definition with rollback SQL
		fullMigration, exists := migrationMap[appliedMigration.Version]
		if !exists {
			return fmt.Errorf("migration %d not found in migration definitions", appliedMigration.Version)
		}

		if fullMigration.Down == "" {
			return fmt.Errorf("migration %d has no rollback script", appliedMigration.Version)
		}

		fmt.Fprintf(os.Stderr, "Rolling back migration %d: %s\n", appliedMigration.Version, appliedMigration.Description)
		err := m.RollbackMigration(fullMigration)
		if err != nil {
			return err
		}
	}

	return nil
}

// calculateChecksum creates a simple checksum for migration content
// In production, you might want to use SHA256 or similar
func calculateChecksum(content string) string {
	// Simple checksum - normalize whitespace and calculate length + first/last chars
	normalized := strings.TrimSpace(strings.ReplaceAll(content, "\n", " "))
	if len(normalized) == 0 {
		return "empty"
	}

	checksum := fmt.Sprintf("%d-%c-%c", len(normalized), normalized[0], normalized[len(normalized)-1])
	return checksum
}