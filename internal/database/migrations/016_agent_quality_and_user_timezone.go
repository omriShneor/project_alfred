package migrations

import "database/sql"

func init() {
	Register(Migration{
		Version: 16,
		Name:    "agent_quality_and_user_timezone",
		Up:      agentQualityAndUserTimezone,
	})
}

func agentQualityAndUserTimezone(db *sql.DB) error {
	if err := AddColumnIfNotExists(db, "users", "timezone", "TEXT DEFAULT 'UTC'"); err != nil {
		return err
	}

	if err := AddColumnIfNotExists(db, "calendar_events", "llm_confidence", "REAL DEFAULT 0"); err != nil {
		return err
	}
	if err := AddColumnIfNotExists(db, "calendar_events", "quality_flags", "TEXT DEFAULT '[]'"); err != nil {
		return err
	}

	if err := AddColumnIfNotExists(db, "reminders", "llm_confidence", "REAL DEFAULT 0"); err != nil {
		return err
	}
	if err := AddColumnIfNotExists(db, "reminders", "quality_flags", "TEXT DEFAULT '[]'"); err != nil {
		return err
	}

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS analysis_traces (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			channel_id INTEGER NOT NULL,
			source_type TEXT NOT NULL,
			trigger_message_id INTEGER,
			intent TEXT NOT NULL,
			router_confidence REAL DEFAULT 0,
			action TEXT,
			confidence REAL DEFAULT 0,
			reasoning TEXT,
			status TEXT NOT NULL,
			details_json TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY(channel_id) REFERENCES channels(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_analysis_traces_user_created ON analysis_traces(user_id, created_at DESC)`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_analysis_traces_channel_created ON analysis_traces(channel_id, created_at DESC)`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_analysis_traces_intent_created ON analysis_traces(intent, created_at DESC)`)

	return nil
}

