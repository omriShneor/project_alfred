package migrations

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
)

// Migration represents a database migration
type Migration struct {
	Version     int
	Name        string
	Up          func(*sql.DB) error
}

// registry holds all registered migrations
var registry []Migration

// Register adds a migration to the registry
func Register(m Migration) {
	registry = append(registry, m)
}

// RunMigrations executes all pending migrations in order
func RunMigrations(db *sql.DB) error {
	// Create schema_migrations table if it doesn't exist
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	// Get applied migrations
	applied := make(map[int]bool)
	rows, err := db.Query("SELECT version FROM schema_migrations")
	if err != nil {
		return fmt.Errorf("failed to query schema_migrations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return fmt.Errorf("failed to scan version: %w", err)
		}
		applied[version] = true
	}

	// Sort migrations by version
	sort.Slice(registry, func(i, j int) bool {
		return registry[i].Version < registry[j].Version
	})

	// Run pending migrations
	for _, m := range registry {
		if applied[m.Version] {
			continue
		}

		log.Printf("Running migration %d: %s", m.Version, m.Name)

		if err := m.Up(db); err != nil {
			return fmt.Errorf("migration %d (%s) failed: %w", m.Version, m.Name, err)
		}

		// Record migration as applied
		_, err := db.Exec(
			"INSERT INTO schema_migrations (version, name) VALUES (?, ?)",
			m.Version, m.Name,
		)
		if err != nil {
			return fmt.Errorf("failed to record migration %d: %w", m.Version, err)
		}

		log.Printf("Migration %d completed successfully", m.Version)
	}

	return nil
}

// addColumnIfNotExists adds a column to a table if it doesn't already exist
func AddColumnIfNotExists(db *sql.DB, table, column, columnDef string) error {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return err
	}
	defer rows.Close()

	columnExists := false
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue interface{}
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return err
		}
		if name == column {
			columnExists = true
			break
		}
	}

	if !columnExists {
		_, err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, columnDef))
		if err != nil {
			return err
		}
	}
	return nil
}

// columnExists checks if a column exists in a table
func ColumnExists(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue interface{}
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, nil
}
