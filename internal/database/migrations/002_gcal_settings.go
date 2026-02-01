package migrations

import (
	"database/sql"
)

func init() {
	Register(Migration{
		Version: 2,
		Name:    "gcal_settings",
		Up:      gcalSettings,
	})
}

func gcalSettings(db *sql.DB) error {
	statements := []string{
		// Google Calendar sync settings - single row table
		`CREATE TABLE IF NOT EXISTS gcal_settings (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			sync_enabled BOOLEAN DEFAULT 0,
			selected_calendar_id TEXT DEFAULT 'primary',
			selected_calendar_name TEXT DEFAULT 'Primary',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		// Insert default row if not exists
		`INSERT OR IGNORE INTO gcal_settings (id) VALUES (1)`,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}

	return nil
}
