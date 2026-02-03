package state

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// TestCompleteRollbackSequence tests that rollback scripts work correctly
// by applying all migrations and then rolling back one by one
func TestCompleteRollbackSequence(t *testing.T) {
	db, cleanup := setupTestMigrationDB(t)
	defer cleanup()

	manager := NewMigrationManager(db)
	err := manager.InitializeMigrationTable()
	require.NoError(t, err)

	migrations := GetAllMigrations()

	// Apply all migrations
	err = manager.MigrateUp(migrations, 0)
	require.NoError(t, err)

	// Verify all migrations are applied
	applied, err := manager.GetAppliedMigrations()
	require.NoError(t, err)
	assert.Len(t, applied, len(migrations))

	// Verify all expected tables exist
	expectedTables := []string{
		"pipeline_state",
		"step_state",
		"pipeline_run",
		"event_log",
		"artifact",
		"cancellation",
		"performance_metric",
		"progress_snapshot",
		"step_progress",
		"pipeline_progress",
		"artifact_metadata",
	}

	for _, tableName := range expectedTables {
		exists := checkTableExists(t, db, tableName)
		assert.True(t, exists, "Table %s should exist after migrations", tableName)
	}

	// Test rollback sequence - roll back one migration at a time
	for targetVersion := len(migrations) - 1; targetVersion >= 0; targetVersion-- {
		t.Run(fmt.Sprintf("Rollback to version %d", targetVersion), func(t *testing.T) {
			err := manager.MigrateDown(migrations, targetVersion)
			assert.NoError(t, err)

			// Check current version
			currentVersion, err := manager.GetCurrentVersion()
			assert.NoError(t, err)
			assert.Equal(t, targetVersion, currentVersion)

			// Verify the correct number of migrations remain
			applied, err := manager.GetAppliedMigrations()
			assert.NoError(t, err)
			assert.Len(t, applied, targetVersion)
		})
	}

	// After complete rollback, only migration tracking table should remain
	currentVersion, err := manager.GetCurrentVersion()
	assert.NoError(t, err)
	assert.Equal(t, 0, currentVersion)

	// Verify that only the schema_migrations table remains
	// Note: SQLite may also create sqlite_sequence for AUTOINCREMENT columns
	tables, err := getAllTables(db)
	assert.NoError(t, err)

	// Check that schema_migrations exists
	assert.Contains(t, tables, "schema_migrations")

	// Check that application tables don't exist
	applicationTables := []string{
		"pipeline_state", "step_state", "pipeline_run", "event_log", "artifact",
		"cancellation", "performance_metric", "progress_snapshot",
		"step_progress", "pipeline_progress", "artifact_metadata",
	}

	for _, appTable := range applicationTables {
		assert.NotContains(t, tables, appTable, "Application table %s should not exist after complete rollback", appTable)
	}
}

// TestRollbackDataIntegrity tests that rollbacks don't lose data integrity
func TestRollbackDataIntegrity(t *testing.T) {
	db, cleanup := setupTestMigrationDB(t)
	defer cleanup()

	manager := NewMigrationManager(db)
	err := manager.InitializeMigrationTable()
	require.NoError(t, err)

	migrations := GetAllMigrations()

	// Apply first few migrations
	err = manager.MigrateUp(migrations[:3], 0)
	require.NoError(t, err)

	// Insert some test data
	testData := map[string]string{
		"INSERT INTO pipeline_state (pipeline_id, pipeline_name, status, input, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)":
			"test-pipeline|test|running|test input|1234567890|1234567890",
		"INSERT INTO pipeline_run (run_id, pipeline_name, status, started_at) VALUES (?, ?, ?, ?)":
			"test-run|test-pipeline|running|1234567890",
		"INSERT INTO performance_metric (run_id, step_id, pipeline_name, started_at, success) VALUES (?, ?, ?, ?, ?)":
			"test-run|test-step|test-pipeline|1234567890|1",
	}

	for query, params := range testData {
		paramSlice := splitParams(params)
		_, err := db.Exec(query, paramSlice...)
		require.NoError(t, err)
	}

	// Verify data exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM pipeline_state").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	err = db.QueryRow("SELECT COUNT(*) FROM pipeline_run").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Rollback to version 2 (should preserve pipeline_state and pipeline_run data)
	err = manager.MigrateDown(migrations, 2)
	require.NoError(t, err)

	// Verify that data in remaining tables is preserved
	err = db.QueryRow("SELECT COUNT(*) FROM pipeline_state").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	err = db.QueryRow("SELECT COUNT(*) FROM pipeline_run").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// Verify that performance_metric table was dropped (migration 3)
	exists := checkTableExists(t, db, "performance_metric")
	assert.False(t, exists, "performance_metric table should not exist after rollback")
}

// TestPartialRollbackAndReapply tests rolling back and then reapplying migrations
func TestPartialRollbackAndReapply(t *testing.T) {
	db, cleanup := setupTestMigrationDB(t)
	defer cleanup()

	manager := NewMigrationManager(db)
	err := manager.InitializeMigrationTable()
	require.NoError(t, err)

	migrations := GetAllMigrations()

	// Apply all migrations
	err = manager.MigrateUp(migrations, 0)
	require.NoError(t, err)

	// Rollback to version 2
	err = manager.MigrateDown(migrations, 2)
	require.NoError(t, err)

	currentVersion, err := manager.GetCurrentVersion()
	require.NoError(t, err)
	assert.Equal(t, 2, currentVersion)

	// Reapply migrations
	err = manager.MigrateUp(migrations, 0)
	require.NoError(t, err)

	finalVersion, err := manager.GetCurrentVersion()
	require.NoError(t, err)
	assert.Equal(t, 5, finalVersion)

	// Verify all tables exist again
	expectedTables := []string{
		"pipeline_state", "step_state", "pipeline_run", "event_log", "artifact",
		"cancellation", "performance_metric", "progress_snapshot",
		"step_progress", "pipeline_progress", "artifact_metadata",
	}

	for _, tableName := range expectedTables {
		exists := checkTableExists(t, db, tableName)
		assert.True(t, exists, "Table %s should exist after reapply", tableName)
	}
}

// TestRollbackWithConstraints tests that foreign key constraints are handled correctly during rollback
func TestRollbackWithConstraints(t *testing.T) {
	db, cleanup := setupTestMigrationDB(t)
	defer cleanup()

	manager := NewMigrationManager(db)
	err := manager.InitializeMigrationTable()
	require.NoError(t, err)

	migrations := GetAllMigrations()

	// Apply migrations through version 2 (includes pipeline_run and artifact tables)
	err = manager.MigrateUp(migrations[:2], 0)
	require.NoError(t, err)

	// Insert data with foreign key relationships
	_, err = db.Exec("INSERT INTO pipeline_run (run_id, pipeline_name, status, started_at) VALUES (?, ?, ?, ?)",
		"test-run", "test-pipeline", "running", 1234567890)
	require.NoError(t, err)

	_, err = db.Exec("INSERT INTO artifact (run_id, step_id, name, path, created_at) VALUES (?, ?, ?, ?, ?)",
		"test-run", "test-step", "test.txt", "/path/test.txt", 1234567890)
	require.NoError(t, err)

	// Rollback to version 1 (should handle foreign key constraints properly)
	err = manager.MigrateDown(migrations, 1)
	require.NoError(t, err)

	// Verify that tables were properly dropped despite having data
	exists := checkTableExists(t, db, "pipeline_run")
	assert.False(t, exists, "pipeline_run table should not exist after rollback")

	exists = checkTableExists(t, db, "artifact")
	assert.False(t, exists, "artifact table should not exist after rollback")

	// Verify pipeline_state still exists
	exists = checkTableExists(t, db, "pipeline_state")
	assert.True(t, exists, "pipeline_state table should exist after rollback to version 1")
}

// Helper functions

func checkTableExists(t *testing.T, db *sql.DB, tableName string) bool {
	var exists bool
	err := db.QueryRow("SELECT 1 FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&exists)
	if err == sql.ErrNoRows {
		return false
	}
	require.NoError(t, err)
	return exists
}

func getAllTables(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		err := rows.Scan(&name)
		if err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}

	return tables, nil
}

func splitParams(paramString string) []interface{} {
	if paramString == "" {
		return nil
	}

	parts := splitString(paramString, '|')
	result := make([]interface{}, len(parts))
	for i, part := range parts {
		result[i] = part
	}
	return result
}

func splitString(s string, delimiter rune) []string {
	var result []string
	var current string

	for _, r := range s {
		if r == delimiter {
			result = append(result, current)
			current = ""
		} else {
			current += string(r)
		}
	}

	if current != "" {
		result = append(result, current)
	}

	return result
}