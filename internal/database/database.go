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

		// User notification preferences - stores per-method settings
		// Single row table (id=1) for all notification preferences
		`CREATE TABLE IF NOT EXISTS user_notification_preferences (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			email_enabled BOOLEAN DEFAULT 0,
			email_address TEXT,
			push_enabled BOOLEAN DEFAULT 0,
			push_token TEXT,
			sms_enabled BOOLEAN DEFAULT 0,
			sms_phone TEXT,
			webhook_enabled BOOLEAN DEFAULT 0,
			webhook_url TEXT,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		// Insert default row if not exists
		`INSERT OR IGNORE INTO user_notification_preferences (id) VALUES (1)`,

		// Gmail settings - single row table for Gmail integration settings
		`CREATE TABLE IF NOT EXISTS gmail_settings (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			enabled BOOLEAN DEFAULT 0,
			poll_interval_minutes INTEGER DEFAULT 5,
			last_poll_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		// Insert default row if not exists
		`INSERT OR IGNORE INTO gmail_settings (id) VALUES (1)`,

		// Processed emails - track which emails have been processed to avoid duplicates
		`CREATE TABLE IF NOT EXISTS processed_emails (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email_id TEXT UNIQUE NOT NULL,
			processed_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_processed_emails_email_id ON processed_emails(email_id)`,

		// Email sources - tracked email sources (similar to WhatsApp channels)
		`CREATE TABLE IF NOT EXISTS email_sources (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL CHECK(type IN ('category', 'sender', 'domain')),
			identifier TEXT NOT NULL,
			name TEXT NOT NULL,
			enabled BOOLEAN DEFAULT 1,
			calendar_id TEXT DEFAULT 'primary',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(type, identifier)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_email_sources_type ON email_sources(type)`,
		`CREATE INDEX IF NOT EXISTS idx_email_sources_identifier ON email_sources(identifier)`,

		// Feature settings - stores feature toggles and setup state
		`CREATE TABLE IF NOT EXISTS feature_settings (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			-- Smart Calendar feature
			smart_calendar_enabled BOOLEAN DEFAULT 0,
			smart_calendar_setup_complete BOOLEAN DEFAULT 0,
			-- Inputs (where to scan for events)
			whatsapp_input_enabled BOOLEAN DEFAULT 0,
			email_input_enabled BOOLEAN DEFAULT 0,
			sms_input_enabled BOOLEAN DEFAULT 0,
			-- Calendars (where to sync events)
			google_calendar_enabled BOOLEAN DEFAULT 0,
			outlook_calendar_enabled BOOLEAN DEFAULT 0,
			-- Timestamps
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		// Insert default row if not exists
		`INSERT OR IGNORE INTO feature_settings (id) VALUES (1)`,

	}

	// Run standard migrations
	for _, migration := range migrations {
		if _, err := d.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	// Add source column to calendar_events if not exists (SQLite-compatible approach)
	if err := d.addColumnIfNotExists("calendar_events", "source", "TEXT DEFAULT 'whatsapp'"); err != nil {
		return fmt.Errorf("failed to add source column: %w", err)
	}

	// Add email_source_id column to calendar_events for linking events to email sources
	if err := d.addColumnIfNotExists("calendar_events", "email_source_id", "INTEGER"); err != nil {
		return fmt.Errorf("failed to add email_source_id column: %w", err)
	}

	// Add alfred_calendar_enabled column to feature_settings (default ON - Alfred calendar always available)
	if err := d.addColumnIfNotExists("feature_settings", "alfred_calendar_enabled", "BOOLEAN DEFAULT 1"); err != nil {
		return fmt.Errorf("failed to add alfred_calendar_enabled column: %w", err)
	}

	// Add calendar_type column to calendar_events to distinguish storage location
	if err := d.addColumnIfNotExists("calendar_events", "calendar_type", "TEXT DEFAULT 'alfred'"); err != nil {
		return fmt.Errorf("failed to add calendar_type column: %w", err)
	}

	// Add onboarding_complete column to feature_settings (new simplified flow)
	if err := d.addColumnIfNotExists("feature_settings", "onboarding_complete", "BOOLEAN DEFAULT 0"); err != nil {
		return fmt.Errorf("failed to add onboarding_complete column: %w", err)
	}

	// Migration: if smart_calendar_setup_complete is true, set onboarding_complete to true
	_, _ = d.Exec(`UPDATE feature_settings SET onboarding_complete = 1 WHERE smart_calendar_setup_complete = 1 AND onboarding_complete = 0`)

	// Phase 0: Add source_type columns for multi-source support (WhatsApp, Telegram, Gmail)
	// Add source_type column to channels table
	if err := d.addColumnIfNotExists("channels", "source_type", "TEXT DEFAULT 'whatsapp'"); err != nil {
		return fmt.Errorf("failed to add source_type to channels: %w", err)
	}

	// Add source_type column to message_history table
	if err := d.addColumnIfNotExists("message_history", "source_type", "TEXT DEFAULT 'whatsapp'"); err != nil {
		return fmt.Errorf("failed to add source_type to message_history: %w", err)
	}

	// Add telegram_input_enabled column to feature_settings
	if err := d.addColumnIfNotExists("feature_settings", "telegram_input_enabled", "BOOLEAN DEFAULT 0"); err != nil {
		return fmt.Errorf("failed to add telegram_input_enabled column: %w", err)
	}

	// Add subject column to message_history for email subjects
	if err := d.addColumnIfNotExists("message_history", "subject", "TEXT"); err != nil {
		return fmt.Errorf("failed to add subject to message_history: %w", err)
	}

	// Create indexes for source_type columns
	_, _ = d.Exec(`CREATE INDEX IF NOT EXISTS idx_channels_source_type ON channels(source_type)`)
	_, _ = d.Exec(`CREATE INDEX IF NOT EXISTS idx_message_history_source_type ON message_history(source_type)`)
	_, _ = d.Exec(`CREATE INDEX IF NOT EXISTS idx_calendar_events_source_type ON calendar_events(source_type)`)

	// Update channels constraint to allow Telegram contact types
	// Note: SQLite doesn't support ALTER TABLE to modify CHECK constraints,
	// so types are validated at application level. Only 'sender' (contacts) supported now.

	// Gmail top contacts cache table for fast discovery
	_, _ = d.Exec(`CREATE TABLE IF NOT EXISTS gmail_top_contacts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT NOT NULL UNIQUE,
		name TEXT,
		email_count INTEGER DEFAULT 0,
		last_updated DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	_, _ = d.Exec(`CREATE INDEX IF NOT EXISTS idx_gmail_top_contacts_email ON gmail_top_contacts(email)`)

	// Add top_contacts_computed_at column to gmail_settings
	if err := d.addColumnIfNotExists("gmail_settings", "top_contacts_computed_at", "DATETIME"); err != nil {
		return fmt.Errorf("failed to add top_contacts_computed_at column: %w", err)
	}

	// Add total_message_count to channels for accurate contact ranking
	// (message_history is capped at 25, so we need actual counts from HistorySync)
	if err := d.addColumnIfNotExists("channels", "total_message_count", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return fmt.Errorf("failed to add total_message_count to channels: %w", err)
	}

	// Add last_message_at to channels for recency-based sorting
	if err := d.addColumnIfNotExists("channels", "last_message_at", "DATETIME"); err != nil {
		return fmt.Errorf("failed to add last_message_at to channels: %w", err)
	}

	// Index for fast top-N queries by message count
	_, _ = d.Exec(`CREATE INDEX IF NOT EXISTS idx_channels_source_count ON channels(source_type, total_message_count DESC)`)

	return nil
}

// addColumnIfNotExists adds a column to a table if it doesn't already exist
func (d *DB) addColumnIfNotExists(table, column, columnDef string) error {
	// Check if column exists by querying the table info
	rows, err := d.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return err
	}
	defer rows.Close()

	columnExists := false
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue interface{}
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return err
		}
		if name == column {
			columnExists = true
			break
		}
	}

	if !columnExists {
		_, err := d.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, columnDef))
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *DB) Close() error {
	return d.DB.Close()
}
