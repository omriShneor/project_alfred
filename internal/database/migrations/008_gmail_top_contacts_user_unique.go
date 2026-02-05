package migrations

import "database/sql"

func init() {
	Register(Migration{
		Version: 8,
		Name:    "gmail_top_contacts_user_unique",
		Up:      gmailTopContactsUserUnique,
	})
}

func gmailTopContactsUserUnique(db *sql.DB) error {
	if _, err := db.Exec(`PRAGMA foreign_keys=OFF`); err != nil {
		return err
	}
	defer func() {
		_, _ = db.Exec(`PRAGMA foreign_keys=ON`)
	}()

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='gmail_top_contacts'`).Scan(&count); err != nil {
		return err
	}

	createTableSQL := `
		CREATE TABLE IF NOT EXISTS gmail_top_contacts_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			email TEXT NOT NULL,
			name TEXT,
			email_count INTEGER DEFAULT 0,
			last_updated DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, email),
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
		INSERT INTO gmail_top_contacts_new (
			id, user_id, email, name, email_count, last_updated
		)
		SELECT
			id, COALESCE(user_id, 0), email, name, email_count, last_updated
		FROM gmail_top_contacts
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`DROP TABLE gmail_top_contacts`); err != nil {
		return err
	}
	if _, err := db.Exec(`ALTER TABLE gmail_top_contacts_new RENAME TO gmail_top_contacts`); err != nil {
		return err
	}

	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_gmail_top_contacts_user ON gmail_top_contacts(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_gmail_top_contacts_email ON gmail_top_contacts(email)`,
	}
	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			// Ignore index errors for resilience
		}
	}

	return nil
}
