package migrations

import (
	"database/sql"
)

func init() {
	Register(Migration{
		Version: 6,
		Name:    "backfill_google_token_scopes",
		Up:      backfillGoogleTokenScopes,
	})
}

func backfillGoogleTokenScopes(db *sql.DB) error {
	// Backfill NULL or empty scopes with ProfileScopes for existing users
	// This ensures users who logged in before the scope persistence fix
	// don't lose their sessions
	_, err := db.Exec(`
		UPDATE google_tokens
		SET scopes = json_array(
			'https://www.googleapis.com/auth/userinfo.email',
			'https://www.googleapis.com/auth/userinfo.profile'
		)
		WHERE scopes IS NULL OR scopes = '' OR scopes = '[]'
	`)
	return err
}
