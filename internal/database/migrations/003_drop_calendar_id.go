package migrations

import (
	"database/sql"
)

func init() {
	Register(Migration{
		Version: 3,
		Name:    "drop_calendar_id_columns",
		Up:      dropCalendarIdColumns,
	})
}

func dropCalendarIdColumns(db *sql.DB) error {
	// Drop calendar_id from channels table (SQLite 3.35.0+)
	// First check if the column exists
	exists, err := ColumnExists(db, "channels", "calendar_id")
	if err != nil {
		return err
	}
	if exists {
		if _, err := db.Exec("ALTER TABLE channels DROP COLUMN calendar_id"); err != nil {
			return err
		}
	}

	// Drop calendar_id from email_sources table
	exists, err = ColumnExists(db, "email_sources", "calendar_id")
	if err != nil {
		return err
	}
	if exists {
		if _, err := db.Exec("ALTER TABLE email_sources DROP COLUMN calendar_id"); err != nil {
			return err
		}
	}

	return nil
}
