package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
}

func New(dbPath string) (*DB, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	d := &DB{db}

	if err := d.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return d, nil
}

func (d *DB) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS channels (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL CHECK(type IN ('sender', 'group')),
			identifier TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			calendar_id TEXT DEFAULT 'primary',
			enabled BOOLEAN DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Indexes
		`CREATE INDEX IF NOT EXISTS idx_channels_identifier ON channels(identifier)`,
		`CREATE INDEX IF NOT EXISTS idx_channels_type ON channels(type)`,

		// Message history table - stores last N messages per channel for context
		`CREATE TABLE IF NOT EXISTS message_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			channel_id INTEGER NOT NULL,
			sender_jid TEXT NOT NULL,
			sender_name TEXT,
			message_text TEXT NOT NULL,
			timestamp DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(channel_id) REFERENCES channels(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_message_history_channel ON message_history(channel_id)`,
		`CREATE INDEX IF NOT EXISTS idx_message_history_timestamp ON message_history(channel_id, timestamp DESC)`,

		// Calendar events table - stores detected events with Google Calendar reference
		`CREATE TABLE IF NOT EXISTS calendar_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			channel_id INTEGER NOT NULL,
			google_event_id TEXT,
			calendar_id TEXT NOT NULL,
			title TEXT NOT NULL,
			description TEXT,
			start_time DATETIME NOT NULL,
			end_time DATETIME,
			location TEXT,
			status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending', 'confirmed', 'synced', 'rejected', 'deleted')),
			action_type TEXT NOT NULL CHECK(action_type IN ('create', 'update', 'delete')),
			original_message_id INTEGER,
			llm_reasoning TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(channel_id) REFERENCES channels(id) ON DELETE CASCADE,
			FOREIGN KEY(original_message_id) REFERENCES message_history(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_calendar_events_channel ON calendar_events(channel_id)`,
		`CREATE INDEX IF NOT EXISTS idx_calendar_events_status ON calendar_events(status)`,
		`CREATE INDEX IF NOT EXISTS idx_calendar_events_google_id ON calendar_events(google_event_id)`,

		// Event attendees table - stores participants for calendar events
		`CREATE TABLE IF NOT EXISTS event_attendees (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id INTEGER NOT NULL,
			email TEXT NOT NULL,
			display_name TEXT,
			optional BOOLEAN DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(event_id) REFERENCES calendar_events(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_event_attendees_event ON event_attendees(event_id)`,
	}

	for _, migration := range migrations {
		if _, err := d.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}

func (d *DB) Close() error {
	return d.DB.Close()
}
