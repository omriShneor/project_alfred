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
