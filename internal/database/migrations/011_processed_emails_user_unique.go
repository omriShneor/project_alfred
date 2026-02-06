package migrations

import "database/sql"

func init() {
	Register(Migration{
		Version: 11,
		Name:    "processed_emails_user_unique",
		Up: func(db *sql.DB) error {
			var count int
			if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='processed_emails'`).Scan(&count); err != nil {
				return err
			}
			if count == 0 {
				return nil
			}

			if _, err := db.Exec(`
				CREATE TABLE IF NOT EXISTS processed_emails_new (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					user_id INTEGER NOT NULL DEFAULT 0,
					email_id TEXT NOT NULL,
					processed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					UNIQUE(user_id, email_id)
				)
			`); err != nil {
				return err
			}

			if _, err := db.Exec(`
				INSERT INTO processed_emails_new (id, user_id, email_id, processed_at)
				SELECT id, COALESCE(user_id, 0), email_id, processed_at
				FROM processed_emails
			`); err != nil {
				return err
			}

			if _, err := db.Exec(`DROP TABLE processed_emails`); err != nil {
				return err
			}

			if _, err := db.Exec(`ALTER TABLE processed_emails_new RENAME TO processed_emails`); err != nil {
				return err
			}

			if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_processed_emails_user ON processed_emails(user_id)`); err != nil {
				return err
			}
			if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_processed_emails_email_id ON processed_emails(email_id)`); err != nil {
				return err
			}

			return nil
		},
	})
}
