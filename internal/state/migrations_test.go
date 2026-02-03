package state

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupTestMigrationDB(t *testing.T) (*sql.DB, func()) {
	tempDir, err := os.MkdirTemp("", "wave-migration-test-*")
	require.NoError(t, err)

	dbPath := filepath.Join(tempDir, "test.db")
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)

	// Configure SQLite
	_, err = db.Exec("PRAGMA journal_mode=WAL")
	require.NoError(t, err)
	_, err = db.Exec("PRAGMA foreign_keys=ON")
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
		os.RemoveAll(tempDir)
	}

	return db, cleanup
}

func TestMigrationManager_InitializeMigrationTable(t *testing.T) {
	db, cleanup := setupTestMigrationDB(t)
	defer cleanup()

	manager := NewMigrationManager(db)

	err := manager.InitializeMigrationTable()
	assert.NoError(t, err)

	// Verify table exists
	var tableExists bool
	err = db.QueryRow("SELECT 1 FROM sqlite_master WHERE type='table' AND name='schema_migrations'").Scan(&tableExists)
	assert.NoError(t, err)
	assert.True(t, tableExists)
}

func TestMigrationManager_ApplyMigration(t *testing.T) {
	db, cleanup := setupTestMigrationDB(t)
	defer cleanup()

	manager := NewMigrationManager(db)
	err := manager.InitializeMigrationTable()
	require.NoError(t, err)

	migration := Migration{
		Version:     1,
		Description: "Create test table",
		Up:          "CREATE TABLE test_table (id INTEGER PRIMARY KEY, name TEXT)",
		Down:        "DROP TABLE test_table",
	}

	// Apply migration
	err = manager.ApplyMigration(migration)
	assert.NoError(t, err)

	// Verify table was created
	var tableExists bool
	err = db.QueryRow("SELECT 1 FROM sqlite_master WHERE type='table' AND name='test_table'").Scan(&tableExists)
	assert.NoError(t, err)
	assert.True(t, tableExists)

	// Verify migration was recorded
	applied, err := manager.GetAppliedMigrations()
	assert.NoError(t, err)
	assert.Len(t, applied, 1)
	assert.Equal(t, 1, applied[0].Version)
	assert.Equal(t, "Create test table", applied[0].Description)
	assert.NotNil(t, applied[0].AppliedAt)
}

func TestMigrationManager_RollbackMigration(t *testing.T) {
	db, cleanup := setupTestMigrationDB(t)
	defer cleanup()

	manager := NewMigrationManager(db)
	err := manager.InitializeMigrationTable()
	require.NoError(t, err)

	migration := Migration{
		Version:     1,
		Description: "Create test table",
		Up:          "CREATE TABLE test_table (id INTEGER PRIMARY KEY, name TEXT)",
		Down:        "DROP TABLE test_table",
	}

	// Apply migration
	err = manager.ApplyMigration(migration)
	require.NoError(t, err)

	// Rollback migration
	err = manager.RollbackMigration(migration)
	assert.NoError(t, err)

	// Verify table was dropped
	var tableExists bool
	err = db.QueryRow("SELECT 1 FROM sqlite_master WHERE type='table' AND name='test_table'").Scan(&tableExists)
	assert.Error(t, err) // Should error because table doesn't exist

	// Verify migration record was removed
	applied, err := manager.GetAppliedMigrations()
	assert.NoError(t, err)
	assert.Len(t, applied, 0)
}

func TestMigrationManager_MigrateUp(t *testing.T) {
	db, cleanup := setupTestMigrationDB(t)
	defer cleanup()

	manager := NewMigrationManager(db)
	err := manager.InitializeMigrationTable()
	require.NoError(t, err)

	migrations := []Migration{
		{
			Version:     1,
			Description: "Create table1",
			Up:          "CREATE TABLE table1 (id INTEGER PRIMARY KEY)",
			Down:        "DROP TABLE table1",
		},
		{
			Version:     2,
			Description: "Create table2",
			Up:          "CREATE TABLE table2 (id INTEGER PRIMARY KEY)",
			Down:        "DROP TABLE table2",
		},
	}

	// Apply all migrations
	err = manager.MigrateUp(migrations, 0)
	assert.NoError(t, err)

	// Verify both tables exist
	for i := 1; i <= 2; i++ {
		var tableExists bool
		err = db.QueryRow("SELECT 1 FROM sqlite_master WHERE type='table' AND name=?", "table"+string(rune('0'+i))).Scan(&tableExists)
		assert.NoError(t, err)
		assert.True(t, tableExists)
	}

	// Verify migrations were recorded
	applied, err := manager.GetAppliedMigrations()
	assert.NoError(t, err)
	assert.Len(t, applied, 2)
}

func TestMigrationManager_MigrateUpWithTarget(t *testing.T) {
	db, cleanup := setupTestMigrationDB(t)
	defer cleanup()

	manager := NewMigrationManager(db)
	err := manager.InitializeMigrationTable()
	require.NoError(t, err)

	migrations := []Migration{
		{
			Version:     1,
			Description: "Create table1",
			Up:          "CREATE TABLE table1 (id INTEGER PRIMARY KEY)",
			Down:        "DROP TABLE table1",
		},
		{
			Version:     2,
			Description: "Create table2",
			Up:          "CREATE TABLE table2 (id INTEGER PRIMARY KEY)",
			Down:        "DROP TABLE table2",
		},
	}

	// Apply only up to version 1
	err = manager.MigrateUp(migrations, 1)
	assert.NoError(t, err)

	// Verify only table1 exists
	var table1Exists, table2Exists bool
	err = db.QueryRow("SELECT 1 FROM sqlite_master WHERE type='table' AND name='table1'").Scan(&table1Exists)
	assert.NoError(t, err)
	assert.True(t, table1Exists)

	err = db.QueryRow("SELECT 1 FROM sqlite_master WHERE type='table' AND name='table2'").Scan(&table2Exists)
	assert.Error(t, err) // Should error because table2 doesn't exist

	// Verify only one migration was applied
	applied, err := manager.GetAppliedMigrations()
	assert.NoError(t, err)
	assert.Len(t, applied, 1)
}

func TestMigrationManager_MigrateDown(t *testing.T) {
	db, cleanup := setupTestMigrationDB(t)
	defer cleanup()

	manager := NewMigrationManager(db)
	err := manager.InitializeMigrationTable()
	require.NoError(t, err)

	migrations := []Migration{
		{
			Version:     1,
			Description: "Create table1",
			Up:          "CREATE TABLE table1 (id INTEGER PRIMARY KEY)",
			Down:        "DROP TABLE table1",
		},
		{
			Version:     2,
			Description: "Create table2",
			Up:          "CREATE TABLE table2 (id INTEGER PRIMARY KEY)",
			Down:        "DROP TABLE table2",
		},
	}

	// Apply all migrations
	err = manager.MigrateUp(migrations, 0)
	require.NoError(t, err)

	// Rollback to version 1
	err = manager.MigrateDown(migrations, 1)
	assert.NoError(t, err)

	// Verify table1 exists but table2 doesn't
	var table1Exists, table2Exists bool
	err = db.QueryRow("SELECT 1 FROM sqlite_master WHERE type='table' AND name='table1'").Scan(&table1Exists)
	assert.NoError(t, err)
	assert.True(t, table1Exists)

	err = db.QueryRow("SELECT 1 FROM sqlite_master WHERE type='table' AND name='table2'").Scan(&table2Exists)
	assert.Error(t, err) // Should error because table2 doesn't exist

	// Verify only one migration remains
	applied, err := manager.GetAppliedMigrations()
	assert.NoError(t, err)
	assert.Len(t, applied, 1)
	assert.Equal(t, 1, applied[0].Version)
}

func TestMigrationManager_GetCurrentVersion(t *testing.T) {
	db, cleanup := setupTestMigrationDB(t)
	defer cleanup()

	manager := NewMigrationManager(db)
	err := manager.InitializeMigrationTable()
	require.NoError(t, err)

	// Initially version should be 0
	version, err := manager.GetCurrentVersion()
	assert.NoError(t, err)
	assert.Equal(t, 0, version)

	// Apply a migration
	migration := Migration{
		Version:     5,
		Description: "Test migration",
		Up:          "CREATE TABLE test (id INTEGER)",
		Down:        "DROP TABLE test",
	}

	err = manager.ApplyMigration(migration)
	require.NoError(t, err)

	// Version should now be 5
	version, err = manager.GetCurrentVersion()
	assert.NoError(t, err)
	assert.Equal(t, 5, version)
}

func TestMigrationManager_PendingMigrations(t *testing.T) {
	db, cleanup := setupTestMigrationDB(t)
	defer cleanup()

	manager := NewMigrationManager(db)
	err := manager.InitializeMigrationTable()
	require.NoError(t, err)

	allMigrations := []Migration{
		{Version: 1, Description: "Migration 1", Up: "CREATE TABLE t1 (id INTEGER)", Down: "DROP TABLE t1"},
		{Version: 2, Description: "Migration 2", Up: "CREATE TABLE t2 (id INTEGER)", Down: "DROP TABLE t2"},
		{Version: 3, Description: "Migration 3", Up: "CREATE TABLE t3 (id INTEGER)", Down: "DROP TABLE t3"},
	}

	// Apply first migration
	err = manager.ApplyMigration(allMigrations[0])
	require.NoError(t, err)

	// Get pending migrations
	pending, err := manager.PendingMigrations(allMigrations)
	assert.NoError(t, err)
	assert.Len(t, pending, 2)
	assert.Equal(t, 2, pending[0].Version)
	assert.Equal(t, 3, pending[1].Version)
}

func TestMigrationManager_ValidateMigrationIntegrity(t *testing.T) {
	db, cleanup := setupTestMigrationDB(t)
	defer cleanup()

	manager := NewMigrationManager(db)
	err := manager.InitializeMigrationTable()
	require.NoError(t, err)

	migration := Migration{
		Version:     1,
		Description: "Test migration",
		Up:          "CREATE TABLE test (id INTEGER)",
		Down:        "DROP TABLE test",
	}

	// Apply migration
	err = manager.ApplyMigration(migration)
	require.NoError(t, err)

	// Validation should pass with the same migration
	err = manager.ValidateMigrationIntegrity([]Migration{migration})
	assert.NoError(t, err)

	// Validation should fail with a modified migration
	modifiedMigration := Migration{
		Version:     1,
		Description: "Test migration",
		Up:          "CREATE TABLE test (id INTEGER, name TEXT)", // Modified
		Down:        "DROP TABLE test",
	}

	err = manager.ValidateMigrationIntegrity([]Migration{modifiedMigration})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")
}

func TestMigrationConfig_LoadFromEnv(t *testing.T) {
	// Save original env and restore after test
	originalEnv := map[string]string{
		"WAVE_MIGRATION_ENABLED":         os.Getenv("WAVE_MIGRATION_ENABLED"),
		"WAVE_AUTO_MIGRATE":              os.Getenv("WAVE_AUTO_MIGRATE"),
		"WAVE_SKIP_MIGRATION_VALIDATION": os.Getenv("WAVE_SKIP_MIGRATION_VALIDATION"),
		"WAVE_MAX_MIGRATION_VERSION":     os.Getenv("WAVE_MAX_MIGRATION_VERSION"),
	}
	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	tests := []struct {
		name      string
		envVars   map[string]string
		expected  *MigrationConfig
	}{
		{
			name:    "defaults",
			envVars: map[string]string{},
			expected: &MigrationConfig{
				EnableMigrations:        true,
				AutoMigrate:             true,
				SkipMigrationValidation: false,
				MaxMigrationVersion:     0,
			},
		},
		{
			name: "custom values",
			envVars: map[string]string{
				"WAVE_MIGRATION_ENABLED":         "false",
				"WAVE_AUTO_MIGRATE":              "false",
				"WAVE_SKIP_MIGRATION_VALIDATION": "true",
				"WAVE_MAX_MIGRATION_VERSION":     "3",
			},
			expected: &MigrationConfig{
				EnableMigrations:        false,
				AutoMigrate:             false,
				SkipMigrationValidation: true,
				MaxMigrationVersion:     3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			for key := range originalEnv {
				os.Unsetenv(key)
			}

			// Set test environment
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			config := LoadMigrationConfigFromEnv()
			assert.Equal(t, tt.expected, config)
		})
	}
}

func TestInitializeWithMigrations_FreshDatabase(t *testing.T) {
	db, cleanup := setupTestMigrationDB(t)
	defer cleanup()

	// Configure SQLite settings (since we're not using NewStateStore)
	_, err := db.Exec("PRAGMA journal_mode=WAL")
	require.NoError(t, err)
	_, err = db.Exec("PRAGMA foreign_keys=ON")
	require.NoError(t, err)

	config := &MigrationConfig{
		EnableMigrations: true,
		AutoMigrate:      true,
		MaxMigrationVersion: 2, // Limit to first 2 migrations
	}

	err = initializeWithMigrations(db, config)
	assert.NoError(t, err)

	// Check that migration tracking table exists
	var tableExists bool
	err = db.QueryRow("SELECT 1 FROM sqlite_master WHERE type='table' AND name='schema_migrations'").Scan(&tableExists)
	assert.NoError(t, err)
	assert.True(t, tableExists)

	// Check current version (should be 2 due to MaxMigrationVersion)
	manager := NewMigrationManager(db)
	version, err := manager.GetCurrentVersion()
	assert.NoError(t, err)
	assert.Equal(t, 2, version)
}

func TestInitializeWithMigrations_ExistingDatabase(t *testing.T) {
	db, cleanup := setupTestMigrationDB(t)
	defer cleanup()

	// Configure SQLite settings
	_, err := db.Exec("PRAGMA journal_mode=WAL")
	require.NoError(t, err)
	_, err = db.Exec("PRAGMA foreign_keys=ON")
	require.NoError(t, err)

	// Create existing tables (simulate old schema)
	_, err = db.Exec("CREATE TABLE pipeline_state (pipeline_id TEXT PRIMARY KEY)")
	require.NoError(t, err)

	config := &MigrationConfig{
		EnableMigrations: true,
		AutoMigrate:      true,
	}

	err = initializeWithMigrations(db, config)
	assert.NoError(t, err)

	// Check that all migrations are marked as applied
	manager := NewMigrationManager(db)
	applied, err := manager.GetAppliedMigrations()
	assert.NoError(t, err)
	assert.Len(t, applied, 5) // All 5 defined migrations
}

func TestInitializeWithMigrations_NoAutoMigrate(t *testing.T) {
	db, cleanup := setupTestMigrationDB(t)
	defer cleanup()

	config := &MigrationConfig{
		EnableMigrations: true,
		AutoMigrate:      false, // Don't auto-migrate
	}

	err := initializeWithMigrations(db, config)
	assert.NoError(t, err)

	// Check that migration tracking table exists but no migrations were applied
	var tableExists bool
	err = db.QueryRow("SELECT 1 FROM sqlite_master WHERE type='table' AND name='schema_migrations'").Scan(&tableExists)
	assert.NoError(t, err)
	assert.True(t, tableExists)

	manager := NewMigrationManager(db)
	version, err := manager.GetCurrentVersion()
	assert.NoError(t, err)
	assert.Equal(t, 0, version) // No migrations applied
}

// Test the actual migration definitions
func TestMigrationDefinitions(t *testing.T) {
	migrations := GetAllMigrations()

	// Should have 5 migrations based on our definition
	assert.Len(t, migrations, 5)

	// Check version sequence
	expectedVersions := []int{1, 2, 3, 4, 5}
	for i, migration := range migrations {
		assert.Equal(t, expectedVersions[i], migration.Version)
		assert.NotEmpty(t, migration.Description)
		assert.NotEmpty(t, migration.Up)
		assert.NotEmpty(t, migration.Down)
	}

	// Test that each migration can be applied and rolled back
	db, cleanup := setupTestMigrationDB(t)
	defer cleanup()

	manager := NewMigrationManager(db)
	err := manager.InitializeMigrationTable()
	require.NoError(t, err)

	// Apply all migrations
	for _, migration := range migrations {
		err := manager.ApplyMigration(migration)
		assert.NoError(t, err, "Failed to apply migration %d: %s", migration.Version, migration.Description)
	}

	// Rollback all migrations in reverse order
	for i := len(migrations) - 1; i >= 0; i-- {
		err := manager.RollbackMigration(migrations[i])
		assert.NoError(t, err, "Failed to rollback migration %d: %s", migrations[i].Version, migrations[i].Description)
	}
}