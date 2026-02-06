package migrations

import "database/sql"

func init() {
	Register(Migration{
		Version: 10,
		Name:    "source_backfill_markers",
		Up: func(db *sql.DB) error {
			if err := AddColumnIfNotExists(db, "channels", "initial_backfill_status", "TEXT"); err != nil {
				return err
			}
			if err := AddColumnIfNotExists(db, "channels", "initial_backfill_at", "DATETIME"); err != nil {
				return err
			}
			if err := AddColumnIfNotExists(db, "email_sources", "initial_backfill_status", "TEXT"); err != nil {
				return err
			}
			if err := AddColumnIfNotExists(db, "email_sources", "initial_backfill_at", "DATETIME"); err != nil {
				return err
			}
			return nil
		},
	})
}
