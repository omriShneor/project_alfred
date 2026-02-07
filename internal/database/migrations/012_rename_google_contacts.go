package migrations

import "database/sql"

func init() {
	Register(Migration{
		Version: 12,
		Name:    "rename_google_contacts",
		Up:      renameGoogleContacts,
	})
}

func renameGoogleContacts(db *sql.DB) error {
	// Rename table
	if _, err := db.Exec(`ALTER TABLE gmail_top_contacts RENAME TO google_contacts`); err != nil {
		return err
	}

	// Drop old indexes and create new ones with updated names
	indexes := []string{
		`DROP INDEX IF EXISTS idx_gmail_top_contacts_user`,
		`DROP INDEX IF EXISTS idx_gmail_top_contacts_email`,
		`CREATE INDEX IF NOT EXISTS idx_google_contacts_user ON google_contacts(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_google_contacts_email ON google_contacts(email)`,
	}
	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			return err
		}
	}

	return nil
}
