package state

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// MigrationConfig controls migration behavior
type MigrationConfig struct {
	// EnableMigrations controls whether the new migration system is used
	EnableMigrations bool

	// AutoMigrate controls whether migrations are applied automatically on startup
	AutoMigrate bool

	// SkipMigrationValidation skips checksum validation for development
	SkipMigrationValidation bool

	// MaxMigrationVersion limits which migrations can be applied (0 = all)
	MaxMigrationVersion int
}

// LoadMigrationConfigFromEnv loads migration configuration from environment variables
func LoadMigrationConfigFromEnv() *MigrationConfig {
	config := &MigrationConfig{
		EnableMigrations:        true, // Default to enabled for new systems
		AutoMigrate:            true, // Default to automatic migration
		SkipMigrationValidation: false,
		MaxMigrationVersion:     0, // No limit
	}

	// WAVE_MIGRATION_ENABLED - enable/disable migration system
	if env := os.Getenv("WAVE_MIGRATION_ENABLED"); env != "" {
		config.EnableMigrations = strings.ToLower(env) == "true"
	}

	// WAVE_AUTO_MIGRATE - enable/disable automatic migration on startup
	if env := os.Getenv("WAVE_AUTO_MIGRATE"); env != "" {
		config.AutoMigrate = strings.ToLower(env) == "true"
	}

	// WAVE_SKIP_MIGRATION_VALIDATION - skip checksum validation (dev only)
	if env := os.Getenv("WAVE_SKIP_MIGRATION_VALIDATION"); env != "" {
		config.SkipMigrationValidation = strings.ToLower(env) == "true"
	}

	// WAVE_MAX_MIGRATION_VERSION - limit migration version (for gradual rollout)
	if env := os.Getenv("WAVE_MAX_MIGRATION_VERSION"); env != "" {
		if version, err := strconv.Atoi(env); err == nil && version > 0 {
			config.MaxMigrationVersion = version
		}
	}

	return config
}

// ShouldUseMigrations determines if the migration system should be used
func (c *MigrationConfig) ShouldUseMigrations() bool {
	return c.EnableMigrations
}

// ShouldAutoMigrate determines if migrations should be applied automatically
func (c *MigrationConfig) ShouldAutoMigrate() bool {
	return c.EnableMigrations && c.AutoMigrate
}

// GetMaxVersion returns the maximum migration version to apply
func (c *MigrationConfig) GetMaxVersion() int {
	return c.MaxMigrationVersion
}

// Validate checks if the configuration is valid
func (c *MigrationConfig) Validate() error {
	if c.MaxMigrationVersion < 0 {
		return fmt.Errorf("max migration version cannot be negative")
	}

	return nil
}