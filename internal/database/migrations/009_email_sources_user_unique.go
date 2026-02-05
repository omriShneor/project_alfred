package migrations

import "database/sql"

func init() {
	Register(Migration{
		Version: 9,
		Name:    "email_sources_user_unique",
		Up:      emailSourcesUserUnique,
	})
}

func emailSourcesUserUnique(db *sql.DB) error {
	if _, err := db.Exec(`PRAGMA foreign_keys=OFF`); err != nil {
		return err
	}
	defer func() {
		_, _ = db.Exec(`PRAGMA foreign_keys=ON`)
	}()

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='email_sources'`).Scan(&count); err != nil {
		return err
	}

	createTableSQL := `
		CREATE TABLE IF NOT EXISTS email_sources_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			type TEXT NOT NULL CHECK(type IN ('category', 'sender', 'domain')),
			identifier TEXT NOT NULL,
			name TEXT NOT NULL,
			enabled BOOLEAN DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, type, identifier),
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`

	if count == 0 {
		_, err := db.Exec(createTableSQL)
		return err
	}

	if _, err := db.Exec(createTableSQL); err != nil {
		return err
	}

	if _, err := db.Exec(`
		INSERT INTO email_sources_new (
			id, user_id, type, identifier, name, enabled, created_at, updated_at
		)
		SELECT
			id, COALESCE(user_id, 0), type, identifier, name, enabled, created_at, updated_at
		FROM email_sources
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`DROP TABLE email_sources`); err != nil {
		return err
	}
	if _, err := db.Exec(`ALTER TABLE email_sources_new RENAME TO email_sources`); err != nil {
		return err
	}

	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_email_sources_user ON email_sources(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_email_sources_type ON email_sources(type)`,
		`CREATE INDEX IF NOT EXISTS idx_email_sources_identifier ON email_sources(identifier)`,
	}
	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			// Ignore index errors for resilience
		}
	}

	return nil
}
