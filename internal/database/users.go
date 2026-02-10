package database

import (
	"database/sql"
	"fmt"
	"time"
)

// User represents a user in the system
type User struct {
	ID          int64
	GoogleID    string
	Email       string
	Name        *string
	AvatarURL   *string
	Timezone    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	LastLoginAt *time.Time
}

// GetAllUsers returns all users in the database
func (d *DB) GetAllUsers() ([]User, error) {
	rows, err := d.Query(`
		SELECT id, google_id, email, name, avatar_url, COALESCE(timezone, 'UTC'), created_at, updated_at, last_login_at
		FROM users
		ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(
			&u.ID,
			&u.GoogleID,
			&u.Email,
			&u.Name,
			&u.AvatarURL,
			&u.Timezone,
			&u.CreatedAt,
			&u.UpdatedAt,
			&u.LastLoginAt,
		); err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	return users, rows.Err()
}

// GetUserTimezone returns a user's preferred timezone.
func (d *DB) GetUserTimezone(userID int64) (string, error) {
	var tz sql.NullString
	err := d.QueryRow(`SELECT timezone FROM users WHERE id = ?`, userID).Scan(&tz)
	if err != nil {
		return "", fmt.Errorf("failed to get user timezone: %w", err)
	}
	if !tz.Valid || tz.String == "" {
		return "UTC", nil
	}
	return tz.String, nil
}

// UpdateUserTimezone updates a user's preferred timezone.
func (d *DB) UpdateUserTimezone(userID int64, timezone string) error {
	_, err := d.Exec(`
		UPDATE users
		SET timezone = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, timezone, userID)
	if err != nil {
		return fmt.Errorf("failed to update user timezone: %w", err)
	}
	return nil
}
