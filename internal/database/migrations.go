package database

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

// getMigrations returns the database migration scripts.
// Migration 1 is the initial schema, handled by InitSchema.
func getMigrations() map[int]string {
	return map[int]string{
		// Future migrations go here starting at version 2
		// Example:
		// 2: `ALTER TABLE entries ADD COLUMN some_field TEXT;`,
	}
}

// InitSchema creates the initial database schema (migration version 1).
func (db *DB) InitSchema() error {
	// Create entries table
	createEntriesTable := `
		CREATE TABLE IF NOT EXISTS entries (
			id TEXT PRIMARY KEY,
			entry_data JSON NOT NULL,
			posted_at DATETIME,
			fetched_at DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_entries_posted_at ON entries(posted_at);
		CREATE INDEX IF NOT EXISTS idx_entries_fetched_at ON entries(fetched_at);
	`

	if _, err := db.conn.Exec(createEntriesTable); err != nil {
		return fmt.Errorf("failed to create entries table: %w", err)
	}

	logrus.Debug("Database schema initialized")
	return nil
}

// RunMigrations applies any pending database migrations.
func (db *DB) RunMigrations() error {
	// Ensure schema_migrations table exists
	if _, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	currentVersion, err := db.GetMigrationVersion()
	if err != nil {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	migrations := getMigrations()

	// If this is a new database (currentVersion = 0) with no entries table, record initial schema version
	if currentVersion == 0 {
		var entriesTableExists int
		err := db.conn.QueryRow(
			"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='entries'",
		).Scan(&entriesTableExists)
		if err == nil && entriesTableExists > 0 {
			// Record version 1 as applied for existing databases
			logrus.Info("Existing database detected, marking initial schema as version 1")
			if _, err := db.conn.Exec("INSERT INTO schema_migrations (version) VALUES (1)"); err != nil {
				return fmt.Errorf("failed to record initial schema version: %w", err)
			}
			currentVersion = 1
		}
	}

	// Determine max migration version
	maxVersion := 1 // We start with migration 1 (initial schema)
	for version := range migrations {
		if version > maxVersion {
			maxVersion = version
		}
	}

	// Check if any migrations are needed
	if currentVersion >= maxVersion {
		return nil // No migrations needed
	}

	logrus.Infof("Checking for database migrations (current version: %d)", currentVersion)

	appliedCount := 0
	for version := currentVersion + 1; version <= maxVersion; version++ {
		if sql, exists := migrations[version]; exists {
			logrus.Infof("Applying migration %d", version)
			if err := db.ApplyMigration(version, sql); err != nil {
				return err
			}
			appliedCount++
		}
	}

	if appliedCount > 0 {
		logrus.Infof("Successfully applied %d migration(s)", appliedCount)
	}

	return nil
}

// ApplyMigration applies a single migration and records it.
func (db *DB) ApplyMigration(version int, sql string) error {
	// Execute the migration
	if _, err := db.conn.Exec(sql); err != nil {
		return fmt.Errorf("failed to apply migration %d: %w", version, err)
	}

	// Record the migration
	if _, err := db.conn.Exec("INSERT INTO schema_migrations (version) VALUES (?)", version); err != nil {
		return fmt.Errorf("failed to record migration %d: %w", version, err)
	}

	return nil
}

// GetMigrationVersion returns the current migration version.
func (db *DB) GetMigrationVersion() (int, error) {
	var version int
	err := db.conn.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	if err != nil {
		// If table doesn't exist, version is 0
		return 0, nil
	}
	return version, nil
}
