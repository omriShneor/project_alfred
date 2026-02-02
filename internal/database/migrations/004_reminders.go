package migrations

import (
	"database/sql"
)

func init() {
	Register(Migration{
		Version: 4,
		Name:    "create_reminders_table",
		Up:      createRemindersTable,
	})
}

func createRemindersTable(db *sql.DB) error {
	// Create reminders table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS reminders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			channel_id INTEGER NOT NULL,
			google_event_id TEXT,
			calendar_id TEXT NOT NULL DEFAULT 'primary',
			title TEXT NOT NULL,
			description TEXT,
			due_date DATETIME NOT NULL,
			reminder_time DATETIME,
			priority TEXT DEFAULT 'normal' CHECK(priority IN ('low', 'normal', 'high')),
			status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending', 'confirmed', 'synced', 'rejected', 'completed', 'dismissed')),
			action_type TEXT NOT NULL CHECK(action_type IN ('create', 'update', 'delete')),
			original_message_id INTEGER,
			llm_reasoning TEXT,
			source TEXT DEFAULT 'whatsapp',
			email_source_id INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(channel_id) REFERENCES channels(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// Create indexes for common queries
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_reminders_status ON reminders(status)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_reminders_due_date ON reminders(due_date)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_reminders_channel ON reminders(channel_id)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_reminders_google_id ON reminders(google_event_id)`)
	if err != nil {
		return err
	}

	return nil
}
