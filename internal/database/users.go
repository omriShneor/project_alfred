package database

import (
	"time"
)

// User represents a user in the system
type User struct {
	ID          int64
	GoogleID    string
	Email       string
	Name        *string
	AvatarURL   *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	LastLoginAt *time.Time
}

// GetAllUsers returns all users in the database
func (d *DB) GetAllUsers() ([]User, error) {
	rows, err := d.Query(`
		SELECT id, google_id, email, name, avatar_url, created_at, updated_at, last_login_at
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
