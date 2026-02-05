package migrations

import "database/sql"

func init() {
	Register(Migration{
		Version: 7,
		Name:    "channels_user_scoped_unique",
		Up:      channelsUserScopedUnique,
	})
}

func channelsUserScopedUnique(db *sql.DB) error {
	// Check if channels table exists
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='channels'`).Scan(&count); err != nil {
		return err
	}

	// Create the new table definition (user-scoped uniqueness)
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS channels_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			source_type TEXT DEFAULT 'whatsapp',
			type TEXT NOT NULL CHECK(type IN ('sender', 'group')),
			identifier TEXT NOT NULL,
			name TEXT NOT NULL,
			enabled BOOLEAN DEFAULT 1,
			total_message_count INTEGER NOT NULL DEFAULT 0,
			last_message_at DATETIME,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, source_type, identifier),
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`

	if count == 0 {
		_, err := db.Exec(createTableSQL)
		return err
	}

	// Create new table
	if _, err := db.Exec(createTableSQL); err != nil {
		return err
	}

	// Copy data over
	_, err := db.Exec(`
		INSERT INTO channels_new (
			id, user_id, source_type, type, identifier, name,
			enabled, total_message_count, last_message_at, created_at
		)
		SELECT
			id, user_id, COALESCE(source_type, 'whatsapp'), type, identifier, name,
			enabled, total_message_count, last_message_at, created_at
		FROM channels
	`)
	if err != nil {
		return err
	}

	// Drop old table and rename new
	if _, err := db.Exec(`DROP TABLE channels`); err != nil {
		return err
	}
	if _, err := db.Exec(`ALTER TABLE channels_new RENAME TO channels`); err != nil {
		return err
	}

	// Recreate indexes
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_channels_user ON channels(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_channels_source_type ON channels(source_type)`,
		`CREATE INDEX IF NOT EXISTS idx_channels_source_count ON channels(source_type, total_message_count DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_channels_type ON channels(type)`,
		`CREATE INDEX IF NOT EXISTS idx_channels_identifier ON channels(identifier)`,
	}
	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			// Ignore index errors to keep migration resilient
		}
	}

	return nil
}
