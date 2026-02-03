package database

import (
	"database/sql"
	"fmt"
	"time"
)

// TelegramSession represents a user's Telegram connection status
type TelegramSession struct {
	UserID      int64
	PhoneNumber string
	Connected   bool
	ConnectedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// GetTelegramSession retrieves the Telegram session info for a user
func (d *DB) GetTelegramSession(userID int64) (*TelegramSession, error) {
	var session TelegramSession
	var connectedAt sql.NullTime

	err := d.QueryRow(`
		SELECT user_id, phone_number, connected, connected_at, created_at, updated_at
		FROM telegram_sessions WHERE user_id = ?
	`, userID).Scan(
		&session.UserID,
		&session.PhoneNumber,
		&session.Connected,
		&connectedAt,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get telegram session: %w", err)
	}

	if connectedAt.Valid {
		session.ConnectedAt = &connectedAt.Time
	}

	return &session, nil
}

// SaveTelegramSession creates or updates a Telegram session record
func (d *DB) SaveTelegramSession(userID int64, phoneNumber string, connected bool) error {
	var connectedAt *time.Time
	if connected {
		now := time.Now()
		connectedAt = &now
	}

	_, err := d.Exec(`
		INSERT INTO telegram_sessions (user_id, phone_number, connected, connected_at, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id) DO UPDATE SET
			phone_number = COALESCE(excluded.phone_number, telegram_sessions.phone_number),
			connected = excluded.connected,
			connected_at = CASE WHEN excluded.connected = 1 AND telegram_sessions.connected = 0 THEN CURRENT_TIMESTAMP ELSE telegram_sessions.connected_at END,
			updated_at = CURRENT_TIMESTAMP
	`, userID, phoneNumber, connected, connectedAt)

	if err != nil {
		return fmt.Errorf("failed to save telegram session: %w", err)
	}

	return nil
}

// UpdateTelegramConnected updates the connection status for a user's Telegram session
func (d *DB) UpdateTelegramConnected(userID int64, connected bool) error {
	if connected {
		_, err := d.Exec(`
			UPDATE telegram_sessions SET
				connected = 1,
				connected_at = CURRENT_TIMESTAMP,
				updated_at = CURRENT_TIMESTAMP
			WHERE user_id = ?
		`, userID)
		return err
	}

	_, err := d.Exec(`
		UPDATE telegram_sessions SET
			connected = 0,
			updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ?
	`, userID)
	return err
}

// DeleteTelegramSession removes a user's Telegram session
func (d *DB) DeleteTelegramSession(userID int64) error {
	_, err := d.Exec(`DELETE FROM telegram_sessions WHERE user_id = ?`, userID)
	if err != nil {
		return fmt.Errorf("failed to delete telegram session: %w", err)
	}
	return nil
}

// ListUsersWithTelegramSession returns user IDs that have connected Telegram sessions
func (d *DB) ListUsersWithTelegramSession() ([]int64, error) {
	rows, err := d.Query(`SELECT user_id FROM telegram_sessions WHERE connected = 1`)
	if err != nil {
		return nil, fmt.Errorf("failed to list users with telegram session: %w", err)
	}
	defer rows.Close()

	var userIDs []int64
	for rows.Next() {
		var userID int64
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, userID)
	}

	return userIDs, rows.Err()
}
