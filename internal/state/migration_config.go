package state

import (
	"fmt"

	"github.com/recinq/wave/internal/config"
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

// LoadMigrationConfigFromEnv loads migration configuration from environment variables.
//
// The four WAVE_MIGRATION_* env vars are read through internal/config so the
// process-wide env-reader contract stays in one place. Boolean fields accept
// "true", "1", or "yes" (case-insensitive); any other non-empty value is
// treated as false. WAVE_MAX_MIGRATION_VERSION must parse as a positive
// integer — malformed or non-positive values are silently ignored to preserve
// historical behaviour while the parse error is surfaced via
// config.MigrationEnv for callers that want to inspect it.
func LoadMigrationConfigFromEnv() *MigrationConfig {
	cfg := &MigrationConfig{
		EnableMigrations:        true, // Default to enabled for new systems
		AutoMigrate:             true, // Default to automatic migration
		SkipMigrationValidation: false,
		MaxMigrationVersion:     0, // No limit
	}

	envCfg := config.LoadMigrationEnv()
	if envCfg.Enabled != nil {
		cfg.EnableMigrations = *envCfg.Enabled
	}
	if envCfg.AutoMigrate != nil {
		cfg.AutoMigrate = *envCfg.AutoMigrate
	}
	if envCfg.SkipValidation != nil {
		cfg.SkipMigrationValidation = *envCfg.SkipValidation
	}
	if envCfg.MaxVersion != nil && *envCfg.MaxVersion > 0 {
		cfg.MaxMigrationVersion = *envCfg.MaxVersion
	}

	return cfg
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
