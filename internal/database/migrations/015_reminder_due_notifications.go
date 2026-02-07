package migrations

import "database/sql"

func init() {
	Register(Migration{
		Version: 15,
		Name:    "reminder_due_notification_tracking",
		Up:      reminderDueNotificationTracking,
	})
}

func reminderDueNotificationTracking(db *sql.DB) error {
	// Skip if reminders table does not exist.
	var tableCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='reminders'`).Scan(&tableCount); err != nil {
		return err
	}
	if tableCount == 0 {
		return nil
	}

	if err := AddColumnIfNotExists(db, "reminders", "due_notification_sent_at", "DATETIME"); err != nil {
		return err
	}

	_, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_reminders_due_notification_queue
		ON reminders(status, due_notification_sent_at, reminder_time, due_date)
	`)
	return err
}
