package migrations

import (
	"database/sql"
	"fmt"
)

func init() {
	Register(Migration{
		Version: 14,
		Name:    "reminders_optional_due_date_and_location",
		Up:      remindersOptionalDueDateAndLocation,
	})
}

func remindersOptionalDueDateAndLocation(db *sql.DB) error {
	// Skip if reminders table does not exist.
	var tableCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='reminders'`).Scan(&tableCount); err != nil {
		return err
	}
	if tableCount == 0 {
		return nil
	}

	hasLocation, err := ColumnExists(db, "reminders", "location")
	if err != nil {
		return err
	}
	dueDateNullable, err := isColumnNullable(db, "reminders", "due_date")
	if err != nil {
		return err
	}

	// Already migrated.
	if hasLocation && dueDateNullable {
		return nil
	}

	if _, err := db.Exec(`PRAGMA foreign_keys=OFF`); err != nil {
		return err
	}
	defer func() {
		_, _ = db.Exec(`PRAGMA foreign_keys=ON`)
	}()

	// Drop rows that cannot satisfy the stronger FK constraints.
	if _, err := db.Exec(`
		DELETE FROM reminders
		WHERE user_id IS NULL
		   OR user_id = 0
		   OR user_id NOT IN (SELECT id FROM users)
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		DELETE FROM reminders
		WHERE channel_id NOT IN (SELECT id FROM channels)
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE reminders_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			channel_id INTEGER NOT NULL,
			google_event_id TEXT,
			calendar_id TEXT NOT NULL DEFAULT 'primary',
			title TEXT NOT NULL,
			description TEXT,
			location TEXT,
			due_date DATETIME,
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
			FOREIGN KEY(channel_id) REFERENCES channels(id) ON DELETE CASCADE,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`); err != nil {
		return err
	}

	locationExpr := "NULL"
	if hasLocation {
		locationExpr = "location"
	}

	copySQL := fmt.Sprintf(`
		INSERT INTO reminders_new (
			id, user_id, channel_id, google_event_id, calendar_id, title, description,
			location, due_date, reminder_time, priority, status, action_type,
			original_message_id, llm_reasoning, source, email_source_id, created_at, updated_at
		)
		SELECT
			id, user_id, channel_id, google_event_id, COALESCE(calendar_id, 'primary'), title, description,
			%s, due_date, reminder_time, COALESCE(priority, 'normal'), COALESCE(status, 'pending'), action_type,
			original_message_id, llm_reasoning, COALESCE(source, 'whatsapp'), email_source_id, created_at, updated_at
		FROM reminders
	`, locationExpr)

	if _, err := db.Exec(copySQL); err != nil {
		return err
	}

	if _, err := db.Exec(`DROP TABLE reminders`); err != nil {
		return err
	}
	if _, err := db.Exec(`ALTER TABLE reminders_new RENAME TO reminders`); err != nil {
		return err
	}

	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_reminders_status ON reminders(status)`,
		`CREATE INDEX IF NOT EXISTS idx_reminders_due_date ON reminders(due_date)`,
		`CREATE INDEX IF NOT EXISTS idx_reminders_channel ON reminders(channel_id)`,
		`CREATE INDEX IF NOT EXISTS idx_reminders_google_id ON reminders(google_event_id)`,
		`CREATE INDEX IF NOT EXISTS idx_reminders_user ON reminders(user_id)`,
	}
	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			return err
		}
	}

	return nil
}

func isColumnNullable(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid       int
			name      string
			ctype     string
			notNull   int
			defaultV  interface{}
			primaryID int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notNull, &defaultV, &primaryID); err != nil {
			return false, err
		}
		if name == column {
			return notNull == 0, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, err
	}

	return false, fmt.Errorf("column %s.%s not found", table, column)
}
