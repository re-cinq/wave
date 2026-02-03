# Wave Database Migrations

Wave uses a comprehensive database migration system to manage schema evolution while maintaining data integrity and providing rollback capabilities.

## Overview

The migration system provides:
- **Version tracking** with checksums for integrity validation
- **Forward migrations** for schema upgrades
- **Rollback migrations** for safe downgrade scenarios
- **Feature flags** for gradual rollout control
- **CLI commands** for manual migration management
- **Automatic migration** during application startup

## Migration Architecture

### Components

1. **MigrationManager** - Core migration orchestration
2. **MigrationRunner** - CLI-friendly wrapper for migration operations
3. **Migration definitions** - Versioned schema changes with up/down SQL
4. **Migration configuration** - Environment-based feature flags
5. **CLI commands** - Manual migration management interface

### Schema Versioning

Migrations are numbered sequentially starting from 1:
- **Version 1**: Initial pipeline and step state tables
- **Version 2**: Ops commands tables (spec 016)
- **Version 3**: Performance metrics tables (spec 018 - part 1)
- **Version 4**: Progress tracking tables (spec 018 - part 2)
- **Version 5**: Artifact metadata extension (spec 018 - part 3)

Each migration includes:
- **Version number** (sequential integer)
- **Description** (human-readable summary)
- **Up SQL** (forward migration script)
- **Down SQL** (rollback migration script)

## Configuration

The migration system is controlled via environment variables:

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `WAVE_MIGRATION_ENABLED` | `true` | Enable/disable migration system |
| `WAVE_AUTO_MIGRATE` | `true` | Apply migrations automatically on startup |
| `WAVE_SKIP_MIGRATION_VALIDATION` | `false` | Skip checksum validation (dev only) |
| `WAVE_MAX_MIGRATION_VERSION` | `0` | Limit migration version for gradual rollout |

### Examples

```bash
# Disable migration system (fallback to old schema loading)
export WAVE_MIGRATION_ENABLED=false

# Disable automatic migration (require manual wave migrate up)
export WAVE_AUTO_MIGRATE=false

# Gradual rollout - only apply up to version 3
export WAVE_MAX_MIGRATION_VERSION=3

# Development mode - skip checksum validation
export WAVE_SKIP_MIGRATION_VALIDATION=true
```

## CLI Commands

### `wave migrate status`
Display current schema version and migration status.

```bash
$ wave migrate status
Current schema version: 3

Migration Status:
================
[x] 1: Create initial pipeline and step state tables (applied 2026-02-03 15:14:29)
[x] 2: Add ops commands tables for run tracking (spec 016) (applied 2026-02-03 15:14:29)
[x] 3: Add performance metrics tables (spec 018 - part 1) (applied 2026-02-03 15:14:29)
[ ] 4: Add progress tracking tables (spec 018 - part 2)
[ ] 5: Add artifact metadata extension (spec 018 - part 3)

2 pending migration(s)
```

### `wave migrate up [target_version]`
Apply pending migrations up to the target version.

```bash
# Apply all pending migrations
$ wave migrate up
Applying migration 4: Add progress tracking tables (spec 018 - part 2)
Applying migration 5: Add artifact metadata extension (spec 018 - part 3)
Migrations applied successfully

# Apply migrations up to version 3 only
$ wave migrate up 3
Applying migration 3: Add performance metrics tables (spec 018 - part 1)
Migrations applied successfully
```

### `wave migrate down <target_version>`
Rollback migrations down to the specified version.

```bash
$ wave migrate down 2
WARNING: Rolling back to version 2. This may result in data loss.
Continue? (y/N): y
Rolling back migration 3: Add performance metrics tables (spec 018 - part 1)
Successfully rolled back to version 2
```

⚠️ **Warning**: Rollback operations can result in data loss. Always backup your database before performing rollbacks.

### `wave migrate validate`
Verify migration integrity by checking applied migration checksums.

```bash
$ wave migrate validate
Migration integrity check passed
```

If a migration has been modified after being applied:
```bash
$ wave migrate validate
Error: migration validation failed: migration 3 checksum mismatch: expected abc123, got def456
```

## Database Files

The migration system uses the following database files:

- **`.wave/state.db`** - Main SQLite database
- **`.wave/state.db-wal`** - Write-ahead log (WAL mode)
- **`.wave/state.db-shm`** - Shared memory file (WAL mode)

The migration tracking table `schema_migrations` is created automatically:

```sql
CREATE TABLE schema_migrations (
    version INTEGER PRIMARY KEY,
    description TEXT NOT NULL,
    applied_at INTEGER NOT NULL,
    checksum TEXT NOT NULL
);
```

## Automatic Migration Behavior

### Fresh Database
When Wave starts with no existing database:
1. Creates `.wave/state.db`
2. Initializes migration tracking table
3. Applies all migrations up to `WAVE_MAX_MIGRATION_VERSION` (if configured)

### Existing Database (No Migration Tracking)
When Wave detects existing tables without migration tracking:
1. Creates migration tracking table
2. Marks all existing migrations as applied (backwards compatibility)
3. Applies any newer migrations

### Existing Database (With Migration Tracking)
When Wave starts with migration tracking:
1. Checks current version
2. Applies pending migrations up to `WAVE_MAX_MIGRATION_VERSION` (if configured)

## Development Workflow

### Adding New Migrations

1. **Create migration definition** in `internal/state/migration_definitions.go`:
   ```go
   {
       Version:     6,
       Description: "Add user authentication tables",
       Up: `
           CREATE TABLE users (
               id INTEGER PRIMARY KEY,
               username TEXT UNIQUE NOT NULL,
               created_at INTEGER NOT NULL
           );
           CREATE INDEX idx_users_username ON users(username);
       `,
       Down: `
           DROP INDEX idx_users_username;
           DROP TABLE users;
       `,
   }
   ```

2. **Write tests** for the new migration
3. **Test rollback** to ensure the down script works correctly
4. **Update documentation** if the migration affects APIs or behavior

### Testing Migrations

```bash
# Test migration system
go test ./internal/state -v -run Migration

# Test rollback functionality specifically
go test ./internal/state -v -run Rollback

# Test CLI commands
./wave migrate status
./wave migrate up 6
./wave migrate validate
./wave migrate down 5
```

## Gradual Rollout Strategy

For production deployments, use `WAVE_MAX_MIGRATION_VERSION` to control rollout:

### Phase 1: Deploy with Limited Migration
```bash
export WAVE_MAX_MIGRATION_VERSION=3
# Deploy application - only applies migrations 1-3
```

### Phase 2: Validate and Extend
```bash
export WAVE_MAX_MIGRATION_VERSION=5
# Restart application - applies migrations 4-5
```

### Phase 3: Remove Limit
```bash
unset WAVE_MAX_MIGRATION_VERSION
# Application now applies all available migrations
```

## Troubleshooting

### Migration Fails to Apply

**Check migration syntax:**
```bash
# Validate SQL manually
sqlite3 .wave/state.db < migration_sql_file.sql
```

**Check migration integrity:**
```bash
wave migrate validate
```

### Rollback Fails

**Check foreign key constraints:**
- Ensure rollback SQL drops dependent tables/indexes in correct order
- Verify foreign key constraints allow cascade deletion

**Manual recovery:**
1. Stop the application
2. Backup the database: `cp .wave/state.db .wave/state.db.backup`
3. Manually fix database schema
4. Update migration tracking: `DELETE FROM schema_migrations WHERE version > X`

### Migration Checksum Mismatch

This indicates a migration was modified after being applied:

**Option 1: Revert changes**
- Restore original migration definition
- Migration system will continue normally

**Option 2: Force acceptance (dangerous)**
```bash
export WAVE_SKIP_MIGRATION_VALIDATION=true
# Only use for development environments
```

## Best Practices

### Migration Design
- **Keep migrations small** - each migration should do one logical change
- **Test rollbacks** - always test that down migrations work correctly
- **Use transactions** - migrations are automatically wrapped in transactions
- **Avoid data migrations** - prefer schema-only changes when possible

### Production Safety
- **Backup before rollback** - always backup database before rollback operations
- **Test in staging** - validate migrations in staging environment first
- **Monitor rollout** - use gradual rollout with `WAVE_MAX_MIGRATION_VERSION`
- **Validate integrity** - run `wave migrate validate` after deployments

### Development Workflow
- **Run tests** - ensure migration tests pass before committing
- **Document changes** - update relevant documentation for user-facing changes
- **Consider backwards compatibility** - new migrations shouldn't break older versions

## Implementation Details

The migration system integrates with Wave's existing SQLite configuration:
- **WAL mode** for better concurrent access
- **Foreign keys enabled** for referential integrity
- **Busy timeout** (5 seconds) for lock contention handling
- **Single connection pool** optimized for SQLite's locking model

Migration transactions are atomic - if any part of a migration fails, all changes are rolled back automatically.