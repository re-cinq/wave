package commands

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/recinq/wave/internal/state"
)

// NewMigrateCmd creates the migrate command
func NewMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Database migration management",
		Long: `Manage database schema migrations for Wave.

This command provides subcommands to apply, rollback, and inspect database migrations.
Migrations are applied automatically during normal operation, but these commands
allow for manual migration management during development or troubleshooting.`,
	}

	// Add subcommands
	cmd.AddCommand(newMigrateUpCmd())
	cmd.AddCommand(newMigrateDownCmd())
	cmd.AddCommand(newMigrateStatusCmd())
	cmd.AddCommand(newMigrateValidateCmd())

	return cmd
}

// newMigrateUpCmd applies pending migrations
func newMigrateUpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "up [target_version]",
		Short: "Apply pending migrations",
		Long: `Apply all pending migrations up to the target version.

If no target version is specified, all pending migrations will be applied.
Use this command to manually upgrade your database schema.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath := getDbPath()

			migrationRunner, err := state.NewMigrationRunner(dbPath)
			if err != nil {
				return fmt.Errorf("failed to create migration runner: %w", err)
			}
			defer migrationRunner.Close()

			var targetVersion int
			if len(args) > 0 {
				targetVersion, err = strconv.Atoi(args[0])
				if err != nil {
					return fmt.Errorf("invalid target version: %s", args[0])
				}
			}

			err = migrationRunner.MigrateUp(targetVersion)
			if err != nil {
				return fmt.Errorf("migration failed: %w", err)
			}

			fmt.Println("Migrations applied successfully")
			return nil
		},
	}
}

// newMigrateDownCmd rolls back migrations
func newMigrateDownCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "down <target_version>",
		Short: "Rollback migrations to target version",
		Long: `Rollback is not supported during the prototype phase.

All migration Down paths have been removed. If you need to reset
the database, delete the state file and let migrations re-apply.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetVersion, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid target version: %s", args[0])
			}

			dbPath := getDbPath()

			migrationRunner, err := state.NewMigrationRunner(dbPath)
			if err != nil {
				return fmt.Errorf("failed to create migration runner: %w", err)
			}
			defer migrationRunner.Close()

			err = migrationRunner.MigrateDown(targetVersion)
			if err != nil {
				return fmt.Errorf("rollback failed: %w", err)
			}

			fmt.Printf("Successfully rolled back to version %d\n", targetVersion)
			return nil
		},
	}
}

// newMigrateStatusCmd shows migration status
func newMigrateStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show migration status",
		Long:  "Display the current schema version and list all available migrations with their status.",
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath := getDbPath()

			migrationRunner, err := state.NewMigrationRunner(dbPath)
			if err != nil {
				return fmt.Errorf("failed to create migration runner: %w", err)
			}
			defer migrationRunner.Close()

			status, err := migrationRunner.GetStatus()
			if err != nil {
				return fmt.Errorf("failed to get migration status: %w", err)
			}

			fmt.Printf("Current schema version: %d\n\n", status.CurrentVersion)
			fmt.Println("Migration Status:")
			fmt.Println("================")

			for _, migration := range status.AllMigrations {
				status := "[ ]"
				appliedAt := ""

				if migration.AppliedAt != nil {
					status = "[x]"
					appliedAt = fmt.Sprintf(" (applied %s)", migration.AppliedAt.Format("2006-01-02 15:04:05"))
				}

				fmt.Printf("%s %d: %s%s\n", status, migration.Version, migration.Description, appliedAt)
			}

			if len(status.PendingMigrations) > 0 {
				fmt.Printf("\n%d pending migration(s)\n", len(status.PendingMigrations))
			} else {
				fmt.Println("\nDatabase is up to date")
			}

			return nil
		},
	}
}

// newMigrateValidateCmd validates migration integrity
func newMigrateValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate migration integrity",
		Long:  "Verify that applied migrations match their expected checksums and are in a consistent state.",
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath := getDbPath()

			migrationRunner, err := state.NewMigrationRunner(dbPath)
			if err != nil {
				return fmt.Errorf("failed to create migration runner: %w", err)
			}
			defer migrationRunner.Close()

			err = migrationRunner.ValidateIntegrity()
			if err != nil {
				return fmt.Errorf("migration validation failed: %w", err)
			}

			fmt.Println("Migration integrity check passed")
			return nil
		},
	}
}

func getDbPath() string {
	// Default to .wave/state.db relative to current directory
	// In production, this would come from configuration
	return filepath.Join(".wave", "state.db")
}