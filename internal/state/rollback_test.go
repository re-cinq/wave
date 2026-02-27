package state

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// TestCompleteRollbackSequence tests that rollback fails with empty Down paths
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

	// Rollback should fail because Down paths have been removed
	for targetVersion := len(migrations) - 1; targetVersion >= 0; targetVersion-- {
		t.Run(fmt.Sprintf("Rollback to version %d", targetVersion), func(t *testing.T) {
			err := manager.MigrateDown(migrations, targetVersion)
			assert.Error(t, err, "rollback should fail with empty Down paths")
			assert.Contains(t, err.Error(), "no rollback script")
		})
	}
}

// TestRollbackDataIntegrity tests that rollback fails with empty Down paths
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

	// Rollback should fail because Down paths have been removed
	err = manager.MigrateDown(migrations, 2)
	assert.Error(t, err, "rollback should fail with empty Down paths")
	assert.Contains(t, err.Error(), "no rollback script")
}

// TestPartialRollbackAndReapply tests that rollback fails with empty Down paths
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

	// Rollback should fail because Down paths have been removed
	err = manager.MigrateDown(migrations, 2)
	assert.Error(t, err, "rollback should fail with empty Down paths")
	assert.Contains(t, err.Error(), "no rollback script")

	// Version should be unchanged since rollback failed
	currentVersion, err := manager.GetCurrentVersion()
	require.NoError(t, err)
	assert.Equal(t, 6, currentVersion)
}

// TestRollbackWithConstraints tests that rollback fails with empty Down paths
func TestRollbackWithConstraints(t *testing.T) {
	db, cleanup := setupTestMigrationDB(t)
	defer cleanup()

	manager := NewMigrationManager(db)
	err := manager.InitializeMigrationTable()
	require.NoError(t, err)

	migrations := GetAllMigrations()

	// Apply migrations through version 2
	err = manager.MigrateUp(migrations[:2], 0)
	require.NoError(t, err)

	// Rollback should fail because Down paths have been removed
	err = manager.MigrateDown(migrations, 1)
	assert.Error(t, err, "rollback should fail with empty Down paths")
	assert.Contains(t, err.Error(), "no rollback script")
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