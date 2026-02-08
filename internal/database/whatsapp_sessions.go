package database

import (
	"database/sql"
	"fmt"
	"time"
)

// WhatsAppSession represents a user's WhatsApp connection status
type WhatsAppSession struct {
	UserID      int64
	PhoneNumber string
	DeviceJID   string
	Connected   bool
	ConnectedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// GetWhatsAppSession retrieves the WhatsApp session info for a user
func (d *DB) GetWhatsAppSession(userID int64) (*WhatsAppSession, error) {
	var session WhatsAppSession
	var connectedAt sql.NullTime

	err := d.QueryRow(`
		SELECT user_id, phone_number, device_jid, connected, connected_at, created_at, updated_at
		FROM whatsapp_sessions WHERE user_id = ?
	`, userID).Scan(
		&session.UserID,
		&session.PhoneNumber,
		&session.DeviceJID,
		&session.Connected,
		&connectedAt,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get whatsapp session: %w", err)
	}

	if connectedAt.Valid {
		session.ConnectedAt = &connectedAt.Time
	}

	return &session, nil
}

// SaveWhatsAppSession creates or updates a WhatsApp session record
func (d *DB) SaveWhatsAppSession(userID int64, phoneNumber, deviceJID string, connected bool) error {
	var connectedAt *time.Time
	if connected {
		now := time.Now()
		connectedAt = &now
	}

	_, err := d.Exec(`
		INSERT INTO whatsapp_sessions (user_id, phone_number, device_jid, connected, connected_at, updated_at)
		VALUES (?, NULLIF(?, ''), NULLIF(?, ''), ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id) DO UPDATE SET
			phone_number = COALESCE(excluded.phone_number, whatsapp_sessions.phone_number),
			device_jid = COALESCE(excluded.device_jid, whatsapp_sessions.device_jid),
			connected = excluded.connected,
			connected_at = CASE WHEN excluded.connected = 1 AND whatsapp_sessions.connected = 0 THEN CURRENT_TIMESTAMP ELSE whatsapp_sessions.connected_at END,
			updated_at = CURRENT_TIMESTAMP
	`, userID, phoneNumber, deviceJID, connected, connectedAt)

	if err != nil {
		return fmt.Errorf("failed to save whatsapp session: %w", err)
	}

	return nil
}

// UpdateWhatsAppConnected updates the connection status for a user's WhatsApp session
func (d *DB) UpdateWhatsAppConnected(userID int64, connected bool) error {
	if connected {
		_, err := d.Exec(`
			UPDATE whatsapp_sessions SET
				connected = 1,
				connected_at = CURRENT_TIMESTAMP,
				updated_at = CURRENT_TIMESTAMP
			WHERE user_id = ?
		`, userID)
		return err
	}

	_, err := d.Exec(`
		UPDATE whatsapp_sessions SET
			connected = 0,
			updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ?
	`, userID)
	return err
}

// UpdateWhatsAppDeviceJID updates the device JID for a user's WhatsApp session
func (d *DB) UpdateWhatsAppDeviceJID(userID int64, deviceJID string) error {
	_, err := d.Exec(`
		INSERT INTO whatsapp_sessions (user_id, device_jid, connected, updated_at)
		VALUES (?, ?, 0, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id) DO UPDATE SET
			device_jid = excluded.device_jid,
			updated_at = CURRENT_TIMESTAMP
	`, userID, deviceJID)

	if err != nil {
		return fmt.Errorf("failed to update whatsapp device JID: %w", err)
	}

	return nil
}

// DeleteWhatsAppSession removes a user's WhatsApp session
func (d *DB) DeleteWhatsAppSession(userID int64) error {
	_, err := d.Exec(`DELETE FROM whatsapp_sessions WHERE user_id = ?`, userID)
	if err != nil {
		return fmt.Errorf("failed to delete whatsapp session: %w", err)
	}
	return nil
}

// ListUsersWithWhatsAppSession returns user IDs that have WhatsApp sessions
func (d *DB) ListUsersWithWhatsAppSession() ([]int64, error) {
	rows, err := d.Query(`SELECT user_id FROM whatsapp_sessions WHERE connected = 1`)
	if err != nil {
		return nil, fmt.Errorf("failed to list users with whatsapp session: %w", err)
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
