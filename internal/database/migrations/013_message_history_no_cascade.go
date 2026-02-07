package migrations

import "database/sql"

func init() {
	Register(Migration{
		Version: 13,
		Name:    "message_history_no_channel_delete_cascade",
		Up:      messageHistoryNoChannelDeleteCascade,
	})
}

func messageHistoryNoChannelDeleteCascade(db *sql.DB) error {
	// Detect whether message_history.channel_id still cascades deletes.
	rows, err := db.Query(`PRAGMA foreign_key_list(message_history)`)
	if err != nil {
		return err
	}

	needsRebuild := false
	for rows.Next() {
		var (
			id       int
			seq      int
			table    string
			from     string
			to       string
			onUpdate string
			onDelete string
			match    string
		)
		if err := rows.Scan(&id, &seq, &table, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			return err
		}
		if table == "channels" && from == "channel_id" && onDelete == "CASCADE" {
			needsRebuild = true
			break
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if !needsRebuild {
		return nil
	}

	if _, err := db.Exec(`PRAGMA foreign_keys=OFF`); err != nil {
		return err
	}
	defer func() {
		_, _ = db.Exec(`PRAGMA foreign_keys=ON`)
	}()

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS message_history_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			channel_id INTEGER NOT NULL,
			sender_jid TEXT NOT NULL,
			sender_name TEXT,
			message_text TEXT NOT NULL,
			timestamp DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			user_id INTEGER NOT NULL,
			source_type TEXT DEFAULT 'whatsapp',
			subject TEXT,
			FOREIGN KEY(channel_id) REFERENCES channels(id),
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		INSERT INTO message_history_new (
			id, channel_id, sender_jid, sender_name, message_text,
			timestamp, created_at, user_id, source_type, subject
		)
		SELECT
			id, channel_id, sender_jid, sender_name, message_text,
			timestamp, created_at, user_id, COALESCE(source_type, 'whatsapp'), subject
		FROM message_history
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`DROP TABLE message_history`); err != nil {
		return err
	}
	if _, err := db.Exec(`ALTER TABLE message_history_new RENAME TO message_history`); err != nil {
		return err
	}

	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_message_history_channel ON message_history(channel_id)`,
		`CREATE INDEX IF NOT EXISTS idx_message_history_timestamp ON message_history(channel_id, timestamp DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_message_history_user ON message_history(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_message_history_source_type ON message_history(source_type)`,
	}
	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			return err
		}
	}

	return nil
}
