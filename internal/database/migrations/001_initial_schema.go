package migrations

import (
	"database/sql"
)

func init() {
	Register(Migration{
		Version: 1,
		Name:    "initial_schema",
		Up:      initialSchema,
	})
}

func initialSchema(db *sql.DB) error {
	statements := []string{
		// Channels table
		`CREATE TABLE IF NOT EXISTS channels (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL CHECK(type IN ('sender', 'group')),
			identifier TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			calendar_id TEXT DEFAULT 'primary',
			enabled BOOLEAN DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_channels_identifier ON channels(identifier)`,
		`CREATE INDEX IF NOT EXISTS idx_channels_type ON channels(type)`,

		// Message history table
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

		// Calendar events table
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

		// Event attendees table
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

		// User notification preferences
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
		`INSERT OR IGNORE INTO user_notification_preferences (id) VALUES (1)`,

		// Gmail settings
		`CREATE TABLE IF NOT EXISTS gmail_settings (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			enabled BOOLEAN DEFAULT 0,
			poll_interval_minutes INTEGER DEFAULT 5,
			last_poll_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`INSERT OR IGNORE INTO gmail_settings (id) VALUES (1)`,

		// Processed emails
		`CREATE TABLE IF NOT EXISTS processed_emails (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email_id TEXT UNIQUE NOT NULL,
			processed_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_processed_emails_email_id ON processed_emails(email_id)`,

		// Email sources
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

		// Feature settings
		`CREATE TABLE IF NOT EXISTS feature_settings (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			smart_calendar_enabled BOOLEAN DEFAULT 0,
			smart_calendar_setup_complete BOOLEAN DEFAULT 0,
			whatsapp_input_enabled BOOLEAN DEFAULT 0,
			email_input_enabled BOOLEAN DEFAULT 0,
			sms_input_enabled BOOLEAN DEFAULT 0,
			google_calendar_enabled BOOLEAN DEFAULT 0,
			outlook_calendar_enabled BOOLEAN DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`INSERT OR IGNORE INTO feature_settings (id) VALUES (1)`,

		// Gmail top contacts cache
		`CREATE TABLE IF NOT EXISTS gmail_top_contacts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL UNIQUE,
			name TEXT,
			email_count INTEGER DEFAULT 0,
			last_updated DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_gmail_top_contacts_email ON gmail_top_contacts(email)`,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}

	// Add columns that were added after initial schema
	columns := []struct {
		table, column, def string
	}{
		{"calendar_events", "source", "TEXT DEFAULT 'whatsapp'"},
		{"calendar_events", "email_source_id", "INTEGER"},
		{"feature_settings", "alfred_calendar_enabled", "BOOLEAN DEFAULT 1"},
		{"calendar_events", "calendar_type", "TEXT DEFAULT 'alfred'"},
		{"feature_settings", "onboarding_complete", "BOOLEAN DEFAULT 0"},
		{"channels", "source_type", "TEXT DEFAULT 'whatsapp'"},
		{"message_history", "source_type", "TEXT DEFAULT 'whatsapp'"},
		{"feature_settings", "telegram_input_enabled", "BOOLEAN DEFAULT 0"},
		{"message_history", "subject", "TEXT"},
		{"gmail_settings", "top_contacts_computed_at", "DATETIME"},
		{"channels", "total_message_count", "INTEGER NOT NULL DEFAULT 0"},
		{"channels", "last_message_at", "DATETIME"},
	}

	for _, col := range columns {
		if err := AddColumnIfNotExists(db, col.table, col.column, col.def); err != nil {
			return err
		}
	}

	// Additional indexes
	additionalIndexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_channels_source_type ON channels(source_type)`,
		`CREATE INDEX IF NOT EXISTS idx_message_history_source_type ON message_history(source_type)`,
		`CREATE INDEX IF NOT EXISTS idx_calendar_events_source_type ON calendar_events(source_type)`,
		`CREATE INDEX IF NOT EXISTS idx_channels_source_count ON channels(source_type, total_message_count DESC)`,
	}

	for _, idx := range additionalIndexes {
		if _, err := db.Exec(idx); err != nil {
			// Ignore errors for indexes that already exist
		}
	}

	// Migration: if smart_calendar_setup_complete is true, set onboarding_complete to true
	_, _ = db.Exec(`UPDATE feature_settings SET onboarding_complete = 1 WHERE smart_calendar_setup_complete = 1 AND onboarding_complete = 0`)

	return nil
}
