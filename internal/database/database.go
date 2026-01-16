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

		`CREATE TABLE IF NOT EXISTS pending_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			sender_jid TEXT NOT NULL,
			source_type TEXT NOT NULL,
			source_id INTEGER NOT NULL,
			event_json TEXT NOT NULL,
			status TEXT DEFAULT 'pending',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS event_source_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			sender_jid TEXT NOT NULL,
			source_type TEXT NOT NULL,
			source_id INTEGER NOT NULL,
			message_text TEXT NOT NULL,
			pending_event_id INTEGER REFERENCES pending_events(id),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS calendar_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pending_event_id INTEGER REFERENCES pending_events(id),
			google_event_id TEXT NOT NULL,
			title TEXT NOT NULL,
			event_date TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Indexes for common queries
		`CREATE INDEX IF NOT EXISTS idx_channels_identifier ON channels(identifier)`,
		`CREATE INDEX IF NOT EXISTS idx_channels_type ON channels(type)`,
		`CREATE INDEX IF NOT EXISTS idx_pending_events_status ON pending_events(status)`,
		`CREATE INDEX IF NOT EXISTS idx_pending_events_sender ON pending_events(sender_jid)`,
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
