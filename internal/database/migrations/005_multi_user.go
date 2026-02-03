package migrations

import (
	"database/sql"
)

func init() {
	Register(Migration{
		Version: 5,
		Name:    "multi_user_support",
		Up:      multiUserSupport,
	})
}

func multiUserSupport(db *sql.DB) error {
	// Phase 1: Create new user-related tables

	// Users table - core user identity
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			google_id TEXT UNIQUE NOT NULL,
			email TEXT UNIQUE NOT NULL,
			name TEXT,
			avatar_url TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_login_at DATETIME
		)
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_users_google_id ON users(google_id)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`)
	if err != nil {
		return err
	}

	// User sessions - app authentication tokens
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			token_hash TEXT UNIQUE NOT NULL,
			expires_at DATETIME NOT NULL,
			device_info TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_user_sessions_user ON user_sessions(user_id)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_user_sessions_token ON user_sessions(token_hash)`)
	if err != nil {
		return err
	}

	// Google tokens - OAuth tokens for Gmail/Calendar per user
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS google_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER UNIQUE NOT NULL,
			access_token_encrypted BLOB NOT NULL,
			refresh_token_encrypted BLOB NOT NULL,
			token_type TEXT DEFAULT 'Bearer',
			expiry DATETIME,
			scopes TEXT,
			email TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// WhatsApp sessions - per-user session tracking (actual session data stored in separate files)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS whatsapp_sessions (
			user_id INTEGER PRIMARY KEY,
			phone_number TEXT,
			device_jid TEXT,
			connected BOOLEAN DEFAULT 0,
			connected_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// Telegram sessions - per-user session tracking (actual session data stored in separate files)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS telegram_sessions (
			user_id INTEGER PRIMARY KEY,
			phone_number TEXT,
			connected BOOLEAN DEFAULT 0,
			connected_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// Phase 2: Add user_id to existing data tables
	// Using DEFAULT 0 which is invalid - fresh start means no existing data to worry about

	dataTablesColumns := []struct {
		table, column, def string
	}{
		{"channels", "user_id", "INTEGER NOT NULL DEFAULT 0"},
		{"message_history", "user_id", "INTEGER NOT NULL DEFAULT 0"},
		{"calendar_events", "user_id", "INTEGER NOT NULL DEFAULT 0"},
		{"reminders", "user_id", "INTEGER NOT NULL DEFAULT 0"},
		{"email_sources", "user_id", "INTEGER NOT NULL DEFAULT 0"},
		{"processed_emails", "user_id", "INTEGER NOT NULL DEFAULT 0"},
		{"gmail_top_contacts", "user_id", "INTEGER NOT NULL DEFAULT 0"},
	}

	for _, col := range dataTablesColumns {
		if err := AddColumnIfNotExists(db, col.table, col.column, col.def); err != nil {
			return err
		}
	}

	// Create indexes for user_id on data tables
	userIndexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_channels_user ON channels(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_message_history_user ON message_history(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_calendar_events_user ON calendar_events(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_reminders_user ON reminders(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_email_sources_user ON email_sources(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_processed_emails_user ON processed_emails(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_gmail_top_contacts_user ON gmail_top_contacts(user_id)`,
	}

	for _, idx := range userIndexes {
		if _, err := db.Exec(idx); err != nil {
			// Ignore errors for indexes that already exist
		}
	}

	// Phase 3: Convert singleton settings tables to per-user
	// SQLite doesn't support removing constraints, so we recreate the tables

	// Convert user_notification_preferences
	if err := convertSettingsTable(db, "user_notification_preferences", `
		CREATE TABLE user_notification_preferences_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER UNIQUE NOT NULL,
			email_enabled BOOLEAN DEFAULT 0,
			email_address TEXT,
			push_enabled BOOLEAN DEFAULT 0,
			push_token TEXT,
			sms_enabled BOOLEAN DEFAULT 0,
			sms_phone TEXT,
			webhook_enabled BOOLEAN DEFAULT 0,
			webhook_url TEXT,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`); err != nil {
		return err
	}

	// Convert gmail_settings
	if err := convertSettingsTable(db, "gmail_settings", `
		CREATE TABLE gmail_settings_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER UNIQUE NOT NULL,
			enabled BOOLEAN DEFAULT 0,
			poll_interval_minutes INTEGER DEFAULT 5,
			last_poll_at DATETIME,
			top_contacts_computed_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`); err != nil {
		return err
	}

	// Convert gcal_settings
	if err := convertSettingsTable(db, "gcal_settings", `
		CREATE TABLE gcal_settings_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER UNIQUE NOT NULL,
			sync_enabled BOOLEAN DEFAULT 0,
			selected_calendar_id TEXT DEFAULT 'primary',
			selected_calendar_name TEXT DEFAULT 'Primary',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`); err != nil {
		return err
	}

	// Convert feature_settings
	if err := convertSettingsTable(db, "feature_settings", `
		CREATE TABLE feature_settings_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER UNIQUE NOT NULL,
			smart_calendar_enabled BOOLEAN DEFAULT 0,
			smart_calendar_setup_complete BOOLEAN DEFAULT 0,
			whatsapp_input_enabled BOOLEAN DEFAULT 0,
			telegram_input_enabled BOOLEAN DEFAULT 0,
			email_input_enabled BOOLEAN DEFAULT 0,
			sms_input_enabled BOOLEAN DEFAULT 0,
			alfred_calendar_enabled BOOLEAN DEFAULT 1,
			google_calendar_enabled BOOLEAN DEFAULT 0,
			outlook_calendar_enabled BOOLEAN DEFAULT 0,
			onboarding_complete BOOLEAN DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`); err != nil {
		return err
	}

	return nil
}

// convertSettingsTable handles the table recreation for settings tables
// SQLite doesn't support ALTER TABLE to remove constraints, so we must recreate
func convertSettingsTable(db *sql.DB, tableName, createNewSQL string) error {
	// Check if table exists
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, tableName).Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		// Table doesn't exist, create the new version directly
		newTableName := tableName + "_new"
		if _, err := db.Exec(createNewSQL); err != nil {
			return err
		}
		// Rename to final name
		_, err := db.Exec(`ALTER TABLE ` + newTableName + ` RENAME TO ` + tableName)
		return err
	}

	// Check if already converted (has user_id column)
	hasUserID, err := ColumnExists(db, tableName, "user_id")
	if err != nil {
		return err
	}
	if hasUserID {
		// Already converted
		return nil
	}

	// Create new table
	if _, err := db.Exec(createNewSQL); err != nil {
		return err
	}

	// Drop old table (fresh start - no data to migrate)
	if _, err := db.Exec(`DROP TABLE ` + tableName); err != nil {
		return err
	}

	// Rename new table
	newTableName := tableName + "_new"
	if _, err := db.Exec(`ALTER TABLE ` + newTableName + ` RENAME TO ` + tableName); err != nil {
		return err
	}

	return nil
}
